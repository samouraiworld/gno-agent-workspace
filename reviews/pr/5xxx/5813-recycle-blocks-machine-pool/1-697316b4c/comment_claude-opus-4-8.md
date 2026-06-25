# Review: PR #5813
Event: APPROVE

## Body
Looks good. Verified on 697316b4c. Moving `noRecycle` back into `bodyStmt`, where it started, makes the FALLTHROUGH clause's `bodyStmt` reassignment clear a defer's flag, and `fallthrough0.gno` then trips the `releaseBlock` defer-parent assert under `-tags debugAssert`. The dedicated `Block` field is what closes that hole.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5813-recycle-blocks-machine-pool/1-697316b4c/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
