# Review: PR #5504
Event: APPROVE

## Body
Looks good. Verified on 6d6ab81a5: the royalty output matches a side-by-side Go mirror across every rate in the table including the overflow fallback, and removing the `derr =` capture in the invalid-token-ID test makes its assertion compare nil so the case stops guarding anything.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5504-royalty-bps-eip2981/2-6d6ab81a5/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno:77 [↗](../../../../../.worktrees/gno-review-5504/examples/gno.land/p/demo/tokens/grc721/grc721_royalty.gno#L77)
The fallback returns a less precise royalty than the exact formula, with no comment marking that as intentional. It only triggers above a sale price of MaxInt64 / 10000, so no real caller reaches it, but a line saying the fallback trades precision for range would stop a future reader from treating the divergence as a bug.
