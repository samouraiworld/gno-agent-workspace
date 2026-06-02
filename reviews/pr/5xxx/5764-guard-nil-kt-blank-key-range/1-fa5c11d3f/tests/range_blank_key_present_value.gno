// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5764 -R gnolang/gno && git checkout fa5c11d3f
curl -fsSL -o gnovm/tests/files/range_blank_key_present_value.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5764-guard-nil-kt-blank-key-range/1-fa5c11d3f/tests/range_blank_key_present_value.gno
go test -v -run 'TestFiles/range_blank_key_present_value.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/range_blank_key_present_value.gno
*/

// Blank key + PRESENT value over slice/array/string/map. evalStaticTypeOf(_)
// returns nil, so kt==nil reaches assertIndexTypeIsInt for slice/array/string;
// the map branch reaches mustAssignableTo(cxt.Key, nil). PR #5764 guards
// assertIndexTypeIsInt against nil and folds the string branch into it.
//
// Result with #5764: all four loops type-check, prints "ok".
// Without the fix (master), the slice loop nil-derefs in assertIndexTypeIsInt
// at type_check.go, called from (*RangeStmt).AssertCompatible — same stack as
// issue #5664. This is also the form PR #5751's widened early-return left
// broken: a present non-blank value skips the both-blank return.
//
// To assert the fix: flip the Output block to expect the panic and it fails.

package main

func main() {
	s := []int{1, 2, 3}
	var sv int
	for _, sv = range s {
	}
	_ = sv

	var a [3]int
	var av int
	for _, av = range a {
	}
	_ = av

	str := "abc"
	var rv rune
	for _, rv = range str {
	}
	_ = rv

	m := map[int]string{1: "a"}
	var mv string
	for _, mv = range m {
	}
	_ = mv

	println("ok")
}

// Output:
// ok
