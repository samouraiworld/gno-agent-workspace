// Asserts the address of a plain `int` named `cur` inside a crossing function
// with a `_ realm` first parameter may be taken.
// Measured: PASS at merge base b7845fb2d, preprocess panic at head 6d88deba7
// claiming the value is realm-typed. Fails at the reviewed head.
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba7
//   cp <this file> gnovm/tests/files/zzjudge3addr.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzjudge3addr.gno$' -v
//   git checkout b7845fb2da9a21bfb895e1c008f016776f8342d0
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzjudge3addr.gno$' -v
//   rm gnovm/tests/files/zzjudge3addr.gno

// PKGPATH: gno.land/r/demo/judge3b
package judge3b

func addrIntCur(_ realm) int {
	var cur int
	p := &cur // &int, refused at head as "realm-typed `cur`"
	*p = 7
	return cur
}

func main(cur realm) {
	println(addrIntCur(cross(cur)))
}

// Output:
// 7
