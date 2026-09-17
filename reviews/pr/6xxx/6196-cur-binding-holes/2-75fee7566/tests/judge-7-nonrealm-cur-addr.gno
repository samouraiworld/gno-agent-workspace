// Asserts that `&cur` on an int named result inside a crossing function stays
// legal: nothing realm-typed, no frame identity. Head 6d88deba7 refuses it with
// the realm-typed wording; base b7845fb2d prints 5.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zzb1addr.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzb1addr.gno$'
//   git checkout b7845fb2da9a21bfb895e1c008f016776f8342d0   # same commands: ok

// PKGPATH: gno.land/r/demo/b1addr
package b1addr

func F(_ realm) (cur int) {
	p := &cur
	*p = 5
	return
}

func main(cur realm) {
	println(F(cross(cur)))
}

// Output:
// 5
