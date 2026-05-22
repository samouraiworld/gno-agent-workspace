# PR #5599: feat(gnovm): support slice-to-array conversion

URL: https://github.com/gnolang/gno/pull/5599
Author: notJoon | Base: master | Files: 32 | +644 -8
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — `(*[N]T)(s)[:]` silently loses slice/array aliasing across persistence; the transient `ArrayValue` view leaks through `SliceValue.Base` into the realm storage walk and gets promoted to an unrelated real object.

## Summary

Implements two Go conversions in GnoVM: `[N]T(s)` (Go 1.20) deep-copies the slice into a new array, and `(*[N]T)(s)` (Go 1.17) returns a pointer whose dereference aliases the slice's backing array. Closes #3501, supersedes the abandoned #5079. Value form lives entirely in `ConvertTo`; pointer form introduces an `ArrayValue` "view" — same struct, two new fields `BaseArray *ArrayValue` / `BaseOffset int` — that forwards element-pointer ops to the base and is reconstructed on reload inside `fillValueTV`. The view is intended to be transient (never persisted), but only the `PointerValue` slot drops it correctly; any other path that pulls the view into a persisted graph (notably `SliceValue.Base` from `(*p)[:]`) lets the realm walker assign it an `ObjectID`, copy its Data/List into a fresh persistent array, and break the alias to the original slice.

```
in-tx:                          across-tx:
  data ──► base [1 2 3 4 5]       data ──► base [1 2 3 4 5]
            ▲   ▲                            ▲
            │   │                            │
  p.TV ──► view (Data=base[:3])    p.TV ──► view rebuilt by fillValueTV  ✓ ok
  g    ──► slice{Base=view}        g    ──► slice{Base=COPY of view}     ✗ no longer aliased
```

## Glossary

- view: transient `ArrayValue` with `BaseArray`/`BaseOffset` set, returned from `newArrayPtrView`; aliases the slice's backing storage.
- `isSliceToArrayPtrView`: discriminator used at reload to decide whether a `PointerValue` pointing into an `ArrayValue` is a slice-to-array-ptr view or a real nested-array element pointer.
- `ConvertTo`: central type-conversion dispatch in `values_conversions.go`.
- `fillValueTV`: lazy hydration entry point that resolves `RefValue` → object and populates `PointerValue.TV`.

## Fix

