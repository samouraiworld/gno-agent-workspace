/* Run: from a gno checkout:
gh pr checkout 5739 -R gnolang/gno && git checkout 155f1a7
curl -fsSL -o gnovm/tests/files/iface_alias_id.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/1-155f1a7/tests/iface_alias_id.gno
go test -v -run 'TestFiles/iface_alias_id.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/iface_alias_id.gno
*/
// An interface embedded in an interface adds only its method set, so in Go
// interface{SAlias} and interface{Stringer} are the identical type. At 155f1a7
// fillEmbeddedName feeds the written alias spelling into the interface TypeID,
// splitting them; asserts the Go result (true), which fails until interface
// embeds stop deriving identity from the source name.
package main

type Stringer interface{ Str() string }
type SAlias = Stringer

func main() {
	// alias embed vs canonical embed of the same interface: identical in Go
	var x, y interface{} = struct{ A interface{ SAlias } }{}, struct{ A interface{ Stringer } }{}
	println("alias==canonical:", x == y)
	// baseline: two canonical embeds stay identical
	var p, q interface{} = struct{ A interface{ Stringer } }{}, struct{ A interface{ Stringer } }{}
	println("canonical==canonical:", p == q)
}

// Output:
// alias==canonical: true
// canonical==canonical: true
