// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/PROBE.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/PROBE.gno$' -v
// Base is b7845fb2da9a21bfb895e1c008f016776f8342d0 (same commands, that sha).
// head 6d88deba7: FAIL  base b7845fb2d: FAIL (pre-existing; the diff sweeps
// this class and misses this member)
//
// zrealm_cur_other_legal.gno pins a named result only under a NON-crossing
// declarer (func namedResult(x int) (cur realm)). Under a crossing declarer
// with a blank first realm parameter the same shape is refused as "the
// crossing cur parameter", which it is not.

// PKGPATH: gno.land/r/demo/curprobea
package curprobea

func F(_ realm) (cur realm) {
	cur = nil
	return cur
}

func main() {
	println("ok")
}

// Output:
// ok
