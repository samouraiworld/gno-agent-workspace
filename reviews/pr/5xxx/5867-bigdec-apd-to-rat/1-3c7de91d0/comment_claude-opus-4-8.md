# Review: PR #5867
Event: REQUEST_CHANGES

## Body
Verified on 3c7de91d0: a 17-line package of repeated integer-valued constant folding takes 72 s and 481 MB at preprocess and is accepted, where Go and gno master reject the same input in about 0.02 s. The same shape with a `1.0/3.0` base panics on the denominator guard, confirming only the denominator is bounded.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/1-3c7de91d0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/op_binary.go:803 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/op_binary.go#L803)
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

## gnovm/tests/files/const63.gno:5 [↗](../../../../../.worktrees/gno-review-5867/gnovm/tests/files/const63.gno#L5)
The comment points at `go run _verify/const63_go/main.go`, but the PR adds no such file, so the Go cross-check it names cannot be run. Drop the reference or add the file.

## gnovm/pkg/gnolang/values_conversions.go:1390 [↗](../../../../../.worktrees/gno-review-5867/gnovm/pkg/gnolang/values_conversions.go#L1390)
The error prints the rational form, so a value written as `1.2` is reported as `6/5 not an exact integer`, which is harder to trace back to the source than the decimal. Rendering the decimal form would match what the user wrote.
