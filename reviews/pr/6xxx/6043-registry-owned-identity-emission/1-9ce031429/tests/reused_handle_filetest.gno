/* Run: from a gno checkout:
gh pr checkout 6043 -R gnolang/gno && git checkout 9ce031429
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/filetests/reused_handle_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6043-registry-owned-identity-emission/1-9ce031429/tests/reused_handle_filetest.gno
go run ./gnovm/cmd/gno test -C examples ./gno.land/r/demo/defi/grc20reg
rm examples/gno.land/r/demo/defi/grc20reg/filetests/reused_handle_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20reuse

// NewTokenWithEmitter checks the namespace of the identifier it gets back, never
// that the handle is fresh. At 9ce031429 both tokens below come out holding
// gno.land/r/demo/grc20reuse.DUP.1, and both mint under grc20reg's pkg_path.
// The golden asserts the second token gets its own identifier.
package grc20reuse

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

type relay struct{ handle grc20.TokenEmitter }

func (r *relay) IssueToken(cur realm, symbol string) grc20.TokenEmitter { return r.handle }

func main(cur realm) {
	r := &relay{handle: grc20reg.Emitter().IssueToken(cross(cur), "DUP")}

	first, _ := grc20.NewTokenWithEmitter("A", "DUP", 6, r, cur)
	second, _ := grc20.NewTokenWithEmitter("B", "DUP", 6, r, cur)

	println("first :", first.ID())
	println("second:", second.ID())
}

// Output:
// first : gno.land/r/demo/grc20reuse.DUP.1
// second: gno.land/r/demo/grc20reuse.DUP.2
