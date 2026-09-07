// Walks every from/to and amount combination BankKeeper.SendCoins accepts and
// prints the balance delta and the event count. The self rows move nothing and
// emit nothing; every zero row returns before the emit. Measured at 639d06bf2.
/* Run: from a gno checkout:
gh pr checkout 6120 -R gnolang/gno && git checkout 639d06bf2
curl -fsSL -o tm2/pkg/sdk/bank/zz_self_transfer_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6120-bank-transfer-events/2-639d06bf2/tests/self_transfer_test.go
go test -count=1 -v -run 'TestSelfTransferCaseSpace' ./tm2/pkg/sdk/bank/
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

func errText(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}
