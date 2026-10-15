// PKGPATH: gno.land/r/demo/regsub

// b3-reach: the Token.ID prefix check accepts a sibling sub-identity token, and
// the index info link for any sub-identity key carries a raw "#".
// Repro from a plain clone of gnolang/gno at 74ec1b3e1ad9406d645443fffbca8ae277b8edf7:
//   cp b3-reach-subrealm-key.gno examples/gno.land/r/demo/defi/grc721reg/z_reachsub_filetest.gno
//   cd examples && gno test -v ./gno.land/r/demo/defi/grc721reg
// PASSES against the golden Output: sub "a" registers the collection whose Token.ID
// is regsub#a.x.x.0000000 under key regsub#a.x, and the info link path, cut at "#"
// as a browser does, renders "Unknown collection".
package regsub

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
)

func main(cur realm) {
	// Sub identity "a.x" creates its own collection "x".
	ax := cur.Sub("a.x")
	tok, _ := grc721.NewToken("Owned by a.x", "x", 0, ax)

	// Sibling sub identity "a" registers it.
	a := cur.Sub("a")
	key := grc721reg.Register(cross(a), tok, "")
	println("token id:", tok.ID())
	println("key:     ", key)

	// A sub identity registering its own collection.
	dao := cur.Sub("dao")
	own, _ := grc721.NewToken("DAO", "SUB", 0, dao)
	daoKey := grc721reg.Register(cross(dao), own, "")
	println("dao key:", daoKey)
	println(grc721reg.Render(""))
	// What the info link's path reaches once a browser strips the "#..." fragment.
	println(grc721reg.Render("gno.land/r/demo/regsub"))
}

// Output:
// token id: gno.land/r/demo/regsub#a.x.x.0000000
// key:      gno.land/r/demo/regsub#a.x
// dao key: gno.land/r/demo/regsub#dao.SUB
// - **Owned by a\.x** - [gno.land/r/demo/regsub#a](/r/demo/regsub#a).x - _core only_ - [info](/r/demo/defi/grc721reg:gno.land/r/demo/regsub#a.x)
// - **DAO** - [gno.land/r/demo/regsub#dao](/r/demo/regsub#dao).SUB - _core only_ - [info](/r/demo/defi/grc721reg:gno.land/r/demo/regsub#dao.SUB)
//
// Unknown collection: gno\.land/r/demo/regsub
