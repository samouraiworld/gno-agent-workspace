// AI generated
/* Run: from a gno checkout:
gh pr checkout 5708 -R gnolang/gno
curl -fsSL -o gnovm/tests/files/ptr12b_zerosize_array_of_zsized.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5708-zerobase-zero-sized-types/1-350d630e3/tests/ptr12b_zerosize_array_of_zsized.gno
go test -v -run 'TestFiles/ptr12b_zerosize_array_of_zsized' -test.short -timeout 60s ./gnovm/pkg/gnolang/
rm gnovm/tests/files/ptr12b_zerosize_array_of_zsized.gno
*/

// run

// Adversarial: [10]struct{} has size 0 in Go (10 * 0 = 0).
// In Go, new([10]struct{}) returns &runtime.zerobase, so two
// calls share the same address. This PR's isZeroSizeType only
// returns true for arrays with Len == 0; it misses arrays of
// any length whose elements are zero-sized. Expected: ok.
// Actual on PR head: FAIL (distinct pointers).

package main

func main() {
	p := new([10]struct{})
	q := new([10]struct{})
	if p != q {
		print("FAIL: new([10]struct{}) returned distinct pointers\n")
		println("FAIL")
		return
	}
	println("ok")
}

// Output:
// ok
