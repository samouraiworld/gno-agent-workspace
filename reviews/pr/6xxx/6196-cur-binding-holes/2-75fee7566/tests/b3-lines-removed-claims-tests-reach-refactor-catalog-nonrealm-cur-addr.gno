// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 6d88deba71165b165641b65541259d00aba20377
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/PROBE.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/PROBE.gno$' -v
// Base is b7845fb2da9a21bfb895e1c008f016776f8342d0 (same commands, that sha).
// head 6d88deba7: FAIL  base b7845fb2d: PASS
//
// Same binding through the address-of rule (preprocess.go:2583). The panic
// reads "cannot take the address of a realm-typed `cur` ... `realm` is an
// interface, so a *realm is never needed" about an int.

// PKGPATH: gno.land/r/demo/curprobeg
package curprobeg

func F(_ realm, cur int) int {
	p := &cur
	*p = 7
	return cur
}

func main() {
	println("ok")
}

// Output:
// ok
