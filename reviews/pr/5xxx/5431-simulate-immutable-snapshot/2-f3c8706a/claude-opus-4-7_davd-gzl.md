# PR #5431: fix(tm2): use separate mutex on ABCI queries client

URL: https://github.com/gnolang/gno/pull/5431
Author: Villaquiranm | Base: master | Files: 14 | +628 -35
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `f3c8706a` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5431 f3c8706a`
Prior review: [`1-02458c712/claude-opus-4-6_davd-gzl.md`](../1-02458c712/claude-opus-4-6_davd-gzl.md)

> **Verdict: APPROVE** — round-2 fixes close the `cacheNodes` race [@omarsy](https://github.com/gnolang/gno/pull/5431#issuecomment-2802541619) flagged and the gas-overestimation regression; remaining items are stale-comment and test-precision nits, none blocking.

## Summary

Round 1 fixed a data race on `app.checkState` by routing `Simulate()` and `handleQueryCustom()` through an immutable IAVL snapshot and an `atomic.Value` for the last block header (already approved by [@tbruyelle](https://github.com/gnolang/gno/pull/5431#pullrequestreview-3017038438) and [@notJoon](https://github.com/gnolang/gno/pull/5431#pullrequestreview-3221068194)). Stress-testing under `-race` then revealed a second race deeper in the stack: GnoVM's root `defaultStore.cacheNodes` is a plain `txlog.GoMap` whose `source` is shared by every concurrent `txLog` wrapper, so a simulate goroutine reading `BlockNode` cache misses raced against a `DeliverTx` commit writing them — a fatal `concurrent map read and map write` ([@omarsy reproducer](https://github.com/gnolang/gno/pull/5431#issuecomment-2802541619)). Round 2 adds `txlog.SyncGoMap` (RWMutex-guarded) and switches root `cacheNodes` to it, then patches a separate gas-overestimation regression in `immut.New` where unconditionally exposing `DepthEstimator` made flat (dbadapter) stores inherit IAVL-style depth multipliers during simulate.

## Glossary

- `cacheNodes` — `defaultStore` field caching `BlockNode`s keyed by `Location`; wrapped per-tx via `txlog.Wrap`.
- `SyncGoMap` — new RWMutex-guarded `Map[K,V]` in `gnovm/pkg/gnolang/internal/txlog`.
- `immutStoreDE` — variant of `immutStore` that forwards `DepthEstimator` only when the parent has one.
- `DepthEstimator` — interface on tree-backed stores (IAVL) exposing expected node-traversal depth for gas accounting.
- `FixedGetReadDepth100` — VM-side `GasContext` override that pins the depth multiplier; if a flat store mistakenly exposes `DepthEstimator`, this kicks in and inflates simulate gas.

## Fix

Round 2 lands two separate corrections on top of round 1. First, `cacheNodes` in the root `defaultStore` becomes a `*SyncGoMap` instead of a bare `GoMap` ([store.go:201](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/store.go#L201) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/store.go#L201)); the `txLog{source,dirty}` wrapper that each transaction installs is unchanged, but every fall-through `Get`/`Set`/`Iterate` on the shared source now goes through `sync.RWMutex` ([txlog.go:154-205](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205)). `Iterate()` deliberately snapshots eagerly under `RLock` and releases the lock before yielding — the comment at [txlog.go:190-193](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/internal/txlog/txlog.go#L190-L193) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L190-L193) calls out that otherwise `txLog.Iterate`'s nested `source.Get` self-deadlocks. Second, `immut.New` now returns the interface `types.Store` and conditionally wraps in `immutStoreDE` only when the parent satisfies `DepthEstimator` ([immut/store.go:28-34](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/store/immut/store.go#L28-L34) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/store/immut/store.go#L28-L34)); the override of `CacheWrap` at [immut/store.go:76-78](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/store/immut/store.go#L76-L78) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/store/immut/store.go#L76-L78) is load-bearing — without it the embedded `immutStore.CacheWrap` would pass the inner type to `cache.New` and drop the depth methods.

## Benchmarks / Numbers

| Test | Master | Round 1 (no SyncGoMap) | Round 2 | Notes |
|---|---|---|---|---|
| `TestSimulateBurstDuringCommit` × 3 | 10/10 PASS | 2/10 FAIL (race) | 3/3 PASS | omarsy reproducer; PR-included |
| `TestSimulateConcurrentWithCommit` | n/a | added | PASS | unit-level, narrower scope |
| `gnokey_gasfee.txtar` addpkg gas | `2_815_758` (strict) | n/a | `\d+` (regex) | simulate now overestimates ~2× |
| `gnokey_gasfee.txtar` call gas | `1_271_083` (strict) | n/a | `\d+` (regex) | same loosening |

## Critical (must fix)

None.

## Warnings (should fix)

- **[stale comment, repeat of prior review]** [`baseapp.go:66-67`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/baseapp.go#L66-L67) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L66-L67), [`baseapp.go:271`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/baseapp.go#L271) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L271) — `lastBlockHeader` doc still claims "Updated atomically in setCheckState() and BeginBlock()" but `BeginBlock` does not touch it.
  <details><summary>details</summary>

  Round 1's review flagged this. `grep -n lastBlockHeader baseapp.go` shows the only write site is [`baseapp.go:255`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/baseapp.go#L255) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L255) inside `setCheckState`, which runs from `Commit` and `InitChain`. The "and BeginBlock()" phrase in both the struct-field comment and the accessor comment is wrong and will mislead future readers debugging header staleness. Fix: drop "and BeginBlock()" from both comments and replace with "(called from Commit and InitChain)".
  </details>

- **[integration tests lost gas-equality assertion]** [`gnokey_gasfee.txtar:15-21`](https://github.com/gnolang/gno/blob/f3c8706a/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar#L15-L21) · [↗](../../../../../.worktrees/gno-review-5431/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar#L15-L21) — strict simulate-vs-deliver gas equality replaced with `\d+` regex.
  <details><summary>details</summary>

  Pre-PR the txtar pinned `GAS USED: 2815758` and used that exact figure on the broadcast line, documenting the "simulate then broadcast at the same gas" workflow. The new test accepts any number and bumps `gas-wanted` from `2_816_000` to `5_005_000` (≈2×). The rationale ("simulation overestimates delivery gas") is sound, but the regression net is now coarse: a future change that makes simulate overestimate by 10× would still pass. `TestGasUsedBetweenSimulateAndDeliverAfterCommit` ([`baseapp_test.go:697-734`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/baseapp_test.go#L697-L734) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp_test.go#L697-L734)) asserts equality but only for a trivial counter handler — it does not exercise VM / amino / IAVL paths. Fix: keep the regex loose if necessary, but add an upper bound (`stdout 'GAS USED:\s+[1-5]\d{6}'` or a follow-up txtar that asserts `simulate_gas <= 2 * deliver_gas` on a realistic call) so a >2× regression actually trips a test. Also worth flagging in the ADR which scenarios overestimate and by how much.
  </details>

- **[no unit tests for SyncGoMap]** [`txlog.go:154-205`](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) — new public type with zero direct test coverage.
  <details><summary>details</summary>

  `grep SyncGoMap` in the txlog package shows only definitions, no tests; the type is exercised only transitively via `TestSimulateBurstDuringCommit`. A direct unit test (2 goroutines hammering `Get`/`Set` + 1 doing `Iterate`, run under `-race`) would catch future regressions if someone "optimizes" `Iterate` back to a lazy implementation that holds `RLock` across `yield` and reintroduces the deadlock the comment warns about. The cost is ~30 lines next to the existing `Test_txLog` in `txlog_test.go`. Fix: add a `Test_SyncGoMap_concurrent` covering at minimum (a) concurrent `Get`/`Set`, (b) `Iterate` not deadlocking when called from a goroutine that also calls `Get`, (c) `Iterate` snapshot stability under writes.
  </details>

- **[pre-existing, not addressed]** [`helpers.go:79`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/helpers.go#L79) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/helpers.go#L79), [`baseapp.go:535`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/baseapp.go#L535) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L535) — `app.consensusParams` and `app.minGasPrices` still read from the query goroutine without synchronization.
  <details><summary>details</summary>

  Carried over from round 1's review. Both fields are written only during `InitChain`/startup and never again under the current code path, so this is latent rather than active. But the whole point of the round-1 work is to make the query goroutine safe in the Go memory model, and these two reads are the same shape as the `checkState` race that motivated the PR — minus a writer for now. If consensus params ever become updatable at `EndBlock` (Cosmos SDK has gone this direction), this turns into a live race. Fix: either store both in `atomic.Value` like `lastBlockHeader`, or add an explicit comment at each read site stating the "write only at InitChain" invariant.
  </details>

## Nits

- [`simulate_concurrent_test.go:143`](https://github.com/gnolang/gno/blob/f3c8706a/gno.land/pkg/gnoclient/simulate_concurrent_test.go#L143) · [↗](../../../../../.worktrees/gno-review-5431/gno.land/pkg/gnoclient/simulate_concurrent_test.go#L143) — stray `var _ = std.Coin{}` and unused `std` import; `grep "std\." simulate_concurrent_test.go` shows the type is referenced nowhere else. Leftover from omarsy's gist that pasted in a single file. Remove the `var` line and drop the `std` import.

- [`txlog.go:194-205`](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/internal/txlog/txlog.go#L194-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L194-L205) — `SyncGoMap.Iterate` clones the entire map eagerly under `RLock` at call time, not at range start; behavioral divergence from `GoMap.Iterate` (lazy) worth a one-line note. For `cacheNodes` with thousands of `BlockNode` entries this is a non-trivial allocation each call. Current callers ([`store.go:295`](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/store.go#L295) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/store.go#L295) in `CopyFromCachedStore`, [`store.go:1177`](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/store.go#L1177) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/store.go#L1177) in `Print`) only run in test/debug contexts so the cost is fine; the documentation should make that contract explicit so a future caller doesn't put `Iterate()` on a hot path.

- ADR section "Follow-on: GnoVM `cacheNodes` data race" calls out that `cacheObjects` and `cacheTypes` aren't affected because tx stores allocate fresh empty maps. Worth adding the `BeginTransaction` line numbers ([`store.go:233-235`](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/store.go#L233-L235) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/store.go#L233-L235)) as receipts so reviewers don't have to re-derive the proof.

## Missing Tests

- **[unit gap]** [`txlog.go:154-205`](https://github.com/gnolang/gno/blob/f3c8706a/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) — see warning above; no direct concurrency unit test for `SyncGoMap`.

- **[scenario gap]** Gas-overestimation upper bound — no test asserts `simulate_gas <= K * deliver_gas` for a non-trivial K on a realistic VM call. `TestGasUsedBetweenSimulateAndDeliverAfterCommit` only exercises a counter handler so it doesn't catch overestimation in the VM/amino path; the txtar `\d+` regex doesn't either. See warning above.

## Suggestions

- The PR title is "fix(tm2): use separate mutex on ABCI queries client", but round 2 also adds a `gnovm/internal/txlog` API and changes `immut.New`'s return type to fix a separate gas-estimation regression. The scope creep is acknowledged in the ADR's follow-on section, which is good. For future readers tracing back from `git log` it would help to either (a) bump the PR title to mention the broader fix, e.g. `fix(tm2,gnovm): immutable snapshot for simulate + sync cacheNodes`, or (b) extract the `immut.New` `DepthEstimator` change into its own PR (the regression test in [`baseapp_test.go:697-734`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/baseapp_test.go#L697-L734) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp_test.go#L697-L734) would move with it).

- [`baseapp.go:1029-1033`](https://github.com/gnolang/gno/blob/f3c8706a/tm2/pkg/sdk/baseapp.go#L1029-L1033) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L1029-L1033) — the `headerSnapshot` wrapper struct exists because `atomic.Value` requires a consistent concrete type and `abci.Header` is an interface. The existing comment captures the "why" succinctly; the round-1 reviewer was already happy with it. Leaving this here as a marker that the pattern is correct and shouldn't be "simplified" away.

## Questions for Author

- Round 1 surfaced [@tbruyelle's concern](https://github.com/gnolang/gno/pull/5431#discussion_r2039879614) that switching simulate to committed state diverges from Cosmos SDK (which reverted the same patch for compatibility). Has there been any follow-up with users whose workflows simulate tx N+1 after broadcasting tx N? Specifically, does any production tooling (gnokey, gnobro, gnodev) rely on simulate seeing mempool sequence updates?
- The two-step overestimation/underestimation arc (`c0b4ddd6` fixed a cacheNodes race introducing overestimation, `7401a2e9` forwarded DepthEstimator to fix underestimation, `c925b9ce` then "fixed overestimation in the immut store") suggests the gas model interacts subtly with which stores expose `DepthEstimator`. Is there a documented invariant somewhere (or worth adding to the ADR) about which store types should/shouldn't expose `DepthEstimator`, so the next person adding a store layer doesn't accidentally reintroduce the same regression?
