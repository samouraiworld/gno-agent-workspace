/* Run: go parity oracle for embed_naming_parity.gno:
go run embed_naming_parity_go.go
Prints the four results the gno filetest asserts: struct embeds keep the written
spelling (alias != target), interface embeds use the flattened method set (alias
== target, embed == explicit).
*/
package main

type Int = int
type Stringer interface{ Str() string }
type SAlias = Stringer

func eq(a, b interface{}) bool { return a == b }

func main() {
	println("struct alias!=target:", eq(struct{ Int }{}, struct{ int }{}))
	println("struct iface-alias!=target:", eq(struct{ SAlias }{}, struct{ Stringer }{}))
	println("iface alias==target:", eq(struct{ A interface{ SAlias } }{}, struct{ A interface{ Stringer } }{}))
	println("iface embed==explicit:", eq(struct{ A interface{ Stringer } }{}, struct{ A interface{ Str() string } }{}))
}
