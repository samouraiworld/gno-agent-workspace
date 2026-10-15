// PKGPATH: gno.land/r/demo/regprobe

// b3-reach: publishedKinds applies its limit before dropping view-less kinds.
// Repro from a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//   cp b3-reach-viewless-cap.gno examples/gno.land/r/demo/defi/grc721reg/z_reach_filetest.gno
//   cd examples && gno test -v ./gno.land/r/demo/defi/grc721reg
// The run PASSES against the golden Output/Events below, which record the defect:
//   A: 17 published kinds (> maxEventKinds 16) registers without "too many extensions",
//      and its register event carries 16 "extension" attrs (k116 is missing).
//   B: one published kind, yet Render(key) shows "_core only_".
//   C: nine published kinds (> maxBadges 8), 8 badges and no elision mark.
package regprobe

import (
	"strconv"
	"strings"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
)

type ext struct{ kind, tokenID string }

func (v *ext) ExtensionKind() string                       { return v.kind }
func (v *ext) TokenID() string                             { return v.tokenID }
func (v *ext) OnMint(address, grc721.TokenID)              {}
func (v *ext) OnTransfer(address, address, grc721.TokenID) {}
func (v *ext) OnBurn(grc721.TokenID)                       {}

func main(cur realm) {
	// A: one hook-only extension first, then 17 with a view (17 > maxEventKinds=16).
	a, aLed := grc721.NewToken("Cap", "CAP", 0, cur)
	aLed.RegisterExtension(&ext{"hook", a.ID()}, nil)
	for i := 0; i < 17; i++ {
		v := &ext{"k" + strconv.Itoa(100+i), a.ID()}
		aLed.RegisterExtension(v, v)
	}
	keyA := grc721reg.Register(cross(cur), a, "")
	println("A registered, published kinds:", len(grc721reg.GetExtensionKinds(keyA)))

	// B: nine hook-only extensions first, then one with a view.
	b, bLed := grc721.NewToken("Badge", "BADGE", 0, cur)
	for i := 0; i < 9; i++ {
		bLed.RegisterExtension(&ext{"h" + strconv.Itoa(i), b.ID()}, nil)
	}
	bv := &ext{"zview", b.ID()}
	bLed.RegisterExtension(bv, bv)
	keyB := grc721reg.Register(cross(cur), b, "")
	println("B published kinds:", len(grc721reg.GetExtensionKinds(keyB)))
	println("B page says core only:", strings.Contains(grc721reg.Render(keyB), "_core only_"))

	// C: one hook-only first, then 9 with a view (9 > maxBadges=8).
	c, cLed := grc721.NewToken("Elide", "ELIDE", 0, cur)
	cLed.RegisterExtension(&ext{"hook", c.ID()}, nil)
	for i := 0; i < 9; i++ {
		v := &ext{"k" + strconv.Itoa(i), c.ID()}
		cLed.RegisterExtension(v, v)
	}
	keyC := grc721reg.Register(cross(cur), c, "")
	pageC := grc721reg.Render(keyC)
	println("C published kinds:", len(grc721reg.GetExtensionKinds(keyC)))
	println("C badges:", strings.Count(pageC, "`k"), "elided mark:", strings.Contains(pageC, "…"))
}

// Output:
// A registered, published kinds: 17
// B published kinds: 1
// B page says core only: true
// C published kinds: 9
// C badges: 8 elided mark: false

// Events:
// [
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/regprobe.CAP.0000000"
//       },
//       {
//         "key": "name",
//         "value": "Cap"
//       },
//       {
//         "key": "symbol",
//         "value": "CAP"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/regprobe.CAP"
//       },
//       {
//         "key": "token_id",
//         "value": "gno.land/r/demo/regprobe.CAP.0000000"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/regprobe"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "CAP"
//       },
//       {
//         "key": "extension",
//         "value": "k100"
//       },
//       {
//         "key": "extension",
//         "value": "k101"
//       },
//       {
//         "key": "extension",
//         "value": "k102"
//       },
//       {
//         "key": "extension",
//         "value": "k103"
//       },
//       {
//         "key": "extension",
//         "value": "k104"
//       },
//       {
//         "key": "extension",
//         "value": "k105"
//       },
//       {
//         "key": "extension",
//         "value": "k106"
//       },
//       {
//         "key": "extension",
//         "value": "k107"
//       },
//       {
//         "key": "extension",
//         "value": "k108"
//       },
//       {
//         "key": "extension",
//         "value": "k109"
//       },
//       {
//         "key": "extension",
//         "value": "k110"
//       },
//       {
//         "key": "extension",
//         "value": "k111"
//       },
//       {
//         "key": "extension",
//         "value": "k112"
//       },
//       {
//         "key": "extension",
//         "value": "k113"
//       },
//       {
//         "key": "extension",
//         "value": "k114"
//       },
//       {
//         "key": "extension",
//         "value": "k115"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc721reg"
//   },
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/regprobe.BADGE.0000000"
//       },
//       {
//         "key": "name",
//         "value": "Badge"
//       },
//       {
//         "key": "symbol",
//         "value": "BADGE"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/regprobe.BADGE"
//       },
//       {
//         "key": "token_id",
//         "value": "gno.land/r/demo/regprobe.BADGE.0000000"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/regprobe"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "BADGE"
//       },
//       {
//         "key": "extension",
//         "value": "zview"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc721reg"
//   },
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/regprobe.ELIDE.0000000"
//       },
//       {
//         "key": "name",
//         "value": "Elide"
//       },
//       {
//         "key": "symbol",
//         "value": "ELIDE"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc721"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/regprobe.ELIDE"
//       },
//       {
//         "key": "token_id",
//         "value": "gno.land/r/demo/regprobe.ELIDE.0000000"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/regprobe"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "ELIDE"
//       },
//       {
//         "key": "extension",
//         "value": "k0"
//       },
//       {
//         "key": "extension",
//         "value": "k1"
//       },
//       {
//         "key": "extension",
//         "value": "k2"
//       },
//       {
//         "key": "extension",
//         "value": "k3"
//       },
//       {
//         "key": "extension",
//         "value": "k4"
//       },
//       {
//         "key": "extension",
//         "value": "k5"
//       },
//       {
//         "key": "extension",
//         "value": "k6"
//       },
//       {
//         "key": "extension",
//         "value": "k7"
//       },
//       {
//         "key": "extension",
//         "value": "k8"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc721reg"
//   }
// ]
