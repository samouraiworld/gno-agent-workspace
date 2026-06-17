# PR #5713: fix(gnovm): panic on comparing uncomparable types via interface

URL: https://github.com/gnolang/gno/pull/5713
Author: ltzmaxwell | Base: master | Files: 16 | +451 -26
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `4646c008b` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5713 4646c008b`

**Verdict: APPROVE** — closes a Go-incompatibility where GnoVM silently returned a pointer-identity result instead of panicking on uncomparable interface comparisons; the fix is correctly gated on static interface type, matches Go's runtime message and short-circuit semantics, and all 14 new filetests plus adversarial probes pass. Only nits below.

## Summary

Go panics with `comparing uncomparable type X` when two interface values whose dynamic type is a `map`/`slice`/`func` are compared with `==`, even when both wrap a nil value. GnoVM did not: in production (non-`debug`) builds, `isEql` fell through to `return lv.V == rv.V` for those kinds, silently producing a pointer-identity boolean instead of trapping. That is a deterministic-but-wrong divergence from Go, reachable from any realm doing `iface == iface`, `[N]iface{} == ...`, `struct{ iface } == ...`, or `switch iface { case ... }`.

The fix threads a `viaInterface`-style flag (`ifaceDyn Type`) through `isEql`. The flag is non-nil only when at least one operand has a static interface type (derived from `ATTR_TYPEOF_VALUE`), and carries the dynamic type to name in the panic. It is re-captured at every interface boundary (array element, struct field) so the message names the inner dynamic type Go would name, not the outer carrier. The map/slice/func arms now panic when `ifaceDyn != nil`, and arrays panic early on a structurally uncomparable element type (covers zero-length arrays Go still rejects). Direct `m == nil` is unaffected because neither operand is statically interface-typed.

```
x == y   (static types)        ifaceDyn          behavior
------------------------------ ----------------- --------------------------
map     == nil                 nil               return ptr-eq (no panic)
iface   == iface (both map)    map type          PANIC "...map[K]V"
[N]iface{map} == ...           re-captured: map  PANIC "...map[K]V" (inner)
iface([N]map{}) == ...         outer array type  PANIC "...[N]map[K]V"
iface(mapA) == iface(sliceB)   differ -> false   return false (checkSame)
```

## Glossary

- `isEql` — recursive structural equality used by `==`/`!=` and `switch` case matching.
- `ifaceDyn` — new param: the dynamic type carried by an interface header at the current recursion level, or nil when no via-interface context applies.
- `ATTR_TYPEOF_VALUE` — preprocess-time attribute holding an expression's static type.
- `isComparable` — type-level predicate ([`type_check.go:1168`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/type_check.go#L1168) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/type_check.go#L1168)); recurses through arrays/structs, treats interface as comparable.

## Fix

Before: [`isEql`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L454) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L454) had no interface context; map/slice/func returned `lv.V == rv.V` in production. After: `doOpEql`/`doOpNeq` compute `ifaceCmpDynType(bx, lv)` ([`op_binary.go:114-135`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L114-L135) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L114-L135)) and pass it down; map/slice/func ([`op_binary.go:594-607`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L594-L607) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L594-L607)) panic when `ifaceDyn != nil`. The load-bearing gate is the static-interface check: a non-interface operand never carries `ifaceDyn`, so `m == nil` and concrete-array equality keep their old behavior. `doOpSwitchClauseCase` wires the same flag using the tag's dynamic type ([`op_exec.go:981-991`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_exec.go#L981-L991) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_exec.go#L981-L991)).

## Critical (must fix)

None.

## Warnings (should fix)

None. Verified the design against Go on the corner cases the threading could get wrong:

- Asymmetric operands (only RHS statically interface, LHS concrete struct with an uncomparable iface field) panic naming the inner `map[int]int`, via the field-level re-capture, not the outer struct named by `ifaceDyn = lv.T`. Confirmed by a local adversarial filetest.
- Differing uncomparable dynamic types (`iface(map) == iface(slice)`) return `false` with no panic, because `checkSame` short-circuits ([`op_binary.go:464`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L464) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L464)) before the kind switch. Matches `go run` output.
- Tag-less `switch {}` is normalized to `X = Nx("true")` ([`go2gno.go:511-521`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/go2gno.go#L511-L521) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/go2gno.go#L511-L521)), so `hasInterfaceStaticType(ss.X)` is false and the switch path adds no false panics.

## Nits

- [`op_binary.go:118-123`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L118-L123) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L118-L123) — `ifaceCmpDynType` returns `lv.T` (left operand's dynamic type) whenever *either* operand is statically interface-typed. When only the right operand is interface-typed and the left is concrete, `ifaceDyn` is seeded with the left concrete type rather than the interface's dynamic type. It happens not to matter: top-level map/slice/func operands are rejected at preprocess unless interface-typed, and nested uncomparables are re-captured at their boundary, so every reachable panic still names the correct type (verified). A one-line comment noting "either-operand seeding is safe because the genuinely-named type is always re-derived at the boundary or is the top-level dynamic type" would save the next reader the same derivation.
- [`op_binary.go:584-593`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L584-L593) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L584-L593) — the new `case InterfaceKind` arm is reached only for a nil interface value held in a struct field / array element; the top-level nil-vs-nil interface compare goes through the `IsUndefined()` early return ([`op_binary.go:457-463`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L457-L463) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L457-L463)) instead. The arm is correct and defensive, just narrower in reach than the comment implies — fine to leave.
- [`op_exec.go:984` and `op_exec.go:994`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_exec.go#L984) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_exec.go#L984) — `ss` is fetched via `PeekStmt1()` at line 984 and re-fetched via `PopStmt()` at line 994 inside the matched branch (shadowing). Pre-existing shadow pattern in the function; not worth churning.

## Missing Tests

- [`op_binary.go:594-607`](https://github.com/gnolang/gno/blob/4646c008b/gnovm/pkg/gnolang/op_binary.go#L594-L607) · [↗](../../../../../.worktrees/gno-review-5713/gnovm/pkg/gnolang/op_binary.go#L594-L607) — no filetest asserts the negative for `slice`/`func` carried directly by an interface (only `map` is covered at the top level by `cmp_uncomp_iface_map*`). `cmp_uncomp_iface_func_arg.gno` covers slice/func/struct-with-slice via the `noCmp` helper, but only checks that a panic happens, not the named type. A `slice`/`func` analogue of `cmp_uncomp_iface_map.gno` asserting `comparing uncomparable type []int` / `func()` would lock the message for those kinds. Low priority — the code path is shared with map and the kind is irrelevant to the message.

## Suggestions

None.

## Questions for Author

- AGENTS.md asks for an ADR on non-trivial AI-assisted PRs. This change is human-authored (ltzmaxwell) so an ADR is not required, but the threading rationale (why re-capture at each boundary, why seed from either operand) is exactly the kind of subtlety an ADR or a longer doc-comment preserves. Worth a short `gnovm/adr/pr5713_*.md`?
