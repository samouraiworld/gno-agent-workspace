// Asserts a plain `int` named `cur`, declared at the top level of a crossing
// function whose first realm parameter is `_`, may be assigned and read.
// Measured: PASS at merge base b7845fb2d, preprocess panic at head 6d88deba7.
// Fails at the reviewed head.
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba7
//   cp <this file> gnovm/tests/files/zzjudge3assign.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzjudge3assign.gno$' -v
//   git checkout b7845fb2da9a21bfb895e1c008f016776f8342d0
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzjudge3assign.gno$' -v
//   rm gnovm/tests/files/zzjudge3assign.gno

// PKGPATH: gno.land/r/demo/judge3a
package judge3a

func localIntCur(_ realm) int {
	var cur int // an int, not a realm; `_` frees the name in this block
	cur = 5     // plain ASSIGN, not DEFINE
	return cur
}

func main(cur realm) {
	println(localIntCur(cross(cur)))
}

// Output:
// 5
