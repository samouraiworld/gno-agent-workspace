# PR [#5902](https://github.com/gnolang/gno/pull/5902): perf(gnovm): load package file blocks lazily

URL: https://github.com/gnolang/gno/pull/5902
Author: omarsy | Base: master | Files: 6 | +367 -7
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: 9a82ed4d1 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5902 9a82ed4d1`

Round 2. Head advanced 0fed10129 → 9a82ed4d1 (rebase onto master 24b6080c2, 6 commits resquashed to 4): the PR's own diff is unchanged apart from the regenerated `stdlib_restart_compare` golden, which tracks a new master baseline; the PR's saving is identical to round 1 on every measure. All round-1 findings carry, none resolved; verdict unchanged.

**TL;DR:** When a multi-file package is read back from the store, the VM used to decode every file's block up front, even for files the transaction never enters. This PR loads each file block only when a function in it is first called, so a transaction that touches a few files of a big package stops paying to decode the rest.

**Verdict: APPROVE** — clean, well-scoped, well-tested; determinism holds warm-vs-cold across a restart; the [#4527](https://github.com/gnolang/gno/issues/4527) panic site is converted to a lazy load, not reintroduced. Non-blocking only: an undocumented lazy-cache invariant on `fBlocksMap`, and two small coverage gaps.

## Summary

`fillPackage` (the single store-load fill point) called `deriveFBlocksMap`, which materialized all N file blocks of a `PackageValue` on every load: N cold object reads and amino-decodes, most often for files the message never runs. This PR drops that eager call for multi-file packages and leaves `fBlocksMap` empty; a file block is materialized on first use through the already-lazy `GetFileBlock`, driven by `FuncValue.GetParent`. Eager derivation is kept on the package-creation path (`RunMemPackage`) and behind a `len(FNames) <= 1` guard so single-file packages keep master's exact gas. Measured here: a 12-file package touched at one file drops from 2123 to 1213 allocs (−43%), and a direct call into a 3-file realm costs 131804 less gas. The PR's ~2.37M saving on the deployed gnoswap pool closure needs the test13 fixture and was not reproduced in this review.

```
master (eager):   GetPackage ─► fillPackage ─► deriveFBlocksMap ─► decode ALL N file blocks
this PR (lazy):   GetPackage ─► fillPackage ─► (multi-file: nothing)
                  first call into file X ─► GetParent ─► GetFileBlock(X) ─► decode ONLY X
                  single-file (guard):  fillPackage ─► deriveFBlocksMap  (unchanged)
