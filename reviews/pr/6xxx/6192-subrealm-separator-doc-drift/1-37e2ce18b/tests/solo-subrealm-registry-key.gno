// One realm, one symbol, three registry entries: grc20reg.Register keys by the
// registering frame's raw pkgpath, so each realm.Sub identity of the same realm
// gets its own key and the overwrite/alias guard in Register's godoc stops
// holding. Reviewed sha 37e2ce18bbf40a60d6b52456621c5b6c324c8d6e.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno.git && cd gno
//   git checkout 37e2ce18bbf40a60d6b52456621c5b6c324c8d6e
//   cp <this file> examples/gno.land/p/nt/grc20/v0/filetests/solo_subreg_dup_filetest.gno
//   cd examples && gno test -v ./gno.land/p/nt/grc20/v0
//
// grc20reg.gno and gnovm are byte-identical at the merge base
// 5d7d8f5f9b7a3688f00cfa70e594b8f50aed8a42, so the behaviour predates this PR.

// PKGPATH: gno.land/r/demo/grc20subdup

package grc20subdup

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
)

func main(cur realm) {
	sa := cur.Sub("a")
	sb := cur.Sub("b")
	a, _ := grc20.NewToken("SubA", "SREG", 4, 0, sa)
	b, _ := grc20.NewToken("SubB", "SREG", 4, 1, sb)
	c, _ := grc20.NewToken("Host", "SREG", 4, 2, cur)
	println("key a:", grc20reg.Register(cross(sa), a, ""))
	println("key b:", grc20reg.Register(cross(sb), b, ""))
	println("key c:", grc20reg.Register(cross(cur), c, ""))
	println("lookup by (realm, symbol):", grc20reg.Get("gno.land/r/demo/grc20subdup.SREG") != nil)
	println("lookup of the sub entry:", grc20reg.Get("gno.land/r/demo/grc20subdup#a.SREG") != nil)
}

// Output:
// key a: gno.land/r/demo/grc20subdup#a.SREG
// key b: gno.land/r/demo/grc20subdup#b.SREG
// key c: gno.land/r/demo/grc20subdup.SREG
// lookup by (realm, symbol): true
// lookup of the sub entry: true
