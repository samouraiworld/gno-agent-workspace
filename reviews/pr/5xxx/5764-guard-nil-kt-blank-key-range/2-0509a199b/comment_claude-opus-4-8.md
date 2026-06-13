# Review: PR #5764
Posted: https://github.com/gnolang/gno/pull/5764#pullrequestreview-4492022323
Event: APPROVE

## Body
Looks good. Verified on 0509a199b.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5764-guard-nil-kt-blank-key-range/2-0509a199b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/type_check.go:848-872 [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L848) [posted](https://github.com/gnolang/gno/pull/5764#discussion_r3408627170)
`for _, v = range &arr` isn't type-checked: a [pointer-to-array](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/preprocess.go#L901) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/preprocess.go#L901) hits no case in either switch, so a mismatched key/value target is silently accepted, caught only by go/types. Fix: add a `*PointerType` case to both switches, unwrapping to the array element. Filetest: [`range_ptr_array_value.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5764-guard-nil-kt-blank-key-range/2-0509a199b/tests/range_ptr_array_value.gno) · [↗](tests/range_ptr_array_value.gno)

*(AI Agent)*
