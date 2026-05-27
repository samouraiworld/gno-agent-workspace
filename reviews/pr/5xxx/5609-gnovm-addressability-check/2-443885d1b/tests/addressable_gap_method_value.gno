// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno && git checkout 443885d1b
curl -fsSL -o gnovm/tests/files/addressable_gap_method_value.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/tests/addressable_gap_method_value.gno
go test -v -run 'TestFiles/addressable_gap_method_value.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/addressable_gap_method_value.gno
*/

// Demonstrates that `&t.M` (method value) survives PR #5609's check.
// `isAddressable(SelectorExpr{X: t})` -> X is *NameExpr -> true. The Kind of
// the *selector result* (FuncKind for a method value) is never inspected.
// Go rejects this at compile time.
//
// Observed: `// Output: 42` passes — `&t.M` yields a pointer whose
// dereference is a callable bound method. Flip the assertion to // Error:
// once SelectorExpr returns false when the selector resolves to a method
// value (selector path or `evalStaticTypeOf(SelectorExpr).Kind() == FuncKind`
// on a non-PointerKind base).

package main

type T struct{ x int }

func (t T) M() int { return t.x }

func main() {
	t := T{x: 42}
	p := &t.M
	println((*p)())
}

// Output:
// 42

// TypeCheckError:
// main/addressable_gap_method_value.gno:29:8: invalid operation: cannot take address of t.M (value of type func() int)
