// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno && git checkout 443885d1b
curl -fsSL -o gnovm/tests/files/addressable_gap_complit_slice.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/tests/addressable_gap_complit_slice.gno
go test -v -run 'TestFiles/addressable_gap_complit_slice.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/addressable_gap_complit_slice.gno
*/

// Demonstrates that PR #5609's slice-of-unaddressable-array check accepts
// `[3]int{1, 2, 3}[:]`. The SliceExpr arm calls isAddressable on the array
// base, which is a `*CompositeLitExpr` and reported addressable. Go rejects
// slicing an unaddressable array value. Observed: `// Output: 3` passes.

package main

func main() {
	s := [3]int{1, 2, 3}[:]
	println(len(s))
}

// Output:
// 3

// TypeCheckError:
// main/addressable_gap_complit_slice.gno:18:7: cannot slice unaddressable value [3]int{…} (value of type [3]int)
