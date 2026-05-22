/* Run: from a gno checkout:
gh pr checkout 5707 -R gnolang/gno && git checkout c7b8227f2
# (save this file at gnovm/tests/files/var_initorder_methodexpr_ptrrcvr.gno)
go test -v -run 'TestFiles/var_initorder_methodexpr_ptrrcvr.gno$' ./gnovm/pkg/gnolang/
*/

// run

// Method expression with pointer receiver: (*T).Method
// Verifies the PointerType branch of the VPField case picks up dt.
package main

type T int

func (r *T) Method() int { return c }

// dummy must initialize after c. (*T).Method is a method expression
// with pointer receiver: tv.Type is *PointerType wrapping *DeclaredType T.
var x T = 0
var dummy = (*T).Method(&x)

var c = identity(7)

func identity(a int) int { return a }

func main() {
	if dummy != 7 {
		panic("dummy != 7")
	}
	println(dummy)
}

// Output:
// 7
