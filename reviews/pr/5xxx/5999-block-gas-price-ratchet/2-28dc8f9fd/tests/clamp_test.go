/* Run: from a gno checkout:
gh pr checkout 5999 -R gnolang/gno && git checkout 28dc8f9fd
curl -fsSL -o tm2/pkg/sdk/auth/zz_clamp_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5999-block-gas-price-ratchet/2-28dc8f9fd/tests/clamp_test.go
go test -v -run 'TestZZClamp' ./tm2/pkg/sdk/auth/
git checkout d1a33f574 -- tm2/pkg/sdk/auth/keeper.go
go test -v -run 'TestZZClamp' ./tm2/pkg/sdk/auth/
git checkout HEAD -- tm2/pkg/sdk/auth/keeper.go
rm tm2/pkg/sdk/auth/zz_clamp_test.go
*/

// Pins the four properties the int64 clamp rests on: it is reached, it keeps the
// denominator and denom, the decrease branch can never reach it even when the
// initial price is itself MaxInt64, and the capped price still decays back to the
// floor. Against master's keeper.go the first test panics with "The min gas price
// is out of int64 range".

package auth

import (
	"math"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func zzPrice(amount int64) std.GasPrice {
	return std.GasPrice{Gas: 1000, Price: std.Coin{Amount: amount, Denom: "ugnot"}}
}

func zzParams(initial int64) Params {
	return Params{
		TargetGasRatio:            70,
		GasPricesChangeCompressor: 10,
		InitialGasPrice:           zzPrice(initial),
	}
}

const zzMaxGas = int64(3_000_000_000) // the MaxBlockMaxGas default

func TestZZClampReached(t *testing.T) {
	gk := GasPriceKeeper{}
	got := gk.calcBlockGasPrice(zzPrice(math.MaxInt64-10), zzMaxGas, zzMaxGas, zzParams(1))
	t.Logf("from MaxInt64-10, one full block -> %d", got.Price.Amount)
	require.Equal(t, int64(math.MaxInt64), got.Price.Amount)
	require.Equal(t, int64(1000), got.Gas, "denominator preserved")
	require.Equal(t, "ugnot", got.Price.Denom, "denom preserved")
}

func TestZZClampFromSustainedCongestion(t *testing.T) {
	gk := GasPriceKeeper{}
	p := zzPrice(1)
	blocks := 0
	for p.Price.Amount != math.MaxInt64 {
		p = gk.calcBlockGasPrice(p, zzMaxGas, zzMaxGas, zzParams(1))
		blocks++
		require.Less(t, blocks, 5000, "cap never reached")
	}
	t.Logf("full blocks from price 1 to the cap: %d", blocks)

	idle := 0
	for p.Price.Amount != 1 {
		p = gk.calcBlockGasPrice(p, 0, zzMaxGas, zzParams(1))
		idle++
		require.Less(t, idle, 5000, "cap is absorbing: price never returns to the floor")
	}
	t.Logf("idle blocks from the cap back to the floor: %d", idle)
}

// Every decrease result lands in [1, max(last, initial)], so the branch cannot
// reach the clamp even with the largest initial price Params.Validate accepts.
func TestZZClampUnreachableWhenDecreasing(t *testing.T) {
	gk := GasPriceKeeper{}
	for _, last := range []int64{1, 2, 1000, math.MaxInt64 / 2, math.MaxInt64 - 1, math.MaxInt64} {
		for _, initial := range []int64{0, 1, 1000, math.MaxInt64} {
			got := gk.calcBlockGasPrice(zzPrice(last), 0, zzMaxGas, zzParams(initial))
			require.LessOrEqual(t, got.Price.Amount, max(last, initial),
				"decrease branch rose above its own bound: last=%d initial=%d", last, initial)
			require.GreaterOrEqual(t, got.Price.Amount, int64(1),
				"decrease branch fell below 1: last=%d initial=%d", last, initial)
		}
	}
}
