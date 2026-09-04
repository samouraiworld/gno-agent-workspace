/* Run: from a gno checkout:
gh pr checkout 6134 -R gnolang/gno && git checkout 7834d5d7e
curl -fsSL -o tm2/pkg/sdk/bank/initcoins_order_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6134-genesis-balance-loading-quadratic/1-7834d5d7e/tests/initcoins_order_test.go
go test -count=1 -run TestInitCoinsWritesTheAccountTierFirst ./tm2/pkg/sdk/bank/
rm tm2/pkg/sdk/bank/initcoins_order_test.go
*/
// Asserts that InitCoins writes the account tier before the split tier, so a
// failure there leaves no split-tier key behind. Passes at 7834d5d7e; fails
// "expected: 0, actual: 9" with the two writes swapped, which the shipped suite
// does not notice.

package bank

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestInitCoinsWritesTheAccountTierFirst(t *testing.T) {
	t.Parallel()

	addr := addrN(4242)
	env := setupTestEnv()

	// A session account rejects any non-zero amount, which is the one reachable
	// way setAccountTierCoins fails.
	sess := &std.BaseSessionAccount{}
	require.NoError(t, sess.SetAddress(addr))
	env.acck.SetAccount(env.ctx, sess)

	amt := std.NewCoins(std.NewCoin(testAccountDenom, 5), std.NewCoin("zzzcoin", 9))
	require.Error(t, env.bankk.InitCoins(env.ctx, addr, amt))
	require.Equal(t, int64(0), env.bankk.GetCoin(env.ctx, addr, "zzzcoin"),
		"account tier first: the failed call must leave no split-tier key")
}
