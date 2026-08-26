/* Run: from a gno checkout, green at 54b78448a: the Events golden shows the cleared payload while Output shows the default still charged:
gh pr checkout 6073 -R gnolang/gno && git checkout 54b78448a
mkdir -p examples/gno.land/p/demo/tokens/grc721/royalty/filetests
curl -fsSL -o examples/gno.land/p/demo/tokens/grc721/royalty/filetests/zprobe_events_filetest.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6073-grc721-metadata-royalty-enumerable/1-54b78448a/tests/royalty_delete_event_filetest.gno
cd examples/gno.land/p/demo/tokens/grc721/royalty && gno test -v .
cd - && rm -r examples/gno.land/p/demo/tokens/grc721/royalty/filetests
*/

// PKGPATH: gno.land/r/demo/zroyevents
package zroyevents

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/royalty"
)

func main(cur realm) {
	_, core := grc721.NewToken("Foo", "FOO", 0, cur)
	roy, led := royalty.NewRoyalty(core, 10000)
	me := cur.Address()

	core.Mint(me, "1")
	led.SetDefaultRoyalty(me, 500)   // collection default, 5%
	led.SetTokenRoyalty("1", me, 100) // per-token override, 1%

	led.DeleteTokenRoyalty("1") // emits receiver "" and bps 0

	_, amount, _ := roy.RoyaltyInfo("1", 10000)
	println("RoyaltyInfo after the delete:", amount)
}

// Output:
// RoyaltyInfo after the delete: 500

// Events:
// [
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/zroyevents.FOO.0000000"
//       },
//       {
//         "key": "name",
//         "value": "Foo"
//       },
//       {
//         "key": "symbol",
//         "value": "FOO"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721"
//   },
//   {
//     "type": "Transfer",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/zroyevents.FOO.0000000"
//       },
//       {
//         "key": "from",
//         "value": ""
//       },
//       {
//         "key": "to",
//         "value": "g15unle0qalt6wpzew8e0h5nx9me66qxpgun7eyh"
//       },
//       {
//         "key": "tokenId",
//         "value": "1"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721"
//   },
//   {
//     "type": "RoyaltyUpdate",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/zroyevents.FOO.0000000"
//       },
//       {
//         "key": "scope",
//         "value": "default"
//       },
//       {
//         "key": "tokenId",
//         "value": ""
//       },
//       {
//         "key": "receiver",
//         "value": "g15unle0qalt6wpzew8e0h5nx9me66qxpgun7eyh"
//       },
//       {
//         "key": "bps",
//         "value": "500"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721/royalty"
//   },
//   {
//     "type": "RoyaltyUpdate",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/zroyevents.FOO.0000000"
//       },
//       {
//         "key": "scope",
//         "value": "token"
//       },
//       {
//         "key": "tokenId",
//         "value": "1"
//       },
//       {
//         "key": "receiver",
//         "value": "g15unle0qalt6wpzew8e0h5nx9me66qxpgun7eyh"
//       },
//       {
//         "key": "bps",
//         "value": "100"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721/royalty"
//   },
//   {
//     "type": "RoyaltyUpdate",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/zroyevents.FOO.0000000"
//       },
//       {
//         "key": "scope",
//         "value": "token"
//       },
//       {
//         "key": "tokenId",
//         "value": "1"
//       },
//       {
//         "key": "receiver",
//         "value": ""
//       },
//       {
//         "key": "bps",
//         "value": "0"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721/royalty"
//   }
// ]
