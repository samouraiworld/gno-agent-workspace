/* Run: from a gno checkout:
gh pr checkout 6027 -R gnolang/gno && git checkout 854b03529
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/filetests/duplicate_token_id_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6027-slug-alias-registrations/1-854b03529/tests/duplicate_token_id_filetest.gno
cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/demo/defi/grc20reg/
rm gno.land/r/demo/defi/grc20reg/filetests/duplicate_token_id_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20reg_dupid

// Two tokens built from the same seqid in one realm carry a byte-identical
// Token.ID(). The assertion below is the merge-base behaviour: the second
// registration is rejected. At 854b03529 both calls succeed and the run
// reaches the final println instead.

package grc20reg_dupid

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

func main(cur realm) {
	first, _ := grc20.NewToken("First", "DUP", 6, 0, cur)
	second, _ := grc20.NewToken("Second", "DUP", 6, 0, cur)

	if first.ID() != second.ID() {
		panic("precondition: both tokens must share one Token.ID()")
	}

	a := grc20reg.Register(cross(cur), first, "a")
	b := grc20reg.Register(cross(cur), second, "b")

	println("both registered under one token id:",
		grc20reg.MustGet(a).ID() == grc20reg.MustGet(b).ID(),
		grc20reg.MustGet(a).GetName(), grc20reg.MustGet(b).GetName())
}

// Error:
// grc20reg: token already registered
