// PKGPATH: gno.land/r/demo/b1local
package b1local

// A plain int local named `cur` in the body block of a crossing function whose
// first realm parameter is unnamed (`_ realm`, the shape used by
// r/gnoland/blog, r/gnoland/boards2/v0, r/demo/counter, r/sys/users ...).
// Nothing realm-typed, nothing any frame runs as, and the function has no
// `cur` parameter at all.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zz_b1_local_int.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_local_int.gno$'
//
// head 6d88deba7: FAIL, at the DEFINE, not the assignment:
//   unexpected panic: gno.land/r/demo/b1local/zz_b1_local_int.gno:9:2-10:
//   cannot reassign the crossing `cur` parameter: it names the realm this
//   frame is executing as, and that binding is fixed for the life of the frame
//
// merge base b7845fb2da9a21bfb895e1c008f016776f8342d0: PASS (prints 2).
// Same commands, checkout b7845fb2d.

func F(_ realm) int {
	cur := 1
	cur = 2
	return cur
}

func main(cur realm) {
	println(F(cur))
}

// Output:
// 2
