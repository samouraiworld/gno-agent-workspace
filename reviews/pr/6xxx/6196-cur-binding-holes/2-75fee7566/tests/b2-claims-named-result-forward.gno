// PKGPATH: gno.land/r/demo/curnamedfwd
package curnamedfwd

// Repro, from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_b2claims_namedforward.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_b2claims_namedforward.gno$'
//
// The ADR's Consequences say "The identity check is now a backstop with no
// known reachable source route". This is such a route, at 6d88deba7 and at
// the merge base b7845fb2d alike: the cur-call provenance check resolves the
// named result `cur` to a crossing FuncDecl and forwards it as a frame
// identity, and the runtime identity check then reports it as a stale
// capture, which is not the mistake written here.
//
//   unexpected panic: crossing function gno.land/r/demo/curnamedfwd.target was
//   entered without cross(), so it takes its caller's identity — but the value
//   passed is not the caller's own cur (a stale capture from another frame, a
//   sibling frame, or a rebound cur). ...
func target(cur realm) {
	_ = cur.IsCurrent()
}

func forward(_ realm) (cur realm) {
	target(cur)
	return
}

func main(cur realm) {
	_ = forward(cur)
	println("ok")
}

// Output:
// ok
