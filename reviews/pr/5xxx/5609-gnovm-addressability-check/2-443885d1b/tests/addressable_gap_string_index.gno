// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno && git checkout 443885d1b
curl -fsSL -o gnovm/tests/files/addressable_gap_string_index.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/tests/addressable_gap_string_index.gno
go test -v -run 'TestFiles/addressable_gap_string_index.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/addressable_gap_string_index.gno
*/

// Demonstrates that PR #5609's `isAddressable` accepts `&s[i]` for a string.
// IndexExpr only special-cases MapKind (false) and SliceKind (true); for
// StringKind it recurses into `isAddressable(cx.X)`, which for a name
// returns true. Go rejects address-of-string-byte at compile time.
//
// Observed: `// Output: 97` passes — `&s[0]` produces a usable byte pointer
// whose deref reads the first byte. Flip the assertion to // Error: once
// `case StringKind: return false` is added to IndexExpr.

package main

func main() {
	s := "abc"
	p := &s[0]
	println(*p)
}

// Output:
// 97

// TypeCheckError:
// main/addressable_gap_string_index.gno:23:8: invalid operation: cannot take address of s[0] (value of type byte)
