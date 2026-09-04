// Asserts that PrivateLedger.UserTeller and Token.UserTellerTrusted build the
// same fnTeller, so the trusted-host widening the UserTellerTrusted doc reserves
// for itself also admits a UserTeller value. Passes at e014175d2.
//
/* Run: from a gno checkout:
gh pr checkout 6123 -R gnolang/gno && git checkout e014175d2
curl -fsSL -o examples/gno.land/p/demo/tokens/grc20/user_teller_identity_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6123-grc20-eoa-writes/1-e014175d2/tests/user_teller_identity_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run TestUserTellerAndUserTellerTrustedAreTheSameValue ./gno.land/p/demo/tokens/grc20
rm gno.land/p/demo/tokens/grc20/user_teller_identity_test.gno
*/

package grc20

import (
	"chain"
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestUserTellerAndUserTellerTrustedAreTheSameValue(cur realm, t *testing.T) {
	const relay = "gno.land/r/demo/defi/grc20reg"

	alice := testutils.TestAddress("alice")
	tok, ledger := newTestToken("Ident", "IDNT", 6, 0, cur)
	ledger.TrustHost(relay)

	// An EOA previous, so only the host half of guardWrite decides.
	eoa := testing.MakeRealm(alice, "", testing.OriginRealm())
	relayFrame := testing.MakeRealm(chain.PackageAddress(relay), relay, eoa)

	priv := ledger.UserTeller().(*fnTeller)
	pub := tok.UserTellerTrusted().(*fnTeller)

	uassert.Equal(t, pub.homeGuard, priv.homeGuard)
	uassert.Equal(t, pub.userOnly, priv.userOnly)
	uassert.True(t, pub.Token == priv.Token)

	// The teller whose doc claims only home confinement is admitted at the relay.
	uassert.NoError(t, priv.guardWrite(0, relayFrame)) // IS:     UserTeller is widened by TrustHost too
	// uassert.ErrorIs(t, priv.guardWrite(0, relayFrame), ErrForeignCallerTeller) // SHOULD: home-guarded means the token's own realm
	uassert.NoError(t, pub.guardWrite(0, relayFrame))
}
