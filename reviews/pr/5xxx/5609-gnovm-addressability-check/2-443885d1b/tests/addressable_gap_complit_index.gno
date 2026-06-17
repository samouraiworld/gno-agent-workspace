// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno && git checkout 443885d1b
curl -fsSL -o gnovm/tests/files/addressable_gap_complit_index.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/tests/addressable_gap_complit_index.gno
go test -v -run 'TestFiles/addressable_gap_complit_index.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/addressable_gap_complit_index.gno
*/

// Demonstrates that PR #5609's `isAddressable` accepts `&[3]int{...}[0]`.
// A composite literal is addressable in Go only as the direct operand of `&`,
// but the helper returns true for `*CompositeLitExpr` everywhere, so the
// IndexExpr arm recurses into the literal base and accepts the address.
// Observed: `// Output: 1` passes; flip to // Error: once a composite-literal
// base is treated as non-addressable.

package main

func main() {
	p := &[3]int{1, 2, 3}[0]
	println(*p)
}

// Output:
// 1

// TypeCheckError:
// main/addressable_gap_complit_index.gno:20:8: invalid operation: cannot take address of [3]int{…}[0] (value of type int)
