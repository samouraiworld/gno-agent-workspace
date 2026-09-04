// Asserts that PrivateLedger.UntrustHost reports nothing when it removes
// nothing, so a token realm cannot tell a completed revocation from a failed
// one. Measured at e014175d2: the mistyped revoke leaves the relay trusted and
// UserTellerTrusted keeps serving it.
//
/* Run: from a gno checkout:
gh pr checkout 6123 -R gnolang/gno && git checkout e014175d2
curl -fsSL -o examples/gno.land/p/demo/tokens/grc20/untrust_host_typo_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6123-grc20-eoa-writes/1-e014175d2/tests/untrust_host_typo_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run TestUntrustHostTypoIsSilent ./gno.land/p/demo/tokens/grc20
rm gno.land/p/demo/tokens/grc20/untrust_host_typo_test.gno
*/

package grc20

import (
	"chain"
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

// TestUntrustHostTypoIsSilent pairs a revoke that works with a revoke that does
// not. Both calls return nothing, so only the guard tells them apart, and the
// guard is not what a token operator reads.
func TestUntrustHostTypoIsSilent(cur realm, t *testing.T) {
	const relay = "gno.land/r/demo/defi/grc20reg"

	alice := testutils.TestAddress("alice")
	tok, ledger := newTestToken("Typo", "TYPO", 6, 0, cur)

	// An EOA previous, so only the host half of guardWrite decides.
	eoa := testing.MakeRealm(alice, "", testing.OriginRealm())
	relayFrame := testing.MakeRealm(chain.PackageAddress(relay), relay, eoa)

	teller := tok.UserTellerTrusted().(*fnTeller)
	ledger.TrustHost(relay)
	uassert.NoError(t, teller.guardWrite(0, relayFrame))

	// A trailing slash is not the stored key. UntrustHost returns nothing, so
	// the operator sees the same result as a revoke that landed.
	ledger.UntrustHost(relay + "/")
	uassert.NoError(t, teller.guardWrite(0, relayFrame)) // IS:     the relay is still trusted after an unreported failed revoke
	// uassert.ErrorIs(t, teller.guardWrite(0, relayFrame), ErrForeignCallerTeller) // SHOULD: UntrustHost returns removed=false and the operator retries

	// The exact key does remove, and nothing in the call distinguishes it from
	// the attempt above.
	ledger.UntrustHost(relay)
	uassert.ErrorIs(t, teller.guardWrite(0, relayFrame), ErrForeignCallerTeller)
}
