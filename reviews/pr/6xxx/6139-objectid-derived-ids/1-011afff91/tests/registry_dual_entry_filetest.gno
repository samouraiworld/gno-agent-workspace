// Repro, from a plain clone of github.com/gnolang/gno:
//
//	mkdir -p examples/gno.land/r/demo/grc20dualreg
//	printf 'module = "gno.land/r/demo/grc20dualreg"\ngno = "0.9"\n' \
//	    > examples/gno.land/r/demo/grc20dualreg/gnomod.toml
//	cp registry_dual_entry_filetest.gno examples/gno.land/r/demo/grc20dualreg/
//	cd examples/gno.land/r/demo/grc20dualreg
//	GNOROOT=$(git rev-parse --show-toplevel) go run ../../../../../gnovm/cmd/gno test -v .
//
// Both copies of a token pass grc20reg.Register's provenance check and land as
// two public entries, and both register events announce one token_id while the
// two ledgers behind them emit two. It passes at the reviewed head: the
// duplicated token_id below is the defect.

// PKGPATH: gno.land/r/demo/grc20dualreg
package grc20dualreg

import (
	"chain"

	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

type Pair struct {
	A grc20.Token
	B grc20.Token
}

var (
	origA, origB     *grc20.Token
	ledgerA, ledgerB *grc20.PrivateLedger
	pair             Pair
)

func init(cur realm) {
	origA, ledgerA = grc20.NewToken("Alpha", "ALPHA", 4, cur)
	origB, ledgerB = grc20.NewToken("Beta", "BETA", 4, cur)
	// origRealm copies with the struct, so the provenance check passes on
	// &pair.A and &pair.B as readily as on the originals.
	pair.A, pair.B = *origA, *origB
}

func main(cur realm) {
	holder := chain.PackageAddress("holder")
	ledgerA.Mint(holder, 111)
	ledgerB.Mint(holder, 222)

	grc20reg.Register(cross(cur), &pair.A, "alpha")
	grc20reg.Register(cross(cur), &pair.B, "beta")
}

// Events:
// [
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "id",
//         "value": ""
//       },
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/grc20dualreg.ALPHA"
//       },
//       {
//         "key": "name",
//         "value": "Alpha"
//       },
//       {
//         "key": "symbol",
//         "value": "ALPHA"
//       },
//       {
//         "key": "decimals",
//         "value": "4"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc20"
//   },
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "id",
//         "value": ""
//       },
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/grc20dualreg.BETA"
//       },
//       {
//         "key": "name",
//         "value": "Beta"
//       },
//       {
//         "key": "symbol",
//         "value": "BETA"
//       },
//       {
//         "key": "decimals",
//         "value": "4"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc20"
//   },
//   {
//     "type": "Transfer",
//     "attrs": [
//       {
//         "key": "id",
//         "value": "g15fn9cy4dutnvkzct356g3j5tf6pdh88f00fvlq"
//       },
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/grc20dualreg.ALPHA"
//       },
//       {
//         "key": "from",
//         "value": ""
//       },
//       {
//         "key": "to",
//         "value": "g18rwjsdf0kcy673ta0q4ujwvsncz4k2a8gcv94t"
//       },
//       {
//         "key": "value",
//         "value": "111"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc20"
//   },
//   {
//     "type": "Transfer",
//     "attrs": [
//       {
//         "key": "id",
//         "value": "g1ssacu4zqmgy90d203z8aw4yehmfurcrfxwvfsj"
//       },
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/grc20dualreg.BETA"
//       },
//       {
//         "key": "from",
//         "value": ""
//       },
//       {
//         "key": "to",
//         "value": "g18rwjsdf0kcy673ta0q4ujwvsncz4k2a8gcv94t"
//       },
//       {
//         "key": "value",
//         "value": "222"
//       }
//     ],
//     "pkg_path": "gno.land/p/demo/tokens/grc20"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_id",
//         "value": "g1a4c52rlfzhx8nmqxgpd4j2vkd7x05p5dvt3ajd"
//       },
//       {
//         "key": "token_key",
//         "value": "gno.land/r/demo/grc20dualreg.alpha"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/grc20dualreg"
//       },
//       {
//         "key": "slug",
//         "value": "alpha"
//       },
//       {
//         "key": "symbol",
//         "value": "ALPHA"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc20reg"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_id",
//         "value": "g1a4c52rlfzhx8nmqxgpd4j2vkd7x05p5dvt3ajd"
//       },
//       {
//         "key": "token_key",
//         "value": "gno.land/r/demo/grc20dualreg.beta"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/grc20dualreg"
//       },
//       {
//         "key": "slug",
//         "value": "beta"
//       },
//       {
//         "key": "symbol",
//         "value": "BETA"
//       }
//     ],
//     "pkg_path": "gno.land/r/demo/defi/grc20reg"
//   }
// ]
