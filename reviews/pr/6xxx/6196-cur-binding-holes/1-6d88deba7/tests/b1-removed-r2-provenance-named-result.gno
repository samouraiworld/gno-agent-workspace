// PKGPATH: gno.land/r/demo/curprov
//
// b1-removed-r2 / PR 6196 @ 6d88deba71165b165641b65541259d00aba20377
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp b1-removed-r2-provenance-named-result.gno gnovm/tests/files/zz_b1_prov_namedresult.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_prov_namedresult.gno$'
//
// Observed at 6d88deba7 (preprocess accepts, the runtime backstop refuses):
//   unexpected panic: crossing function gno.land/r/demo/curprov.Target was
//   entered without cross(), so it takes its caller's identity — but the value
//   passed is not the caller's own cur (a stale capture from another frame, a
//   sibling frame, or a rebound cur). ...
//   installInheritedCur ... op_call.go:521
//
// Wanted: the preprocess-time error the control file gets,
//   "only the `cur` argument of a containing crossing function maybe passed by
//   cross-call", since `cur` here is a named RESULT, not the parameter.
//
// The pair to run beside it is b1-removed-r2-provenance-control.gno: identical
// but for a NON-crossing declarer, refused at preprocess. Only the declaring
// function's crossing-ness differs, which is what shows the check keys on the
// declaring block node and never on the binding.
package curprov

func Target(cur realm) {
	_ = cur.IsCurrent()
}

func F(_ realm) (cur realm) {
	Target(cur)
	return
}

func main(cur realm) {
	F(cur)
	println("ok")
}
