# PR #4892: fix(gnovm): include missing field in shallow size calculation + add overflow protection

URL: https://github.com/gnolang/gno/pull/4892
Author: davd-gzl | Base: master | Files: 25 | +338 -54
Reviewed by: davd-gzl | Model: claude-opus-4-7 (self-review — flag for second human reviewer)

**Verdict: REQUEST CHANGES** — premise largely superseded by master (constants already fixed in [`alloc.go:82-90`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L82-L90) on master, `_allocPackageValue` now `296` due to interrealm v2 PkgID), `AllocatePackageValue` becomes dead code, `slice_alloc.gno` flipped from success-at-threshold to failure-just-above-threshold (weakens coverage), `alloc_11a.gno` now triggers the "should not happen" panic branch instead of the regular alloc-limit panic; rebase onto current master and re-evaluate whether anything beyond the `PkgName` accounting and `AddFileBlock` allocator-plumbing still applies.

## Summary

The PR fixes [issue #4791](https://github.com/gnolang/gno/issues/4791) (missing `PkgName` in `PackageValue.GetShallowSize`), corrects 9 of 11 `_alloc*Value` constants that had drifted from `unsafe.Sizeof(...)`, plumbs `*Allocator` through `AddFileBlock` so file-block memory is charged at creation time, and adds a shared `packageValueSize()` used by both the allocation and GC-recount paths. It also introduces `overflow.Addp/Mulp` wrapping in several `GetShallowSize` implementations. The work was started 2025-11-17 and has not been rebased onto major master changes since (e.g. [#5642 preprocess hardening](https://github.com/gnolang/gno/pull/5642), [#5498 alloc constructor validation](https://github.com/gnolang/gno/pull/5498), [#5607 restore alloc counting](https://github.com/gnolang/gno/pull/5607), [#5291 gas calibration](https://github.com/gnolang/gno/pull/5291)), which makes the PR `CONFLICTING` per `gh pr view`.

## Glossary

- `GetShallowSize()` — returns the bytes a value is reported to occupy; called during GC re-walk and store-load.
- `Recount(size)` — adds `size` to `alloc.bytes` without charging gas (GC re-walk path).
- `packageValueSize(...)` — new shared helper that computes a `PackageValue`'s shallow cost from its `PkgName`, `PkgPath`, and `FNames`.
- `fileBlockEntrySize(fname)` — incremental cost of one file-block entry: filename string + interface slot + map-entry pointer.
- `AddFileBlock` — appends a file block to a `PackageValue`; before PR took no allocator, now takes `*Allocator` so the cost is charged at creation.
- `nilAllocator` — `(*Allocator)(nil)` sentinel; `Allocate()` short-circuits on nil. Used in `PreprocessFiles` and `Machine.runDeclarationFor` lint paths.

## Fix

Before: `PackageValue.GetShallowSize()` returned bare `allocPackage`, ignoring `PkgName`, `PkgPath`, `FNames`, `FBlocks`, and `fBlocksMap` — the original audit finding in #4791. `AddFileBlock` performed no allocation. 9 of 11 `_alloc*Value` constants didn't match `unsafe.Sizeof(...)` for the matching struct.

After: a single `packageValueSize(pkgName, pkgPath, fnames)` is the source of truth for `PackageValue` shallow cost; it is called from both `Allocator.NewPackageValue` (main pkg creation), the non-main branch of `PackageNode.NewPackage`, and `PackageValue.GetShallowSize` — guaranteeing symmetry between allocation and GC-recount. `AddFileBlock(alloc, fname, fb)` charges one `fileBlockEntrySize(fname)` per append. Numeric constants are realigned to `unsafe.Sizeof`. Several `GetShallowSize` implementations gain `overflow.Addp/Mulp` wrapping.

The load-bearing constraint: `pv.GetShallowSize()` must equal the cumulative amount the allocator charged for that `pv` during creation, otherwise `loadObjectSafe` at [`store.go:462-464`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/store.go#L462-L464) under-charges or over-charges, and GC `Recount` either reclaims less than was allocated (bytes never returns to zero) or attempts to reclaim more (negative drift / GC stop early).

## Benchmarks / Numbers

Test golden values change (a sample, sourced from the PR diff):

| Test | Before | After | Delta |
|------|--------|-------|-------|
| `gas_test.go TestAddPkgDeliverTx` | 226778 | 227880 | +1102 (+0.5%) |
| `gc.txtar GAS USED` | 1048705180 | 1048705980 | +800 |
| `gnokey_gasfee.txtar addpkg simulate` | 270022 | 271124 | +1102 |
| `gnokey_gasfee.txtar call simulate` | 113965 | 114069 | +104 |
| `issue_4983.txtar` | 252934 | 253062 | +128 |
| `restart_gas.txtar bar` | 593679 | 595593 | +1914 |
| `simulate_gas.txtar` | 110247 | 110351 | +104 |
| `tests/files/gas/nested_alloc.gno` | 24810947885 | 27211459933 | +9.7% |

The +9.7% on `nested_alloc.gno` is the largest single delta and reflects the per-block size increase (`_allocBlock 472 → 528`, +56 bytes). Master already lives at `_allocBlock = 528` independently, so the cost increase is not net-new on top of current master once rebased.

## Critical (must fix)

- **[premise superseded by master]** [`gnovm/pkg/gnolang/alloc.go:82-90`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L24-L49) — most of the constant-drift fix is already in master via the separate `unsafe.Sizeof`-anchored refactor; `_allocPackageValue` is `296` (not `272`) on master after interrealm v2.
  <details><summary>details</summary>

  Master `gnovm/pkg/gnolang/alloc.go:82-90` defines `_allocStructValue = 176`, `_allocArrayValue = 200`, `_allocMapValue = 168`, `_allocBoundMethodValue = 200`, `_allocBlock = 528`, `_allocFuncValue = 352` — identical to this PR. Master goes further: `_allocPackageValue = 296` (was 272 here, off by 24 — `Hashlet + alignment` for the new `PkgID` field added during interrealm v2 in [#5669](https://github.com/gnolang/gno/pull/5669)). Master also introduced `_allocHeap = 32` replacing `_allocBase + _allocPointer`, with structured comments classifying by-value vs by-pointer types, and a `TestAllocConstSizes` check function. Once rebased, the constant block in this PR will largely conflict-resolve to master's existing values; the unique remaining contributions are the `PkgName`/`PkgPath`/`FNames` accounting in `packageValueSize`, the `AddFileBlock(*Allocator, ...)` plumbing, and the `overflow.Addp/Mulp` wrapping. Fix: rebase onto current master, drop everything that already exists there, and re-justify the surviving deltas against master's allocator framework.
  </details>

- **[GC re-walk early-stop now reachable for tight `MAXALLOC`]** [`gnovm/tests/files/alloc_11a.gno:18`](../../../../../.worktrees/gno-review-4892/gnovm/tests/files/alloc_11a.gno#L18) — test expected error changed from `"allocation limit exceeded"` to `"should not happen, allocation limit exceeded while gc."`.
  <details><summary>details</summary>

  Mechanism: with the larger per-block cost, the recursion in `alloc_11a.gno` (MAXALLOC=1500) hits the early-stop branch in `GCVisitorFn` at [`garbage_collector.go:158-161`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/garbage_collector.go#L158-L161) — a single object's shallow size on its own would push `curBytes+size > maxBytes`, the visitor returns `stop=true`, `GarbageCollect` returns `ok=false`, and `Allocate` panics with the "should not happen" message at [`alloc.go:146`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L146). That panic message is contradictory: it asserts the case "should not happen" but a deterministic file-test now relies on it firing. Either the message is wrong (this is a normal outcome for tight budgets) or the early-stop should be reframed as a recoverable limit-hit. Fix: pick one — either rename the panic to a regular "allocation limit exceeded (during GC)" with no "should not happen" framing, or raise `MAXALLOC` in `alloc_11a.gno` to land back on the normal post-retry panic path. Don't ship a test asserting a "should not happen" panic.
  </details>

## Warnings (should fix)

- **[`slice_alloc.gno` flipped from success-at-threshold to failure-at-threshold+1]** [`gnovm/tests/files/gas/slice_alloc.gno:5-17`](../../../../../.worktrees/gno-review-4892/gnovm/tests/files/gas/slice_alloc.gno#L5-L17) — test no longer covers the "exactly at MAXALLOC passes" path; comment still says "is the threshold to reach the allocation limit" but the test now expects `allocation limit exceeded`.
  <details><summary>details</summary>

  Master's version allocs `12499872` items, passes, asserts gas `70970781`. This PR allocs `12499879` items and asserts `allocation limit exceeded`, with the same comment "is the threshold to reach the allocation limit". Either intent — boundary success or boundary failure — is valid coverage; mixing the two with a stale comment is not. The commit message [`9b931f63`](https://github.com/gnolang/gno/pull/4892/commits/9b931f631dbbb7df9fa0fc044f595ccad2ccd5a1) calls out Go 1.24 vs 1.25 gas drift (7127 vs 7111) as the reason for moving the boundary, but the proper fix is to find an `n` that succeeds across Go versions, not to flip the test's intent. Fix: split into two file-tests — one at `n-1` asserting success with the canonical gas value, one at `n` asserting `allocation limit exceeded`. Update the comments accordingly.
  </details>

- **[`AllocatePackageValue` becomes dead code]** [`gnovm/pkg/gnolang/alloc.go:213-215`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L213-L215) — no callers after the PR; `NewPackageValue` and `NewPackage` both call `Allocate(packageValueSize(...))` directly.
  <details><summary>details</summary>

  `grep -rn AllocatePackageValue gnovm/ gno.land/ tm2/` returns only the function definition and the ADR mentioning it. Keeping a public-receiver alloc helper around as documentation drift bait is worse than deleting it. If callers want the "package without FNames" shape, they should call the shared `packageValueSize(pkgName, pkgPath, nil)` directly, just like the PR does. Fix: remove `AllocatePackageValue`.
  </details>

- **[asymmetry remains for `PreprocessFiles` lint path]** [`gnovm/pkg/gnolang/machine.go:508`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/machine.go#L508) and [`:511`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/machine.go#L511) — `PreprocessFiles` calls `AddFileBlock(nilAllocator, ...)` and `PrepareNewValues(nilAllocator, pv)`, so the file-block costs are never charged, but the same `pv` if later persisted will have a `GetShallowSize` that includes them.
  <details><summary>details</summary>

  This is the same shape the ADR claims to have fixed. The lint/import-only contract for `PreprocessFiles` (no allocator, no transaction) means the discrepancy is harmless inside the lint binary. But the ADR's "single source of truth, no asymmetry" claim is too strong — there is still an asymmetry, the path is just narrow. Either: (a) acknowledge it in the ADR with the rationale "lint contexts never persist `pv`, so the asymmetry is sound by construction", or (b) wire `m.Alloc` through `PreprocessFiles` (more invasive, breaks the lint contract). Fix: option (a), edit `gnovm/adr/pr4892_package_alloc_consistency.md` to call this out explicitly so a future reader doesn't misread the invariant.
  </details>

- **[ADR claims removed overflow protection that is still present]** [`gnovm/adr/pr4892_package_alloc_consistency.md:42-50`](../../../../../.worktrees/gno-review-4892/gnovm/adr/pr4892_package_alloc_consistency.md#L42-L50) — ADR section "Overflow protection removed from GetShallowSize" but the code at [`alloc.go:405-422`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L405-L422), [`:447`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L447), [`:454-465`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L454-L465), [`:509`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L509) still uses `overflow.Addp`/`overflow.Mulp` throughout `fileBlockEntrySize`, `packageValueSize`, `Block.GetShallowSize`, `ArrayValue.GetShallowSize`, `StructValue.GetShallowSize`, `MapValue.GetShallowSize`, `StringValue.GetShallowSize`.
  <details><summary>details</summary>

  The ADR's overflow-removal section reflects an interim design that was reverted but the prose was not updated. A maintainer reading the ADR will trust the wrong story. Fix: update `gnovm/adr/pr4892_package_alloc_consistency.md` to match the current code state — overflow protection is retained as a defense-in-depth measure in `GetShallowSize` despite `Allocate()` already validating, and the prose section that argues against it should be deleted or rewritten.
  </details>

- **[`getFBlocksMap()` XXX comment unaddressed]** [`gnovm/pkg/gnolang/values.go:822`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/values.go#L822) — `// XXX, pass in allocator` still present even though the PR claims to have closed this gap for file blocks.
  <details><summary>details</summary>

  The PR plumbs the allocator into `AddFileBlock`, so the *first* lazy materialization of `fBlocksMap` is charged when `AddFileBlock` calls `pv.getFBlocksMap()[fname] = fb`. But `getFBlocksMap` itself, called from `GetFileBlock` lookup post-load, still materializes the map without charging — which is fine because `packageValueSize` already accounted for `len(FNames)` map entries during `loadObjectSafe`. So the XXX comment is now obsolete, not unfixed. Fix: delete the `// XXX, pass in allocator` comment and replace with a brief note "map entries are counted via packageValueSize(FNames); allocator is not needed here".
  </details>

## Nits

- [`gnovm/pkg/gnolang/alloc_test.go:23`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc_test.go#L23) — `TestAllocSizes` uses `println` to stdout; master's `TestAllocConstSizes` is the structured form. Once rebased these tests collide; consolidate into one.
- [`gnovm/pkg/gnolang/alloc_test.go:60-87`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc_test.go#L60-L87) — `TestAllocConstantsMatchActualSizes` overlaps with master's similar check. Drop in favor of master's version on rebase.
- [`gnovm/adr/pr4892_package_alloc_consistency.md:17-19`](../../../../../.worktrees/gno-review-4892/gnovm/adr/pr4892_package_alloc_consistency.md#L17-L19) — claim that "Non-main packages constructed in `PackageNode.NewPackage()` skipped `AllocatePackageValue()` entirely (comment: other packages are allocated while loading from store)" is accurate, but worth noting that the comment was deleted in the same PR at [`nodes.go:1343`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/nodes.go#L1341-L1345).

## Missing Tests

- **[no failing test that demonstrates the asymmetry the PR fixes]** [`gnovm/pkg/gnolang/alloc_test.go`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc_test.go) — the new tests verify `unsafe.Sizeof` parity and `packageValueSize` arithmetic but none exercise the alloc-vs-recount divergence the PR is meant to close.
  <details><summary>details</summary>

  Shape: a regression test should construct a `PackageValue` via `NewPackageValue`, add a few file blocks, snapshot `alloc.bytes`, force `GarbageCollect` to recount, and assert `alloc.bytes` (post-GC) equals the snapshot. Without that, a future change that re-introduces the asymmetry (e.g. adding a new field to `PackageValue` and forgetting to update `packageValueSize`) will silently regress. Fix: add such a test. Bonus: do the same for `Block` and `FuncValue`, the other types where `GetShallowSize` does non-trivial arithmetic.
  </details>

- **[no store round-trip test for `GetShallowSize` consistency]** [`gnovm/pkg/gnolang/store.go:462-464`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/store.go#L462-L464) — the load-then-recount path is the primary consumer of `GetShallowSize` outside GC; no test exercises it end-to-end.
  <details><summary>details</summary>

  The `loadObjectSafe` path calls `oo.GetShallowSize() + internalRefSize(oo)` and feeds it to `Allocate`. A test that creates a package, persists it, drops it from cache, reloads, then asserts the post-load `alloc.bytes` increment matches `GetShallowSize` would catch any future drift between `packageValueSize` and the actual struct shape.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/alloc.go:213-215`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L213-L215) — delete `AllocatePackageValue`; it is dead code and bait for future drift.
- [`gnovm/pkg/gnolang/nodes.go:1346-1347`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/nodes.go#L1346-L1347) — the non-main branch of `NewPackage` now does `alloc.Allocate(packageValueSize(...)) + alloc.AllocateBlock(...)`; this is the same two-call shape as `NewPackageValue` but duplicated. Move the shared body into `alloc.NewPackageValue` and call it unconditionally; the only diverging concern is whether `pn.PkgName == "main"` gets a `Realm` later, which is already handled outside `NewPackage`.
- [`gnovm/pkg/gnolang/alloc.go:402-407`](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L402-L407) — `fileBlockEntrySize` adds `_allocValue+_allocName+_allocPointer` (40 bytes) as a single literal; document the breakdown inline as `// FBlocks[] interface slot (_allocValue) + fBlocksMap entry (_allocName key + _allocPointer value)` so a future reader doesn't have to reverse-engineer the count.

## Questions for Author

- Have you confirmed `_allocPackageValue` should be 272 vs master's 296? Master added 24 bytes for interrealm v2 PkgID; if you rebase your `_allocPackageValue` will need to follow, and `packageValueSize` will need to be re-validated against the new struct layout.
- Why retain `overflow.Addp/Mulp` in `GetShallowSize` paths after the ADR argues they're unnecessary? If the position has changed, update the ADR. If not, remove the wrappers and rely on `Allocate`'s overflow check.
- Was the `slice_alloc.gno` test's flip from success-at-threshold to failure-at-threshold+1 intentional? If so, please also keep the success-at-threshold case as a separate file-test, otherwise we lose the boundary-success assertion entirely.
- This PR is `CONFLICTING` per `gh pr view` and 5+ months behind master with substantial allocator changes landed in the interim. Are you planning to rebase, or is this PR a candidate for closure/supersession by a fresh PR targeting current master?

---

Self-review caveat: this review was produced under the GitHub account of the PR author. Treat as a sanity check, not as approval. A second human reviewer (preferably @ltzmaxwell, who left the original review comments, or @thehowl per the comment thread) should re-evaluate before merge.
