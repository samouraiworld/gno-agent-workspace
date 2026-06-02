// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno && git checkout 61fc396e4
curl -fsSL -o gnovm/tests/files/decltype_derived_empty_base.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/1-61fc396e4/tests/decltype_derived_empty_base.gno
go test -v -run 'TestFiles/decltype_derived_empty_base.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_derived_empty_base.gno
*/

// Mechanism: `type T2 T1`, where T1 is declared first and carries a *T2
// field plus data fields, finalizes T2's base from the still-empty
// predefined T1 in the TypeDecl TRANS_LEAVE handler (preprocess.go:3066,
// declareWith -> baseOf(tmp)). T2's Base collapses to struct{} instead of
// copying T1's fields. PR #5763 removed the tryPredefine guard that
// previously turned this exact shape into a loud "should not happen"
// panic; with the guard gone the program compiles but T2 is structurally
// corrupt.
//
// Observed on 61fc396e4: runtime panic
//   "struct type struct{} has no field Val" at the a.Next.Val read.
// Master (2c7f1abe3, guard present) panics earlier at predefine with
//   "should not happen". Either way `type T2 T1` (T1-first, mutual) never
// works; the PR only changes WHEN it fails. decltype_mutual.gno (the PR's
// own test) passes ONLY because it never reads a field on T2.
//
// To flip to the post-fix expectation, replace the Error directive with
//   // Output:
//   // 9
//   // ok
// once T2's base correctly resolves to T1's struct.

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

// Error:
// main/decltype_derived_empty_base.gno:43:15-18: struct type struct{} has no field Val
