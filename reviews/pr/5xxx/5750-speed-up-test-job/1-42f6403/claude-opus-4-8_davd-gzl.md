# PR #5750: perf(ci): speed up the gno.land test job (in-memory parallel txtars + per-package fixes)

URL: https://github.com/gnolang/gno/pull/5750
Author: thehowl | Base: master | Files: 14 | +175 -171
Reviewed by: davd-gzl | Model: claude-opus-4-8 (1M) | Commit: `42f6403` (stale — +24 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5750 42f6403`

**Verdict: APPROVE** — the one consensus-relevant change (GC gas exclusion of `.uverse`) is correct, deterministic, and test-guarded; the rest is test-infra/CI plumbing that builds and passes. Only open item: the `.uverse` exclusion is a second consensus-visible effect (it also drops `.uverse` from the GC alloc recount, not just gas) that the description frames as gas-only — confirm + document.

## Summary

CI-perf PR cutting the `ci / gnoland` `main / test` job from ~22 min to ~7.2 min by running the `gno.land` txtar integration suite in-memory **and in parallel** (the local exec time drops 1204s → 177s on 4 cores), plus boot-once speedups for the next-three-slowest packages (`gnoweb/markdown` 426s→~1s, `gnoweb` 302s→~24s, `cmd/gnoland` ~76s→~30s). Parallel in-memory was previously disabled for nondeterminism; the PR fixes the two remaining causes — a shared-`.uverse` GC-gas race and a port-reuse race on restart — then makes parallel the default and removes the now-dead subprocess machinery (`commandKindBin`, `INMEMORY_TS`). The only behavior-visible change to the chain is GC gas: `.uverse` (a process-global singleton) is no longer counted in per-tx GC traversal, lowering `gc.txtar`'s expected gas by 174 (≈6 GC passes × 29 gas/visit for the one uverse block).

```
GC visitor (per object v):
  before:  vis(uverseBlock) -> count++, alloc.Recount(size), SetLastGCCycle  ── shared object mutated
  after:   vis(uverseBlock) -> return false                                  ── never counted, never stamped
                                                  │
   machine A cycle N stamps uverseBlock.LastGCCycle=N ─┐
   machine B cycle N sees N, skips count  ── race ─────┘  (removed: uverse never counted on any machine)
```

## Glossary

- `.uverse` — the universe package: process-global singleton (`Uverse()` / `SetCachePackage(Uverse())`) holding all builtins, shared by every Machine in the process.
- GC gas — gas charged for a garbage-collection traversal, proportional to objects visited (`gcVisitGas`).
- `LastGCCycle` / `GCCycle` — per-object stamp vs per-machine counter used to dedup objects within one GC pass.
- in-memory node / txtar — integration test runs a full node inside the test process (vs. a subprocess); `.txtar` files are the scripted node test cases.
- `commandkind` — testscript node-launch mode: `Bin` (removed), `Testing` (subprocess re-exec), `InMemory` (now the default).

## Fix

The GC dedups visited objects with a per-object `LastGCCycle` compared to the per-machine `GCCycle`. `.uverse`'s `PackageValue` and `Block` are a single process-global instance shared across all parallel machines, so one machine stamping its cycle onto the shared block makes a concurrent machine (same cycle number) skip counting it — nondeterministic GC gas. The fix adds [`isUverseValue`](https://github.com/gnolang/gno/blob/42f6403/gnovm/pkg/gnolang/garbage_collector.go#L153-L165) · [↗](../../../../../.worktrees/gno-review-5750/gnovm/pkg/gnolang/garbage_collector.go#L153-L165) and returns early from the GC visitor for those two objects ([`garbage_collector.go:185-187`](https://github.com/gnolang/gno/blob/42f6403/gnovm/pkg/gnolang/garbage_collector.go#L185-L187) · [↗](../../../../../.worktrees/gno-review-5750/gnovm/pkg/gnolang/garbage_collector.go#L185-L187)), so they are never counted on any machine — uniform and deterministic. `.uverse`'s contents were already excluded from traversal, so skipping the two container objects changes only the count, not what gets reached. The port-reuse race is fixed by resetting RPC/P2P listen addresses to `:0` on every (re)start ([`process.go:91-98`](https://github.com/gnolang/gno/blob/42f6403/gno.land/pkg/integration/process.go#L91-L98) · [↗](../../../../../.worktrees/gno-review-5750/gno.land/pkg/integration/process.go#L91-L98)).

## Critical (must fix)

None.

## Warnings (should fix)

- **[consensus effect beyond gas — confirm + document]** [`gnovm/pkg/gnolang/garbage_collector.go:185-187`](https://github.com/gnolang/gno/blob/42f6403/gnovm/pkg/gnolang/garbage_collector.go#L185-L187) · [↗](../../../../../.worktrees/gno-review-5750/gnovm/pkg/gnolang/garbage_collector.go#L185-L187) — the early `return false` skips `alloc.Recount(size)`, not just the gas `visitCount`, so the GC's recomputed live-set shrinks by `.uverse`'s shallow size and the per-tx alloc cap is effectively relaxed.
  <details><summary>details</summary>

  The visitor returns at [`garbage_collector.go:185`](https://github.com/gnolang/gno/blob/42f6403/gnovm/pkg/gnolang/garbage_collector.go#L185) · [↗](../../../../../.worktrees/gno-review-5750/gnovm/pkg/gnolang/garbage_collector.go#L185) before both `*visitCount++` (line 200, → gas) and `alloc.Recount(size)` (line 214, → live-byte accounting). A GC is only triggered from [`alloc.go:312`](https://github.com/gnolang/gno/blob/42f6403/gnovm/pkg/gnolang/alloc.go#L312) · [↗](../../../../../.worktrees/gno-review-5750/gnovm/pkg/gnolang/alloc.go#L312) when an allocation would exceed `maxBytes`; after GC, `alloc.bytes` is the recounted live set and the next allocation is re-checked against it. Dropping `.uverse` from that recount means a tx sitting exactly at the memory boundary now has marginally more headroom — so GC gas is not the only consensus-visible effect; the OOM/`panic("allocation limit exceeded")` boundary shifts by a small constant too.

  This is deterministic and uniform across nodes (it does not reintroduce nondeterminism), and excluding shared, always-reachable infrastructure from a tx's own budget is arguably more correct. But the description states "only `gc.txtar`'s expected value changes" and frames the change as a "tiny … GC gas" reduction, which could lead a future consensus audit to miss the alloc-accounting side effect. Fix: confirm the headroom shift is intended and add one sentence to the PR/commit noting the alloc-recount effect alongside the gas one (magnitude is negligible — one uverse block's shallow size against `MaxAllocBytes` — so no code change is required, just documentation).
  </details>

## Nits

- [`gno.land/pkg/integration/testscript_gnoland.go:804`](https://github.com/gnolang/gno/blob/42f6403/gno.land/pkg/integration/testscript_gnoland.go#L804) · [↗](../../../../../.worktrees/gno-review-5750/gno.land/pkg/integration/testscript_gnoland.go#L804) — with `commandKindBin` gone and [`testdata_test.go:33`](https://github.com/gnolang/gno/blob/42f6403/gno.land/pkg/integration/testdata_test.go#L33) · [↗](../../../../../.worktrees/gno-review-5750/gno.land/pkg/integration/testdata_test.go#L33) now forcing `commandKindInMemory`, **nothing** in the tree sets `commandKindTesting` (grepped). The `case commandKindTesting` branch in `setupNode` is now unreachable via testscript — the same dead-via-no-setter shape this PR just removed for `commandKindBin`. The underlying `RunNode`/`RunNodeProcess` are still exercised directly by `TestNodeProcess`, so they stay; only the testscript dispatch to them is dead. Either drop the branch too, or add a one-line comment that it's retained for manual/external switching.

- [`gno.land/pkg/integration/process.go:89`](https://github.com/gnolang/gno/blob/42f6403/gno.land/pkg/integration/process.go#L89) · [↗](../../../../../.worktrees/gno-review-5750/gno.land/pkg/integration/process.go#L89) — pre-existing, not introduced here: `nodecfg.TMConfig.DBPath = pcfg.DBDir` is dead because the very next line `nodecfg.TMConfig = pcfg.TMConfig` overwrites the whole struct. Sits directly above the new port-reset lines, so it's a cheap drive-by cleanup while the hunk is open (verify `pcfg.TMConfig.DBPath` is already set before deleting).

## Missing Tests

- **[determinism not directly asserted]** [`gnovm/pkg/gnolang/garbage_collector.go:153-165`](https://github.com/gnolang/gno/blob/42f6403/gnovm/pkg/gnolang/garbage_collector.go#L153-L165) · [↗](../../../../../.worktrees/gno-review-5750/gnovm/pkg/gnolang/garbage_collector.go#L153-L165) — `gc.txtar` pins the post-fix gas value, so a regression that re-counts `.uverse` would flip the gas and fail that txtar. But the actual invariant — `isUverseValue` matches the uverse block + package value, and GC visitCount excludes them — has no direct unit test. A small `gnolang` unit test asserting `isUverseValue(Uverse())` / `isUverseValue(Uverse().Block)` is true (and false for a normal package) would lock the contract independent of the gas golden. Optional; the txtar guard is sufficient for now.

## Suggestions

None.

## Questions for Author

- The `.uverse` exclusion drops the two objects from the GC alloc recount as well as from GC gas (see Warning) — intended, and worth a sentence in the description? (Both gas-used and the alloc-cap boundary are consensus-visible.)
- The description notes "only the pre-existing `or_f0` golden mismatch" on master under `go test ./gnovm/pkg/gnolang/ -run Files -test.short`. I observe three (`types/eql_0b4`, `types/eql_0f0`, `types/or_f0`), all type-checker error-message drift, all present on master with the GC file reverted — i.e. unrelated to this PR. Just flagging the count is higher than stated; nothing for this PR to fix.

---

### Verification

`gc.txtar` passes with the updated gas value (consensus-relevant change confirmed green):

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5750 -R gnolang/gno
go test ./gno.land/pkg/integration/ -run 'TestTestdata/gc$' -v 2>&1 | grep -E 'GAS USED|--- (PASS|FAIL)'
```

```
        GAS USED:   151380803
    --- PASS: TestTestdata/gc (3.64s)
```

Also run locally, all green: `go test ./gno.land/pkg/sdk/vm/ -run Gas` (3.5s); `go test ./gnovm/pkg/gnolang/ -run 'GC|GarbageCollect|Gas|Alloc'`; `go test ./gno.land/pkg/gnoweb/ -run 'TestRoutes|TestAnalytics|TestHealthEndpoints|TestStaticMarkdownDevLinks'` (24s, one node boot); `go test ./gno.land/pkg/gnoweb/markdown/ -run 'TestSanitizeIntegration|TestBlockRichSetext'` (0.8s). The 3 `Files -short` mismatches reproduce on master with `garbage_collector.go` reverted, confirming they predate this PR.
