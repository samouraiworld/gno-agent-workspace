// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno && git checkout 443885d1b
curl -fsSL -o gnovm/tests/files/addressable_gap_complit_selector.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/tests/addressable_gap_complit_selector.gno
go test -v -run 'TestFiles/addressable_gap_complit_selector.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/addressable_gap_complit_selector.gno
*/

// Demonstrates that PR #5609's `isAddressable` accepts `&struct{ x int }{}.x`.
// The SelectorExpr arm recurses into the base, and the base is a
// `*CompositeLitExpr`, which the helper reports addressable unconditionally.
// Go allows addressing a composite literal only when it is the direct operand
// of `&`, not as a selector base. Observed: `// Output: 0` passes.

package main

func main() {
	p := &struct{ x int }{}.x
	println(*p)
}

// Output:
// 0

// TypeCheckError:
// main/addressable_gap_complit_selector.gno:19:8: invalid operation: cannot take address of struct{x int}{}.x (value of type int)
