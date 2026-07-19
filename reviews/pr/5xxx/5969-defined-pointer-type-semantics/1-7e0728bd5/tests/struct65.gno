/* Run: from a gno checkout:
gh pr checkout 5969 -R gnolang/gno && git checkout 7e0728bd5
curl -fsSL -o gnovm/tests/files/struct65.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5969-defined-pointer-type-semantics/1-7e0728bd5/tests/struct65.gno
go test -v -run 'TestFiles/struct65.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/struct65.gno
*/

// Go rejects an embedded pointer to an interface type at declaration, from the
// same struct-field check that rejects an embedded pointer type. At 7e0728bd5
// fillEmbeddedName accepts it, and selecting a name through it panics
// "should not happen".
package main

type I interface{ M() string }

type S struct{ *I }

func main() {
	var s S
	_ = s.M
}

// Error:
// main/struct65.gno:19:8-18: embedded field type cannot be a pointer to an interface: *main.I

// TypeCheckError:
// main/struct65.gno:19:16: embedded field type cannot be a pointer to an interface
