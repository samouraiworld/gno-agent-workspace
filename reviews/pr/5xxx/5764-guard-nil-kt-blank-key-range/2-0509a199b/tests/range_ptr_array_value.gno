/* Run: from a gno checkout:
gh pr checkout 5764 -R gnolang/gno && git checkout 0509a199b
curl -fsSL -o gnovm/tests/files/types/range_ptr_array_value.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5764-guard-nil-kt-blank-key-range/2-0509a199b/tests/range_ptr_array_value.gno
go test -v -run 'TestFiles/types/range_ptr_array_value.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/types/range_ptr_array_value.gno
*/
// Pointer-to-array range hits no case in (*RangeStmt).AssertCompatible, so the value
// target is unchecked: at 0509a199b this runs and prints 3 (int into a string var),
// caught only by go/types. A *PointerType case unwrapping to the array elem fixes it.
package main

func main() {
	var v string
	arr := [3]int{1, 2, 3}
	for _, v = range &arr {
	}
	println(v)
}

// Error:
// main/range_ptr_array_value.gno:6:2-7:3: cannot use int as string

// TypeCheckError:
// main/range_ptr_array_value.gno:6:9: cannot use v (value of type int) as string value in assignment
