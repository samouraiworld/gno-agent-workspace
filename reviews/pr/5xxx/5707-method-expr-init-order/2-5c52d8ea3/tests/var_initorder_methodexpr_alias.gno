// AI generated
/* Run: from a gno checkout:
gh pr checkout 5707 -R gnolang/gno && git checkout 5c52d8ea3
curl -fsSL -o gnovm/tests/files/var_initorder_methodexpr_alias.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5707-method-expr-init-order/2-5c52d8ea3/tests/var_initorder_methodexpr_alias.gno
go test -v -run 'TestFiles/var_initorder_methodexpr_alias.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/var_initorder_methodexpr_alias.gno
*/
package main

type T int

func (r T) M() int { return c }

type U = T // type alias to a declared type

var dummy = U.M(0)

var c = identity(13)

func identity(a int) int { return a }

func main() {
	if dummy != 13 {
		panic("dummy != 13")
	}
	println(dummy)
}

// Output:
// 13
