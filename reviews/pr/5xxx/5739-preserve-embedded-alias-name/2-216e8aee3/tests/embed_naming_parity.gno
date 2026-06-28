/* Run: from a gno checkout:
gh pr checkout 5739 -R gnolang/gno && git checkout 216e8aee3
curl -fsSL -o gnovm/tests/files/embed_naming_parity.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/2-216e8aee3/tests/embed_naming_parity.gno
go test -v -run 'TestFiles/embed_naming_parity.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/embed_naming_parity.gno
*/
// Embedded-field naming parity with Go in one assertion bundle. Struct embeds
// take the written spelling (alias stays distinct from its target); interface
// embeds take the flattened method set (alias collapses to its target, and an
// embed equals the explicit method). All values below match a side-by-side Go
// run; embed_naming_parity_go.go is the oracle.
package main

type Int = int
type Stringer interface{ Str() string }
type SAlias = Stringer

func eq(a, b interface{}) bool { return a == b }

func main() {
	// struct: alias spelling kept, distinct from the target type
	println("struct alias!=target:", eq(struct{ Int }{}, struct{ int }{}))
	// struct embedding an interface alias: spelling kept, distinct
	println("struct iface-alias!=target:", eq(struct{ SAlias }{}, struct{ Stringer }{}))
	// interface: alias embed collapses to its target (method set identity)
	println("iface alias==target:", eq(struct{ A interface{ SAlias } }{}, struct{ A interface{ Stringer } }{}))
	// interface: embed equals the explicit method (name leaves the TypeID)
	println("iface embed==explicit:", eq(struct{ A interface{ Stringer } }{}, struct{ A interface{ Str() string } }{}))
}

// Output:
// struct alias!=target: false
// struct iface-alias!=target: false
// iface alias==target: true
// iface embed==explicit: true
