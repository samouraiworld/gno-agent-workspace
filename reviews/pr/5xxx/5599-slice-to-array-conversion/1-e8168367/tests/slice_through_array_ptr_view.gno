/* Run:
 *   cp reviews/pr/5xxx/5599-slice-to-array-conversion/1-e8168367/tests/slice_through_array_ptr_view.gno \
 *      .worktrees/gno-review-5599/gnovm/tests/files/slice_through_array_ptr_view.gno
 *   go test -C .worktrees/gno-review-5599/gnovm -v -run 'TestFiles/slice_through_array_ptr_view' ./pkg/gnolang/
 *   rm .worktrees/gno-review-5599/gnovm/tests/files/slice_through_array_ptr_view.gno
 */
// Slicing through a slice-to-array pointer: (*p)[low:high] should produce
// a slice whose backing is the original slice's array, not the transient view.
package main

func main() {
	s := []byte{1, 2, 3, 4, 5}
	p := (*[3]byte)(s)
	s2 := (*p)[:]
	s2[0] = 99
	println(s[0], s2[0])
}

// Output:
// 99 99
