/* Run: from a gno checkout:
gh pr checkout 5975 -R gnolang/gno && git checkout e3b7a7934
curl -fsSL -o examples/gno.land/p/demo/parimutuel/rake_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5975-parimutuel-payout-math/1-e3b7a7934/tests/rake_test.gno
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/p/demo/parimutuel
cd .. && rm examples/gno.land/p/demo/parimutuel/rake_test.gno
*/

// CalculatePayoutWithRake passes the post-rake netPool into CalculatePayout,
// which then compares the gross winningPool against it. At e3b7a7934 any
// outcome holding more than 10000-rakeBps of the pool panics with
// "winningPool cannot exceed totalPool"; the fix returns the reduced payout.
package parimutuel

import "testing"

func TestRakeWithEveryoneOnTheWinner(t *testing.T) {
	// 1% rake, all 1000 staked on the outcome that wins.
	if got := CalculatePayoutWithRake(1000, 1000, 1000, 100); got != 990 {
		t.Fatalf("got %d, want 990", got)
	}
}

func TestRakeWithDominantWinningPool(t *testing.T) {
	// 5% rake leaves 950; a winning pool of 960 is still a valid market.
	if got := CalculatePayoutWithRake(960, 1000, 960, 500); got != 950 {
		t.Fatalf("got %d, want 950", got)
	}
}
