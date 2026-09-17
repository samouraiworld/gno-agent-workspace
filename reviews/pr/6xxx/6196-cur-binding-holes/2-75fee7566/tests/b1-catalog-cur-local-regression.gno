// Repro: PR 6196 (head 6d88deba7) rejects a function-scope name `cur` of ANY
// type inside a crossing function whose first realm parameter is not named
// `cur` (the common `_ realm` shape). `isCrossingCurParam` keys on the name
// plus the declaring node and dropped the base rule's realm-type test, and the
// `n.Op != DEFINE` carve-out was removed, so even the declaration `cur := 0`
// is refused.
//
// Head:  preprocess panic at the `cur := 0` line —
//        "cannot reassign the crossing `cur` parameter: ..."
// Base (b7845fb2d): ok, prints 6.
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba7
//   cp <this file> gnovm/tests/files/zzb1intlocal.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzb1intlocal.gno$'
//   # head: FAIL, preprocess panic. Then:
//   git checkout b7845fb2da9a21bfb895e1c008f016776f8342d0
//   cp <this file> gnovm/tests/files/zzb1intlocal.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzb1intlocal.gno$'
//   # base: ok

// PKGPATH: gno.land/r/demo/curzz6
package curzz6

func F(_ realm, xs []int) int {
	cur := 0
	for _, x := range xs {
		cur = cur + x
	}
	return cur
}

func main(cur realm) {
	println(F(cross(cur), []int{1, 2, 3}))
}

// Output:
// 6
