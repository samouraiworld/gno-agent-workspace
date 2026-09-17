// PKGPATH: gno.land/r/demo/curargref
package curargref

// b1-claims-r2: `&cur` on a NON-realm binding inside a crossing function.
//
// Repro from a plain clone, at 6d88deba71165b165641b65541259d00aba20377:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 6d88deba71165b165641b65541259d00aba20377
//	cp <this file> gnovm/tests/files/zrealm_cur_argname_ref.gno
//	go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_argname_ref.gno$'
//
// preprocess.go's RefExpr rule used to read
//
//	if ne, ok := n.X.(*NameExpr); ok && ne.Name == "cur" { if xt == gRealmType { panic(...) } }
//
// and now reads `ok && isCrossingCurParam(store, last, ne)`, which tests the
// DECLARING NODE and never the type nor the first parameter's name. F's first
// realm parameter is unnamed (`.arg0`), which the FuncTypeExpr rule allows, so
// the int parameter `cur` resolves to a crossing FuncDecl and `&cur` is refused
// as the frame identity, with a message that calls it realm-typed.
//
// Head, 6d88deba7:
//
//	--- FAIL: TestFiles/zrealm_cur_argname_ref.gno (0.00s)
//	    files_test.go:135: unexpected panic:
//	    gno.land/r/demo/curargref/zrealm_cur_argname_ref.gno:27:7-11: cannot take
//	    the address of a realm-typed `cur`: the binding is fixed for the life of
//	    the frame, ...
//	    (preprocess.go:2584)
//
// Merge base, b7845fb2da9a21bfb895e1c008f016776f8342d0: `ok
// github.com/gnolang/gno/gnovm/pkg/gnolang 0.046s` — the file prints 5, because
// the removed type test `xt == gRealmType` is false for an int.

func F(_ realm, cur int) int {
	p := &cur
	*p = 5
	return cur
}

func main(cur realm) {
	println(F(cur, 1))
}

// Output:
// 5
