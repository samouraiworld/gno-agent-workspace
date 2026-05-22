/* Run:
WT=/home/davd/Projects/gno-skills/.worktrees/gno-review-5708
cp ptr12b_zerosize_array_of_zsized.gno "$WT/gnovm/tests/files/"
go test -C "$WT" ./gnovm/pkg/gnolang/ -run 'TestFiles/ptr12b_zerosize_array_of_zsized' -test.short -v -timeout 60s
rm "$WT/gnovm/tests/files/ptr12b_zerosize_array_of_zsized.gno"
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
