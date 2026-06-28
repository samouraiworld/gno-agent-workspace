/* Run: from a gno checkout:
gh pr checkout 5737 -R gnolang/gno && git checkout ccb6c94ad
curl -fsSL -o gnovm/tests/files/concrete_defer_eager.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5737-defer-nil-receiver-panic/2-ccb6c94ad/tests/concrete_defer_eager.gno
go test -v -run 'TestFiles/concrete_defer_eager.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/concrete_defer_eager.gno
*/

// Round-1 Critical (concrete-receiver eager-deref regression) — now fixed.
// A concrete nil *T value method derefs at bind, so `defer pt.M()` panics
// eagerly and r=1 never runs: f()==0, matching Go. Round 1 (sha 4c57c37e4)
// regressed this to call-time (printed 1); the redesign moved interface
// dispatch to VPInterface and left the concrete VPDerefValMethod eager.
package main

type T struct{ x int }

func (T) M() {}

var pt *T

func f() (r int) {
	defer func() { recover() }()
	defer pt.M()
	r = 1
	return
}

func main() {
	println(f())
}

// Output:
// 0
