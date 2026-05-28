// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5738 -R gnolang/gno && git checkout 6b091a0
curl -fsSL -o gnovm/tests/files/recover_byte_array_oob.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5738-fix-array-index-recover/1-6b091a0/tests/recover_byte_array_oob.gno
go test -v -run 'TestFiles/recover_byte_array_oob.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/recover_byte_array_oob.gno
*/

// Covers ArrayValue.GetPointerAtIndexInt2's `Data != nil` branch — fixed-size
// byte arrays use the Data slot rather than List. The PR fix added bounds
// checks there too, but the supplied recover25.gno only exercises the
// `Data == nil` (List) path.

package main

var b [8]byte

func main() {
	defer func() {
		println("recover:", recover())
	}()
	i := 10
	_ = b[i]
}

// Output:
// recover: runtime error: index out of range [10] with length 8
