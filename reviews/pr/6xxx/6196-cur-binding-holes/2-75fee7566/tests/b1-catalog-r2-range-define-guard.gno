// Repro for: the `n.Op != DEFINE` wrapper on the RangeStmt cur rule
// (gnovm/pkg/gnolang/preprocess.go:3113) is dead at PR 6196 head 6d88deba7.
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head && git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zz_cur_rangedefine.gno
//   # 1. unmodified head: the range-DEFINE shadow is legal
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_cur_rangedefine.gno$'
//   # 2. delete the `if n.Op != DEFINE {` wrapper (and its closing brace) around
//   #    the range-clause cur rule in preprocess.go, leaving the loop, then:
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_cur_rangedefine.gno$'
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_range.gno$'
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_shadow_range.gno$'
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_legal.gno$'
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_shadow.gno$'
//
// Measured at 6d88deba7: all five read `ok` with the wrapper and `ok` without
// it. A range DEFINE binds its key/value in the RangeStmt's own block, so
// isCrossingCurParam resolves the name to a *RangeStmt and returns false
// whatever the op; the wrapper filters nothing. The same wrapper was deleted
// from the AssignStmt rule in this diff, which is where the asymmetry shows.

// PKGPATH: gno.land/r/demo/currangedefine
package currangedefine

func main(cur realm) {
	xs := []int{7, 8}
	for cur := range xs {
		println(cur)
	}
	println(cur.PkgPath())
}

// Output:
// 0
// 1
// gno.land/r/demo/currangedefine
