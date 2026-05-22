/* Run:
 *   cp reviews/pr/5xxx/5599-slice-to-array-conversion/1-e8168367/tests/restored_g.gno \
 *      .worktrees/gno-review-5599/gnovm/tests/files/convert_slice_to_array_g_restored.gno
 *   go test -C .worktrees/gno-review-5599/gnovm -v -run 'TestFiles/convert_slice_to_array_g_restored' ./pkg/gnolang/
 *   rm .worktrees/gno-review-5599/gnovm/tests/files/convert_slice_to_array_g_restored.gno
 */
// Restored from convert_slice_to_array_g.gno (deleted in commit e81683679).
// List-backed byte slice fallback in ConvertTo:
// slice literals go through NewListArray, so sBase.Data == nil.
package main

type B byte

func main() {
	s := []B{1, 2, 3, 4}
	a := [3]B(s[1:])
	println(a[0], a[1], a[2])
}

// Output:
// (2 main.B) (3 main.B) (4 main.B)
