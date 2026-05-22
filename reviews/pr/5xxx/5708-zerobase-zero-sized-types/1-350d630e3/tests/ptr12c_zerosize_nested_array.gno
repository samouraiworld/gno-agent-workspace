// AI generated
/* Run: from a gno checkout:
gh pr checkout 5708 -R gnolang/gno
curl -fsSL -o gnovm/tests/files/ptr12c_zerosize_nested_array.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5708-zerobase-zero-sized-types/1-350d630e3/tests/ptr12c_zerosize_nested_array.gno
go test -v -run 'TestFiles/ptr12c_zerosize_nested_array' -test.short -timeout 60s ./gnovm/pkg/gnolang/
rm gnovm/tests/files/ptr12c_zerosize_nested_array.gno
*/

// run

// Adversarial: [5][0]int has size 5 * (0 * sizeof(int)) = 0.
// In Go, two new([5][0]int) share zerobase. This PR's
// isZeroSizeType.*ArrayType branch only handles ct.Len == 0,
// not arrays of zero-sized elements. Expected: ok.
// Actual on PR head: FAIL.

package main

func main() {
	p := new([5][0]int)
	q := new([5][0]int)
	if p != q {
		print("FAIL: new([5][0]int) returned distinct pointers\n")
		println("FAIL")
		return
	}
	println("ok")
}

// Output:
// ok
