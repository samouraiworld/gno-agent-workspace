/* Run:
 *   cp reviews/pr/5xxx/5599-slice-to-array-conversion/1-e8168367/tests/value_conversion_struct_copy.gno \
 *      .worktrees/gno-review-5599/gnovm/tests/files/value_conversion_struct_copy.gno
 *   go test -C .worktrees/gno-review-5599/gnovm -v -run 'TestFiles/value_conversion_struct_copy' ./pkg/gnolang/
 *   rm .worktrees/gno-review-5599/gnovm/tests/files/value_conversion_struct_copy.gno
 */
// In Go, [N]T(s) shallow-copies elements: the elements are copies, but
// the contents of pointer/slice/map fields inside elements are shared.
// Mutating a[0].P (the field itself, not *a[0].P) MUST NOT touch s[0].P.
package main

type Box struct {
	P *int
}

func main() {
	x := 1
	y := 2
	s := []Box{{P: &x}}
	a := [1]Box(s)
	a[0].P = &y
	println(*s[0].P) // expects 1 — s[0].P should still point to x
}

// Output:
// 1
