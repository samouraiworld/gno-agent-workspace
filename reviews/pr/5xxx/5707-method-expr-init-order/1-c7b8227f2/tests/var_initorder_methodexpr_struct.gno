/* Run: from a gno checkout:
gh pr checkout 5707 -R gnolang/gno
# (save this file at gnovm/tests/files/var_initorder_methodexpr_struct.gno)
go test -v -run 'TestFiles/var_initorder_methodexpr_struct.gno$' ./gnovm/pkg/gnolang/
*/

// run

// Struct field selector (VPField path, but n.X is a value, not a TypeType).
// The new gate `_, isType := n.X.GetAttribute(ATTR_TYPEOF_VALUE).(*TypeType)`
// must drop through and NOT register a fake dep "S.x" — S is not a type,
// it's a value, and x is a struct field, not a method.
package main

type S struct{ X int }

var s = S{X: 42}

// This reads s.X (VPField, depth=1 likely). The walker must not record
// a method-dep "S.X" — there's no method S.X. If the gate is broken,
// the resolver would panic in resolveDeclDep ("method X not found in type S").
var dummy = s.X

var k = identity(5)

func identity(a int) int { return a }

func main() {
	if dummy != 42 {
		panic("dummy != 42")
	}
	if k != 5 {
		panic("k != 5")
	}
	println(dummy)
	println(k)
}

// Output:
// 42
// 5
