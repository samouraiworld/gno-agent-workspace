# PR #5082: feat(gnovm): Reduce string slicing allocation cost with reference-based StringValue

**URL:** https://github.com/gnolang/gno/pull/5082
**Author:** notJoon | **Base:** master | **Files:** 12 | **+213 -18**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR changes `StringValue` from a simple Go type alias (`type StringValue string`) to a struct with two modes: **owner** (`ref=false`) and **reference** (`ref=true`). The motivation is to make string slicing O(1) in the GnoVM allocator instead of O(n), since Go's string slicing already shares the underlying byte array and no data copy occurs.

Key structural changes:
- `StringValue` is now `struct { data string; ref bool }` with constructors `NewStringValue(s)` (owner) and `NewStringValueRef(s)` (reference/slice).
- The allocator gains `NewStringRef()` and `AllocateStringRef()` which charge a fixed 24-byte overhead instead of `24 + len(s)`.
- `GetSlice()` in `values.go` now calls `alloc.NewStringRef()` instead of `alloc.NewString()`.
- `GetShallowSize()` returns mode-appropriate sizes (fixed 24 bytes for refs, 24 + len for owners).
- Custom `MarshalAmino`/`UnmarshalAmino` methods ensure correct serialization (ref flag is not persisted; deserialized values are always owner mode).
- All call sites migrated from `StringValue(s)` cast to `NewStringValue(s)` constructor.

Files affected: `values.go` (core struct change), `alloc.go` (allocator methods and size calculation), `values_string.go` (String() method), `package.go` (amino registration), `uverse.go`, `convert.go`, `native.go`, `testing_runtime.go`, `context_testing.go` (call site updates), plus tests.

Related to issue #4885 (WIP fix for correctly counting strings in alloc and GC).

## Test Results
- **Existing tests:** PASS (3 pre-existing failures in `TestFiles` for error message formatting diffs — unrelated to this PR, confirmed they also fail on master)
- **New tests:** `TestAllocatorBytesForSlice` with short string, longer string, and chained slice cases — all PASS
- **Benchmark:** `BenchmarkStringSliceAlloc` confirms constant 24 `alloc_bytes/op` regardless of string length (5 to 320 chars)
- **CI:** All checks pass

## Critical (must fix)

