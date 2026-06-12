# Review: PR #5790
Event: APPROVE

## Body
Looks good. Verified on the current head (44cd6b494): the four new filetests each fail when only `machine.go`+`op_assign.go` are reverted to the merge-base and pass with the fix, the single-LHS fast path and the `doOpRef` read path are semantically unchanged, and the benchmarks confirm 0 allocs/op. Two leftover comments name `PopAsPointer2` as the only non-nil-realm write path into `GetPointerAtIndex`, which is no longer true now that `doOpAssign`'s multi-LHS loop calls `resolvePointer` directly — behavior is fine, just the comments are stale (inline).

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5790-tuple-assign-inplace/1-44cd6b494/review_claude-opus-4-8_davd-gzl.md

*(AI Agent)*

## gnovm/pkg/gnolang/values.go:2054 [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/values.go#L2054)
The comment says only `PopAsPointer2` reaches this map-key attach with a non-nil realm, but the multi-LHS assign now gets here via `resolvePointer` called straight from `doOpAssign`. The readonly check still runs, so behavior is correct; the comment is just out of date. Worth widening it to mention the multi-LHS write path.

*(AI Agent)*

## gnovm/pkg/gnolang/realm.go:271 [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/realm.go#L271)
Same stale claim in the `DidUpdate` caller list: `doOpAssign`'s multi-LHS loop is now a second non-nil-realm write path into `GetPointerAtIndex`, alongside `PopAsPointer2`. Add it to the list.

*(AI Agent)*

## gnovm/pkg/gnolang/op_assign.go:80-82 [↗](../../../../../.worktrees/gno-review-5790/gnovm/pkg/gnolang/op_assign.go#L80-L82)
The "value stack must not grow mid-loop" invariant is asserted only under `debug`, so a future change that makes `resolvePointer`/`Assign2`/`DidUpdate` push onto `m.Values` would silently corrupt the aliased operand frames in production builds. Safe today (none of those paths push); flagging the latent fragility for whoever touches them next. No change needed here.

*(AI Agent)*
