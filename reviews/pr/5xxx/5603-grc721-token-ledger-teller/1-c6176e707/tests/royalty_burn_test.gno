/* Run:
  WORKTREE=/home/davd/Projects/gno-skills/.worktrees/gno-review-5603
  PKG=$WORKTREE/examples/gno.land/p/demo/tokens/grc721/royalty
  cp /home/davd/Projects/gno-skills/reviews/pr/5xxx/5603-grc721-token-ledger-teller/1-c6176e707/tests/royalty_burn_test.gno "$PKG/royalty_burn_test.gno"
  (cd "$WORKTREE/examples" && go run ../gnovm/cmd/gno test -v -run 'TestRoyaltyInfo_AfterBurn|TestRoyaltyInfo_Remint_Inherits' ./gno.land/p/demo/tokens/grc721/royalty/)
  rm "$PKG/royalty_burn_test.gno"
*/
package royalty

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// TestRoyaltyInfo_AfterBurn: after burning a token, RoyaltyInfo still
// returns the stored royalty without error. RoyaltyInfo looks up
// l.royalties directly with no OwnerOf guard, mirroring the core
// TokenURI leak. The metadata.TokenMetadata path correctly guards on
// OwnerOf — only royalty and the core TokenURI lack the gate.
func TestRoyaltyInfo_AfterBurn(t *testing.T) {
	nft, ledger, rLedger := NewRoyaltyNFT("R", "R", 1000)
	alice := testutils.TestAddress("alice")
	payee := testutils.TestAddress("payee")

	uassert.NoError(t, ledger.Mint(alice, grc721.TokenID("1")))
	uassert.NoError(t, rLedger.SetTokenRoyalty(grc721.TokenID("1"), RoyaltyInfo{
		PaymentAddress: payee,
		Bps:            500,
	}))

	uassert.NoError(t, ledger.Burn(grc721.TokenID("1")))

	_, err := nft.OwnerOf(grc721.TokenID("1"))
	uassert.ErrorIs(t, err, grc721.ErrInvalidTokenId)

	// Bug: royalty info for the burned token is still returned.
	addr, amount, err := nft.RoyaltyInfo(grc721.TokenID("1"), 10000)
	uassert.NoError(t, err)
	uassert.Equal(t, payee.String(), addr.String())
	uassert.Equal(t, int64(500), amount)
}

// TestRoyaltyInfo_Remint_Inherits: re-minting the same TokenID returns
// the previous owner's royalty configuration without explicit reset.
func TestRoyaltyInfo_Remint_Inherits(t *testing.T) {
	nft, ledger, rLedger := NewRoyaltyNFT("R", "R", 1000)
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	payeeAlice := testutils.TestAddress("payee_alice")

	uassert.NoError(t, ledger.Mint(alice, grc721.TokenID("1")))
	uassert.NoError(t, rLedger.SetTokenRoyalty(grc721.TokenID("1"), RoyaltyInfo{
		PaymentAddress: payeeAlice,
		Bps:            500,
	}))
	uassert.NoError(t, ledger.Burn(grc721.TokenID("1")))

	// Re-mint same tid to bob; do not set royalty.
	uassert.NoError(t, ledger.Mint(bob, grc721.TokenID("1")))

	// Bob's freshly minted token still pays Alice's payee.
	addr, amount, err := nft.RoyaltyInfo(grc721.TokenID("1"), 10000)
	uassert.NoError(t, err)
	uassert.Equal(t, payeeAlice.String(), addr.String())
	uassert.Equal(t, int64(500), amount)
}
