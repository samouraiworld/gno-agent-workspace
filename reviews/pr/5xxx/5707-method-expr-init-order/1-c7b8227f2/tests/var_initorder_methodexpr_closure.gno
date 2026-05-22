/* Run: from a gno checkout:
gh pr checkout 5707 -R gnolang/gno
# (save this file at gnovm/tests/files/var_initorder_methodexpr_closure.gno)
go test -v -run 'TestFiles/var_initorder_methodexpr_closure.gno$' ./gnovm/pkg/gnolang/
*/

// run

// Method expression hidden inside a func literal initializer.
// The dependency walker must still discover T.M as a dep.
package main

type T int

func (r T) M() int { return d }

var dummy = func() int { return T.M(0) }()

var d = identity(11)

func identity(a int) int { return a }

func main() {
	if dummy != 11 {
		panic("dummy != 11")
	}
	println(dummy)
}

// Output:
// 11
