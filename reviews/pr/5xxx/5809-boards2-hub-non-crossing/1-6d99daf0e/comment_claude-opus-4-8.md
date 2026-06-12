# Review: PR #5809
Event: APPROVE

## Body
Looks good. Verified on the current head (6d99daf0e) that this is a behavior-preserving move: every retained `Get*` view returns the same data as before, hidden-thread lookup still works, and the only deliberate change is dropping the caller-namespace gate on reads, matching the new "no caller authorization" header.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5809-boards2-hub-non-crossing/1-6d99daf0e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## examples/gno.land/r/gnoland/boards2/v1/hub.gno:142-145 [↗](../../../../../.worktrees/gno-review-5809/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L142)
The doc says GetComments returns "all thread comments and replies", but it iterates only `thread.Replies`, which holds top-level comments, so a thread with one comment and one nested reply returns one item. Behavior is unchanged from before the PR; only the doc is off. Consider "top-level thread comments" and pointing to `Comment.IterateReplies` for the rest.

*(AI Agent)*

## examples/gno.land/r/gnoland/boards2/v1/hub.gno:16-22 [↗](../../../../../.worktrees/gno-review-5809/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L16)
The `Member` and `Flag` type aliases are re-exported but no realm function returns them: `GetMembers` returns `[]boards.User` and `GetFlags` returns `[]boards.Flag`. Worth confirming the aliases are intentional API surface rather than leftover.

*(AI Agent)*
