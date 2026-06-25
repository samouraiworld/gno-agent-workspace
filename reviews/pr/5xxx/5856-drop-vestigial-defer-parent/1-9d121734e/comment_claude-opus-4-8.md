# Review: PR #5856
Event: APPROVE

## Body
Ran all four `defer_block_recycle.gno` cases as a Go program on 9d121734e: output is byte-identical to the filetest, including the named-return-through-a-recycled-block case. Ran the same filetest and `fallthrough0` under `-tags debugAssert`: the new GarbageCollect recycle-safety assert does not fire. Measured `sizeof(Block)` at 528, which the alloc.go `init` assert enforces, so a later field add that desyncs the constant fails at startup.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5856-drop-vestigial-defer-parent/1-9d121734e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
