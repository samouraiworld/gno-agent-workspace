/* Run: from a gno checkout:
gh pr checkout 5963 -R gnolang/gno && git checkout a94c9098
curl -fsSL -o gnovm/tests/files/goto_switch_loops.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5963-gotojump-nested-loop-crash/1-a94c9098/tests/goto_switch_loops.gno
go test -v -run 'TestFiles/goto_switch_loops.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/goto_switch_loops.gno
*/
package main

// Backward goto out of two loops nested in a switch clause to a func-body label.
// GotoJump crosses a SwitchClauseStmt frame plus two ForStmt frames (depth 3);
// before the fix the stmt-stack truncation underflowed to [:-1].

func main() {
	i := 0
top:
	println("i =", i)
	switch {
	case true:
		for a := 0; a < 2; a++ {
			for b := 0; b < 2; b++ {
				i++
				if i < 3 {
					goto top
				}
			}
		}
	}
	println("done", i)
}

// Output:
// i = 0
// i = 1
// i = 2
// done 6
