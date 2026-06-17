# PR #5431: fix(tm2): use separate mutex on ABCI queries client

URL: https://github.com/gnolang/gno/pull/5431
Author: Villaquiranm | Base: master | Files: 14 | +628 -35
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `b84ee8bdb` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5431 b84ee8bdb`
Prior reviews: [`1-02458c712/claude-opus-4-6_davd-gzl.md`](../1-02458c712/claude-opus-4-6_davd-gzl.md), [`2-f3c8706a/claude-opus-4-7_davd-gzl.md`](../2-f3c8706a/claude-opus-4-7_davd-gzl.md)

> **Verdict: APPROVE** — round 3 is the round-2 head with master merged on top of it (`b84ee8bdb`); no PR code changes, all round-2 findings still apply as non-blocking. Already approved by [@tbruyelle](https://github.com/gnolang/gno/pull/5431#pullrequestreview-3017038438) and [@notJoon](https://github.com/gnolang/gno/pull/5431#pullrequestreview-3221068194). Failing CI checks (`gno-checks/lint`, `main/test`) are master regressions from [PR #5669 (interrealm Phase 3)](https://github.com/gnolang/gno/pull/5669), not introduced by this PR.

## Summary

Round 1 fixed a data race on `app.checkState` by routing `Simulate()` and `handleQueryCustom()` through an immutable IAVL snapshot and an `atomic.Value` for the last block header. Round 2 closed [@omarsy](https://github.com/gnolang/gno/pull/5431#issuecomment-2802541619)'s follow-up `cacheNodes` race (`fatal error: concurrent map read and map write` in the GnoVM root store) by adding `txlog.SyncGoMap`, then patched a gas-overestimation regression in `immut.New` where unconditionally exposing `DepthEstimator` made flat (dbadapter) stores inherit IAVL-style depth multipliers during simulate. Round 3 (`b84ee8bdb`) is a clean `git merge master` on top of round 2 — `git diff $(git merge-base origin/master HEAD)..HEAD` shows the same 14 files as round 2 with byte-for-byte identical content. The two new failing checks on master (`gno-checks/lint` on `gno.land/r/test/sealviolation`, `main/test` on `TestTestdata/params_valset_rotation_throttle`) both trace to commits introduced by [PR #5669](https://github.com/gnolang/gno/pull/5669) (Phase 3 interrealm + valset rotation throttle), which landed in master after round 2.

## Glossary

- `cacheNodes` — `defaultStore` field caching `BlockNode`s keyed by `Location`; wrapped per-tx via `txlog.Wrap`.
- `SyncGoMap` — RWMutex-guarded `Map[K,V]` in `gnovm/pkg/gnolang/internal/txlog`.
- `immutStoreDE` — variant of `immutStore` that forwards `DepthEstimator` only when the parent has one.
- `DepthEstimator` — interface on tree-backed stores (IAVL) exposing expected node-traversal depth for gas accounting.

## Fix

No new code changes since round 2. The round-2 architecture stands: `cacheNodes` initialized as `*SyncGoMap` ([`store.go:210`](https://github.com/gnolang/gno/blob/b84ee8bdb/gnovm/pkg/gnolang/store.go#L210) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/store.go#L210)), `SyncGoMap.Iterate` snapshots eagerly under `RLock` to avoid `txLog.Iterate`'s nested `source.Get` deadlock ([`txlog.go:190-205`](https://github.com/gnolang/gno/blob/b84ee8bdb/gnovm/pkg/gnolang/internal/txlog/txlog.go#L190-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L190-L205)), `immut.New` returns `types.Store` and conditionally wraps in `immutStoreDE` only when the parent satisfies `DepthEstimator` ([`immut/store.go:28-34`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/store/immut/store.go#L28-L34) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/store/immut/store.go#L28-L34)), `Simulate()` loads an immutable snapshot at the atomic-header height ([`helpers.go:56-83`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/sdk/helpers.go#L56-L83) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/helpers.go#L56-L83)).

## Test verification (locally re-run on round-3 HEAD)

| Test | Result | Notes |
|---|---|---|
| `TestSimulateConcurrentWithCommit` | PASS (-race) | `tm2/pkg/sdk`, 1.0s |
| `TestGasUsedBetweenSimulateAndDeliverAfterCommit` | PASS (-race) | `tm2/pkg/sdk`, regression guard for the immutStore DepthEstimator fix |
| `TestGasUsedBetweenSimulateAndDeliver` | PASS (-race) | `tm2/pkg/sdk` |
| `TestNewDepthEstimatorForwarding` (3 subtests) | PASS (-race) | `tm2/pkg/store/immut`, 1.0s |
| `TestSimulateBurstDuringCommit` | PASS (-race, count=1) | `gno.land/pkg/gnoclient`, 37s; omarsy's reproducer |

## Critical (must fix)

None.

## Warnings (should fix) — carried from round 2, unchanged

- **[stale comment, repeat of round-1 and round-2]** [`baseapp.go:65-67`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/sdk/baseapp.go#L65-L67) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L65-L67), [`baseapp.go:270-273`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/sdk/baseapp.go#L270-L273) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L270-L273) — `lastBlockHeader` doc still claims "Updated atomically in setCheckState() and BeginBlock()" but `BeginBlock` does not touch it.
  <details><summary>details</summary>

  `grep -n lastBlockHeader baseapp.go` shows the only write site is [`baseapp.go:255`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/sdk/baseapp.go#L255) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L255) inside `setCheckState`, which runs from `Commit` and `InitChain`. The "and BeginBlock()" phrase in both the struct-field comment and the accessor comment is wrong and will mislead future readers debugging header staleness. Now flagged across three rounds. Fix: drop "and BeginBlock()" from both comments and replace with "(called from Commit and InitChain)".
  </details>

- **[integration tests lost gas-equality assertion]** [`gnokey_gasfee.txtar:15-21`](https://github.com/gnolang/gno/blob/b84ee8bdb/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar#L15-L21) · [↗](../../../../../.worktrees/gno-review-5431/gno.land/pkg/integration/testdata/gnokey_gasfee.txtar#L15-L21) — strict simulate-vs-deliver gas equality replaced with `\d+` regex.
  <details><summary>details</summary>

  Pre-PR the txtar pinned `GAS USED: 2815758` and used that exact figure on the broadcast line. The new test accepts any number and bumps `gas-wanted` from `2_816_000` to `5_005_000` (≈2×). The rationale (simulation overestimates delivery gas) is sound, but the regression net is now coarse: a future change that makes simulate overestimate by 10× would still pass. `TestGasUsedBetweenSimulateAndDeliverAfterCommit` ([`baseapp_test.go:697-737`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/sdk/baseapp_test.go#L697-L737) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp_test.go#L697-L737)) asserts equality but only for a trivial counter handler — it does not exercise VM / amino / IAVL paths. Fix: keep the regex loose if necessary, but add an upper bound (`stdout 'GAS USED:\s+[1-5]\d{6}'` or a follow-up txtar that asserts `simulate_gas <= 2 * deliver_gas` on a realistic call) so a >2× regression actually trips a test.
  </details>

- **[no unit tests for SyncGoMap]** [`txlog.go:154-205`](https://github.com/gnolang/gno/blob/b84ee8bdb/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) — new public type with zero direct test coverage.
  <details><summary>details</summary>

  `grep SyncGoMap` in the txlog package shows only definitions, no tests; the type is exercised only transitively via `TestSimulateBurstDuringCommit`. A direct unit test (2 goroutines hammering `Get`/`Set` + 1 doing `Iterate`, run under `-race`) would catch future regressions if someone "optimizes" `Iterate` back to a lazy implementation that holds `RLock` across `yield` and reintroduces the deadlock the comment warns about. The cost is ~30 lines next to the existing `Test_txLog` in `txlog_test.go`. Fix: add a `Test_SyncGoMap_concurrent` covering at minimum (a) concurrent `Get`/`Set`, (b) `Iterate` not deadlocking when called from a goroutine that also calls `Get`, (c) `Iterate` snapshot stability under writes.
  </details>

- **[pre-existing, not addressed]** [`helpers.go:79-80`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/sdk/helpers.go#L79-L80) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/helpers.go#L79-L80), [`baseapp.go:280-283`](https://github.com/gnolang/gno/blob/b84ee8bdb/tm2/pkg/sdk/baseapp.go#L280-L283) · [↗](../../../../../.worktrees/gno-review-5431/tm2/pkg/sdk/baseapp.go#L280-L283) — `app.consensusParams` and `app.minGasPrices` still read from the query goroutine without synchronization.
  <details><summary>details</summary>

  Carried from round 1's review. Both fields are written only during `InitChain`/startup and never again under the current code path, so this is latent rather than active. But the whole point of the round-1 work is to make the query goroutine safe in the Go memory model, and these two reads are the same shape as the `checkState` race that motivated the PR — minus a writer for now. If consensus params ever become updatable at `EndBlock` (Cosmos SDK has gone this direction), this turns into a live race. Fix: either store both in `atomic.Value` like `lastBlockHeader`, or add an explicit comment at each read site stating the "write only at InitChain" invariant.
  </details>

## Nits

- [`simulate_concurrent_test.go:22`](https://github.com/gnolang/gno/blob/b84ee8bdb/gno.land/pkg/gnoclient/simulate_concurrent_test.go#L22) · [↗](../../../../../.worktrees/gno-review-5431/gno.land/pkg/gnoclient/simulate_concurrent_test.go#L22), [`simulate_concurrent_test.go:143`](https://github.com/gnolang/gno/blob/b84ee8bdb/gno.land/pkg/gnoclient/simulate_concurrent_test.go#L143) · [↗](../../../../../.worktrees/gno-review-5431/gno.land/pkg/gnoclient/simulate_concurrent_test.go#L143) — stray `var _ = std.Coin{}` and unused `std` import still present. Leftover from omarsy's gist that pasted in a single file. Remove the `var` line and drop the `std` import.

- [`txlog.go:194-205`](https://github.com/gnolang/gno/blob/b84ee8bdb/gnovm/pkg/gnolang/internal/txlog/txlog.go#L194-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L194-L205) — `SyncGoMap.Iterate` clones the entire map eagerly under `RLock` at call time, not at range start; behavioral divergence from `GoMap.Iterate` (lazy) worth a one-line note. For `cacheNodes` with thousands of `BlockNode` entries this is a non-trivial allocation each call. Current callers ([`store.go:306`](https://github.com/gnolang/gno/blob/b84ee8bdb/gnovm/pkg/gnolang/store.go#L306) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/store.go#L306) in `CopyFromCachedStore`) only run in test/setup contexts so the cost is fine; the documentation should make that contract explicit so a future caller doesn't put `Iterate()` on a hot path.

- ADR section "Follow-on: GnoVM `cacheNodes` data race" calls out that `cacheObjects` and `cacheTypes` aren't affected because tx stores allocate fresh empty maps. Worth adding the `BeginTransaction` line numbers ([`store.go:243-244`](https://github.com/gnolang/gno/blob/b84ee8bdb/gnovm/pkg/gnolang/store.go#L243-L244) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/store.go#L243-L244)) as receipts so reviewers don't have to re-derive the proof.

## Missing Tests — unchanged from round 2

- **[unit gap]** [`txlog.go:154-205`](https://github.com/gnolang/gno/blob/b84ee8bdb/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) · [↗](../../../../../.worktrees/gno-review-5431/gnovm/pkg/gnolang/internal/txlog/txlog.go#L154-L205) — see warning above; no direct concurrency unit test for `SyncGoMap`.

- **[scenario gap]** Gas-overestimation upper bound — no test asserts `simulate_gas <= K * deliver_gas` for a non-trivial K on a realistic VM call. `TestGasUsedBetweenSimulateAndDeliverAfterCommit` only exercises a counter handler so it doesn't catch overestimation in the VM/amino path; the txtar `\d+` regex doesn't either.

## Suggestions

- **[CI status]** The two failing checks on round 3 are inherited from master, not introduced by this PR:
  - `gno-checks/lint`: `gno.land/r/test/sealviolation/z_seal_violation_filetest.gno:31:6-36` — `*sealviolation.foreignImpl does not implement seal.Sealed (missing method isSealed)`. File was added by `1bed667a3 feat(interrealm)!: Phase 3 (#5669)`, present only on master post-`f3c8706a`.
  - `main/test`: `TestTestdata/params_valset_rotation_throttle` (testdata/params_valset_rotation_throttle.txtar) — `rotation throttled: try again later`. Also from #5669 (valset rotation throttle). The throttle test fires non-deterministically based on block-time deltas; this looks like a flake on master that should be filed separately, not blocked on this PR.
  Repro to confirm both are master-side, not PR-side, runnable from a local clone of gnolang/gno:
  ```bash
  # from a local clone of gnolang/gno:
  git fetch origin master
  git checkout origin/master
  go build ./gno.land/r/test/sealviolation/... 2>&1 | tail -5   # same lint error
  go test -run 'TestTestdata/params_valset_rotation_throttle' -count=5 ./gno.land/pkg/integration/   # flakes the same way
  ```

- The PR title is still "fix(tm2): use separate mutex on ABCI queries client", but the diff also adds a `gnovm/internal/txlog` API and changes `immut.New`'s return type to fix a separate gas-estimation regression. For future readers tracing back from `git log` it would help to either bump the PR title to mention the broader fix, e.g. `fix(tm2,gnovm): immutable snapshot for simulate + sync cacheNodes`, or merge with a squashed message that lists all three subsystems touched.

## Questions for Author — carried, no new information since round 2

- [@tbruyelle's concern](https://github.com/gnolang/gno/pull/5431#discussion_r2039879614) that switching simulate to committed state diverges from Cosmos SDK (which reverted the same patch for compatibility): any follow-up with users whose workflows simulate tx N+1 after broadcasting tx N? Specifically, does any production tooling (gnokey, gnobro, gnodev) rely on simulate seeing mempool sequence updates?
- The two-step overestimation/underestimation arc (`c0b4ddd6` → `7401a2e9` → `c925b9ce`) suggests the gas model interacts subtly with which stores expose `DepthEstimator`. Is there a documented invariant somewhere (or worth adding to the ADR) about which store types should/shouldn't expose `DepthEstimator`, so the next person adding a store layer doesn't accidentally reintroduce the same regression?
