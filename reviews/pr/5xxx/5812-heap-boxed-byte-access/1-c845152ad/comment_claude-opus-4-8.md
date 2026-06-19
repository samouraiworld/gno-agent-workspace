# Review: PR #5812
Event: APPROVE

## Body
Looks good.

Verified on c845152ad. Removing the array bounds checks in `GetByteAtIndexInt` makes `recover25b`'s out-of-range read crash the runner as an uncatchable Go panic, the #5738 regression, so the checks are load-bearing. A side-by-side gno-vs-Go run of byte index, range, and overlapping `copy` gave identical output. The local `alloc_0/3/4` MemStats mismatches reproduce on pristine master, so they are not from this PR.

Nit, [`recover25b.gno:2`](https://github.com/gnolang/gno/blob/c845152ad/gnovm/tests/files/recover25b.gno#L2) [↗](../../../../../.worktrees/gno-review-5812/gnovm/tests/files/recover25b.gno#L2): the comment names `ArrayValue.GetPointerAtIndexInt2`, renamed here to `GetElementPointer`. `a[i]` on a byte array now routes through `GetByteAtIndexInt`, so the name points at the wrong path.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5812-heap-boxed-byte-access/1-c845152ad/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:2010 [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/values.go#L2010)
Optional: only the array branch here has a recover guard, `recover25b`. The string and byte-slice branches carry the same out-of-range `*Exception` checks, but nothing exercises their recoverability, so a later edit dropping them slips through.
