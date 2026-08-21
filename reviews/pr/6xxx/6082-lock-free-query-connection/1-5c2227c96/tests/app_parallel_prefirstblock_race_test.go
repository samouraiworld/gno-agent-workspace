package gnoland

/* Run: from a gno checkout:
gh pr checkout 6082 -R gnolang/gno && git checkout 5c2227c96
curl -fsSL -o gno.land/pkg/gnoland/app_parallel_prefirstblock_race_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6082-lock-free-query-connection/1-5c2227c96/tests/app_parallel_prefirstblock_race_test.go
go test -race -count=1 -run 'TestPreFirstBlockSimulate' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/app_parallel_prefirstblock_race_test.go
*/

// Before the first block commits, BaseApp.Simulate falls back to the shared
// checkState (tm2/pkg/sdk/helpers.go:59-64), so concurrent .app/simulate calls
// share one gas meter and one cache store.
//
// IS:     TestPreFirstBlockSimulate_ReadOnlyConn    -> DATA RACE, gas.go / cache/store.go
// SHOULD: TestPreFirstBlockSimulate_SerialisedConn  -> clean, and is the pre-6082 shape
//
// The window is InitChain until the first block commits. gnodev and the
// in-memory integration node both set CreateEmptyBlocks=false, so they sit in
// it until someone sends a tx. Fails at 5c2227c96.

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	"github.com/gnolang/gno/tm2/pkg/amino"
	abcicli "github.com/gnolang/gno/tm2/pkg/bft/abci/client"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bftCfg "github.com/gnolang/gno/tm2/pkg/bft/config"
	"github.com/gnolang/gno/tm2/pkg/bft/proxy"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/pebbledb"
	"github.com/gnolang/gno/tm2/pkg/events"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/sdk/bank"
	"github.com/gnolang/gno/tm2/pkg/sdk/config"
	"github.com/gnolang/gno/tm2/pkg/std"
)

const (
	preBlockChainID  = "dev"
	preBlockQueriers = 6
	preBlockRounds   = 40
)

