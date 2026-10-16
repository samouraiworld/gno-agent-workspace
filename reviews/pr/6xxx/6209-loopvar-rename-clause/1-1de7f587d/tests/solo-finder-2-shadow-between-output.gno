// Repro, from a plain clone of gnolang/gno:
//   cp <this file> gnovm/tests/files/zz_shadow_between.gno
//   cd gnovm/pkg/gnolang && go test -short -run 'TestFiles/zz_shadow_between.gno' .
// at 1de7f587d (head) and at bbd9b2ffe (merge base).
// Head: passes, prints 5 6 (Go prints 5 6).
// Merge base: preprocesses and runs, Output diff "+0 +1 +2 +3 +4" before 5 6:
// the inner init RHS x was renamed to the outer loop's x.loopvar.
// A program master accepts changes output (and the work it does) under this PR.
package main

func main() {
	for x := 0; x < 1; x++ {
		x := 5
		for x := x; x < 7; x++ {
			println(x)
		}
	}
}

// Output:
// 5
// 6
