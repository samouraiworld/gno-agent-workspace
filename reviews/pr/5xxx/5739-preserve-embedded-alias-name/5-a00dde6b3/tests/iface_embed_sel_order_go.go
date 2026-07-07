/* Go parity oracle for iface_embed_sel_order.gno. Go accepts BOTH orderings
   (binding x.sec() to main's own sec); gno at a00dde6b3 rejects the embed-first
   form. Run from an empty dir:

d=$(mktemp -d); cd "$d"; printf 'module oracle\n\ngo 1.23\n' > go.mod
mkdir -p ifaceext
printf 'package ifaceext\ntype Sec interface{ sec() int }\n' > ifaceext/ifaceext.go
# drop the body of this file below (from "package main") into main.go, then:
go build ./...   # -> succeeds: Go accepts both method orders
cd /; rm -rf "$d"
*/

package main

import "oracle/ifaceext"

// embed-first: identical to gno selEmbedFirst; Go accepts it.
func selEmbedFirst(x interface {
	ifaceext.Sec
	sec() int
}) int {
	return x.sec()
}

// own-first: identical to gno selOwnFirst.
func selOwnFirst(x interface {
	sec() int
	ifaceext.Sec
}) int {
	return x.sec()
}

func main() {
	_, _ = selEmbedFirst, selOwnFirst
	println("both orders compile in Go")
}
