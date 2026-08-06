/* Run: from a gno checkout:
gh pr checkout 6043 -R gnolang/gno && git checkout 9ce031429
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/filetests/forged_token_event_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6043-registry-owned-identity-emission/1-9ce031429/tests/forged_token_event_filetest.gno
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/forged_token_event_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20forge

// Emitter() hands any realm a TokenEmitter, and tokenEmitter.Emit forwards its
// attrs untouched. At 9ce031429 the event below carries the token attribute
// this realm typed, gno.land/r/demo/victim.VIC.1, under grc20reg's pkg_path.
// The golden asserts the handle's own bound identifier instead.
package grc20forge

import (
	"gno.land/r/demo/defi/grc20reg"
)

func main(cur realm) {
	handle := grc20reg.Emitter().IssueToken(cross(cur), "X")
	handle.Emit("Transfer",
		"token", "gno.land/r/demo/victim.VIC.1",
		"from", "g1victim000000000000000000000000000000",
		"to", "g1attacker0000000000000000000000000000",
		"value", "1000000")
}

// Events:
// [
//   {
//     "type": "Transfer",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/grc20forge.X.1"
//       },
//       {
//         "key": "from",
//         "value": "g1victim000000000000000000000000000000"
//       },
//       {
//         "key": "to",
//         "value": "g1attacker0000000000000000000000000000"
//       },
//       {
//         "key": "value",
//         "value": "1000000"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc20reg"
//   }
// ]
