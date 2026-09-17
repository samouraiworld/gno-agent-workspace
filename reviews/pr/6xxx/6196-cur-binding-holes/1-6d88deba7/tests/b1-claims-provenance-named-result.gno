// PKGPATH: gno.land/r/demo/b1prov
package b1prov

// The provenance half of the same resolution: a named result `cur` in a
// crossing function with an unnamed first realm parameter passes the cur-call
// check (its declaring node is a crossing FuncDecl), is forwarded into a
// callee's frame identity, and is caught only at runtime by the identity
// check, whose message describes a stale capture instead.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zz_b1_provenance.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_provenance.gno$'
//
// head 6d88deba7 and merge base b7845fb2d both FAIL identically:
//   unexpected panic: crossing function gno.land/r/demo/b1prov.Target was
//   entered without cross(), so it takes its caller's identity — but the value
//   passed is not the caller's own cur (a stale capture from another frame, a
//   sibling frame, or a rebound cur) ...

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

// Output:
// ok
