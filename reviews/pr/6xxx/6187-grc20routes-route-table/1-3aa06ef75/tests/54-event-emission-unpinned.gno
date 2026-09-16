// PKGPATH: gno.land/r/demo/routeevents

// Pins the two events grc20routes emits, "register" and "proven", which nothing
// in the PR asserts: deleting both chain.Emit calls leaves the package suite ok
// in 3.63s, and grc20routes_keyed_proof.txtar carries no EVENTS line to redden.
// This filetest passes at
// 3aa06ef757b87d731a3e554e96c7540a209fd9a3 and reddens on that deletion.
//
// From a plain clone of github.com/gnolang/gno:
//
//   git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//   go build -o /tmp/gno ./gnovm/cmd/gno
//   cp <this file> examples/gno.land/r/nt/grc20routes/v0/route_events_filetest.gno
//   (cd examples && /tmp/gno test ./gno.land/r/nt/grc20routes/v0)     # ok
//   # now delete both chain.Emit calls in grc20routes.gno (lines 313-324 and 418)
//   (cd examples && /tmp/gno test ./gno.land/r/nt/grc20routes/v0)     # FAIL, events mismatch
//   git checkout examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
//   rm examples/gno.land/r/nt/grc20routes/v0/route_events_filetest.gno

package routeevents

import (
	"gno.land/p/nt/grc20/v0"

	"gno.land/r/nt/grc20reg/v0"
	routes "gno.land/r/nt/grc20routes/v0"
)

func main(cur realm) {
	// Two tokens on one realm: the declared path needs one registered in
	// grc20reg, the keyed proof needs two at the same realm.
	tokA, ledA := grc20.NewToken("Alpha", "ALFA", 6, 0, cur)
	tokB, ledB := grc20.NewToken("Beta", "BETA", 6, 1, cur)
	keyA := grc20reg.Register(cross(cur), tokA, "")
	keyB := grc20reg.Register(cross(cur), tokB, "")

	// DECLARED path, emits "register". "deposit" is there so ops is not the
	// canonical three, which a byte-for-byte golden would otherwise hide.
	routes.Register(cross(cur), "ALFA", routes.Canonical().With("deposit", "Deposit"))

	// PROVEN path, four calls, emits "proven" on the last. prover is this
	// realm's address: BeginKeyedProof reads cur.Previous().Address().
	prover := cur.Address()
	routes.BeginKeyedProof(cross(cur), keyA, keyB, 111, 222)
	if err := ledA.Approve(prover, routes.ProbeAddress(), 111); err != nil {
		panic(err)
	}
	if err := ledB.Approve(prover, routes.ProbeAddress(), 222); err != nil {
		panic(err)
	}
	routes.FinishKeyedProof(cross(cur), keyA, keyB)
	println("done")
}

// Output:
// done

// Events:
// [
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.ALFA.0000000"
//       },
//       {
//         "key": "name",
//         "value": "Alpha"
//       },
//       {
//         "key": "symbol",
//         "value": "ALFA"
//       },
//       {
//         "key": "decimals",
//         "value": "6"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "NewToken",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.BETA.0000001"
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
//         "value": "6"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/routeevents.ALFA"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "ALFA"
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20reg/v0"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_path",
//         "value": "gno.land/r/demo/routeevents.BETA"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "slug",
//         "value": ""
//       },
//       {
//         "key": "symbol",
//         "value": "BETA"
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20reg/v0"
//   },
//   {
//     "type": "register",
//     "attrs": [
//       {
//         "key": "token_key",
//         "value": "gno.land/r/demo/routeevents.ALFA"
//       },
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "ops",
//         "value": "approve,deposit,transfer,transfer_from"
//       },
//       {
//         "key": "prefix",
//         "value": ""
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20routes/v0"
//   },
//   {
//     "type": "Approval",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.ALFA.0000000"
//       },
//       {
//         "key": "owner",
//         "value": "g13s9pqxx2nza4v8ngw46veuss0sq6xdtrzrtx7f"
//       },
//       {
//         "key": "spender",
//         "value": "g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep"
//       },
//       {
//         "key": "value",
//         "value": "111"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "Approval",
//     "attrs": [
//       {
//         "key": "token",
//         "value": "gno.land/r/demo/routeevents.BETA.0000001"
//       },
//       {
//         "key": "owner",
//         "value": "g13s9pqxx2nza4v8ngw46veuss0sq6xdtrzrtx7f"
//       },
//       {
//         "key": "spender",
//         "value": "g1v58us0p32wfaxfnhzqwd72hpu56fg4ktfkhyep"
//       },
//       {
//         "key": "value",
//         "value": "222"
//       }
//     ],
//     "pkg_path": "gno.land/p/nt/grc20/v0"
//   },
//   {
//     "type": "proven",
//     "attrs": [
//       {
//         "key": "pkgpath",
//         "value": "gno.land/r/demo/routeevents"
//       },
//       {
//         "key": "prover",
//         "value": "g13s9pqxx2nza4v8ngw46veuss0sq6xdtrzrtx7f"
//       }
//     ],
//     "pkg_path": "gno.land/r/nt/grc20routes/v0"
//   }
// ]
