# PR #5765: fix(gnovm): evaluate LHS operands then assign left-to-right in tuple assignment

URL: https://github.com/gnolang/gno/pull/5765
Author: ltzmaxwell | Base: master | Files: 3 | +46 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `28164a5d1` (stale — +3 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5765 28164a5d1`

**Verdict: APPROVE** — correct, Go-spec-faithful fix for tuple-assignment evaluation order; verified parity against equivalent Go on five shapes (last-writer-wins aliasing, mixed-arity LHS, operand side-effects, 8-wide aliasing stress, mid-statement panic). Reorder is contained to `doOpAssign`, gas accounting is unchanged, and the `frames` buffer is stack-allocated on the hot path. Open items are non-blocking: a hidden 3-way sync invariant (`numStackValuesForPointer` ↔ `PushForPointer` ↔ `PopAsPointer2`) and a dropped single-LHS fast path. The one e2e CI failure (`scenario-01-four-validators-reset-three`) is a TM2 consensus scenario unrelated to this diff.

## Summary

In `m[k], *p = 42, 2` with `p == nil`, the old `doOpAssign` resolved and assigned LHS targets right-to-left, so `*p` was dereferenced (panic) before `m[k]` was ever written — leaving the map untouched. Go's spec (§Assignment statements) requires operands and RHS to be evaluated first, then assignments to proceed left-to-right; the equivalent Go program leaves `m[k] == 42` (the panic happens while *assigning* the later target). The fix pops each LHS's already-evaluated operand frame off the value stack in reverse, then re-pushes / resolves (`PopAsPointer`) / assigns one target at a time in forward LHS order, so a mid-statement nil-deref / OOB / nil-map leaves earlier assignments intact.

```
Old (reverse resolve+assign):   i = N-1 .. 0:  PopAsPointer(L_i); assign     → write order N-1..0, panic on L_i kills L_0..L_{i-1}
New (buffer frames, fwd order): i = N-1 .. 0:  frames[i] = pop operand vals  (LIFO off the value stack)
                                i = 0 .. N-1:  re-push frames[i]; PopAsPointer(L_i); assign  → write order 0..N-1
```

## Glossary

- `PushForPointer` — during `op_exec`, pushes the operand sub-expressions of an LHS (e.g. `X` and `Index` for `IndexExpr`) onto the value stack as `OpEval`s; their side effects run before `doOpAssign`.
- `PopAsPointer` / `PopAsPointer2` — consume an LHS's operand values off the value stack and construct the `PointerValue` to write through; for maps it creates the entry, for `*p`/nil it panics.
- `numStackValuesForPointer` — new helper: how many `TypedValue`s `PushForPointer` left on the stack for a given LHS expr (NameExpr 0, IndexExpr 2, others 1).
- frame — the slice of raw operand `TypedValue`s belonging to one LHS target, buffered so it can be re-pushed in the right order.

## Fix

Relationship to [#5720](https://github.com/gnolang/gno/pull/5720) (reviewed, **not merged** — absent from `origin/master`): #5720 resolved *all* LHS pointers up front into `lvs[]`, then assigned forward. That fixes the aliasing/write-order bug but is stricter than Go on panics: it leaves *no* partial assignment when a later LHS panics during resolution. #5765 supersedes it with a finer interleave — buffer the raw operand frames, then resolve-and-assign one target at a time — so a panic on `L_i` leaves `L_0..L_{i-1}` written, matching Go exactly. Both produce the same last-writer-wins result for the aliasing case ([`a, a, a = 1, 2, 3`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/op_assign.go#L35-L44) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/op_assign.go#L35-L44) → `a == 3`); #5765 additionally gets the panic-partiality right.

The reverse-pop is load-bearing: operand frames sit on the value stack bottom-to-top as `L_0 … L_{N-1}` (RHS already popped at [`op_assign.go:29`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/op_assign.go#L29) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/op_assign.go#L29)), so to slice them apart they must be popped top-first ([`op_assign.go:32-34`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/op_assign.go#L32-L34) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/op_assign.go#L32-L34)). The forward loop ([`op_assign.go:35-44`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/op_assign.go#L35-L44) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/op_assign.go#L35-L44)) then re-pushes each frame just before `PopAsPointer` consumes it, exactly mimicking the stack layout that resolution expects.

Crucially, this reorder does **not** change operand-evaluation order: the index/selector/`X` sub-expressions and the RHS are all evaluated during `op_exec` (`PushForPointer` at [`op_exec.go:518-521`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/op_exec.go#L518-L521) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/op_exec.go#L518-L521)), before `doOpAssign` runs. The PR only reorders pointer *resolution* (map-entry creation, the deref check) and the writes — which is the Go-spec-relevant ordering.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/machine.go:2616-2630`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/machine.go#L2616-L2630) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/machine.go#L2616-L2630) — `numStackValuesForPointer` introduces a silent 3-way invariant.
  <details><summary>details</summary>

  The arities here must stay byte-for-byte aligned with the pushes in `PushForPointer` ([`machine.go:2586-2614`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/machine.go#L2586-L2614) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/machine.go#L2586-L2614)) and the pops in `PopAsPointer2` ([`machine.go:2712-2775`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/machine.go#L2712-L2775) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/machine.go#L2712-L2775)). They match today (NameExpr 0, IndexExpr 2, SelectorExpr/StarExpr/CompositeLitExpr 1, default panic). The comment names the constraint, which is good, but the count is duplicated structural knowledge: a future LHS expr type added to `PushForPointer` without touching this helper desyncs the value stack silently — wrong pointer resolved, no panic, a determinism bug that only some operand shapes expose. Cheap guard: a table-driven test that, for each LHS expr kind, asserts `numStackValuesForPointer == (stack depth after PushForPointer)`. Locks all three together without restructuring.
  </details>

## Missing Tests

- [`gnovm/tests/files/assign_tuple_order.gno`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/tests/files/assign_tuple_order.gno) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/tests/files/assign_tuple_order.gno) — the single filetest covers only the nil-deref-on-`*p` shape; the headline aliasing fix and multi-arity reorder are untested.
  <details><summary>details</summary>

  The shipped test proves panic-partiality for `m[k], *p`. It does not exercise: (a) the last-writer-wins ordering that is the *other* half of the Go-spec fix (`m[k], m[k] = 1, 2` → `2`; `*p, *p = 1, 2` → `2`); (b) a multi-target statement mixing `IndexExpr` (2 operand vals), `SelectorExpr`/`StarExpr` (1) and `NameExpr` (0) in one go, which is what stresses the `frames`/re-push reorder; (c) OOB and nil-map panic variants alongside nil-deref. I verified all of these pass and match Go — adding them as filetests would gate future regressions of the reorder, not just the one panic shape. Adversarial tests written and confirmed green: [`assign_tuple_alias_order.gno`](tests/assign_tuple_alias_order.gno), [`assign_tuple_mixed_lhs.gno`](tests/assign_tuple_mixed_lhs.gno), [`assign_tuple_alias_stress.gno`](tests/assign_tuple_alias_stress.gno), [`assign_tuple_operand_sideeffects.gno`](tests/assign_tuple_operand_sideeffects.gno).
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/op_assign.go:31`](https://github.com/gnolang/gno/blob/28164a5d1/gnovm/pkg/gnolang/op_assign.go#L31) · [↗](../../../../../.worktrees/gno-review-5765/gnovm/pkg/gnolang/op_assign.go#L31) — single-LHS no longer gets a fast path; #5720 had one.
  <details><summary>details</summary>

  `x = expr` (one NameExpr LHS) is the overwhelmingly dominant assignment shape, and it now runs the full buffer machinery: `make([][]TypedValue, 1)`, a reverse-pop of a zero-length frame, an empty inner push loop. Escape analysis (`go build -gcflags=-m`) reports the `frames` make at this line "does not escape", so there is **no heap allocation** on the hot path — the cost is a stack slice header plus the extra loop bookkeeping, not GC pressure. Minor, but #5720's explicit `len(s.Lhs) == 1` branch sidestepped it entirely. If a microbench shows it, restore a one-liner fast path: `if len(s.Lhs) == 1 { rvs := m.PopValue(); lv := m.PopAsPointer(s.Lhs[0]); … ; return }`. Skip if the numbers don't move; no assign benchmark exists in this tree (the `BenchmarkOpAssign_*` referenced in the #5720 review was added by #5720, which isn't merged).
  </details>
- ADR: `AGENTS.md:83-101` requires an ADR for non-trivial AI-assisted PRs. The fix is small but the change touches VM evaluation-order semantics and adds a sync-sensitive helper. If AI-assisted, add `gnovm/adr/pr5765_tuple_assign_order.md` capturing the bug shape, the reverse-pop / forward-assign-with-frames decision, and the relationship to #5720. If not, ignore. (Same ask was open on #5720.)

## Questions for Author

- This supersedes #5720, correct? If so, will #5720 be closed, and should its `assign41.gno` ordering tests (which #5765 doesn't carry over) be folded into the new filetest?
- Is this AI-assisted? If so, an ADR is required per `AGENTS.md`.

## Verification

Equivalent-Go parity confirmed for every shape below; Gno output matched Go (`go run`) exactly. From a local clone of gnolang/gno:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5765 -R gnolang/gno

# Shipped filetest (nil-deref leaves earlier assignment intact):
go test -run 'TestFiles/assign_tuple_order.gno$' ./gnovm/pkg/gnolang/

# Last-writer-wins aliasing (Go: 3 30 300):
cat > gnovm/tests/files/_alias.gno <<'EOF'
package main
func main() {
	m := map[string]int{}
	arr := []int{0}
	x := 0
	p := &x
	m["k"], m["k"], m["k"] = 1, 2, 3
	arr[0], arr[0], arr[0] = 10, 20, 30
	*p, *p, *p = 100, 200, 300
	println(m["k"], arr[0], x)
}
// Output:
// 3 30 300
EOF
go test -run 'TestFiles/_alias.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/_alias.gno
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	(assign_tuple_order.gno)
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	(_alias.gno)
```

Note: `go test ./gnovm/pkg/gnolang/ -run Files -test.short` shows one failure, `TestFiles/types/or_f0.gno` — a `go/types` error-message-format mismatch (`cannot convert ... to type interface{...}` vs `invalid operation ... mismatched types`) under local Go 1.26.3. It reproduces identically with this PR's `op_assign.go`/`machine.go` reverted to the PR base, so it is a pre-existing toolchain artifact, unrelated to this diff.
