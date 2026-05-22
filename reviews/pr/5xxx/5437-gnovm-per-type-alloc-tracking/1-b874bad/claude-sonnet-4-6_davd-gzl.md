# PR #5437: feat(gnovm): add per-type GC allocation tracking in debug builds

**URL:** https://github.com/gnolang/gno/pull/5437
**Author:** omarsy | **Base:** master | **Files:** 5 | **+124 -17**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR adds per-type allocation tracking to the GnoVM allocator, gated behind the `debug` build tag, as a follow-up diagnostic tool to the GC mismatch bug fixed in PR #5436. The motivation: when GC recount exceeds `alloc.bytes`, there was no way to identify which value type was under-tracked; diagnosing PR #5436 required ad-hoc instrumentation.

The implementation adds a `typeCounts map[string]int64` field to `Allocator`, initialized only in debug builds (`NewAllocator` checks the `debug` constant). All `Allocate` and `Recount` call sites gain a `typeName string` parameter. The wrapper functions (`AllocateString`, `AllocateBlock`, etc.) pass compile-time string literals. In `GCVisitorFn`, a new `allocTypeName(v Value) string` helper uses `reflect.TypeOf` to derive the type name at runtime. `store.go` similarly uses `reflect.TypeOf(oo).Elem().Name()` at the call site.

`SnapshotTypeCounts()` copies the map before GC. `GarbageCollect()` snapshots before `Reset()`, then a new deferred function compares post-recount `typeCounts` against the snapshot and logs any type where after > before to stderr. The ADR (`gnovm/adr/pr5437_gc_debug_tracking.md`) documents context, decision, alternatives, and consequences.

Five files change: the ADR (new), `alloc.go`, `garbage_collector.go`, `store.go`, and `gnovm/tests/files/gas/slice_alloc.gno` (gas value updated +16).

## Test Results

- **Existing tests:** Gas tests (gas/const.gno, gas/nested_alloc.gno, gas/slice_alloc.gno) PASS. Alloc filetests (alloc_0 through alloc_9, heap_alloc_*) PASS. The full `./gnovm/...` suite times out on the slow alloc_10_long tests (pre-existing infrastructure issue, unrelated to this PR).
- **Edge-case tests:** skipped — no debug-build tests exist for the new tracking path.

## Critical (must fix)

- [ ] `gnovm/pkg/gnolang/garbage_collector.go:191` — `allocTypeName(v)` is called unconditionally on every GC-visited object in **all** builds, not just debug. `reflect.TypeOf(v)` is a runtime call with non-trivial cost (interface dispatch + type descriptor lookup). The result is passed to `Recount`, which discards it immediately when `typeCounts == nil`. The ADR explicitly states "Non-debug builds have zero runtime overhead" — that claim is false for the GC path. Every GC visit incurs one `reflect.TypeOf` call for nothing. Fix: gate the call with `if alloc.typeCounts != nil { alloc.Recount(size, allocTypeName(v)) } else { alloc.Recount(size, "") }`, or pass an empty string and let Recount nil-check guard the map write.

