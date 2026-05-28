// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5738 -R gnolang/gno && git checkout 6b091a0
curl -fsSL -o gnovm/tests/files/recover_array_neg_int.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5738-fix-array-index-recover/1-6b091a0/tests/recover_array_neg_int.gno
go test -v -run 'TestFiles/recover_array_neg_int.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/recover_array_neg_int.gno
*/

// Covers ArrayValue.GetPointerAtIndexInt2's `Data == nil && ii < 0` branch:
// a non-byte array indexed by a negative int. The provided recover25.gno only
// exercises positive OOB; this round-trips the negative branch.

package main

var a [4]int

func main() {
	defer func() {
		println("recover:", recover())
	}()
	i := -1
	_ = a[i]
}

// Output:
// recover: runtime error: index out of range [-1]
