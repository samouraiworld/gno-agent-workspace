# Review: PR #5790
Posted: https://github.com/gnolang/gno/pull/5790#pullrequestreview-4491657042
Event: APPROVE

## Body
Looks good. Verified on 44cd6b494: each of the four new filetests fails with `machine.go`+`op_assign.go` reverted to merge-base, so they guard this fix; the single-LHS fast path and `doOpRef` read path are unchanged; benchmarks show 0 allocs/op.

Two stale comments now omit the multi-LHS write path this PR adds; both sit outside the diff, so flagging here rather than inline:
- [values.go:2054](https://github.com/gnolang/gno/blob/44cd6b494/gnovm/pkg/gnolang/values.go#L2054) [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/values.go#L2054): the comment claims only `PopAsPointer2` reaches this map-key attach with a non-nil realm, but multi-LHS assign now arrives via `resolvePointer` from `doOpAssign`; widen it to cover the multi-LHS write path.
- [realm.go:271](https://github.com/gnolang/gno/blob/44cd6b494/gnovm/pkg/gnolang/realm.go#L271) [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/realm.go#L271): the `DidUpdate` caller list omits `doOpAssign`'s multi-LHS loop, now a second non-nil-realm write path into `GetPointerAtIndex`; add it.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5790-tuple-assign-inplace/1-44cd6b494/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## SKIP gnovm/pkg/gnolang/op_assign.go:80-82 [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/op_assign.go#L80-L82)
The "value stack must not grow mid-loop" invariant is asserted only under `debug`. If a future change makes `resolvePointer`/`Assign2`/`DidUpdate` push onto `m.Values`, production builds would silently corrupt the aliased operand frames. Safe today; flagging for whoever touches these paths next. No change needed.

*(AI Agent)*
