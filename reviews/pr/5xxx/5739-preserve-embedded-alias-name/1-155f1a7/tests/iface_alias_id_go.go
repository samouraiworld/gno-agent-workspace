/* Run: go parity for iface_alias_id.gno (the embedded-interface identity case):
go run iface_alias_id_go.go
Prints "alias==canonical: true": Go treats interface{SAlias} and
interface{Stringer} as the identical type, the result the gno filetest asserts.
*/
package main

type Stringer interface{ Str() string }
type SAlias = Stringer

func main() {
	var x, y interface{} = struct{ A interface{ SAlias } }{}, struct{ A interface{ Stringer } }{}
	println("alias==canonical:", x == y)
	var p, q interface{} = struct{ A interface{ Stringer } }{}, struct{ A interface{ Stringer } }{}
	println("canonical==canonical:", p == q)
}
