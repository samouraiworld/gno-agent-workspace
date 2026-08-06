/* Run: from a gno checkout:
gh pr checkout 6043 -R gnolang/gno && git checkout 9ce031429
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/filetests/unbound_symbol_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6043-registry-owned-identity-emission/1-9ce031429/tests/unbound_symbol_filetest.gno
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/unbound_symbol_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20sym

// NewToken builds the identifier as pkgPath.symbol.seq, so it always names the
// token's own symbol. NewTokenWithEmitter checks the realm prefix and nothing
// else, so at 9ce031429 the token below reports symbol BBB and emits under the
// identifier gno.land/r/demo/grc20sym.AAA.1.
package grc20sym

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

type relay struct{ handle grc20.TokenEmitter }

func (r *relay) IssueToken(cur realm, symbol string) grc20.TokenEmitter { return r.handle }

func main(cur realm) {
	r := &relay{handle: grc20reg.Emitter().IssueToken(cross(cur), "AAA")}

	token, _ := grc20.NewTokenWithEmitter("Bee", "BBB", 6, r, cur)

	println("symbol:", token.GetSymbol())
	println("id    :", token.ID())
}

// Output:
// symbol: BBB
// id    : gno.land/r/demo/grc20sym.BBB.1
