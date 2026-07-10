/* Run: from a gno checkout:
gh pr checkout 5721 -R gnolang/gno && git checkout 16d5227b9
curl -fsSL -o gnovm/tests/files/embed_iface_ambiguous.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5721-shallowest-match-embedded-lookup/1-16d5227b9/tests/embed_iface_ambiguous.gno
go test -v -run 'TestFiles/embed_iface_ambiguous.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/embed_iface_ambiguous.gno
*/

// Covers the VerifyImplementedBy path, which the PR's selector-expression
// tests never exercise: a type whose method is ambiguous (two embedded
// siblings provide Foo at the same depth) does not satisfy an interface
// requiring it. The rejection is correct and matches Go. The VM message
// says "missing method Foo"; go/types says "ambiguous selector C.Foo",
// and the direct-selector path in this PR reports "ambiguous selector".
package main

type I interface{ Foo() string }

type A struct{}

func (A) Foo() string { return "a" }

type B struct{}

func (B) Foo() string { return "b" }

type C struct {
	A
	B
}

func main() {
	var i I = C{}
	_ = i
}

// Error:
// main/embed_iface_ambiguous.gno:33:6-15: main.C does not implement main.I (missing method Foo)

// TypeCheckError:
// main/embed_iface_ambiguous.gno:33:12: cannot use C{} (value of struct type C) as I value in variable declaration: C does not implement I (ambiguous selector C.Foo)
