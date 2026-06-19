# Review: PR #5812
Event: APPROVE

## Body
Looks good. Verified on c845152ad: removing the array bounds checks in `GetByteAtIndexInt` makes `recover25b`'s out-of-range read crash the runner as an uncatchable Go panic (the #5738 regression), so the checks are load-bearing; a side-by-side gno-vs-Go run of byte index, range, and `copy` (including an overlapping `copy(o[2:], o[:6])`) produces identical output; and the local `alloc_0/3/4` MemStats mismatches reproduce on pristine master, so they are not from this PR.

One stale comment, in a file the diff doesn't touch: [`recover25b.gno:2`](https://github.com/gnolang/gno/blob/c845152ad/gnovm/tests/files/recover25b.gno#L2) [↗](../../../../../.worktrees/gno-review-5812/gnovm/tests/files/recover25b.gno#L2) names `ArrayValue.GetPointerAtIndexInt2`, renamed here to `GetElementPointer`, and `a[i]` on a byte array now routes through `GetByteAtIndexInt` anyway, so point it there at the path it actually guards.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5812-heap-boxed-byte-access/1-c845152ad/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:2010 [↗](../../../../../.worktrees/gno-review-5812/gnovm/pkg/gnolang/values.go#L2010)
Only the array branch here has a recover guard (`recover25b`); the string and byte-slice branches carry the same out-of-range `*Exception` checks but nothing exercises their recoverability, so a later edit dropping them would slip through. Two small `recover()` filetests, `s[i]` and `b[i]` on a byte slice both out of range, would cover them; I confirmed both currently recover with the right messages.
