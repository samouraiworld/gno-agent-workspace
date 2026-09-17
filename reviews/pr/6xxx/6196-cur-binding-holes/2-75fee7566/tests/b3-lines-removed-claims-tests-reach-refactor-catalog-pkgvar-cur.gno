// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/PROBE.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/PROBE.gno$' -v
// Base is b7845fb2da9a21bfb895e1c008f016776f8342d0 (same commands, that sha).
// head 6d88deba7: PASS  base b7845fb2d: FAIL
//
// The package-level `var cur realm` the PR body names as fixed. The committed
// fixture zrealm_cur_other_legal.gno spells this arm `pkgCur`, a name
// isCrossingCurParam rejects on its first line, so that arm is green on the
// base too and pins nothing. Mutation check, head with isCrossingCurParam
// replaced by `return nx.Name == "cur"`: this file FAILs, the pkgCur spelling
// still PASSes.

// PKGPATH: gno.land/r/demo/curprobec
package curprobec

var cur realm

func set(x int, r realm) {
	cur = r
	cur = nil
}

func main() {
	set(0, nil)
	println("ok")
}

// Output:
// ok
