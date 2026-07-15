/* Run: from a gno checkout:
gh pr checkout 5867 -R gnolang/gno
curl -fsSL -o gnovm/tests/files/types/bigdec_float32_subnormal.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/5-79ebc4058/tests/bigdec_float32_subnormal.gno
go test -v -run 'TestFiles/types/bigdec_float32_subnormal.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/types/bigdec_float32_subnormal.gno
*/

// bigdec -> float32 must round once from the exact value. This constant sits
// just above a subnormal float32 halfway point: the 0x1p-200 term pushes it
// past the tie, so it must round up to bits 3. Rounding through a wider
// float first collapses it onto the tie and round-half-to-even gives bits 2.
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
