# Review: PR [#5867](https://github.com/gnolang/gno/pull/5867)
Event: REQUEST_CHANGES

## Body
Verified on adbb5ac60: bit-exact against the Go compiler across ~1200 untyped-float-constant expressions covering exact rationals, the promotion boundary, cancellation, hex floats, and float64 conversions including subnormals.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/4-adbb5ac60/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values_conversions.go:1454-1474 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1454)
The float32 conversion rounds at 24-bit precision, which double-rounds subnormal float32 constants and disagrees with Go. A 24-bit float equals a normal float32 exactly, so the normal case is fixed, but a subnormal holds fewer than 24 bits, so pre-rounding to 24 bits and then to the subnormal grid rounds twice. Round once from the exact value with [`bdv.V.Float32()`](https://github.com/gnolang/gno/blob/adbb5ac60/gnovm/pkg/gnolang/values_conversions.go#L1496-L1506) · [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1496) / `bdv.F.Float32()`, the way the float64 path already does.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5867 -R gnolang/gno
cat > /tmp/sub.gno <<'EOF'
package main
import "math"
func main() {
	const s float32 = 0x1.4p-148 + 0x1p-200
	println(math.Float32bits(s))
}
EOF
cp /tmp/sub.gno /tmp/sub.go
go run ./gnovm/cmd/gno run /tmp/sub.gno   # gno
go run /tmp/sub.go                        # go
rm /tmp/sub.gno /tmp/sub.go
```

```
2
3
```
</details>

## gnovm/pkg/gnolang/op_binary.go:862-876 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L862)
`parseBigdecLiteral` calls `big.Rat.SetString` on every float literal, so an extreme-exponent literal like `1e999999` spends ~16 ms materializing a 3.3 Mbit integer before `ratOverflows` discards it. A package of ~500 such consts is ~6.5 KB and folds in ~7.9 s of preprocess CPU that no gas meters, reachable by any user through `maketx run`. Parse the float first with `big.ParseFloat` and build the exact `big.Rat` only when its magnitude fits under 4096 bits.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5867 -R gnolang/gno
go build -o /tmp/gno ./gnovm/cmd/gno
printf 'package main\nconst x = 1e999999\nfunc main(){ println(x==x) }\n' > /tmp/one.gno
{ echo 'package main'; echo 'const ('
  for i in $(seq 1 500); do echo "  x$i = 1e999999"; done
  echo ')'; echo 'func main(){ println(x1==x1) }'; } > /tmp/many.gno
time /tmp/gno run /tmp/one.gno
time /tmp/gno run /tmp/many.gno
rm /tmp/one.gno /tmp/many.gno /tmp/gno
```

```
# one literal:   real  0m0.04s
# 500 literals:  real  0m7.9s   (linear, unmetered; master folds the same in ~0.02s)
```
</details>

## gnovm/pkg/gnolang/values.go:128 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values.go#L128)
Nit: this comment says values fit "within ratGuard's bit limits", but `ratGuard` was renamed to `ratOverflows` / `ratOverflowBits` in this PR. Only stale reference left.

## gnovm/pkg/gnolang/values.go:198-227 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values.go#L198)
Missing test: the `f:`-prefixed `big.Float` amino form has no coverage, though a package-level `const X = 1e5000` persists through it. A float case can't be appended to `parity_test.go` as-is: `AssertCodecParity`'s deep-equal step trips on a decoded `big.Float` normalizing its mantissa slice. Assert numeric equality plus re-marshal stability instead.

<details><summary>test cases</summary>

Ready-to-add, green at adbb5ac60: [`bigdec_float_amino_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/4-adbb5ac60/tests/bigdec_float_amino_test.go) · [↗](tests/bigdec_float_amino_test.go) covers both rat and float forms, asserting `Marshal → Unmarshal → Marshal` is a fixed point and the value is preserved.
</details>