Preprocess: a new branch in [`preprocess.go:1687-1702`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/preprocess.go#L1687-L1702) accepts `[]T` → `[N]T` and `[]T` → `*[N]T` when the source slice and target array element TypeIDs match exactly, deferring length validation to runtime. Runtime: [`values_conversions.go:1085-1156`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values_conversions.go#L1085-L1156) handles both forms — the array case allocates a fresh `[N]T` (Data path for `Uint8Kind`, List path otherwise) and shallow-copies; the pointer case panics on length shortfall, returns nil for `*[0]T` from a nil slice, and otherwise returns a `PointerValue{TV: view, Base: baseAV, Index: sv.Offset}`. The view machinery is in [`values.go:303-332`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L303-L332) and the reload reconstruction in [`values.go:2783-2791`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L2783-L2791). `ArrayValue.GetPointerAtIndexInt2` ([`values.go:335-338`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L335-L338)) gains a one-line forward to base so writes through the view hit dirty tracking on the real backing.

## Benchmarks / Numbers

`_allocArrayValue` grows from 200 to 216 bytes ([`alloc.go:63`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/alloc.go#L63)) — every `ArrayValue` in every realm now pays the view tax.

| test                                  | before     | after      | Δ       |
|---------------------------------------|-----------:|-----------:|--------:|
| `gas/nested_alloc.gno` (10k arrays)   | 8559690082 | 8962092798 | +4.70%  |
| `gas/slice_alloc.gno`                 |   70970775 |   70970782 | +0.00%  |
| `alloc_1.gno` bytes                   |       9232 |       9248 | +16 B   |
| `gc.txtar` GAS USED                   |  151379058 |  151379086 | +28     |
| `stdlib_restart_compare.txtar` EXACT  |    1974912 |    1974916 | +4      |

## Critical (must fix)

- **[view leaks through SliceValue.Base and breaks alias semantics across persistence]** [`values.go:2244-2252`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L2244-L2252) — `(*p)[:]` produces a `SliceValue{Base: view, …}`; when that slice is assigned to a realm-persisted variable, the storage walk promotes the view to a real object and copies its Data, severing the alias to the original slice's backing array. Mutations through the persisted slice no longer hit the original slice and vice versa, silently violating Go's `(*[N]T)(s)` semantics.
  <details><summary>details</summary>

  Repro is [`tests/slice_through_view_alias.txtar`](tests/slice_through_view_alias.txtar): `data = []byte{1,2,3,4,5}; p := (*[3]byte)(data); g = (*p)[:]` in `Init`, then `g[0] = 99` in a later tx, then read `data` in a third tx. Expected (Go): `99 2 3 4 5`. Actual: `1 2 3 4 5`. No panic — the realm walker silently turns the view into a freestanding array.

  The view is an `Object` ([`ownership.go:139`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/ownership.go#L139)) but starts with no `ObjectID`, so when it becomes a child of `g` (a real `SliceValue`), `incRefCreatedDescendants` ([`realm.go:510`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/realm.go#L510)) calls `assignNewObjectID` on it and queues it as created. `copyValueWithRefs` on a view ([`realm.go:1425-1440`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/realm.go#L1425-L1440)) ignores `BaseArray`/`BaseOffset` and persists the view's `Data`/`List` slices verbatim — those slices alias the base in memory, but `cp(cv.Data)` makes a fresh copy in the persistent form, so reload yields a disconnected array. The PointerValue special-case in `fillValueTV` ([`values.go:2786`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L2786)) only covers the case where the view sits at the immediate `PointerValue.TV` slot (dropped by [`realm.go:1410-1424`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/realm.go#L1410-L1424)); every other path that reaches the view is broken.

  Same hazard for `(*p)[low:high]`, `&(*p)[i]` if the resulting pointer's `Base` ends up the view (it doesn't here — `view.GetPointerAtIndexInt2` forwards to base — but any future caller forgetting to forward would land here), and passing `(*p)[:]` as an argument that ends up stored. In-tx (no persistence) all of these work; the bug manifests only at tx boundaries, which is the worst class for realm bugs.

  Fix: either (a) at `(*p)[:]` slice construction, set `SliceValue.Base = view.BaseArray` and adjust `Offset += view.BaseOffset` so the slice tracks the real backing; or (b) make the view a separate type that `realm.go` knows to skip during the descendant walk and at `toRefValue`; or (c) reject persistence of view objects at `incRefCreatedDescendants` with a clear error rather than silently promoting them. Option (a) preserves Go semantics with the smallest surface change.
  </details>

## Warnings (should fix)

- **[every ArrayValue pays a 16-byte tax for a niche feature]** [`values.go:254-263`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L254-L263) — `BaseArray *ArrayValue` + `BaseOffset int` push `_allocArrayValue` from 200 to 216 bytes; `gas/nested_alloc.gno` jumps 4.7% (8.56G → 8.96G). Every realm allocating arrays pays for a feature ~zero realms use today.
  <details><summary>details</summary>

  16 bytes × 10k arrays in the nested-alloc benchmark is the entire delta. The same 1.6 KB/k-arrays footprint applies to every realm in production. A separate `ArrayPtrView` type implementing the `Value`/`Object` interfaces alongside `ArrayValue` would isolate the cost to views, and also makes the "this thing must not be persisted as a plain ArrayValue" invariant explicit at the type level — directly addressing the Critical above.
  </details>

- **[deleted file test regresses coverage of named-byte-type value conversion]** [`convert_slice_to_array_g.gno`](https://github.com/gnolang/gno/blob/bbde77944/gnovm/tests/files/convert_slice_to_array_g.gno) — added in `bbde77944`, deleted in `e81683679` with no explanation. Covered `type B byte; s := []B{...}; a := [3]B(s[1:])` — the List-backed source path for `Uint8Kind` (named) elements in `ConvertTo`. `convert_slice_to_array_ptr_f.gno` covers the *pointer* form for the same scenario but not the value form, leaving the `copyListToData` branch [`values_conversions.go:1112`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values_conversions.go#L1112) without an end-to-end test.
  <details><summary>details</summary>

  Restored as [`tests/restored_g.gno`](tests/restored_g.gno) — still passes. No reason to drop it. Restore the file or merge its assertions into another value-form test.
  </details>

- **[CPU gas not charged for the O(N) copy in `[N]T(s)`]** [`values_conversions.go:1106-1118`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values_conversions.go#L1106-L1118) — `copy(av.Data, sBase.Data[sOff:sOff+cat.Len])` and `copyListToData` scale linearly in `cat.Len`, but only `alloc.NewDataArray`/`NewListArray` (memory) gets billed. Other O(N) conversions in this file (e.g., `[]rune(string)`) charge through the allocator too, so this is consistent — but for very large `cat.Len` the per-element work is real and uncharged.
  <details><summary>details</summary>

  Not a blocker on its own; raise it explicitly so reviewers don't approve the gas changes without considering whether `OpCPUSlope*` should fire here. If existing comparable conversions skip CPU gas, fine — but document the decision.
  </details>

## Nits

- [`convert_slice_to_array_g.gno`](../../../../../.worktrees/gno-review-5599/gnovm/tests/files/) absent (deleted) and `convert_slice_to_array_ptr_h.gno` absent — non-sequential lettering invites future grep confusion. Renumber or document.
- [`values.go:319-323`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L319-L323) — `isSliceToArrayPtrView` comment is dense; the load-bearing claim is "a real `*[N]T` element pointer into a List-backed array always has `cbv.List[index].T.TypeID() == at.TypeID()`". Worth stating that exactly, plus pointing at the `et.(*ArrayType)` pre-filter in [`values.go:2786`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L2786) that makes the Data-backed branch trivially correct.
- PR title says "support slice-to-array conversion" but the second commit adds the substantially more complex pointer form. Title and description should call that out explicitly so reviewers know to scrutinize the view machinery, not just the value copy.
- [`preprocess.go:1680-1685`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/preprocess.go#L1680-L1685) — comment says "Element types must be identical (Go spec)" but the code uses `TypeID()` equality, which is identity by string canonicalization. For named types these coincide; worth a one-line note that TypeID equality is the chosen encoding of "identical".

## Missing Tests

- **[realm aliasing through `(*p)[:]`]** [`testdata/`](../../../../../.worktrees/gno-review-5599/gno.land/pkg/integration/testdata/) — `slice_to_array_ptr_persist.txtar` exercises `(*ptr)[0] = X` and `data[0] = X` separately, but not the slice-derived alias. See [`tests/slice_through_view_alias.txtar`](tests/slice_through_view_alias.txtar) — failing.
  <details><summary>details</summary>

  Adding a green-on-Go version of this test would have caught the Critical above. Should be part of the PR once the underlying alias is preserved.
  </details>

- **[non-byte view persistence]** [`testdata/`](../../../../../.worktrees/gno-review-5599/gno.land/pkg/integration/testdata/) — every existing `*persist*.txtar` uses `[]byte`/`[]int` with byte-like semantics. No coverage for `[]Struct{...}` or `[]*T` through a `*[N]T` pointer, which exercises the List-backed view path with non-trivial element TypedValues. Risk: persistence of the view's List (containing pointers) may invoke escape/refcount paths not covered by Data-backed tests.

- **[capacity-bound conversion]** No test for `(*[N]T)(s[lo:hi:cap])` with `cap > hi` — the PR ignores `Maxcap` per Go spec, but a regression test pins the behavior.

## Suggestions

- [`values.go:303-317`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values.go#L303-L317) — `newArrayPtrView` populates both `Data`/`List` AND `BaseArray`. The `Data`/`List` aliases are only used as a "convenient `GetLength`/`GetReadonlyBytes` short-circuit"; if everything routed through `BaseArray`, the view would carry only `(base, offset, length)` and no shared-slice hazard. Worth considering as part of the broader "make view a separate type" refactor.
  <details><summary>details</summary>

  Today `GetLength`, `GetCapacity`, `GetReadonlyBytes`, `Copy` all happen to work because the view's `Data`/`List` are populated. But that's a pile of implicit correctness — every future method on `ArrayValue` needs to ask "does this work for views?". A `ArrayPtrView` type implementing the same interface explicitly removes that footgun.
  </details>

- [`values_conversions.go:1146-1155`](../../../../../.worktrees/gno-review-5599/gnovm/pkg/gnolang/values_conversions.go#L1146-L1155) — for `(*[0]T)(s)` with a non-nil empty slice, the spec wants a non-nil pointer ([`convert_slice_to_array_ptr_e.gno`](../../../../../.worktrees/gno-review-5599/gnovm/tests/files/convert_slice_to_array_ptr_e.gno) covers this). The current code constructs a `PointerValue` whose `Base` is `baseAV` (which for `make([]int, 0)` is a real empty array). Fine — but if `baseAV` is ever nil for a non-nil zero-length slice, the `&((*p)[0])`-equivalent paths would NPE. Add a guard or document the invariant that `sv.GetBase` is non-nil for non-nil slices.

## Questions for Author

- Why was `convert_slice_to_array_g.gno` removed in `e81683679`? It still passes locally.
- What's the intended persistence story for slices and pointers *derived* from a view (`(*p)[:]`, `&(*p)[i]`)? The PR description focuses on the `ptr` itself; the derived-reference case is where the alias breaks.
- Was a separate `ArrayPtrView` type considered? The 16-byte cost on every `ArrayValue` is real and falls on every realm, not just users of this feature.
- Codecov reports 76.92% patch coverage with 21 lines missing — which lines, and are they in the unreachable-by-design paths or genuine gaps?