```

## Glossary

- cold object read: load of a persisted object not in the VM object cache, charged `ReadCostFlat` (59k gas) + 17 gas/byte; the skipped-file saving is skipped cold reads.
- Allocator: VM component charging allocation gas; its gas meter is wired only when a Machine is created, not at package-load time. Load-bearing for the gas note below.
- transactionStore: per-message Store wrapper from `BeginTransaction` with fresh per-message caches; why a partial `fBlocksMap` can't leak across messages.

## Fix

Master's `fillPackage` always called `deriveFBlocksMap`; the PR calls it only when `len(pv.FNames) <= 1`, so multi-file packages return with an empty `fBlocksMap` at [`store.go:629`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/store.go#L629) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/store.go#L629). The former panic-on-miss in `GetParent` becomes `pv.GetFileBlock(store, fv.FileName)` at [`values.go:640`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/values.go#L640) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/values.go#L640), which resolves the file's `RefValue` from the `FBlocks` slice on demand and caches it. The load-bearing constraint: only `GetParent` read the map assuming full population; every other `FBlocks` consumer walks the slice, which is unchanged.

## Benchmarks / Numbers

`BenchmarkPackageLoadFromStore` — load an N-file package from a fresh transaction store, touch one file. Base 24b6080c2 vs this branch, both re-run at this head.

| files | master allocs | branch allocs | master B/op | branch B/op | delta |
|---|---|---|---|---|---|
| 1 (guard) | 330 | 330 | 17592 | 17556 | none (no regression) |
| 4 | 819 | 573 | 39475 | 28137 | −30% allocs |
| 12 | 2123 | 1213 | 101934 | 57301 | −43% allocs, −44% mem |

`stdlib_restart_compare` gas golden: `EXACT_GAS` 2012646 → 1986776 (−25870), the skipped `strconv`/`strings` file-block cold reads. Master's baseline moved from round 1's 2176646, so the golden's absolute value is regenerated; the PR's own −25870 is unchanged.

End-to-end `MsgCall` into a 3-file target realm touching one file, the case no committed golden covers (scratch txtar, base vs branch):

| | master (24b6080c2) | branch | delta |
|---|---|---|---|
| GAS USED | 1816880 | 1685076 | −131804 |

The two skipped file blocks dominate, at roughly a `ReadCostFlat` cold read each. Identical to round 1's −131804 despite master shifting the absolute baseline by 258000.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[error message loses precision]** [`values.go:953-956`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/values.go#L953-L956) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/values.go#L953) — `GetParent`'s miss now routes to `GetFileBlock`'s `fmt.Sprintf("file %v not found in package %v", fname, pv)`, which formats the whole `PackageValue` with `%v`; master's terse `"file block missing for file %q"` was easier to read. Pre-existing in `GetFileBlock`, only reachable on already-corrupt state (a `FileName` not in `FNames`), so optional. Not posted.

## Missing Tests

- **[first-file-skip is unproven]** [`store_test.go:180`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/store_test.go#L180) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/store_test.go#L180) — every added test and the benchmark touch the first `FNames` entry (`a.gno` / `f0.gno`). None proves that touching *only a later* file skips the earlier ones, so a regression that eagerly materialized `FBlocks[0]` would pass. A variant calling into the last file and asserting `spy.reads[a.gno]==0` closes it (written and run green at this head, see `tests/`). Modest value; the core [#4527](https://github.com/gnolang/gno/issues/4527) path is covered.
- **[boundary FNames==2 untested]** [`store.go:629`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/store.go#L629) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/store.go#L629) — the guard branches at `len <= 1`; tests use 1 and 3 files, never the smallest lazy case (2). Low priority.

## Suggestions

- **[lazy-cache invariant invisible at the field]** [`values.go:864`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/values.go#L864) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/values.go#L864) — `fBlocksMap` has no doc comment. After this PR it carries a load-bearing invariant: on a store-loaded multi-file package it is empty or partial, a lazy first-touch cache, and must be read only through `GetFileBlock`. One direct reader already exists at [`nodes.go:1441`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/nodes.go#L1441) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/nodes.go#L1441) (miss-tolerant, creation path only); a future direct `fBlocksMap[...]` read on a load path reintroduces the exact [#4527](https://github.com/gnolang/gno/issues/4527) panic class. The `GetParent` comment explains the consumer, but the field is where a future editor looks. Fix: a one-line field comment stating the invariant and pointing reads at `GetFileBlock`.

## Verified

All re-run at 9a82ed4d1 against base 24b6080c2.

- **[#4527](https://github.com/gnolang/gno/issues/4527) regression guard is real.** Reverting `GetParent` to master's `pv.fBlocksMap[fv.FileName]` panic-on-miss makes [`TestLazyFileBlocksSkipUnusedStoreReads`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/store_test.go#L180) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/store_test.go#L180) fail with the exact incident stack: `panic: file block missing for file "a.gno"` through `DeclaredType.GetValueAt → FuncValue.GetParent`. The test exercises the method-dispatch nil-`Parent` path that fired in production, and would go red on a revert.
- **Determinism, warm vs cold.** `TestTestdata/stdlib_restart_compare` passes: a no-restart `Convert` and a post-`gnoland restart` `Convert` both charge exactly `EXACT_GAS=1986776`. Gas is independent of in-memory cache warmth; a leaked `fBlocksMap` would make the warm call cheaper and fail the test.
- **The rebase preserved the PR's effect exactly.** The PR's diff against its own merge-base is byte-identical to round 1 except the golden's absolute value; the measured saving is unchanged on both the golden (−25870) and the 3-file target call (−131804), despite master moving the baselines by 164000 and 258000. Master's drift touches `store.go` around `fillPackage` (mempackage prod/test blob split, [#5891](https://github.com/gnolang/gno/pull/5891)) and adds type-check gas metering at AddPackage and Run ([#5892](https://github.com/gnolang/gno/pull/5892)); neither changes the `Call` path's allocator wiring, which still loads the target before the meter is wired.
- **Cross-message isolation.** Every consensus handler (`AddPackage`, `Call`, `Run`) obtains the store via `getGnoTransactionStore`, which calls `ClearObjectCache` at [`keeper.go:417`](https://github.com/gnolang/gno/blob/9a82ed4d1/gno.land/pkg/sdk/vm/keeper.go#L417) · [↗](../../../../../.worktrees/gno-review-5902/gno.land/pkg/sdk/vm/keeper.go#L417); it rebuilds `cacheObjects` wholesale at [`store.go:1359`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/store.go#L1359) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/store.go#L1359), so a partial `fBlocksMap` cannot survive a message boundary.
- **Persisted bytes unchanged.** `GetFileBlock` caches the resolved block into the transient map only, never back into `pv.FBlocks[i]`, so `FBlocks` entries stay `RefValue` after load; GC ([`garbage_collector.go:360`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/garbage_collector.go#L360) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/garbage_collector.go#L360)) and the realm crawl ([`realm.go:1399`](https://github.com/gnolang/gno/blob/9a82ed4d1/gnovm/pkg/gnolang/realm.go#L1399) · [↗](../../../../../.worktrees/gno-review-5902/gnovm/pkg/gnolang/realm.go#L1399)) walk that slice, not the map.
- **The uncovered shape is a win, not a regression.** A direct `MsgCall` into a 3-file target realm touching one file, the one shape no committed golden pins, charges 1685076 gas on the branch against 1816880 on base (−131804). This is the case where the metering shift in Open questions could in principle have cost gas; measured, the skipped cold reads swamp it.
- Green at 9a82ed4d1: the three added store tests, `stdlib_restart_compare`.

## Open questions

- **A metering shift sits inside the win for a multi-file `MsgCall`/`MsgRun` target, and no golden covers that shape.** The target package loads at [`keeper.go:837`](https://github.com/gnolang/gno/blob/9a82ed4d1/gno.land/pkg/sdk/vm/keeper.go#L837) · [↗](../../../../../.worktrees/gno-review-5902/gno.land/pkg/sdk/vm/keeper.go#L837), before the Machine wires the allocator's gas meter at [`keeper.go:890`](https://github.com/gnolang/gno/blob/9a82ed4d1/gno.land/pkg/sdk/vm/keeper.go#L890) · [↗](../../../../../.worktrees/gno-review-5902/gno.land/pkg/sdk/vm/keeper.go#L890). On master the target's file blocks are allocated there, un-metered; here they are allocated during `m.Eval`, metered. So a touched target file block is charged allocation gas master never charged, while every skipped file saves a cold read. The skipped reads dominate: the 3-file target call above measures −131804 net, against a per-touched-block allocation charge in the low hundreds. Deterministic either way (meter-wiring timing is identical on every node; the guard branches on serialized `len(FNames)`), so not a defect, and the direction is a large decrease. Noting it only because no committed golden pins this shape: `stdlib_restart_compare`'s target `myrealm` is single-file (the guard), and the multi-file `strconv`/`strings` there are imports, which are metered on both sides. Not posted: the net is a clear win and nothing here asks the author to act.
