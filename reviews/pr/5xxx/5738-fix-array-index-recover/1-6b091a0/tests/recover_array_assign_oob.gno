// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5738 -R gnolang/gno && git checkout 6b091a0
curl -fsSL -o gnovm/tests/files/recover_array_assign_oob.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5738-fix-array-index-recover/1-6b091a0/tests/recover_array_assign_oob.gno
go test -v -run 'TestFiles/recover_array_assign_oob.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/recover_array_assign_oob.gno
*/

// Confirms the assignment LHS also routes through GetPointerAtIndexInt2 and
// is therefore recoverable post-fix. Before the fix, `a[i] = v` with i out of
// range would equally have produced a native Go panic that escaped recover().

package main

var a [4]int

func main() {
	defer func() {
		println("recover:", recover())
	}()
	i := 99
	a[i] = 1
}

// Output:
// recover: runtime error: index out of range [99] with length 4
