// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5737 -R gnolang/gno && git checkout 4c57c37e4
curl -fsSL -o gnovm/tests/files/concrete_defer_regression.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/1-4c57c37e4/tests/concrete_defer_regression.gno
go test -v -run 'TestFiles/concrete_defer_regression.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/concrete_defer_regression.gno
*/

// PR #5737 makes the VPDerefValMethod-nil path defer the nil-deref panic to
// call time for ALL receivers. That is correct for an interface-boxed nil *T
// (Go also defers it), but WRONG for a *concrete* nil *T: Go evaluates the
// value-receiver copy *pt when the method value is formed, so `defer pt.M()`
// panics EAGERLY at the defer statement and r=1 never runs => f()==0.
//
// Go reference (go run): f()==0. master: 0. This PR: 1 (test FAILS here).
// Flip is one char: change "// Output: 0" to "1" to confirm the PR's value.

package main

type T struct{ x int }

func (T) M() {}

var pt *T

func f() (r int) {
	defer func() { recover() }()
	defer pt.M() // concrete nil *T value method
	r = 1
	return
}

func main() {
	println(f())
}

// Output:
// 0
