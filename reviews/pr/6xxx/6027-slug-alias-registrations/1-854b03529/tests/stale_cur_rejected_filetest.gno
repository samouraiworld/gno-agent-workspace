/* Run: from a gno checkout:
gh pr checkout 6027 -R gnolang/gno && git checkout 854b03529
curl -fsSL -o examples/gno.land/r/demo/defi/grc20reg/filetests/stale_cur_rejected_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6027-slug-alias-registrations/1-854b03529/tests/stale_cur_rejected_filetest.gno
cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/demo/defi/grc20reg/
rm gno.land/r/demo/defi/grc20reg/filetests/stale_cur_rejected_filetest.gno
*/

// PKGPATH: gno.land/r/demo/grc20reg_stale

// A cur captured from a sibling frame that has already returned. cross rejects
// it before Register runs, so Register's own cur.IsCurrent() never sees it.
// The panic below comes from the cross builtin, not from grc20reg.

package grc20reg_stale

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)

func main(cur realm) {
	tok, _ := grc20.NewToken("Stale", "STL", 4, 0, cur)

	var stale realm
	func(cur realm) { stale = cur }(cross(cur))

	grc20reg.Register(cross(stale), tok, "x")
	println("reached Register")
}

// Error:
// cross: rlm is not the current cur (stale capture or sibling frame)
