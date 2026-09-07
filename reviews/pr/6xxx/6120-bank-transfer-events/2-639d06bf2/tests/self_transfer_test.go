// Walks every from/to and amount combination BankKeeper.SendCoins accepts and
// prints the balance delta, the event count and the session SpendLimit consumed.
// The self-transfer rows move nothing and emit nothing, and still spend the
// session limit. Measured at 639d06bf2; the second assertion fails there.
/* Run: from a gno checkout:
gh pr checkout 6120 -R gnolang/gno && git checkout 639d06bf2
curl -fsSL -o tm2/pkg/sdk/bank/zz_self_transfer_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/self_transfer_test.go
go test -count=1 -v -run 'TestSelfTransferCaseSpace|TestSelfTransferSpendsSessionLimit' ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/zz_self_transfer_test.go
*/
package bank

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
)

// TestSelfTransferCaseSpace enumerates the cells the fromAddr != toAddr guard
// partitions: sender identity crossed with zero, partial, whole and
// over-balance amounts.
func TestSelfTransferCaseSpace(t *testing.T) {
	t.Parallel()

	self := crypto.AddressFromPreimage([]byte("self"))
	other := crypto.AddressFromPreimage([]byte("other"))
	balance := std.NewCoins(std.NewCoin("ugnot", 100))

	cases := []struct {
		name     string
		from, to crypto.Address
		amt      std.Coins
	}{
		{"self, zero coins", self, self, std.NewCoins()},
		{"self, partial balance", self, self, std.NewCoins(std.NewCoin("ugnot", 40))},
		{"self, whole balance", self, self, balance},
		{"self, over balance", self, self, std.NewCoins(std.NewCoin("ugnot", 101))},
		{"other, zero coins", self, other, std.NewCoins()},
		{"other, partial balance", self, other, std.NewCoins(std.NewCoin("ugnot", 40))},
		{"other, whole balance", self, other, balance},
		{"other, over balance", self, other, std.NewCoins(std.NewCoin("ugnot", 101))},
	}

	for _, tc := range cases {
		// One env per row: a failed row leaves a half-written balance behind.
		env := setupTestEnv()
		require.NoError(t, env.bankk.SetCoins(env.ctx, self, balance))

		err := env.bankk.SendCoins(env.ctx, tc.from, tc.to, tc.amt)
		events := env.ctx.EventLogger().Events()
		t.Logf("%-22s err=%-26s events=%d self=%-8s other=%s",
			tc.name, errText(err), len(events),
			env.bankk.GetCoins(env.ctx, self).String(),
			env.bankk.GetCoins(env.ctx, other).String())
	}
}

// TestSelfTransferSpendsSessionLimit sends a session master's coins to itself.
// Nothing moves and no event records it, and the session's SpendLimit pays for
// it anyway, so a second self-transfer inside the same limit is refused.
func TestSelfTransferSpendsSessionLimit(t *testing.T) {
	t.Parallel()

	env := setupTestEnv()
	ctx, master, da := setupSessionCtx(t, env,
		std.NewCoins(std.NewCoin("ugnot", 1000)),
		std.NewCoins(std.NewCoin("ugnot", 500)))

	before := env.bankk.GetCoins(ctx, master).AmountOf("ugnot")
	require.NoError(t, env.bankk.SendCoins(ctx, master, master,
		std.NewCoins(std.NewCoin("ugnot", 400))))
	after := env.bankk.GetCoins(ctx, master).AmountOf("ugnot")

	require.Equal(t, before, after, "a self-transfer moves nothing")
	require.Empty(t, ctx.EventLogger().Events(), "and reports nothing")

	require.Equal(t, int64(0), da.GetSpendUsed().AmountOf("ugnot")) //     SHOULD: a no-op costs no session allowance
	// require.Equal(t, int64(400), da.GetSpendUsed().AmountOf("ugnot")) // IS: 400 of the 500 limit is gone

	// The allowance the first call burned is what refuses this one.
	err := env.bankk.SendCoins(ctx, master, master,
		std.NewCoins(std.NewCoin("ugnot", 200)))
	t.Logf("second 200ugnot self-transfer: err=%v", err)
}

func errText(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}
