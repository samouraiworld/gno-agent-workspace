/* Run: from a gno checkout:
gh pr checkout 5999 -R gnolang/gno && git checkout 29fb53a1e
curl -fsSL -o gno.land/pkg/gnoland/zz_abci_apphash_ab_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5999-block-gas-price-ratchet/1-29fb53a1e/tests/abci_apphash_ab_test.go
go test -v -run 'TestZZE2E' ./gno.land/pkg/gnoland/
git checkout d14a03770 -- tm2/pkg/sdk/auth/keeper.go
go test -v -run 'TestZZE2E' ./gno.land/pkg/gnoland/
git checkout HEAD -- tm2/pkg/sdk/auth/keeper.go
rm gno.land/pkg/gnoland/zz_abci_apphash_ab_test.go
*/

// Drives InitChain plus N blocks of BeginBlock/DeliverTx/EndBlock/Commit and prints
// the app hash and stored gas price per height. Run twice, once at the branch and
// once with keeper.go reverted to master, then diff the two outputs. At MaxGas
// 1000000 every app hash matches; at MaxGas 0 master panics in EndBlock at height 2;
// at MaxGas -1 master emits a new app hash every idle block.

package gnoland

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func zzReadGasPrice(t *testing.T, app abci.Application) std.GasPrice {
	t.Helper()
	qr := app.Query(abci.RequestQuery{Path: ".store/main/key", Data: []byte("gasPrice")})
	var gp std.GasPrice
	if len(qr.Value) == 0 {
		return gp
	}
	require.NoError(t, amino.Unmarshal(qr.Value, &gp))
	return gp
}

// zzRun drives maxGas through InitChain + N blocks, each burning gasPerBlock,
// and reports the app hash and stored gas price per height.
func zzRun(t *testing.T, maxGas int64, gasPerBlock []int64) {
	t.Helper()
	app := newGasPriceTestApp(t)
	gnoGen := gnoGenesisState(t)
	app.InitChain(abci.RequestInitChain{
		AppState:        gnoGen,
		ChainID:         "test-chain",
		ConsensusParams: &abci.ConsensusParams{Block: &abci.BlockParams{MaxGas: maxGas}},
	})
	baseApp := app.(*sdk.BaseApp)

	for i, gas := range gasPerBlock {
		h := int64(i + 1)
		header := &bft.Header{ChainID: "test-chain", Height: h}
		app.BeginBlock(abci.RequestBeginBlock{Header: header})
		if gas > 0 {
			tx := newCounterTx(gas)
			tx.Fee = std.Fee{GasWanted: 2000000, GasFee: sdk.Coin{Amount: 1000, Denom: "ugnot"}}
			txBytes, err := amino.Marshal(tx)
			require.NoError(t, err)
			res := app.DeliverTx(abci.RequestDeliverTx{Tx: txBytes})
			require.True(t, res.IsOK(), fmt.Sprintf("%v", res))
		}
		var panicked string
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = fmt.Sprint(r)
				}
			}()
			app.EndBlock(abci.RequestEndBlock{})
		}()
		if panicked != "" {
			t.Logf("maxGas=%d h=%d gasUsed=%d -> EndBlock PANIC %q", maxGas, h, gas, panicked)
			return
		}
		commit := app.Commit()
		t.Logf("maxGas=%-9d h=%d gasUsed=%-8d apphash=%s gasPrice=%s",
			maxGas, h, gas, hex.EncodeToString(commit.Data), zzReadGasPrice(t, app).Price.String())
		_ = baseApp
	}
}

func TestZZE2EMaxGasZero(t *testing.T) {
	zzRun(t, 0, []int64{0, 20000, 20000})
}

func TestZZE2EMaxGasMinusOne(t *testing.T) {
	zzRun(t, -1, []int64{0, 0, 0, 0, 0})
}

// Shipped shape: a positive MaxGas with a real target. The app hashes here must
// be byte-identical between master and this PR for a rolling upgrade to be safe.
func TestZZE2EShippedShape(t *testing.T) {
	zzRun(t, 1_000_000, []int64{800000, 800000, 600000, 20000, 0, 0, 900000})
}
