// PKGPATH: gno.land/r/demo/curnamedres
package curnamedres

// Repro, from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_b2claims_namedresult.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_b2claims_namedresult.gno$'
//
// Expected per the ADR ("Package-level vars, named results, and block-scoped
// shadows stay writable") and isCrossingCurParam's godoc ("a named result ...
// is not any frame's identity, and stays writable"): PASS, printing `ok`.
// Observed at 6d88deba7 AND at the merge base b7845fb2d:
//
//   unexpected panic: .../zrealm_b2claims_namedresult.gno:N: cannot reassign
//   the crossing `cur` parameter: it names the realm this frame is executing
//   as, and that binding is fixed for the life of the frame
//
// Same shape as the PR's own zrealm_cur_other_legal.gno namedResult, except
// the enclosing function's first parameter is an UNNAMED realm, which the
// FuncTypeExpr rule allows through its `.arg` prefix (preprocess.go:2827).
// IsCrossing() only compares Params[0].Type to gRealmType, so
// isCrossingCurParam resolves this named result to a crossing FuncDecl and
// refuses the write.
func namedResult(_ realm) (cur realm) {
	cur = nil
	return cur
}

func main(cur realm) {
	_ = namedResult(cur)
	println("ok")
}

// Output:
// ok
