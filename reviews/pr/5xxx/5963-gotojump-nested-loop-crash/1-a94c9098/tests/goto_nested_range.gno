/* Run: from a gno checkout:
gh pr checkout 5963 -R gnolang/gno && git checkout a94c9098
curl -fsSL -o gnovm/tests/files/goto_nested_range.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5963-gotojump-nested-loop-crash/1-a94c9098/tests/goto_nested_range.gno
go test -v -run 'TestFiles/goto_nested_range.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/goto_nested_range.gno
*/
package main

// Backward goto out of three nested RANGE loops to a func-body label.
// GotoJump crosses three RangeStmt frames; before the fix the stmt-stack
// truncation underflowed to [:-1]. goto10.gno covers only ForStmt frames.

func main() {
	i := 0
	s := []int{0, 1}
top:
	println("i =", i)
	for range s {
		for range s {
			for range s {
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
// done 10
