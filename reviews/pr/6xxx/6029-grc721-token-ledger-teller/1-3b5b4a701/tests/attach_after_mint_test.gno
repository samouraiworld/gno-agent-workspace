/* Run: from a gno checkout:
gh pr checkout 6029 -R gnolang/gno && git checkout 3b5b4a701
curl -fsSL -o examples/gno.land/p/demo/tokens/grc721/enumerable/attach_after_mint_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6029-grc721-token-ledger-teller/1-3b5b4a701/tests/attach_after_mint_test.gno
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/p/demo/tokens/grc721/enumerable -run TestAttachAfterMint
rm gno.land/p/demo/tokens/grc721/enumerable/attach_after_mint_test.gno
*/

// RegisterExtension accepts an extension at any point in a token's life, so an
// enumerable attached after the first mint indexes only the later tokens.
// At 3b5b4a701 the core reports two tokens while the enumerable reports one,
// and moving a pre-attach token leaves an owner entry the global index cannot
// reach. Both assertions pass once attaching after the first mint is rejected.
package enumerable

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestAttachAfterMint(cur realm, t *testing.T) {
	owner := testutils.TestAddress("attach-owner")
	next := testutils.TestAddress("attach-next")

	tok, core := newCore("Late", "LATE", 0, cur)
	urequire.NoError(t, core.Mint(owner, "1"))

	enum, _ := NewEnumerable(core)
	urequire.NoError(t, core.Mint(owner, "2"))

	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())

	first, err := enum.TokenByIndex(0)
	urequire.NoError(t, err)
	uassert.Equal(t, "1", first.String())

	urequire.NoError(t, core.TransferFrom(owner, owner, next, "1"))
	uassert.Equal(t, tok.TotalSupply(), enum.TotalSupply())
}
