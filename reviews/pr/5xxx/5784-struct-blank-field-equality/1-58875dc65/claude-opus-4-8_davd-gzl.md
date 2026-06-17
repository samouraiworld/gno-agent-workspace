# PR #5784: fix(gnovm): ignore blank fields in struct equality and map keys

URL: https://github.com/gnolang/gno/pull/5784
Author: omarsy | Base: master | Files: 5 | +88 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `58875dc65` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5784 58875dc65`

**Verdict: APPROVE** — correct, spec-conformant fix with verified Go parity and good tests; one theoretical migration edge (pre-existing persisted maps whose struct keys differ only in blank fields collapse on load) is worth acknowledging but is not a blocker given there is no live chain state with that shape.

## Summary

Go ignores blank (`_`) struct fields when comparing structs and when encoding structs as map keys: two values equal under the spec must compare equal and hash to the same key. Gno was comparing and encoding blank fields, so `struct{ _ int; X int }{1,2} == {3,2}` wrongly returned `false`, and a map keyed on such a struct missed entries that the Go spec says are equal. The PR skips blank fields in both the struct-equality path and the struct map-key encoding, keeping the two consistent. Comparability is unaffected: a struct containing `_ []int` is still rejected at compile time, matching Go.

The two paths must stay in lockstep. Equality decides "are these the same key"; map-key encoding decides "do these land in the same bucket". Fixing one without the other would make lookups disagree with `==`.

## Glossary
- `isEql` — runtime struct/array/value equality in the VM, backing the `==` operator.
- `ComputeMapKey` — serializes a `TypedValue` to a `MapKey` string used as the in-memory map index.
- `vmap` — a `MapValue`'s `map[MapKey]*MapListItem` index, rebuilt from the persisted entry list on load.
- `blankIdentifier` — the constant `"_"` (preprocess.go:18).

## Fix

In [`isEql`](https://github.com/gnolang/gno/blob/58875dc65/gnovm/pkg/gnolang/op_binary.go#L523-L533) · [↗](../../../../../.worktrees/gno-review-5784/gnovm/pkg/gnolang/op_binary.go#L523-L533) the `StructKind` loop now `continue`s past any field whose type-level name is `_`, so the blank field's value never participates in `==`. `lt := baseOf(lv.T).(*StructType)` moves out of the `debug` block since it is now needed unconditionally to read field names. In [`ComputeMapKey`](https://github.com/gnolang/gno/blob/58875dc65/gnovm/pkg/gnolang/values.go#L1663-L1685) · [↗](../../../../../.worktrees/gno-review-5784/gnovm/pkg/gnolang/values.go#L1663-L1685) the `*StructType` branch skips blank fields and switches the comma separator from index-based (`i != sl-1`) to a `count`-based "comma before each entry after the first", so the encoding stays well-formed regardless of which fields are dropped.

## Verification

Go parity confirmed empirically against the toolchain (`go run` / `go vet`), matching every assertion in the new filetests:

```
T{1,2} == T{3,2}            => true    # blank ignored
T{1,2} == T{1,3}            => false
Outer{T{1,2}}==Outer{T{3,2}}=> true    # nested, via recursion
m[T{3,2}] (m has T{1,2})    => "a"      # equal key found
m[T{3,2}]="b"; len(m)       => 1        # overwrite, not insert
m[T{1,2}]                   => "b"
struct{ _ []int }           => compile error "struct containing []int cannot be compared"
F{NaN,2} as/at map key      => "nan", len 1   # NaN in a *blank* field is ignored, key stays comparable
```

The NaN case is the subtle one and is correct: the `isNaN` early-return now only fires for non-blank fields, so a NaN sitting in a blank field is dropped before it can poison the key. Gno's `map49.gno` asserts exactly this and passes.

Tests run green in the worktree: `TestFiles/map49`, `TestFiles/types/cmp_struct_h`, `TestFiles/types/cmp_struct_i`, the full `TestFiles/map*` set, and `TestComputeMapKey`.

## Critical (must fix)
None.

## Warnings (should fix)
- **[pre-existing maps with blank-only-differing keys silently collapse on load]** [`gnovm/pkg/gnolang/realm.go:1895-1906`](https://github.com/gnolang/gno/blob/58875dc65/gnovm/pkg/gnolang/realm.go#L1895-L1906) · [↗](../../../../../.worktrees/gno-review-5784/gnovm/pkg/gnolang/realm.go#L1895-L1906) — the map-key encoding change is a behavioral change for already-persisted maps, not just new code.
  <details><summary>details</summary>

  A `MapValue` persists its entries as a list (`List`); the `vmap` index is not stored, it is recomputed via `ComputeMapKey` every time the value is loaded. After this PR, a previously-stored map keyed on `struct{ _ int; X int }` that under the old encoding held two distinct entries differing only in the blank field (e.g. `T{1,2}` and `T{3,2}`, both inserted because the old code treated them as different keys) will, on load, encode both to the same new `MapKey`. The rebuild loop does `cv.vmap[mk] = cur`, so the later entry overwrites the earlier in the index while `List.Size` still counts both. Result: `len(m)` reports 2, lookups resolve to only one entry, and range iteration yields both — a quietly inconsistent map.

  This is deterministic across nodes (everyone runs the same new encoder), so it is not a consensus split, and gno.land has no live mainnet state that could already be in this shape, so the practical blast radius today is essentially nil. It is flagged because it is the one non-obvious consequence of changing a persisted-value encoding, and it is the kind of thing that belongs in an ADR or PR note so a future reader does not rediscover it as a "bug". Fix: no code change required for correctness going forward; document the encoding change (an ADR per `AGENTS.md`, since this is an AI-assisted VM-semantics change), and optionally note that any map persisted before the fix rebuilds its index under the new rules.
  </details>

## Nits
- [`gnovm/pkg/gnolang/op_binary.go:527`](https://github.com/gnolang/gno/blob/58875dc65/gnovm/pkg/gnolang/op_binary.go#L527) · [↗](../../../../../.worktrees/gno-review-5784/gnovm/pkg/gnolang/op_binary.go#L527) — `m.incrCPU(OpCPUEql)` and the per-byte gas in `ComputeMapKey` are now skipped for blank fields, so structs with blank fields cost marginally less CPU/gas than before. Deterministic and harmless, but it is a (tiny) gas-schedule change riding along with the semantic fix; worth a one-line mention in the PR body.

## Missing Tests
- **[low]** [`gnovm/tests/files/types/cmp_struct_h.gno`](https://github.com/gnolang/gno/blob/58875dc65/gnovm/tests/files/types/cmp_struct_h.gno) · [↗](../../../../../.worktrees/gno-review-5784/gnovm/tests/files/types/cmp_struct_h.gno) — no case for an all-blank struct (`struct{ _, _ int }`, always equal, encodes to `{}`) or a blank field in the last position (exercises the `count`-based comma boundary). Both follow from the logic, but they are the cheap edge cases that would lock the comma/encoding behavior in place. The collision-collapse path in `realm.go` (the Warning above) has no test at all.

## Suggestions
None.

## Questions for Author
- Was an ADR considered? `AGENTS.md` asks for one on non-trivial AI-assisted PRs and exempts "trivial bug fixes" — this is a small diff but it changes a persisted-value encoding and VM equality semantics, which sits on the non-trivial side of that line.
