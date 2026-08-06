/* Run: from a gno checkout:
gh pr checkout 6043 -R gnolang/gno && git checkout 9ce031429
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/filetests/forged_provenance_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6043-registry-owned-identity-emission/1-9ce031429/tests/forged_provenance_filetest.gno
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/forged_provenance_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20prov

// IssuedTo records the identifier string, not the handle the token holds, so a
// realm can take an identifier from IssueToken and build its token on a
// counterfeit Emitter carrying the same string. At 9ce031429 Register reports
// emitter gno.land/r/demo/defi/grc20reg for the token below, whose Mint emits
// nothing. The golden asserts the empty emitter that token has earned.
package grc20prov

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

type fake struct{ id string }

func (f *fake) IssueToken(cur realm, symbol string) grc20.TokenEmitter { return f }
func (f *fake) TokenID() string                                        { return f.id }
func (f *fake) Emit(kind string, attrs ...string)                      {}

func main(cur realm) {
	handle := grc20reg.Emitter().IssueToken(cross(cur), "VIC")
	token, ledger := grc20.NewTokenWithEmitter("Victim", "VIC", 6, &fake{id: handle.TokenID()}, cur)
	grc20reg.Register(cross(cur), token, "")
	ledger.Mint(cur.Address(), 1)
}

// Events:
// [
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/grc20prov.VIC"
//       },
//       {
//         "key": "token_id",
//         "value": "gno.land/r/demo/grc20prov.VIC.1"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/grc20prov"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "VIC"
//       },
//       {
//         "key": "emitter",
//         "value": ""
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc20reg"
//   }
// ]
