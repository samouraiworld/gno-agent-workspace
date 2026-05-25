// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 4931 -R gnolang/gno && git checkout bd39a8c15
curl -fsSL -o examples/gno.land/r/samcrew/subscriptions/inactive_double_claim_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4931-subscription-package/1-bd39a8c15/tests/inactive_double_claim_test.gno
cd examples/gno.land/r/samcrew/subscriptions && gno test -v -run 'TestInactiveSubscriber_VaultStateDivergence' .
rm inactive_double_claim_test.gno
*/

// Mechanism: subscription.dueAmount returns (reducedDue, false) early when the
// vault cannot cover the next full period. The early-return path skips the
// `if charge` block, so vault and lastChargedAt are never updated even when
// the caller passed charge=true. ServiceClaimVault uses that reduced due to
// SendCoins from the realm address, draining real funds without decrementing
// the subscriber's vault. Subscriber's vault becomes a lie: it still claims
// to hold what was already paid to the owner. Subsequent Unsubscribe / Withdraw
// either panic in the banker (realm has no coins to back the vault) or let the
// owner double-collect until realm runs dry.

package subscriptions

import (
	"chain"
	"chain/banker"
	"testing"
	"time"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestInactiveSubscriber_VaultStateDivergence(t *testing.T) {
	defer resetState()

	owner := testutils.TestAddress("owner")
	subscriber := testutils.TestAddress("subscriber")
	subscriptionsAddr := chain.PackageAddress("gno.land/r/samcrew/subscriptions")

	testing.SetOriginCaller(owner)
	testing.SetRealm(testing.NewUserRealm(owner))
	// 1 ugnot/hour service.
	NewServiceSimple(cross, "Hourly", "desc", 1*time.Hour, "ugnot", 1000, "", 0)
	svcKey := serviceKey(owner, "Hourly")

	// Subscriber deposits 2500 — enough for 2 full periods plus initial, but
	// not 3. After enough time elapses they become inactive at period 3.
	testing.SetOriginCaller(subscriber)
	testing.SetRealm(testing.NewUserRealm(subscriber))
	deposit := chain.NewCoins(chain.NewCoin("ugnot", 2500))
	testing.IssueCoins(subscriptionsAddr, deposit)
	testing.SetOriginSend(deposit)
	Subscribe(cross, svcKey, "", 0)

	svc := getService(svcKey)
	subRaw, _ := svc.subscribers.Get(subscriber.String())
	sub := subRaw.(*subscription)
	uassert.Equal(t, int64(2500), sub.vault.AmountOf("/native/ugnot"))

	// Skip ~3 hours of blocks (>= 3 renewal periods worth of wall-clock time).
	testing.SkipHeights(720 * 4)

	// Owner claims. Subscriber is inactive (needs 3000 to cover 3 periods,
	// only has 2500). dueAmount(true) returns 2000 but does NOT update state.
	testing.SetOriginCaller(owner)
	testing.SetRealm(testing.NewUserRealm(owner))
	ServiceClaimVault(cross, svcKey)

	// Vault still says 2500 even though realm just paid 2000 to owner.
	uassert.Equal(t, int64(2500), sub.vault.AmountOf("/native/ugnot"),
		"BUG: subscriber vault not decremented after owner claimed inactive-subscriber debt")

	// Realm actually has 500 left (2500 deposited, 2000 sent to owner).
	b := banker.NewBanker(banker.BankerTypeReadonly)
	realmBalance := b.GetCoins(subscriptionsAddr).AmountOf("ugnot")
	uassert.Equal(t, int64(500), realmBalance,
		"realm balance reflects the actual transfer, but vault accounting does not")

	// Confirm dueAmount(true) still reports the same 2000 — proof that the
	// charge=true path silently skipped state update. A second owner claim
	// would attempt to drain the same 2000 again (bounded only by realm
	// having coins to send), and any Unsubscribe call by the subscriber will
	// trigger a banker panic ("cannot send 2000ugnot", realm only has 500),
	// leaving the user permanently unable to exit.
	due2, active := sub.dueAmount(true, svc.renewalPeriod, svc.price)
	uassert.False(t, active, "sub still inactive")
	uassert.Equal(t, int64(2000), due2.AmountOf("/native/ugnot"),
		"BUG: dueAmount(charge=true) returns the same uncharged debt — state never advances")
	uassert.Equal(t, int64(2500), sub.vault.AmountOf("/native/ugnot"),
		"BUG: charge=true did not deduct vault — invariant decay")
}
