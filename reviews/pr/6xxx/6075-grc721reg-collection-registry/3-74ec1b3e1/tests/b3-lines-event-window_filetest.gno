// PKGPATH: gno.land/r/demo/lines_evt

// b3-lines-event-window: the register event's extension list is cut from the
// first maxEventKinds+1 kinds the issuer registered, view-less hooks included,
// so a published kind registered after 17 hooks is missing from the event while
// GetExtensionKinds reports it. The golden below is what head emits: Output
// lists one published kind, and the "register" event carries no "extension"
// attribute. Green at head is the defect reproduced.
//
// Repro from a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//
//   cp b3-lines-event-window_filetest.gno examples/gno.land/r/demo/defi/grc721reg/zz_lines_event_filetest.gno
//   cd examples && gno test ./gno.land/r/demo/defi/grc721reg -v 2>&1 | grep -A1 zz_lines_event
//   rm gno.land/r/demo/defi/grc721reg/zz_lines_event_filetest.gno
package lines_evt

import (
	"strconv"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
)

type ext struct{ kind, id string }

func (e *ext) ExtensionKind() string                       { return e.kind }
func (e *ext) TokenID() string                             { return e.id }
func (e *ext) OnMint(address, grc721.TokenID)              {}
func (e *ext) OnTransfer(address, address, grc721.TokenID) {}
func (e *ext) OnBurn(grc721.TokenID)                       {}

func main(cur realm) {
	token, ledger := grc721.NewToken("Evt", "EVT", 0, cur)
	for i := 0; i < 17; i++ {
		ledger.RegisterExtension(&ext{"hook" + strconv.Itoa(i), token.ID()}, nil)
	}
	pub := &ext{"published", token.ID()}
	ledger.RegisterExtension(pub, pub)

	key := grc721reg.Register(cross(cur), token, "")
	println("published kinds:", grc721reg.GetExtensionKinds(key))
}

// Output:
// published kinds: slice[("published" string)]

// Events:
// [
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/lines_evt.EVT.0000000"
//       },
//       {
//         "key": "name",
//         "value": "Evt"
//       },
//       {
//         "key": "symbol",
//         "value": "EVT"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/lines_evt.EVT"
//       },
//       {
//         "key": "token_id",
//         "value": "gno.land/r/demo/lines_evt.EVT.0000000"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/lines_evt"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "EVT"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc721reg"
//   }
// ]
