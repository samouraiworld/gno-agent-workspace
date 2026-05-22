# PR #5431: fix(tm2): use separate mutex on ABCI queries client

**URL:** https://github.com/gnolang/gno/pull/5431
**Author:** Villaquiranm | **Base:** master | **Files:** 6 | **+237 -10**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a data race and a zero-cost DoS vector in the `.app/simulate` RPC endpoint. The core issue: `Simulate()` read from `app.checkState`, a live pointer replaced during `Commit()`. With the query connection using a separate mutex from consensus/mempool, there was no synchronization protecting this access — a textbook data race.

The fix has three parts:

1. **Separate query mutex** (`tm2/pkg/bft/proxy/client.go`, `tm2/pkg/bft/appconn/multi_app_conn.go`): The `ClientCreator` interface gains a `NewReadOnlyABCIClient()` method that returns a client backed by an independent `queryMtx`. The query connection in `multi_app_conn.go` now uses this instead of sharing the consensus/mempool mutex.

2. **Immutable snapshot for `Simulate()`** (`tm2/pkg/sdk/helpers.go`): Instead of reading from `app.checkState`, `Simulate()` now loads an immutable IAVL snapshot via `MultiImmutableCacheWrapWithVersion(height)` — the same copy-on-write pattern already used by `handleQueryCustom` and `handleQueryStore`. Falls back to the old `getContextForTx` path pre-first-commit (height < 1).

3. **Atomic header storage** (`tm2/pkg/sdk/baseapp.go`): An `atomic.Value` field `lastBlockHeader` stores the latest block header, updated in `setCheckState()`. This is read lock-free by `Simulate()` and `handleQueryCustom` (which was also fixed — it previously read `app.checkState.ctx.BlockHeader()` without the consensus mutex).

The PR also includes an ADR document explaining the decision and a concurrent race test (`TestSimulateConcurrentWithCommit`).

## Test Results
- **Existing tests:** PASS — all 22 tests in `tm2/pkg/sdk` pass with `-race` enabled.
- **Edge-case tests:** The new `TestSimulateConcurrentWithCommit` spawns 4 goroutines continuously calling `Simulate()` while the main goroutine runs 9 block cycles. Passes cleanly with the race detector.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `baseapp.go:67-68` — **Inaccurate comment.** The comment says `lastBlockHeader` is "Updated atomically in setCheckState() and BeginBlock()." However, `BeginBlock()` (line 541) does NOT update `lastBlockHeader`. It only updates `deliverState`. The `lastBlockHeader` is only updated in `setCheckState()` (called from `Commit` and `InitChain`). The comment should read "Updated atomically in setCheckState() (called from Commit and InitChain)."

- [ ] `baseapp.go:274` — **Comment also says "BeginBlock".** Same issue: `getLastBlockHeader` docs say "updated in setCheckState() and BeginBlock()" but BeginBlock doesn't touch it.

- [ ] `helpers.go:47` — **`app.consensusParams` read without synchronization.** `Simulate()` reads `app.consensusParams` on the query goroutine (via `.WithConsensusParams(app.consensusParams)`), while it could theoretically be written from the consensus goroutine. Currently this is practically safe because `consensusParams` is only written during `InitChain` (before queries are accepted) and at startup in `initFromMainStore`. However, this is not formally race-free under the Go memory model. If a future change allows consensus param updates during `EndBlock` or `Commit`, this would become an active data race. Consider storing `consensusParams` in an `atomic.Value` similar to `lastBlockHeader`, or documenting this assumption explicitly. Note: this is a pre-existing issue, not introduced by this PR.

- [ ] `baseapp.go:520` — **`handleQueryCustom` also reads `app.minGasPrices` without synchronization.** Same concern as `consensusParams` — it's set at init and never changed, but the access pattern is not formally safe. Pre-existing issue.

## Nits

- [ ] `proxy/client.go:42-43` — The comment "Query calls never contend with consensus or mempool operations" is slightly misleading — query calls still call the same `app` methods, just under a different mutex. The safety comes from the immutable snapshot pattern, not the mutex separation alone.

- [ ] `multi_app_conn.go:20-27` — The `ClientCreator` interface is duplicated verbatim in both `appconn` and `proxy` packages. This is pre-existing, but adding `NewReadOnlyABCIClient` to both duplicated interfaces increases the maintenance burden. Consider whether one package should import the other's interface.

## Missing Tests

- [ ] No test for `NewReadOnlyABCIClient` directly — it's only tested indirectly through the integration test. A unit test verifying that `NewReadOnlyABCIClient` returns a client with a different mutex than `NewABCIClient` would be valuable. `proxy/client.go:44`.
- [ ] No test for `Simulate` behavior when `MultiImmutableCacheWrapWithVersion` returns an error (e.g., pruned height). The error path at `helpers.go:38-42` is untested.
- [ ] No test verifying that `handleQueryCustom` now correctly reads the atomic header instead of the old `checkState.ctx.BlockHeader()`. `baseapp.go:520`.

## Suggestions

- The `headerSnapshot` wrapper struct (line 998) is needed because `atomic.Value` requires a consistent concrete type. This is correct but worth a brief inline comment explaining why the header itself can't be stored directly (it's an interface `abci.Header`). The existing comment is good.
- The ADR document is well-written and provides excellent context. Consider adding it to a docs index if one exists.
- The test `TestSimulateConcurrentWithCommit` could be strengthened by asserting that at least some `Simulate` calls succeed (currently all results are discarded with `_ = result`). Even a simple counter of successes would increase confidence.

## Questions for Author

- The PR description mentions this is related to a HackenProof security report (NEWTENDG-170). Is there a coordinated disclosure timeline, and should this PR be fast-tracked?
- `Simulate()` now reads committed state rather than `checkState`. This means it won't see pending mempool changes (e.g., sequence number updates from `CheckTx`). The ADR acknowledges this matches Cosmos SDK behavior — has this been validated against existing users/integrations that might depend on seeing pending state?

## Verdict

APPROVE — The fix correctly eliminates the data race by switching to immutable IAVL snapshots and atomic header access. The approach is sound and consistent with existing query path patterns. All tests pass with the race detector. The warnings about inaccurate comments should be fixed but are non-blocking.
