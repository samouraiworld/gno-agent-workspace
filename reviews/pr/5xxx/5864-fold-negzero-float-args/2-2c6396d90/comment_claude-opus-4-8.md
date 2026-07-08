# Review: PR [#5864](https://github.com/gnolang/gno/pull/5864)
Event: APPROVE

## Body
Verified on 2c6396d90. Reverting the fold reproduces negative zero for -0, -0.0, and the float32 underflow -1e-50. Every NaN spelling, including a payload NaN, collapses to Go's canonical NaN bits. The +Inf, signaling NaN, and overflow literals are rejected. Accepting NaN and Inf therefore leaves no malleable encoding. The VM produces NaN and Inf at runtime, so a realm float parameter can already hold them.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5864-fold-negzero-float-args/2-2c6396d90/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/sdk/vm/convert_goparity_test.go:25-36 [↗](../../../../../.worktrees/gno-review-5864-h/gno.land/pkg/sdk/vm/convert_goparity_test.go#L25)
Missing test: the float32-underflow route into the fold is uncovered. The zeros table only tests strings that are already zero. No case feeds a nonzero magnitude that underflows to -0 at float32 precision, like `-1e-50`.

<details><summary>test cases</summary>

`float32(-1e-50)` is `0x0` in Go, and the arg path folds it the same way, so this is a Go-parity case that belongs in this file. Add after the zeros loop:

```go
// "-1e-50" is a normal negative float64 that underflows to -0 at float32
// precision; Go folds float32(-1e-50) to +0, and so must the arg path.
if got := convertFloat("-1e-50", 32); math.Signbit(got) {
	t.Errorf("convertFloat(%q, 32) has sign bit set, want +0", "-1e-50")
}
```
</details>
