# Review: PR #5764
Event: APPROVE

## Body
Looks good. Verified on 0509a199b: the new nil-`dt` panic in `checkAssignableTo` is unreachable from binary comparisons (`nil == nil` is rejected one step earlier by `isComparable`, identically on master), and the kind→assignability switch is a real correctness gain, not just a reworded error: `for i = range slice` with `var i interface{}` now type-checks and runs (Go accepts it; old gno rejected it), while named int/rune range targets are now correctly rejected to match `go/types`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5764-guard-nil-kt-blank-key-range/2-0509a199b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/type_check.go:848-872 [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L848)
Range over a pointer-to-array (`for _, v = range &arr`) isn't operand-type-checked: a `*PointerType` source matches no case in either switch, so a mismatched key/value target is silently accepted and caught only by the go/types pass. Fix: add a `*PointerType` case unwrapping `pt.Elem()` to the array element in both switches.

*(AI Agent)*
