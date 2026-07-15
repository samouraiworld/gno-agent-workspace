# Review: PR [#5867](https://github.com/gnolang/gno/pull/5867)
Event: APPROVE

## Body
Verified on 79ebc4058: reverting `bigdecToFloat32` to the 24-bit intermediate reproduces the subnormal float32 double round, and the direct path matches the Go compiler across an 800-case float32/float64 scan including the subnormal band. The literal-parse exponent guard changes no in-band literal's representation, so `TestAppHashCrossrealm38` holds unchanged.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/5-79ebc4058/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values_conversions.go:1500-1509 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1500)
Missing test: the subnormal float32 path this commit fixed has no filetest, so a future refactor of `bigdecToFloat32` could reintroduce the double round with the suite still green. A regression filetest asserting `math.Float32bits(0x1.4p-148 + 0x1p-200) == 3` closes it: it passes here and gives `bits: 2` on the 24-bit path.

<details><summary>test cases</summary>

Ready-to-add under `gnovm/tests/files/types/`, green at 79ebc4058, red (`bits: 2`) at adbb5ac60: [`bigdec_float32_subnormal.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/5-79ebc4058/tests/bigdec_float32_subnormal.gno) · [↗](tests/bigdec_float32_subnormal.gno).

```go
package main

import "math"

func main() {
	const s float32 = 0x1.4p-148 + 0x1p-200
	println("bits:", math.Float32bits(s))
	println("value:", s)
}

// Output:
// bits: 3
// value: 4e-45
```
</details>

## gnovm/pkg/gnolang/values.go:198-227 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values.go#L198)
Missing test: the `f:`-prefixed `big.Float` amino form has no coverage, though a package-level `const X = 1e5000` persists through it. A float case can't be appended to `parity_test.go` as-is: `AssertCodecParity`'s deep-equal step trips on a decoded `big.Float` normalizing its mantissa slice. Assert numeric equality plus re-marshal stability instead.

<details><summary>test cases</summary>

Ready-to-add, green at this head: [`bigdec_float_amino_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/4-adbb5ac60/tests/bigdec_float_amino_test.go) · [↗](../4-adbb5ac60/tests/bigdec_float_amino_test.go) covers both rat and float forms, asserting `Marshal → Unmarshal → Marshal` is a fixed point and the value is preserved.
</details>

## gnovm/pkg/gnolang/values.go:128 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values.go#L128)
Nit: this comment says the rational form holds values "within ratGuard's bit limits", but the guard is named `ratOverflows` / `ratOverflowBits`. Only stale reference left in the code.

## gnovm/tests/files/types/bigdec_float32_no_double_round.gno:1-2 [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/types/bigdec_float32_no_double_round.gno#L1)
Nit: the header says the conversion "must round once at 24-bit precision", but this commit removed the 24-bit intermediate and now rounds directly from the exact rational to float32.
