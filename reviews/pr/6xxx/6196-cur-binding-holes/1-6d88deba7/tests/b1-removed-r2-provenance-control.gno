// PKGPATH: gno.land/r/demo/curprovctl
//
// b1-removed-r2 / PR 6196 @ 6d88deba71165b165641b65541259d00aba20377
// Control for b1-removed-r2-provenance-named-result.gno.
//
//   cp b1-removed-r2-provenance-control.gno gnovm/tests/files/zz_b1_prov_control.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_prov_control.gno$'
//
// Observed at 6d88deba7:
//   unexpected panic: .../zz_b1_prov_control.gno:13:2-13: only the `cur`
//   argument of a containing crossing function maybe passed by cross-call
//
// Same named result, same forward, non-crossing declarer: refused at
// preprocess. The finding file differs only in `G(x int)` becoming
// `F(_ realm)`.
package curprovctl

func Target(cur realm) {
	_ = cur.IsCurrent()
}

func G(x int) (cur realm) {
	Target(cur)
	return
}

func main(cur realm) {
	_ = G(0)
	println("ok")
}
