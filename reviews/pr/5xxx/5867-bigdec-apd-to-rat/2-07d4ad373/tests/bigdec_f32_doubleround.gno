/* Run: from a gno checkout:
gh pr checkout 5867 -R gnolang/gno && git checkout 07d4ad373
curl -fsSL -o gnovm/tests/files/bigdec_f32_doubleround.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/2-07d4ad373/tests/bigdec_f32_doubleround.gno
go test -v -run 'TestFiles/bigdec_f32_doubleround.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/bigdec_f32_doubleround.gno
*/
// ConvertUntypedBigdecTo narrows float32 constants as rat -> float64 -> float32,
// which double-rounds. This value sits just above the float32 midpoint 1+2^-24
// but rounds to exactly that midpoint in float64, so the second rounding ties to
// even and yields 1. Go's compiler single-rounds the constant and yields 1.0000001.
package main

func main() {
	const c = 0x1.000001000000001p0
	var f float32 = c
	println(f)
}

// Output:
// 1.0000001
