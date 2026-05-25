// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 4931 -R gnolang/gno && git checkout bd39a8c15
curl -fsSL -o examples/gno.land/r/samcrew/normalizedcoins/less_coins_than_bypass_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4931-subscription-package/1-bd39a8c15/tests/less_coins_than_bypass_test.gno
cd examples/gno.land/r/samcrew/normalizedcoins && gno test -v -run 'TestLessCoinsThan_MissingDenomBypass' .
rm less_coins_than_bypass_test.gno
*/

// Mechanism: LessCoinsThan iterates only `a`. If the first coin in `a` has a
// denom that is NOT in `b` and amount > 0, the function returns `false` (early
// exit), regardless of whether `b` contains required denoms missing from `a`.
// Net effect: any consumer using LessCoinsThan(have, want) as a "have >= want"
// check is bypassed whenever `have` contains a single denom that is irrelevant
// to `want`. Subscribe and dueAmount both rely on this invariant.

package normalizedcoins

import (
	"chain"
	"testing"
)

func TestLessCoinsThan_MissingDenomBypass(t *testing.T) {
	// Have only native; want only GRC20. Have clearly cannot cover want.
	have := chain.NewCoins(chain.NewCoin("/native/ugnot", 5000))
	want := chain.NewCoins(chain.NewCoin("/grc20/r/demo/foo20", 1000))

	got := LessCoinsThan(have, want)

	// IS:     LessCoinsThan returns false — caller thinks `have >= want`.
	// SHOULD: LessCoinsThan returns true — `have` is missing /grc20/r/demo/foo20.
	if got != false {
		t.Fatalf("expected false (current buggy behavior), got %v", got)
	}
	// To assert the fix, flip the line above to: if got != true { ... }
}

func TestLessCoinsThan_EmptyHaveBypass(t *testing.T) {
	have := chain.Coins{}
	want := chain.NewCoins(chain.NewCoin("/native/ugnot", 1000))

	got := LessCoinsThan(have, want)

	// IS:     LessCoinsThan returns false on empty `have`.
	// SHOULD: LessCoinsThan returns true — empty cannot cover anything.
	if got != false {
		t.Fatalf("expected false (current buggy behavior), got %v", got)
	}
}
