// AI generated
/* Run: from a gno checkout:
gh pr checkout 5707 -R gnolang/gno && git checkout 5c52d8ea3
curl -fsSL -o gnovm/tests/files/var_initorder_methodexpr_closure.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5707-method-expr-init-order/2-5c52d8ea3/tests/var_initorder_methodexpr_closure.gno
go test -v -run 'TestFiles/var_initorder_methodexpr_closure.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/var_initorder_methodexpr_closure.gno
*/
package main

type T int

func (r T) M() int { return c }

var dummy = func() int { return T.M(0) }()

var c = identity(11)

func identity(a int) int { return a }

func main() {
	if dummy != 11 {
		panic("dummy != 11")
	}
	println(dummy)
}

// Output:
// 11
