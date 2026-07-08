# Review: PR [#5867](https://github.com/gnolang/gno/pull/5867)
Event: REQUEST_CHANGES

## Body
Verified on 07d4ad373: float32(0x1.000001000000001p0) evaluates to 1, where Go's gc compiler and big.Rat.Float32 both give 1.0000001, because the conversion rounds to float64 before narrowing to float32. A package that repeatedly squares an integer-valued constant materializes gigabytes at preprocess and returns ok. The same shape built from 1.0/3.0 panics on the denominator guard.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/2-07d4ad373/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## SKIP gnovm/pkg/gnolang/op_binary.go:806 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L806)
Already raised: https://github.com/gnolang/gno/pull/5867#discussion_r3507150834
Posted in round 1, unaddressed at 07d4ad373; keeping it out of this round's post so it is not duplicated.

`ratGuard` checks `r.Denom().BitLen()` but never the numerator, so an integer-valued `big.Rat` keeps `Denom() == 1` and passes unbounded. Repeated squaring of such a constant doubles the numerator each step, and folding runs on a gasless machine, so a roughly 20-line package allocates gigabytes at `AddPkg` preprocess. Bound the numerator the way the denominator already is.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5867 -R gnolang/gno

# A) denominator growth IS bounded by ratGuard:
{ echo 'package main'; echo 'const a0 = 1.0 / 3.0'
  for i in $(seq 1 12); do echo "const a$i = a$((i-1)) * a$((i-1))"; done
  echo 'func main() { println("ok") }'; } > /tmp/denomA.gno

# B) numerator growth is NOT bounded — same shape, integer-valued base:
{ echo 'package main'; echo 'const a0 = 1e10000'
  for i in $(seq 1 12); do echo "const a$i = a$((i-1)) * a$((i-1))"; done
  echo 'func main() { println("ok") }'; } > /tmp/numB.gno

go run ./gnovm/cmd/gno run /tmp/denomA.gno   # a12 denom = 3^4096 ≈ 6492 bits
go run ./gnovm/cmd/gno run /tmp/numB.gno      # a12 = 1e40960000, numerator ≈ 136 Mbit
rm /tmp/denomA.gno /tmp/numB.gno
```
```
# A:
panic: constant expression result too large: denominator exceeds 4096 bits

# B:
ok
# numB accepted: a12 = 1e40960000 folded and materialized (~17 MB) at preprocess,
# no panic, no gas. Adding lines scales it 4x each (k=14 ≈ 481 MB / 72 s);
# Go and gno-master both reject this input in ~0.02 s.
```
</details>

## gnovm/pkg/gnolang/values_conversions.go:1437-1447 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1437)
float32 constant conversion rounds to float64 before narrowing to float32, so it double-rounds and diverges from Go for values on a float32 rounding boundary. float32(0x1.000001000000001p0) evaluates to 1 here, where Go gives 1.0000001. Round the rational straight to float32 so it single-rounds and stays deterministic.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5867 -R gnolang/gno

cat > /tmp/dr.gno <<'EOF'
package main
func main() {
	const c = 0x1.000001000000001p0
	var f float32 = c
	println(f)
}
EOF
cat > /tmp/dr.go <<'EOF'
package main
func main() {
	const c = 0x1.000001000000001p0
	var f float32 = c
	println(f)
}
EOF
echo -n 'gno: '; go run ./gnovm/cmd/gno run /tmp/dr.gno
echo -n 'go:  '; go run /tmp/dr.go
rm /tmp/dr.gno /tmp/dr.go
```
```
gno: 1
go:  1.0000001
```
</details>
