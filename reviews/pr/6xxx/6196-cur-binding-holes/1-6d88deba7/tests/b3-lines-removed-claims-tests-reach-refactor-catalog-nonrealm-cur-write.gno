// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/PROBE.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/PROBE.gno$' -v
// Base is b7845fb2da9a21bfb895e1c008f016776f8342d0 (same commands, that sha).
// head 6d88deba7: FAIL  base b7845fb2d: PASS
//
// isCrossingCurParam (preprocess.go:6111) resolves the written name to its
// declaring node and tests only IsCrossing(); it checks neither the declared
// type nor that the first parameter is named cur. The first realm parameter
// may be blank or unnamed (preprocess.go:2828 admits a ".arg" prefix), so an
// ordinary int parameter named cur is declared by a crossing FuncDecl and is
// refused. The three write rules on the base each gated on
// evalStaticTypeOf(...) == gRealmType, which excluded it.

// PKGPATH: gno.land/r/demo/curprobef
package curprobef

func F(_ realm, cur int) int {
	cur = cur + 1
	return cur
}

func main() {
	println("ok")
}

// Output:
// ok
