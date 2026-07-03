# PR [#5901](https://github.com/gnolang/gno/pull/5901): perf(vm): lazily clone the type-check cache per transaction

URL: https://github.com/gnolang/gno/pull/5901
Author: omarsy | Base: master | Files: 1 | +25 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: cf9cdf9f5 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5901 cf9cdf9f5`

**TL;DR:** Before every gno.land transaction the node made a full copy of its shared type-check cache, but only package-deploy and `run` transactions ever read that copy. This PR makes the copy happen the first time a transaction actually needs it, so the common transactions (plain contract calls and read-only queries) skip the copy entirely.

**Verdict: APPROVE** — correct and safe: the clone stays off the gas-metered path, per-transaction isolation is preserved, and the deferral only skips work no non-type-checking transaction used. One latent-fragility note on the nil-base sentinel, non-blocking.

## Summary
`MakeGnoTransactionStore` runs on every transaction through the `BeginTxHook` and used to eagerly `maps.Clone(vm.typeCheckCache)` (a map of ~stdlib-package count entries). Only [`AddPackage`](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L664) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L664) and [`Run`](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L1057) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L1057) call `getTypeCheckCache`; `Call` and the query-eval paths never do, so their clone was pure waste. The PR stores a `typeCheckCacheHolder{base}` in the context and clones lazily on first `get()`, so non-type-checking transactions do zero clones and deploy/run transactions still get exactly one clone shared across a transaction's messages.

## Glossary
- type-check: go/types validation of gno source (`TypeCheckMemPackage`), distinct from preprocessing.
- TypeCheckCache: map of already-type-checked imported packages passed via `TypeCheckOptions.Cache` to skip re-checking.
- transactionStore: the per-transaction Store wrapper carrying per-tx caches and the tx-scoped gas meter.

## Fix
Old code wrote `maps.Clone(vm.typeCheckCache)` directly into the context at [keeper.go:426](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L426) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L426). New code writes a `&typeCheckCacheHolder{base: vm.typeCheckCache}` instead, and [`getTypeCheckCache`](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L434-L436) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L434) calls `holder.get()`, which clones once and memoizes. The load-bearing constraint is that the consumer writes newly type-checked packages back into the passed cache ([gotypecheck.go:343](https://github.com/gnolang/gno/blob/cf9cdf9f5/gnovm/pkg/gnolang/gotypecheck.go#L343) · [↗](../../../../../.worktrees/gno-review-5901/gnovm/pkg/gnolang/gotypecheck.go#L343)), so each transaction must write to its own clone and never to the shared `base`; the holder preserves that.

Verified on cf9cdf9f5:
- Reverting the holder to the eager `maps.Clone` at the call site and adding a clone counter, non-type-checking transactions dropped from 1 clone to 0 while `AddPackage`/`Run` stayed at exactly 1 per transaction.
- Per-transaction isolation holds: a write into one holder's clone is invisible to `base` and to a sibling holder's clone (`gotypecheck.go:343` is the write that made isolation load-bearing). See [`holder_isolation_test.go`](../../../../../reviews/pr/5xxx/5901-lazy-typecheck-cache-clone/1-cf9cdf9f5/tests/holder_isolation_test.go).
- The clone stays off the metered path at its new location: in both `AddPackage` and `Run`, `TypeCheckMemPackage` (where `get()` fires) runs before `SetPreprocessAllocator`/`SetGasMeter` wire the gas meter to the store ([keeper.go:773-775](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L773-L775) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L773)), and the type-check path calls no `ConsumeGas`. No gas goldens changed.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- **[isolation is the invariant the PR rests on, and it's untested]** [keeper.go:417](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L417) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L417) — no test asserts a transaction's clone write stays out of the shared base or a sibling transaction's clone.
  <details><summary>details</summary>

  The existing suite exercises `AddPackage`/`Run` end to end, so it catches a clone that drops entries, but nothing asserts the isolation property directly: that the consumer's write into the clone ([gotypecheck.go:343](https://github.com/gnolang/gno/blob/cf9cdf9f5/gnovm/pkg/gnolang/gotypecheck.go#L343) · [↗](../../../../../.worktrees/gno-review-5901/gnovm/pkg/gnolang/gotypecheck.go#L343)) never reaches `base` or another holder. A future change to `get()` (say, returning `base` directly when the clone would be empty) would pass the whole suite while silently sharing state across transactions. The ready-to-add internal test in [`holder_isolation_test.go`](../../../../../reviews/pr/5xxx/5901-lazy-typecheck-cache-clone/1-cf9cdf9f5/tests/holder_isolation_test.go) pins lazy-clone, single-clone-per-holder, and both isolation directions; run: `cp` it into `gno.land/pkg/sdk/vm/` and `go test -run TestHolder ./gno.land/pkg/sdk/vm/`.
  </details>

## Suggestions
- [keeper.go:417-422](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L417-L422) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L417) — the `cloned == nil` sentinel silently degrades to re-clone-per-access if `base` is ever a nil map.
  <details><summary>details</summary>

  `maps.Clone(nil)` returns nil, so if `base` were a nil map, `get()` would set `cloned = nil` and re-clone on every call, quietly worse than the original eager code. Not reachable today: `NewVMKeeper` sets `typeCheckCache: gno.TypeCheckCache{}` ([keeper.go:129](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L129) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L129)) and `LoadStdlibCached` sets `maps.Clone(cachedInitTypeCheckCache)` over a `make(...)` map ([keeper.go:266](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L266) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L266)), both non-nil. Confirmed behaviorally: `maps.Clone` of a non-nil empty map is a non-nil empty map, so the sentinel latches; only a nil source breaks it. A separate `bool` flag instead of the nil sentinel would make `get()` robust to a nil base, or the `base` field could carry a doc note that it must be non-nil. Optional; no change needed for correctness today.
  </details>

## Open questions
None.