// newPreFirstBlockApp boots the app and stops after InitChain, so
// getLastBlockHeader() still reports height 0 and Simulate takes the
// shared-checkState branch. It returns the app, the creator, and the
// pre-signed simulate payloads.
func newPreFirstBlockApp(t *testing.T) (abci.Application, proxy.ClientCreator, [][]byte) {
	t.Helper()

	simKeys := make([]ed25519.PrivKeyEd25519, preBlockQueriers)
	balances := make([]Balance, 0, preBlockQueriers+1)
	funded := std.Coins{std.NewCoin("ugnot", 100_000_000)}
	for i := range simKeys {
		simKeys[i] = ed25519.GenPrivKey()
		balances = append(balances, Balance{Address: simKeys[i].PubKey().Address(), Amount: funded})
	}
	sink := ed25519.GenPrivKey().PubKey().Address()
	balances = append(balances, Balance{
		Address: sink,
		Amount:  std.Coins{std.NewCoin("ugnot", 1)},
	})

	db, err := dbm.NewDB("gnolang", dbm.PebbleDBBackend,
		filepath.Join(t.TempDir(), bftCfg.DefaultDBDir))
	require.NoError(t, err)
	appCfg := config.DefaultAppConfig()
	genesisCfg := NewTestGenesisAppConfig()
	app, err := NewAppWithOptions(&AppOptions{
		DB:          db,
		Logger:      log.NewNoopLogger(),
		EventSwitch: events.NewEventSwitch(),
		InitChainerConfig: InitChainerConfig{
			GenesisTxResultHandler: PanicOnFailingTxResultHandler,
			StdlibDir:              filepath.Join(gnoenv.RootDir(), "gnovm", "stdlibs"),
		},
		MinGasPrices:               appCfg.MinGasPrices,
		SkipGenesisSigVerification: genesisCfg.SkipSigVerification,
		PruneStrategy:              appCfg.PruneStrategy,
	})
	require.NoError(t, err)

	creator := proxy.NewLocalClientCreator(app)
	cons, err := creator.NewABCIClient()
	require.NoError(t, err)
	require.NoError(t, cons.Start())
	t.Cleanup(func() { cons.Stop() })

	genState := DefaultGenState()
	genState.Balances = append(genState.Balances, balances...)
	_, err = cons.InitChainSync(abci.RequestInitChain{
		ChainID: preBlockChainID,
		Time:    time.Now(),
		ConsensusParams: &abci.ConsensusParams{
			Block: &abci.BlockParams{MaxGas: 100_000_000},
		},
		AppState: genState,
	})
	require.NoError(t, err)
	_, err = cons.CommitSync()
	require.NoError(t, err)

	// The precondition the whole fixture rests on. Genesis is committed, so the
	// multistore is at version 1 — but Commit() republishes the header it was
	// handed, and after InitChain that is initHeader, built with no Height
	// (tm2/pkg/sdk/baseapp.go:356, :250-256). So getLastBlockHeader() still
	// reports 0 here and Simulate takes the shared-checkState branch. The
	// window closes at the first real BeginBlock, not at this Commit.
	base := app.(*sdk.BaseApp)
	require.EqualValues(t, 1, base.LastBlockHeight(),
		"fixture precondition: genesis committed, no block produced")

	sendAmount := std.Coins{std.NewCoin("ugnot", 1_000_000)}
	fee := std.NewFee(2_000_000, std.NewCoin("ugnot", 10_000_000))
	txs := make([][]byte, preBlockQueriers)
	for i, key := range simKeys {
		addr := key.PubKey().Address()
		tx := std.Tx{
			Msgs: []std.Msg{bank.NewMsgSend(addr, sink, sendAmount)},
			Fee:  fee,
		}
		// Account numbers are unqueryable before the first block; simulate does
		// not verify signatures, so any well-formed signature does.
		signBytes, err := tx.GetSignBytes(preBlockChainID, 0, 0)
		require.NoError(t, err)
		sig, err := key.Sign(signBytes)
		require.NoError(t, err)
		tx.Signatures = []std.Signature{{PubKey: key.PubKey(), Signature: sig}}
		txs[i] = amino.MustMarshal(tx)
	}
	return app, creator, txs
}

// hammerSimulate fires preBlockQueriers goroutines, each simulating its own
// pre-signed tx preBlockRounds times, all released from one barrier.
func hammerSimulate(t *testing.T, query abcicli.Client, txs [][]byte) {
	t.Helper()

	var start, wg sync.WaitGroup
	start.Add(1)
	errs := make([]error, preBlockQueriers)
	for g := range preBlockQueriers {
		wg.Go(func() {
			start.Wait()
			for range preBlockRounds {
				res, err := query.QuerySync(abci.RequestQuery{
					Path: ".app/simulate",
					Data: txs[g],
				})
				if err != nil {
					errs[g] = err
					return
				}
				if res.IsErr() {
					errs[g] = fmt.Errorf("simulate query: %w (log: %q)", res.Error, res.Log)
					return
				}
			}
		})
	}
	start.Done()
	wg.Wait()
	for g := range preBlockQueriers {
		require.NoError(t, errs[g], "querier %d", g)
	}
}

// The read-only connection this PR ships: no lock, so N simulates enter the
// shared checkState together. Fails under -race at 5c2227c96.
func TestPreFirstBlockSimulate_ReadOnlyConn(t *testing.T) {
	_, creator, txs := newPreFirstBlockApp(t)
	query, err := creator.NewReadOnlyABCIClient()
	require.NoError(t, err)
	require.NoError(t, query.Start())
	defer query.Stop()

	hammerSimulate(t, query, txs)
}

// The pre-6082 shape: same app, same fallback branch, one lock. Clean.
func TestPreFirstBlockSimulate_SerialisedConn(t *testing.T) {
	app, _, txs := newPreFirstBlockApp(t)
	query := abcicli.NewLocalClient(new(sync.Mutex), app)
	require.NoError(t, query.Start())
	defer query.Stop()

	hammerSimulate(t, query, txs)
}
