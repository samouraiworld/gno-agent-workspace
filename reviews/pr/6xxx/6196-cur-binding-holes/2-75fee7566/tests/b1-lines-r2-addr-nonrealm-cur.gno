// Repro, from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_cur_addr_int.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_addr_int.gno$'
//
// Observed at 6d88deba7:
//   --- FAIL: TestFiles/zrealm_cur_addr_int.gno
//   files_test.go:135: unexpected panic:
//     gno.land/r/demo/curaddrint/zrealm_cur_addr_int.gno:10:7-11: cannot take the
//     address of a realm-typed `cur`: the binding is fixed for the life of the
//     frame, ... panic raised from preprocess.go:2584
//
// The value is an int. The address-of rule at preprocess.go:2583 now asks
// isCrossingCurParam, which tests only the declaring block node, so it refuses
// any name spelled `cur` declared in a crossing function's own block and the
// message names a type the value does not have.
//
// Counterfactual, same body under a non-crossing declarer, run at the same sha:
//   func helper() int { var cur int = 42; p := &cur; *p = 7; return cur }
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_addr_int2.gno$'
//   ok  github.com/gnolang/gno/gnovm/pkg/gnolang  0.040s
// so the rejection keys on the crossing declarer, not on the value's type.

// PKGPATH: gno.land/r/demo/curaddrint
package curaddrint

func main(_ realm) {
	var cur int = 42
	p := &cur
	*p = 7
	println(cur)
}

// Output:
// 7
