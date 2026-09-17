// Repro, from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_cur_range_def.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/(zrealm_cur_range_def|zrealm_cur_range|zrealm_cur_shadow_range).gno$'
//   # ok  github.com/gnolang/gno/gnovm/pkg/gnolang  0.052s
//
//   # now delete the guard the RangeStmt rule still carries:
//   sed -i '3113s/if n.Op != DEFINE {/if true {/' gnovm/pkg/gnolang/preprocess.go
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/(zrealm_cur_range_def|zrealm_cur_range|zrealm_cur_shadow_range).gno$'
//   # ok  github.com/gnolang/gno/gnovm/pkg/gnolang  0.044s
//   git checkout -- gnovm/pkg/gnolang/preprocess.go
//
// Same three results with the guard and without it: a range DEFINE binds the
// name in the RangeStmt's own block, which isCrossingCurParam can never resolve
// to a crossing FuncDecl or FuncLitExpr, so `n.Op != DEFINE` can no longer
// change the rule's answer. The sibling assign rule dropped the same guard at
// preprocess.go:3011.

// PKGPATH: gno.land/r/demo/currangedef
package currangedef

func main(_ realm) {
	xs := []int{10, 20}
	for cur := range xs {
		println(cur)
	}
}

// Output:
// 0
// 1
