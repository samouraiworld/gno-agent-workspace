// The mirror case: a token created under cur.Sub("vault") belongs to the host
// realm (origRealm drops the subpath), but Token.ID() keeps "#vault", so the
// host frame cannot register its own token. Reviewed sha
// 37e2ce18bbf40a60d6b52456621c5b6c324c8d6e.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno.git && cd gno
//   git checkout 37e2ce18bbf40a60d6b52456621c5b6c324c8d6e
//   cp <this file> examples/gno.land/p/nt/grc20/v0/filetests/solo_subreg_host_filetest.gno
//   cd examples && gno test -v ./gno.land/p/nt/grc20/v0

// PKGPATH: gno.land/r/demo/grc20subhost

package grc20subhost

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
)

func main(cur realm) {
	tok, _ := grc20.NewToken("SubHost", "SHOS", 4, 0, cur.Sub("vault"))
	println("id:", tok.ID())
	println("key:", grc20reg.Register(cross(cur), tok, ""))
}

// Output:
// id: gno.land/r/demo/grc20subhost#vault.SHOS.0000000

// Error:
// grc20reg: token must be registered from its own realm
