# PR #5790: fix(gnovm): tuple assignment — resolve LHS in-place, assign left-to-right

URL: https://github.com/gnolang/gno/pull/5790
Author: ltzmaxwell | Base: master | Files: 8 | +406 -20
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `44cd6b494` (stale — +4 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5790 44cd6b494`

**TL;DR:** When Gno code assigns several things at once (`a, b = 1, 2`), the VM was committing the writes right-to-left, the reverse of Go. This breaks two visible cases: `a, a, a = 1, 2, 3` left `a == 1` instead of `3`, and a panic partway through a multi-assign (e.g. `m[k], *p = 42, 2` where `*p` is a nil deref) discarded the earlier `m[k]` write instead of keeping it. This PR makes the VM resolve and commit each target left-to-right, matching Go, with no extra memory allocation.

**Verdict: APPROVE** — correct fix for golang/go#23017 semantics, the four new filetests each fail on master and pass here, single-LHS fast path and the read path (`doOpRef`) are semantically unchanged, 0 allocs/op confirmed; only two stale code comments left behind (Nits).

## Summary

`doOpAssign` previously looped `for i := len-1; i>=0; i--`, popping and writing the rightmost LHS first. Go requires left-to-right: last write wins on aliased targets, and a mid-assignment panic must leave earlier writes committed. The fix pops the RHS block and the LHS-operand block off `m.Values` once, then resolves+assigns each LHS in increasing index order, reading each LHS's operand frame in place from a slice window over the popped region. `PopAsPointer2`'s body is extracted into a pure `resolvePointer(lx, operands)` so both the stack path (`PopAsPointer2`, unchanged callers) and `doOpAssign` (in-place) share one resolver. Single-LHS keeps a dedicated fast path; multi-LHS adds ~6% (IndexExpr) to ~13% (NameExpr) CPU per op and 0 allocs.

## Glossary
- **LHS operand frame** — the value-stack entries `PushForPointer` pushes to later resolve one assignment target (IndexExpr pushes 2: X then Index; Selector/Star/CompositeLit push 1; Name pushes 0).
- **`m.Values`** — the VM's value stack; `PopValues(n)` returns a slice aliasing its live backing array, so the popped region is reused by any later push.

## Fix

Before, [`op_assign.go` looped right-to-left](https://github.com/gnolang/gno/blob/b738c108/gnovm/pkg/gnolang/op_assign.go) so the last-resolved LHS (index 0) was written last and any panic on a higher index discarded lower-index writes. After, [`op_assign.go:51-83`](https://github.com/gnolang/gno/blob/44cd6b494/gnovm/pkg/gnolang/op_assign.go#L51-L83) · [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/op_assign.go#L51-L83) pops `rvs` (RHS) then `lhsOperands` (all LHS frames) and resolves+assigns each LHS forward from `lhsOperands[offset:offset+sz]`. The load-bearing constraint: `rvs` and `lhsOperands` are views into the live `m.Values` backing array, so nothing in the loop may push onto the stack or it overwrites not-yet-resolved frames; [`resolvePointer`](https://github.com/gnolang/gno/blob/44cd6b494/gnovm/pkg/gnolang/machine.go#L2784) · [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/machine.go#L2784) and `Assign2` run no bytecode, so they never push, and a `debug`-only assert guards the invariant.

## Benchmarks / Numbers

Measured on the PR head (`go test -bench BenchmarkDoOpAssign -benchmem -benchtime=200x`):

| bench | ns/op | allocs/op |
|---|---|---|
| Name_N1 | 171 | 0 |
| Name_N2 | 312 | 0 |
| Name_N3 | 354 | 0 |
| Name_N5 | 597 | 0 |
| Index_N1 | 238 | 0 |
| Index_N2 | 465 | 0 |
| Index_N3 | 628 | 0 |
| Index_N5 | 821 | 0 |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[comment names a function that's no longer the only write path]** [`values.go:2054`](https://github.com/gnolang/gno/blob/44cd6b494/gnovm/pkg/gnolang/values.go#L2054) · [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/values.go#L2054) — comment says "Only PopAsPointer2 (write path) reaches here with non-nil rlm," but the multi-LHS write path now reaches the map `GetPointerAtIndex` via `resolvePointer` called directly from `doOpAssign`, not through `PopAsPointer2`. Behavior is correct (the readonly check still runs, in `resolvePointer`'s map branch and the `if ro` in the loop); only the comment is stale. Fix: widen it to "the write paths (`PopAsPointer2` and `doOpAssign`'s multi-LHS loop) reach here with non-nil rlm and check readonly first."
- **[same stale claim in DidUpdate's caller list]** [`realm.go:271`](https://github.com/gnolang/gno/blob/44cd6b494/gnovm/pkg/gnolang/realm.go#L271) · [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/realm.go#L271) — the "Indirect callers via GetPointerAtIndex" list names `PopAsPointer2 (write path)` as the only non-nil-realm path; `doOpAssign`'s multi-LHS loop is now a second one. Fix: add `doOpAssign multi-LHS (write path): checks readonly via resolvePointer's map branch / the loop's ro check` to the list.

## Missing Tests
None. The four new filetests each fail on master and pass on this head (confirmed: reverting only `machine.go`+`op_assign.go` to the merge-base makes all four fail — `assign40` panics with `a==1`, `assign_tuple_order` prints `0 0`, `assign_tuple_order2` prints `0`). The Go-level unit tests cover the NameExpr last-write-wins and the IndexExpr per-frame offset arithmetic. `assign_tuple_order2.gno` additionally covers a panic during LHS *resolution* (out-of-range `s[9]`), distinct from `assign_tuple_order.gno`'s panic during the *assignment* (nil `*p`).

## Suggestions
- `op_assign.go:80-82` · [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/op_assign.go#L80-L82) — the no-push aliasing invariant is enforced only under `debug`. Confirmed behaviorally safe at this head: `resolvePointer`, `Assign2`, `GetPointerAtIndex`, and `DidUpdate` contain no `PushValue`/`PushOp`/`PushExpr`, so production builds are correct today. The note is forward-looking: a future change that makes any of those push onto `m.Values` would silently corrupt the aliased frames in non-debug builds, and the assert wouldn't catch it in production. No action required for this PR; flagging the latent fragility for whoever touches those paths next.

## Open questions
None.
