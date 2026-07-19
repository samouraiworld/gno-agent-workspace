/* Run: from a gno checkout:
gh pr checkout 5946 -R gnolang/gno && git checkout 037a90410

# the package does not build at this commit; apply the avl.Get arity fix first
sed -i 's/raw, exists := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/if !exists {/if raw == nil {/' \
  examples/gno.land/r/demo/predictionmarket/predictionmarket.gno
sed -i 's/raw, exists := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/raw, _ := markets.Get(\(.*\))/raw := markets.Get(\1)/; s/if !exists {/if raw == nil {/' \
  examples/gno.land/r/demo/predictionmarket/predictionmarket_test.gno

curl -fsSL -o examples/gno.land/r/demo/predictionmarket/payout_overflow_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5946-prediction-market-demo-realm/1-037a90410/tests/payout_overflow_test.gno

cd examples/gno.land/r/demo/predictionmarket && go run ../../../../../gnovm/cmd/gno test -v .
cd - && git checkout examples/gno.land/r/demo/predictionmarket/
rm examples/gno.land/r/demo/predictionmarket/payout_overflow_test.gno
*/

// ClaimWinnings computes payout as b.amount * totalPool / winningPool, so the
// product wraps int64 once the two factors reach the low billions of ugnot.
// At 5000 GNOT per side the claim panics inside bankerSendCoins with
// "non-positive coin amount: -1068046444" and the escrow can never leave the
// realm. A fix that divides before multiplying, or widens the intermediate,
// makes both assertions below pass.
package predictionmarket

import (
	"chain"
	"chain/banker"
	"strconv"
	"testing"
)

const realmPath = "gno.land/r/demo/predictionmarket"

// seed builds a resolved market directly, bypassing PlaceBet's origin-call
// guard, which no in-package test can satisfy.
func seedResolved(cur realm, bettor address, stake int64) int {
	id := CreateMarket(cur, "overflow probe")
	m := markets.Get(strconv.Itoa(id)).(*Market)
	m.pools[OutcomeHome] = stake
	m.pools[OutcomeAway] = stake
	m.bets = append(m.bets, &Bet{bettor: bettor, outcome: OutcomeHome, amount: stake})
	m.resolved = true
	m.winner = OutcomeHome
	return id
}

func TestClaimPayoutDoesNotWrap(cur realm, t *testing.T) {
	rlm := chain.PackageAddress(realmPath)
	testing.IssueCoins(rlm, chain.Coins{{"ugnot", 1_000_000_000_000}})

	winner := cur.Previous().Address()
	before := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot")

	// 5000 GNOT on each side: the sole Home bettor takes the whole 10000 GNOT.
	const stake = int64(5_000_000_000)
	id := seedResolved(cur, winner, stake)
	ClaimWinnings(cur, id)

	got := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot") - before
	if got != 2*stake {
		t.Fatalf("payout: got %d, want %d", got, 2*stake)
	}
}

func TestClaimPayoutSmallStakeUnchanged(cur realm, t *testing.T) {
	rlm := chain.PackageAddress(realmPath)
	testing.IssueCoins(rlm, chain.Coins{{"ugnot", 1_000_000_000_000}})

	winner := cur.Previous().Address()
	before := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot")

	const stake = int64(1_000_000)
	id := seedResolved(cur, winner, stake)
	ClaimWinnings(cur, id)

	got := banker.NewReadonlyBanker().GetCoins(winner).AmountOf("ugnot") - before
	if got != 2*stake {
		t.Fatalf("payout: got %d, want %d", got, 2*stake)
	}
}
