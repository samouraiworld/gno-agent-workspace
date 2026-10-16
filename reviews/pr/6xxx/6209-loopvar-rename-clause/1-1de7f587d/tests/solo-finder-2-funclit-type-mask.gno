// Repro, from a plain clone of gnolang/gno:
//   git checkout 1de7f587da469abad9ce4cf5fa945f3328aa0e40
//   cp <this file> gnovm/tests/files/zz_funclit_type_mask.gno
//   cd gnovm/pkg/gnolang && go test -short -run 'TestFiles/zz_funclit_type_mask.gno' .
// Head 1de7f587d: Output diff "-2 +3", no error: the result type [len(x)]int
// resolves x to the outer [3]int instead of the loop's [2]int.
// Merge base bbd9b2ffe (preprocess.go + transcribe.go reverted): the same "-2 +3".
// Go: `go run` of the same main prints 2.
// Fix shape (solo-finder-2-funclit-type-mask.patch): push the FuncLitExpr frame
// in initStaticBlocks1 at TRANS_BLOCK instead of TRANS_ENTER, so params and
// results are masked only after the FuncTypeExpr is walked: prints 2, and the
// param variant below preprocesses; 372 TestFiles matching
// (loopvar|closure|heap|range|for|switch|typeswitch|fun|var|block|shadow) stay
// green apart from switch13.gno, whose go/types message differs on the local
// toolchain with or without the patch.
//
// Param variant, refused instead of miscompiled (same at head and base):
//   for x := [2]int{}; x[0] < 1; x[0]++ { f := func(x [len(x)]int) int { return len(x) }; println(f([2]int{})) }
//   -> main/zz.gno:5:20-21: name x not declared   (Go prints 2)
package main

func main() {
	x := [3]int{}
	_ = x
	for x := [2]int{}; x[0] < 1; x[0]++ {
		f := func() (x [len(x)]int) { return }
		println(len(f()))
	}
}

// Output:
// 2
