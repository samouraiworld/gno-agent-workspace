// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5737 -R gnolang/gno && git checkout 4c57c37e4
curl -fsSL -o gnovm/tests/files/concrete_methodval_regression.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/1-4c57c37e4/tests/concrete_methodval_regression.gno
go test -v -run 'TestFiles/concrete_methodval_regression.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/concrete_methodval_regression.gno
*/

// Same root cause as concrete_defer_regression, via a plain method-value
// assignment. In Go, `g := pt.M` on a concrete nil *T (value receiver) copies
// *pt to form the method value, so it panics EAGERLY at the assignment —
// "bound ok" / "calling" never print. master matches Go. This PR binds a
// BoundMethodValue{nilReceiverPanic:true} and defers the panic to g(), so it
// prints "bound ok" then "calling" before panicking — divergence.
//
// Active "// Output:" is Go's/master's behavior; this PR produces the 3-line
// form (bound ok / calling / panicked at bind), so the test FAILS on the PR.

package main

type T struct{ x int }

func (T) M() string { return "called" }

var pt *T

func main() {
	defer func() {
		if r := recover(); r != nil {
			println("panicked at bind")
		}
	}()
	g := pt.M // concrete nil *T: Go panics HERE
	println("bound ok")
	_ = g
	println("calling")
	println(g())
}

// Output:
// panicked at bind
