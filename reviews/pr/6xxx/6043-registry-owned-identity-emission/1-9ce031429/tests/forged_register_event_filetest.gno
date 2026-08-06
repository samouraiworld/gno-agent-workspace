/* Run: from a gno checkout:
gh pr checkout 6043 -R gnolang/gno && git checkout 9ce031429
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/filetests/forged_register_event_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6043-registry-owned-identity-emission/1-9ce031429/tests/forged_register_event_filetest.gno
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/forged_register_event_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20regforge

// Emit forwards kind as well as attrs, so a handle emits the registry's own
// register event. At 9ce031429 the row below is byte-identical to one Register
// would produce for wugnot, from a realm that owns neither that namespace nor
// that symbol, while the registry holds no such key. The golden asserts the
// empty event stream a handle bound to token events would leave.
package grc20regforge

import (
	"gno.land/r/demo/defi/grc20reg"
)

func main(cur realm) {
	handle := grc20reg.Emitter().IssueToken(cross(cur), "junk")

	handle.Emit("register",
		"token_path", "gno.land/r/gnoland/wugnot.wugnot",
		"token_id", "gno.land/r/gnoland/wugnot.wugnot.1",
		"pkgpath", "gno.land/r/gnoland/wugnot",
		"slug", "",
		"symbol", "wugnot",
		"emitter", "gno.land/r/demo/defi/grc20reg")

	println("registry has wugnot:", grc20reg.Get("gno.land/r/gnoland/wugnot.wugnot") != nil)
}

// Output:
// registry has wugnot: false

// Events:
// null
