// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 4931 -R gnolang/gno && git checkout bd39a8c15
curl -fsSL -o examples/gno.land/r/samcrew/subscriptions/subscribe_wrong_denom_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4931-subscription-package/1-bd39a8c15/tests/subscribe_wrong_denom_test.gno
cd examples/gno.land/r/samcrew/subscriptions && gno test -v -run 'TestSubscribe_PayWrongDenomBypassesPriceGate' .
rm subscribe_wrong_denom_test.gno
*/

// Mechanism: Subscribe gates the initial deposit with LessCoinsThan(sCoins,
// svc.price). LessCoinsThan only iterates the deposit, so depositing a single
// irrelevant denom (e.g., native ugnot) trivially passes the gate even when
// the service is priced in GRC20 tokens the subscriber never pays.
// Result: subscriber is enrolled in a GRC20-priced service without ever
// transferring any GRC20 token, and dueAmount keeps reporting "active" because
// LessCoinsThan(vault, due) hits the same bypass.

package subscriptions

import (
	"chain"
	"testing"
	"time"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestSubscribe_PayWrongDenomBypassesPriceGate(t *testing.T) {
	defer resetState()

	owner := testutils.TestAddress("owner")
	subscriber := testutils.TestAddress("subscriber")
	subscriptionsAddr := chain.PackageAddress("gno.land/r/samcrew/subscriptions")
	svcKey := serviceKey(owner, "GRC20 Only Service")

	testing.SetOriginCaller(owner)
	testing.SetRealm(testing.NewUserRealm(owner))
	// Service is priced ONLY in a fictional GRC20 token. No native price.
	NewServiceSimple(cross, "GRC20 Only Service", "desc", 24*time.Hour,
		"", 0,
		"gno.land/r/demo/foo20", 1000)

	// Subscriber sends only native coins, no GRC20 (fqName empty, amount 0).
	testing.SetOriginCaller(subscriber)
	testing.SetRealm(testing.NewUserRealm(subscriber))
	deposit := chain.NewCoins(chain.NewCoin("ugnot", 5000))
	testing.IssueCoins(subscriptionsAddr, deposit)
	testing.SetOriginSend(deposit)

	// IS: Subscribe accepts native-only payment for a GRC20-priced service.
	Subscribe(cross, svcKey, "", 0)

	svc := getService(svcKey)
	uassert.True(t, svc.subscribers.Has(subscriber.String()),
		"BUG: subscriber enrolled in GRC20-priced service after paying only native ugnot")

	// And IsSubscribed reports "active" because LessCoinsThan(vault, due)
	// returns false when vault holds an irrelevant denom.
	uassert.True(t, IsSubscribed(cross, svcKey, subscriber.String()),
		"BUG: subscriber reported active without ever paying the GRC20 price")
}
