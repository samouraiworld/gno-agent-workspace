/* Run: from a gno checkout:
gh pr checkout 5975 -R gnolang/gno && git checkout e3b7a7934
curl -fsSL -o examples/gno.land/p/demo/parimutuel/overflow_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5975-parimutuel-payout-math/1-e3b7a7934/tests/overflow_test.gno
cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/p/demo/parimutuel
cd .. && rm examples/gno.land/p/demo/parimutuel/overflow_test.gno
*/

// mulDiv evaluates a*b before dividing, so the intermediate product wraps
// once userBet*totalPool passes 2^63-1. At e3b7a7934 a 4000 GNOT winner-takes-all
// pool returns -611686018 instead of 4000000000; the fix makes it either
// return the true value or panic.
package parimutuel

import "testing"

func TestPayoutDoesNotWrapAtLargePools(t *testing.T) {
	const pool = int64(4_000_000_000) // 4000 GNOT, in ugnot

	got := CalculatePayout(pool, pool, pool)
	if got != pool {
		t.Fatalf("winner takes all: got %d, want %d", got, pool)
	}
}

func TestImpliedProbabilityDoesNotWrapAtLargePools(t *testing.T) {
	const pool = int64(1_000_000_000_000_000) // 1e9 GNOT, in ugnot

	if got := ImpliedProbabilityBps(pool, pool); got != 10000 {
		t.Fatalf("whole pool on one outcome: got %d bps, want 10000", got)
	}
	if got := ImpliedProbabilityBps(pool/4, pool); got != 2500 {
		t.Fatalf("quarter pool on one outcome: got %d bps, want 2500", got)
	}
}
