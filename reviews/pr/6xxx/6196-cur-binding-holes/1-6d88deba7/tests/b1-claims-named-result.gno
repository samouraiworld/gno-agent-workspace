// PKGPATH: gno.land/r/demo/b1named
package b1named

// A realm-typed named result `cur` in a crossing function whose first realm
// parameter is unnamed. isCrossingCurParam's godoc says a named result "is not
// any frame's identity, and stays writable"; the write rule refuses it,
// because the result is declared by the crossing FuncDecl's own block.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zz_b1_named_result.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_named_result.gno$'
//
// head 6d88deba7: FAIL
//   unexpected panic: gno.land/r/demo/b1named/zz_b1_named_result.gno:9:2-11:
//   cannot reassign the crossing `cur` parameter: ...
// merge base b7845fb2d: FAIL with the same message (predates the branch; the
// branch sweeps this class and misses this member).

func F(_ realm) (cur realm) {
	cur = nil
	return cur
}

func main(cur realm) {
	println("ok")
}

// Output:
// ok
