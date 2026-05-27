// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5728 -R gnolang/gno && git checkout c463023cb
curl -fsSL -o examples/gno.land/p/demo/tokens/grc721/royalty/royalty_burn_leak_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5728-grc721-private-ledger-teller/1-c463023cb/tests/royalty_burn_leak_test.gno
(cd examples && go run ../gnovm/cmd/gno test -v -run 'TestHostForget' ./gno.land/p/demo/tokens/grc721/royalty)
rm examples/gno.land/p/demo/tokens/grc721/royalty/royalty_burn_leak_test.gno
*/

// Mechanism: royalty.PrivateLedger.RoyaltyInfo at royalty/token.gno:49-58
// reads led.royalties directly with no OwnerOf gate. As with the metadata
// extension, the burn contract is host-realm-enforced — if the host
// forgets royalty.PrivateLedger.Burn(tid) after grc721.PrivateLedger.Burn(tid),
// a marketplace querying RoyaltyInfo on the burned tid silently routes
// payments to the previous owner's payee at the previous bps.
//
// Result: stale royalty entry leaks across burn AND across re-mint. A
// marketplace honoring EIP-2981 has no way to detect the leak from the
// outside.
//
// To flip the assertion, either (a) add `if _, err := led.token.OwnerOf(tid); err != nil { return "", 0, err }`
// at the top of royalty.PrivateLedger.RoyaltyInfo, or (b) introduce an
// OnBurn hook on grc721.PrivateLedger so extensions clean up automatically.

package royalty

import (
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestHostForget_RoyaltyInfo_LeaksAfterCoreBurn(cur realm, t *testing.T) {
	_, coreLedger, rLedger := newTestStack(0, cur, 1000)
	alice := testutils.TestAddress("alice")
	payee := testutils.TestAddress("payee")
	tid := grc721.TokenID("1")

	uassert.NoError(t, coreLedger.Mint(alice, tid))
	uassert.NoError(t, rLedger.SetTokenRoyalty(tid, RoyaltyInfo{
		PaymentAddress: payee,
		Bps:            500, // 5%
	}))

	// Host calls core Burn but forgets the extension Burn.
	uassert.NoError(t, coreLedger.Burn(tid))

	addr, amount, err := rLedger.RoyaltyInfo(tid, 10000)
	uassert.NoError(t, err, "RoyaltyInfo returns nil error on a burned token")
	uassert.Equal(t, payee.String(), addr.String(),
		"burned-payee=alive owner-of-burned=invalid (royalty leaks across burn)")
	uassert.Equal(t, int64(500), amount)
}

func TestHostForget_RoyaltyInfo_PersistsAcrossRemint(cur realm, t *testing.T) {
	_, coreLedger, rLedger := newTestStack(0, cur, 1000)
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")
	alicePayee := testutils.TestAddress("alicePayee")
	tid := grc721.TokenID("1")

	uassert.NoError(t, coreLedger.Mint(alice, tid))
	uassert.NoError(t, rLedger.SetTokenRoyalty(tid, RoyaltyInfo{
		PaymentAddress: alicePayee,
		Bps:            500,
	}))
	uassert.NoError(t, coreLedger.Burn(tid)) // host forgot rLedger.Burn(tid)

	// Bob mints same tid; never sets royalty.
	uassert.NoError(t, coreLedger.Mint(bob, tid))

	addr, amount, err := rLedger.RoyaltyInfo(tid, 10000)
	uassert.NoError(t, err)
	uassert.Equal(t, alicePayee.String(), addr.String(),
		"re-minted token inherits alice's payee: bob-owns-tid=true alice-still-paid=true")
	uassert.Equal(t, int64(500), amount)
}