- [ ] `gnovm/pkg/gnolang/alloc.go:506-513` — **GC recount undercharges for ref-mode StringValue, causing allocator drift.** When the GC walks live objects via `GCVisitorFn` (garbage_collector.go:152), it calls `v.GetShallowSize()` and passes the result to `alloc.Recount()`. For a ref-mode `StringValue`, this returns only 24 bytes. But the original allocation (before GC) charged the **full** cost of the parent string (`24 + len(parentString)`) via `alloc.NewString()`. After GC recounts the ref at 24 bytes, the allocator's byte count is lower than it should be — the parent string's data bytes are effectively forgotten. Over multiple GC cycles, this could allow a transaction to allocate more memory than `maxBytes` permits. **However**, note that `StringValue` is not an `Object` (it doesn't implement `ObjectInfo`), so it is only visited as an associated value of its parent container (e.g., `TypedValue.V` inside an `ArrayValue.List` element). The parent container's `GetShallowSize()` would already account for itself. The question is whether `StringValue.GetShallowSize()` is actually called during GC recount. Looking at the GC visitor flow: `vis(v)` is called on each `Value` in the graph, including non-Object values like `StringValue`. So yes, the ref-mode `StringValue` will be recounted at 24 bytes instead of whatever was originally charged. This is a real discrepancy, but its practical impact depends on how many string slices survive a GC cycle. This needs careful analysis and is likely the concern behind the related issue #4885.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/values.go:94-97` — **Struct size increase may cause heap escapes.** The PR description acknowledges that `B/op` increased in benchmarks (from 32 to 48 per op). The old `StringValue` was just a `string` (16 bytes), now it's a struct with `string` + `bool` + padding = 24 bytes. Since `StringValue` is stored as an `interface{}` (`Value`) in `TypedValue.V`, it must be boxed on the heap. The larger struct may affect cache locality and allocation patterns across the VM. The benchmark shows ~15% ns/op regression (59ns vs 27ns on the author's machine). This is a real trade-off worth documenting in the PR.

- [ ] `gnovm/pkg/gnolang/values.go:94-97` — **The `ref` field is exported behavior on an unexported field.** The `ref` field affects allocation accounting and `GetShallowSize()` semantics but is private. Consider whether `ref` should survive value copies in contexts like `copyValueWithRefs` (realm.go:1323) where the struct is copied as-is, preserving `ref=true`. After amino round-trip `ref` becomes `false`, but in-memory copies during realm operations may retain `ref=true`. This means a single StringValue could be counted differently depending on whether it was just deserialized or still in memory. The behavior is defensible but should be explicitly documented as an invariant.

- [ ] `gnovm/pkg/gnolang/values.go:131-135` — **`UnmarshalAmino` always sets `ref=false`, which is correct, but the `MarshalAmino` discards the `ref` flag silently.** This means amino round-tripping changes the identity of the value (a ref StringValue becomes an owner StringValue). For determinism, this is fine since the `data` field is identical. But it means `GetShallowSize()` can return different values before and after serialization for the same logical string, which could affect allocator accounting during persistence operations.

## Nits

- [ ] `gnovm/pkg/gnolang/values.go:99` — Missing period at end of doc comment: `// NewStringValue creates a new StringValue in owner mode` should end with `.`
- [ ] `gnovm/pkg/gnolang/values.go:104` — Same: `// NewStringValueRef creates a new StringValue in reference mode`
- [ ] `gnovm/pkg/gnolang/bench_test.go:35` — `benchmarkSliceSink` is a package-level var used to prevent compiler optimization, which is correct, but it should have a comment explaining that (like `sink` on line 9).
- [ ] `gnovm/pkg/gnolang/alloc_test.go:87,113` — `_ = result.GetString()` / `_ = s3.GetString()` are dead calls that don't affect the test outcome. The allocation is already tracked by `GetSlice`. These could be removed or replaced with assertions on the string content for added correctness verification.

## Missing Tests

- [ ] No test for `MarshalAmino`/`UnmarshalAmino` round-trip — verify that a ref-mode `StringValue` serializes and deserializes correctly, with `ref` becoming `false` after round-trip. This is the most critical serialization path.
- [ ] No test for `GetShallowSize()` on both owner and ref modes — verify that owner mode returns `24 + len(data)` and ref mode returns `24`.
- [ ] No test for string concatenation of ref-mode strings — verify that concatenating two ref-mode `StringValue`s produces an owner-mode result with correct allocation accounting.
- [ ] No test for `Len()` and `Value()` accessors on both modes.
- [ ] No test for `String()` (values_string.go:65) output format on both modes.
- [ ] No filetests exercising string slice allocation from Gno code — e.g., a `.gno` test that slices a string and verifies correct behavior.

## Suggestions

- Consider whether the `ref` flag should instead be tracked outside `StringValue`, perhaps in the allocator or as metadata, to avoid changing the struct size and heap behavior of every `StringValue` in the system. The bool + padding adds 8 bytes to every string in the VM, not just slices. An alternative: use a separate `StringSliceValue` type that wraps `StringValue`, or use a bitflag in `TypedValue` itself.
- The `allocStringRef = _allocBase` constant is identical to `allocString`. Consider making `allocStringRef` reference `allocString` directly (`allocStringRef = allocString`) to make the relationship explicit and avoid future divergence.
- The benchmark creates a new `Allocator` per iteration (`NewAllocator(1024 * 1024)` inside the `b.N` loop). This means allocator creation overhead is included in the measurement. Consider hoisting the allocator creation and using `alloc.Reset()` per iteration for a cleaner signal.

## Questions for Author

- How does this interact with the GC recount mechanism? When the GC walks live objects and calls `GetShallowSize()` on a ref-mode `StringValue`, it gets 24 bytes instead of the original allocation cost. Is this intentional? Does it align with the design in #4885?
- Should the `ref` flag be considered part of the value's identity for equality comparisons? Currently, `NewStringValue("abc") == NewStringValueRef("abc")` returns `false` in Go (because `ref` differs). This doesn't affect Gno-level equality (which uses `GetString()`), but it could matter for internal comparisons like the `assert.Equal` in `machine_test.go:62`.
- What happens when a ref-mode string is used as a map key? `MapKeyBytes` (values.go:1155-1156) calls `tv.GetString()` which returns `sv.data` regardless of mode, so correctness is preserved. But the allocator may under-count the actual memory cost of the key's `StringValue`.

## Verdict

NEEDS DISCUSSION — The core optimization is sound and well-implemented: string slicing should not charge for data that isn't copied. However, the interaction between the `ref` flag and the GC recount mechanism needs clarification, as it could lead to allocator drift over GC cycles. The PR should explicitly address how this fits with the broader allocation/GC fix in #4885 before merging. The 8-byte struct size increase for all StringValues (not just slices) is a meaningful trade-off that deserves discussion on whether the `ref` distinction should live at a different level.
