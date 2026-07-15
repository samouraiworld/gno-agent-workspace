/* Run: from a gno checkout:
gh pr checkout 5867 -R gnolang/gno && git checkout adbb5ac60
curl -fsSL -o gnovm/tests/files/types/bigdec_dual_form_arith.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5867-bigdec-apd-to-rat/4-adbb5ac60/tests/bigdec_dual_form_arith.gno
go test -v -run 'TestFiles/types/bigdec_dual_form_arith.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/types/bigdec_dual_form_arith.gno
*/
// Exercises the dual-representation paths bigdec_extreme_magnitude.gno leaves
// uncovered: float-form comparison (bigdecCmp float branch), mixed rat/float
// arithmetic (bigdecArith float branch), and float-form negation. Each line
// matches go/constant, which switches from big.Rat to a 512-bit big.Float at
// the same 4096-bit threshold. Green at adbb5ac60.

package main

func main() {
	const big1 = 1e5000  // float form
	const big2 = 1e5001  // float form
	println(big1 < big2) // float-form comparison
	println(big2 > big1)
	println(big1 == big1)
	println(big1 <= big1)
	const half = 1.5           // rat form
	println(big1*half > big1)  // mixed rat*float
	println(big1+half == big1) // 1.5 is below the 512-bit ULP at 1e5000
	println(big1/big1 == 1.0)  // float/float folds to exactly 1
	const negBig = -1e5000     // float-form negation
	println(negBig < 0.0)
	println(negBig+big1 == 0.0) // float+float cancels to 0
	const sum = big1 + big1
	println(sum > big1)
	println(sum/2.0 == big1)
}

// Output:
// true
// true
// true
// true
// true
// true
// true
// true
// true
// true
// true