- [ ] `gnovm/pkg/gnolang/store.go:489` — `reflect.TypeOf(oo).Elem().Name()` is evaluated unconditionally in all builds. Same zero-overhead violation as above. If `oo` holds a non-pointer concrete type (unlikely given amino always unmarshals to pointers, but not guaranteed), `.Elem()` panics. Fix: use a compile-time string literal or gate behind `if debug`.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/alloc.go:152–159` — `Fork()` creates a new `Allocator` without copying or initialising `typeCounts`. When `BeginTransaction` calls `ds.alloc.Fork().Reset()`, the transaction-scoped store allocator gets `typeCounts == nil` even in debug builds. Objects loaded via `loadObjectSafe` during a transaction allocate through `ds.alloc` (the fork), so they never appear in `beforeCounts`. But GC recounts them via `m.Alloc.Recount` (the original alloc, which does have `typeCounts`). Result: after GC, those types have `after > before = 0`, firing **spurious mismatch alerts** for every transaction that loads objects from the store. This undermines the usefulness of the diagnostic. Fix: either initialise `typeCounts` in `Fork()` when `debug` is set, or document that the tracking is only valid for non-transaction (test/REPL) scenarios.

- [ ] `gnovm/tests/files/gas/slice_alloc.gno:14` — Gas value increased from 500003087 to 500003103 (+16) with no explanation in the commit message. In non-debug builds `typeCounts` is nil throughout, so the map operations don't run. The only runtime change is `allocTypeName` being called per GC visit (`reflect.TypeOf`). However gas is charged via `alloc.gasMeter.ConsumeGas`, not via Go-level reflection. +16 gas = either 2 more GnoVM objects visited during GC (`VisitCpuFactor=8`) or 16 more GnoVM-allocated bytes, neither of which is explained by the diff. The author should clarify.

## Nits

- [ ] `gnovm/pkg/gnolang/garbage_collector.go:68–69` — The mismatch-log defer accesses `m.Alloc.typeCounts` directly rather than going through `SnapshotTypeCounts` or a `GetTypeCounts()` accessor. Since both files are in the same package this is legal, but it leaks internal representation. Minor, since the whole feature is debug-only.

- [ ] `gnovm/adr/pr5437_gc_debug_tracking.md:52` — The ADR states: "Non-debug builds have zero runtime overhead: `typeCounts` is nil, the nil check short-circuits, and the `typeName` string argument is not used." This is inaccurate: (a) `allocTypeName(v)` with `reflect.TypeOf` IS evaluated before the nil-check guard, and (b) `reflect.TypeOf(oo).Elem().Name()` in `store.go` is also unconditional. The consequences section should be corrected.

## Missing Tests

- [ ] No unit test for `SnapshotTypeCounts` — basic clone/nil semantics are untested (`gnovm/pkg/gnolang/alloc.go:100`).
- [ ] No unit test for `allocTypeName` — pointer vs value type dispatch is untested (`gnovm/pkg/gnolang/garbage_collector.go:13`).
- [ ] No test (under `-tags debug`) for the mismatch-detection path — there is no test that triggers GC in a debug build and asserts stderr output for a mismatch condition. The entire new feature is unexercised by the test suite.

## Suggestions

- To make `allocTypeName` overhead-free in non-debug builds, move the call inside a `if alloc.typeCounts != nil` guard rather than passing the result as an argument. Go's inliner will then trivially dead-code-eliminate it. Example: `if alloc.typeCounts != nil { alloc.Recount(size, allocTypeName(v)) } else { alloc.Recount(size, "") }` — or better, add a `RecountDebug(size int64, name string)` method that inlines both the nil check and the map write, keeping the calling code clean.
- Consider a `TestSnapshotTypeCounts` unit test that constructs a debug-build `Allocator`, calls a few `Allocate*` wrappers, snapshots, resets, and verifies the snapshot is unchanged. This would exercise the feature without needing a full filetest. Can be placed in `alloc_test.go` or a new `alloc_debug_test.go` with `-tags debug`.

## Questions for Author

- What causes the +16 gas increase in `slice_alloc.gno`? In a non-debug build the new code paths should be nil-gated. If the change is expected, add a comment to the test explaining the delta.
- Was the transaction-scoped allocator (Fork + Reset in `BeginTransaction`) intentionally excluded from type tracking? If so, document the limitation in the ADR and add a comment in `Fork()`.
- The mismatch logging writes to `os.Stderr` with `fmt.Fprintf`. For a debug facility used during development, is `fmt.Fprintf(os.Stderr, ...)` the right output channel, or would `debug.Printf` (which already manages the format and is already imported) be more consistent with the rest of the debug output in this package?

## Verdict

REQUEST CHANGES — The zero-overhead guarantee claimed by the ADR is violated: `allocTypeName` (via `reflect.TypeOf`) and the `reflect.TypeOf(oo).Elem().Name()` call in `store.go` execute unconditionally in all builds, adding measurable cost to every GC visit and every store object load. Additionally, `Fork()` silently drops `typeCounts`, causing spurious mismatch alerts for any blockchain transaction that loads objects — the most common real-world scenario. Fix the conditional gating first; the rest of the feature (ADR, mapping design, snapshot/compare approach) is sound.
