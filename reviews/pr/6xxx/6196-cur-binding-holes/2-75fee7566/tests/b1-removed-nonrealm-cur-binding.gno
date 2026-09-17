// PKGPATH: gno.land/r/demo/curnonrealmlocal
package curnonrealmlocal

// b1-removed: isCrossingCurParam (preprocess.go:6111) dropped the realm static
// type test the three write rules carried, so every name `cur` declared inside
// a crossing function's own block is now unwritable whatever its type. A
// crossing function whose first realm parameter is unnamed (`_ realm`, 56
// occurrences under examples/ at this sha) owns that block, so an ordinary
// cursor local named `cur` no longer compiles.
//
// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6196/head
//	git checkout 6d88deba71165b165641b65541259d00aba20377
//	cp <this file> gnovm/tests/files/zrealm_cur_nonrealm_local.gno
//	go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_nonrealm_local.gno$' -v
//
// head 6d88deba7:
//
//	unexpected panic: gno.land/r/demo/curnonrealmlocal/zrealm_cur_nonrealm_local.gno:8:2-10:
//	cannot reassign the crossing `cur` parameter: it names the realm this frame is
//	executing as, and that binding is fixed for the life of the frame
//
// merge base b7845fb2d, same file, base preprocess.go only:
//
//	git checkout b7845fb2da9a21bfb895e1c008f016776f8342d0 -- gnovm/pkg/gnolang/preprocess.go
//	go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_nonrealm_local.gno$' -v
//	--- PASS: TestFiles/zrealm_cur_nonrealm_local.gno
//
// Three more shapes of the same class, each PASS on b7845fb2d and FAIL on
// 6d88deba7, all inside `func f(_ realm) (cur int)`:
//
//	cur = 3                        -> cannot reassign the crossing `cur` parameter
//	p := &cur                      -> cannot take the address of a realm-typed `cur`
//	for _, cur = range []int{7} {} -> cannot assign to a realm-typed `cur` in a range clause
//
// Run them together with -run 'TestFiles/zrealm_cur_nonrealm'.

func sum(_ realm, n int) int {
	cur := 0
	for i := 0; i < n; i++ {
		cur += i
	}
	return cur
}

func main(cur realm) {
	println(sum(cross(cur), 4))
}

// Output:
// 6
