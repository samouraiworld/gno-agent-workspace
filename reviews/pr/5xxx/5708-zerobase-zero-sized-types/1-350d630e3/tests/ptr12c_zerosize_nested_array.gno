/* Run:
WT=/home/davd/Projects/gno-skills/.worktrees/gno-review-5708
cp ptr12c_zerosize_nested_array.gno "$WT/gnovm/tests/files/"
go test -C "$WT" ./gnovm/pkg/gnolang/ -run 'TestFiles/ptr12c_zerosize_nested_array' -test.short -v -timeout 60s
rm "$WT/gnovm/tests/files/ptr12c_zerosize_nested_array.gno"
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
