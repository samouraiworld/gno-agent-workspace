/* Run: from a gno checkout:
gh pr checkout 5946 -R gnolang/gno && git checkout 037a90410
mkdir -p /tmp/pmparity && cd /tmp/pmparity && go mod init pmparity
curl -fsSL -o payout_go_parity_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5946-prediction-market-demo-realm/1-037a90410/tests/payout_go_parity_test.go
go test -v ./...
cd - && rm -rf /tmp/pmparity
*/

// ClaimWinnings' payout expression under Go int64 semantics. The wrapped
// results here match what the GnoVM produces for the same inputs, so the
// negative payout is plain int64 overflow rather than a VM divergence.
// A fix that keeps the intermediate in range makes the want column exact.
package pmparity

import "testing"

// payoutBuggy mirrors predictionmarket.gno's ClaimWinnings arithmetic.
func payoutBuggy(amount, totalPool, winningPool int64) int64 {
	return (amount * totalPool) / winningPool
}

// payoutFixed splits the multiplication so the intermediate stays in range
// for a sole winner taking the whole pool.
func payoutFixed(amount, totalPool, winningPool int64) int64 {
	return (amount/winningPool)*totalPool + ((amount%winningPool)*totalPool)/winningPool
}

func TestPayoutOverflowParity(t *testing.T) {
	cases := []struct {
		name        string
		stake       int64
		wantBuggy   int64
		wantCorrect int64
	}{
		{"1 GNOT per side", 1_000_000, 2_000_000, 2_000_000},
		{"3000 GNOT per side", 3_000_000_000, -148914691, 6_000_000_000},
		{"5000 GNOT per side", 5_000_000_000, -1068046444, 10_000_000_000},
	}
	for _, tc := range cases {
		total := 2 * tc.stake
		if got := payoutBuggy(tc.stake, total, tc.stake); got != tc.wantBuggy {
			t.Errorf("%s: payoutBuggy = %d, want %d", tc.name, got, tc.wantBuggy)
		}
		if got := payoutFixed(tc.stake, total, tc.stake); got != tc.wantCorrect {
			t.Errorf("%s: payoutFixed = %d, want %d", tc.name, got, tc.wantCorrect)
		}
	}
}
