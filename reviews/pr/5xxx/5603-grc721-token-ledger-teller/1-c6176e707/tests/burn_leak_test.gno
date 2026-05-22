// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5603 -R gnolang/gno
# (save this file at examples/gno.land/p/demo/tokens/grc721/burn_leak_test.gno)
(cd examples && go run ../gnovm/cmd/gno test -v -run 'TestBurn_Leak|TestBurn_TokenURI_AfterBurn|TestBurn_TokenURI_Mint_Same_ID|TestBurn_OperatorApproval_Persists' ./gno.land/p/demo/tokens/grc721/)
rm examples/gno.land/p/demo/tokens/grc721/burn_leak_test.gno
*/
package grc721

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// TestBurn_TokenURI_AfterBurn demonstrates that TokenURI returns the
// stale URI of a burned token instead of ErrInvalidTokenId. Burn cleans
// up owners, balances, and tokenApprovals but never touches tokenURIs.
func TestBurn_TokenURI_AfterBurn(t *testing.T) {
	nft, ledger := NewNFT("X", "X")
	alice := testutils.TestAddress("alice")

	uassert.NoError(t, ledger.Mint(alice, TokenID("1")))
	uassert.NoError(t, ledger.SetTokenURI(TokenID("1"), TokenURI("ipfs://x")))

	uassert.NoError(t, ledger.Burn(TokenID("1")))

	// OwnerOf correctly reports the burned token as invalid.
	_, err := nft.OwnerOf(TokenID("1"))
	uassert.ErrorIs(t, err, ErrInvalidTokenId)

	// But TokenURI still returns the burned token's URI.
	uri, err := nft.TokenURI(TokenID("1"))
	uassert.NoError(t, err, "expected TokenURI to error on burned token")
	uassert.Equal(t, "ipfs://x", uri, "TokenURI leaks burned-token URI")
}

// TestBurn_TokenURI_Mint_Same_ID shows the consequence: re-minting the
// same TokenID gets the previous owner's URI without explicit reset.
func TestBurn_TokenURI_Mint_Same_ID(t *testing.T) {
	nft, ledger := NewNFT("X", "X")
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")

	uassert.NoError(t, ledger.Mint(alice, TokenID("1")))
	uassert.NoError(t, ledger.SetTokenURI(TokenID("1"), TokenURI("ipfs://alice")))
	uassert.NoError(t, ledger.Burn(TokenID("1")))

	// Re-mint same tid to bob; do NOT set URI.
	uassert.NoError(t, ledger.Mint(bob, TokenID("1")))

	uri, err := nft.TokenURI(TokenID("1"))
	uassert.NoError(t, err)
	// Bug: bob's freshly minted token inherits alice's URI.
	uassert.Equal(t, "ipfs://alice", uri, "re-minted token inherits stale URI")
}

// TestBurn_OperatorApproval_Persists confirms that owner-level operator
// approvals correctly survive burn (this is desired: they are per-owner,
// not per-token). Included here so the contrast with tokenURIs is clear.
func TestBurn_OperatorApproval_Persists(t *testing.T) {
	nft, ledger := NewNFT("X", "X")
	owner := testutils.TestAddress("owner")
	op := testutils.TestAddress("op")

	uassert.NoError(t, ledger.SetApprovalForAll(owner, op, true))
	uassert.NoError(t, ledger.Mint(owner, TokenID("1")))
	uassert.NoError(t, ledger.Burn(TokenID("1")))

	uassert.True(t, nft.IsApprovedForAll(owner, op), "operator approval correctly persists")
}
