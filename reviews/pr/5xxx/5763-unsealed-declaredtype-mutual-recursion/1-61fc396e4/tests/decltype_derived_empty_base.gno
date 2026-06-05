// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno && git checkout 61fc396e4
curl -fsSL -o gnovm/tests/files/decltype_derived_empty_base.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/1-61fc396e4/tests/decltype_derived_empty_base.gno
go test -v -run 'TestFiles/decltype_derived_empty_base.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_derived_empty_base.gno
*/

// Asserts the CORRECT behaviour: `type T2 T1` (T1 declared first, carrying a
// *T2 field plus data fields) is valid Go/Gno, so reading the data field back
// through T2 must print 9 then ok. The // Output directive below is the
// post-fix expectation; the test goes green the moment the bug is fixed.
//
// Mechanism of the bug: in the TypeDecl TRANS_LEAVE handler (preprocess.go:3066,
// declareWith -> baseOf(tmp)) T2's base is finalized from the still-empty
// predefined T1, so T2.Base collapses to struct{} instead of copying T1's
// fields. PR #5763 removed the tryPredefine guard that previously turned this
// exact shape into a loud "should not happen" panic; with the guard gone the
// program compiles but T2 is structurally corrupt.
//
// On 61fc396e4 this test FAILS: the a.Next.Val read panics
//   "struct type struct{} has no field Val".
// (Master, guard present, fails earlier at predefine with "should not happen".)
// The PR's own decltype_mutual.gno never reads a field on T2, so it cannot
// notice this.

package main

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

func main() {
	var a T1
	a.Next = &T2{Val: 9}
	println(a.Next.Val)
	println("ok")
}

// Output:
// 9
// ok
