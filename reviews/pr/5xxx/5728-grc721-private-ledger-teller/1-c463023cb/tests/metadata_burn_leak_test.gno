// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5728 -R gnolang/gno && git checkout c463023cb
curl -fsSL -o examples/gno.land/p/demo/tokens/grc721/metadata/metadata_burn_leak_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5728-grc721-private-ledger-teller/1-c463023cb/tests/metadata_burn_leak_test.gno
(cd examples && go run ../gnovm/cmd/gno test -v -run 'TestHostForget' ./gno.land/p/demo/tokens/grc721/metadata)
rm examples/gno.land/p/demo/tokens/grc721/metadata/metadata_burn_leak_test.gno
*/

// Mechanism: the extension Burn contract is host-realm-enforced — core
// PrivateLedger.Burn never notifies extensions. metadata.TokenURI() reads
// led.tokenURIs directly with no OwnerOf gate (compare TokenMetadata at
// metadata/token.gno:50-61 which does gate). If a host realm forgets to
// call mLedger.Burn(tid) right after coreLedger.Burn(tid), TokenURI keeps
// returning the burned token's URI silently.
//
// Result: stale tokenURI leaks across burn; if the same tid is re-minted
// to a different owner the new token inherits the previous URI until
// SetTokenURI is called again. TokenMetadata is defensive and returns
// ErrInvalidTokenId in the same path — the inconsistency surfaces here.
//
// To flip the assertion, either (a) add `if _, err := led.token.OwnerOf(tid); err != nil { return "", err }`
// at the top of metadata.PrivateLedger.TokenURI, or (b) introduce an
// OnBurn hook on grc721.PrivateLedger so extensions clean up automatically.

package metadata

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestHostForget_TokenURI_LeaksAfterCoreBurn(cur realm, t *testing.T) {
	_, coreLedger, mLedger := newTestStack(0, cur)
	alice := testutils.TestAddress("alice")
	tid := grc721.TokenID("1")

	uassert.NoError(t, coreLedger.Mint(alice, tid))
	uassert.NoError(t, mLedger.SetTokenURI(tid, grc721.TokenURI("ipfs://leak")))

	// Host realm calls core Burn but forgets the extension Burn.
	uassert.NoError(t, coreLedger.Burn(tid))

	// TokenMetadata gates on OwnerOf and correctly reports the burn.
	_, err := mLedger.TokenMetadata(tid)
	uassert.ErrorIs(t, err, grc721.ErrInvalidTokenId)

	// TokenURI does NOT gate on OwnerOf and returns the stale URI.
	uri, err := mLedger.TokenURI(tid)
	uassert.NoError(t, err, "TokenURI returns nil error on a burned token")
	uassert.Equal(t, "ipfs://leak", uri, "TokenURI leaks burned-token URI")
}

func TestHostForget_TokenURI_PersistsAcrossRemint(cur realm, t *testing.T) {
	_, coreLedger, mLedger := newTestStack(0, cur)
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	tid := grc721.TokenID("1")

	uassert.NoError(t, coreLedger.Mint(alice, tid))
	uassert.NoError(t, mLedger.SetTokenURI(tid, grc721.TokenURI("ipfs://alice-art")))
	uassert.NoError(t, coreLedger.Burn(tid)) // host forgot mLedger.Burn(tid)

	// Re-mint the same tid to bob without setting a new URI.
	uassert.NoError(t, coreLedger.Mint(bob, tid))

	uri, err := mLedger.TokenURI(tid)
	uassert.NoError(t, err)
	uassert.Equal(t, "ipfs://alice-art", uri,
		"re-minted token inherits previous owner's URI: alice-tx-1=ipfs://alice-art bob-tx-2=ipfs://alice-art")
}
