# Review: PR #5818
Event: APPROVE

## Body
Looks good. Verified on the current head (cd7b76ca9): the issue MRE returns `847895691526144` for both add and sub via `gno run`; the generator reproduces both committed files byte-for-byte under a different Go toolchain (anchor guard works); and a sweep of 30M+ add/sub/mul/div/narrowing pairs shows 0 mismatches vs hardware, while reverting the fix reproduces the bug.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5818-softfloat-subnormal-cancellation/1-cd7b76ca9/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64_test.go:52-53 [↗](../../../../../.worktrees/gno-review-5818/gnovm/pkg/gnolang/internal/softfloat/runtime_softfloat64_test.go#L52)
These regression operands are bare `Float64frombits` literals, unlike the neighbours that carry their value in a comment (`// first normal`). Appending the decimals (`-2.662e-301` / `2.662e-301`) makes the test data self-documenting.

*(AI Agent)*
