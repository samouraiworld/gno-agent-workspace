/* Run: from a gno checkout:
gh pr checkout 5739 -R gnolang/gno && git checkout a00dde6b3
curl -fsSL -o gnovm/tests/files/iface_embed_sel_order.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/5-a00dde6b3/tests/iface_embed_sel_order.gno
go test -v -run 'TestFiles/iface_embed_sel_order.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/iface_embed_sel_order.gno
*/
// FindEmbeddedFieldType binds a method selection to the FIRST flattened entry
// whose name matches, gating only that one on origin package. At a00dde6b3 the
// embed-first form is rejected ("cannot access ...sec from main") because the
// foreign ifaceext.sec is hit before main's own sec; the own-first form
// compiles. Go binds x.sec() to main's own sec regardless of order, so both
// selections should type-check and this prints "ok".
package main

import "filetests/extern/ifaceext"

// embed-first: foreign sealed ifaceext.sec listed before main's own sec.
func selEmbedFirst(x interface {
	ifaceext.Sec
	sec() int
}) int {
	return x.sec()
}

// own-first: main's own sec listed before the foreign embed (compiles today).
func selOwnFirst(x interface {
	sec() int
	ifaceext.Sec
}) int {
	return x.sec()
}

func main() {
	_, _ = selEmbedFirst, selOwnFirst
	println("ok")
}

// Output:
// ok
