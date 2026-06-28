# Review: PR #5813
Posted: https://github.com/gnolang/gno/pull/5813#pullrequestreview-4587606679
Event: APPROVE

## Body
Verified on becc5fa87.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5813-recycle-blocks-machine-pool/2-becc5fa87/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## SKIP gnovm/pkg/gnolang/op_call.go:678 [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/op_call.go#L678)
The no-recycle exclusion for a defer-origin block hangs on this one `setNoRecycle()`. `Defer.Parent` feeds only the recount GC visitor, never defer execution, so a future defer path that omits this call would recycle a still-referenced block in production, where the `debugAssert` guard is compiled out. Acceptable here; [#5856](https://github.com/gnolang/gno/pull/5856) drops `Defer.Parent`, removing the flag and the field.

## gnovm/pkg/gnolang/values.go:2487-2488 [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/values.go#L2487) [posted](https://github.com/gnolang/gno/pull/5813#discussion_r3488472028)
The comment says allocation accounting "is by numNames, independent of capacity," but `newPooledBlock` charges `AllocateBlock(max(numNames, 14))`, by capacity. Update the sentence to match.
