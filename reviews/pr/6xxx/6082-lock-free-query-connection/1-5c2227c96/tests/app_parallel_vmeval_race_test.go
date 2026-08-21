package gnoland

/* Run: from a gno checkout:
gh pr checkout 6082 -R gnolang/gno && git checkout 5c2227c96
curl -fsSL -o gno.land/pkg/gnoland/app_parallel_vmeval_race_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6082-lock-free-query-connection/1-5c2227c96/tests/app_parallel_vmeval_race_test.go
go test -race -count=1 -run 'TestParallelVMEval' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/app_parallel_vmeval_race_test.go
*/

// Two concurrent vm/qeval calls on the read-only ABCI connection race on the
// lazily memoised fields of process-shared gnolang.Type objects.
//
// IS:     TestParallelVMEval_ReadOnlyConn    -> DATA RACE at types.go:1151 / :1288
// SHOULD: TestParallelVMEval_SerialisedConn  -> clean, and is the pre-6082 shape
//
// Both tests drive the SAME app through the SAME handleQueryCustom snapshot
// path; the only difference is the Locker the query client holds, so the delta
// is attributable to this PR's diff and nothing else. Fails at 5c2227c96.

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/amino"
	abcicli "github.com/gnolang/gno/tm2/pkg/bft/abci/client"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bftCfg "github.com/gnolang/gno/tm2/pkg/bft/config"
	"github.com/gnolang/gno/tm2/pkg/bft/proxy"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/pebbledb"
	"github.com/gnolang/gno/tm2/pkg/events"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/sdk/config"
	vmm "github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/gnolang/gno/tm2/pkg/std"
)

const (
	vmEvalRealmPath = "gno.land/r/zzparallel"
	// pairs is the number of (interface, struct) pairs the realm declares and
	// never converts between. Each pair's conversion is forced for the first
	// time by the query, so the memo fields are cold when the goroutines hit
	// them. One pair is one shot at the race; 60 keeps the test near a minute.
	vmEvalPairs = 60
	// queriers is how many goroutines hit the same cold pair at once.
	vmEvalQueriers = 6
)

// vmEvalRealmSource declares `pairs` independent interface/struct pairs. The
// realm itself never assigns a T to an I, so InterfaceType.VerifyImplementedBy
// -> FuncType.TypeID / FuncType.BoundType are unreached at deploy time and
// still nil when the queriers arrive.
func vmEvalRealmSource(pairs int) string {
	var b strings.Builder
	b.WriteString("package zzparallel\n")
	for i := range pairs {
		fmt.Fprintf(&b, "type I%d interface{ F%d() int }\n", i, i)
		fmt.Fprintf(&b, "type T%d struct{ A int; B string }\n", i)
		fmt.Fprintf(&b, "func (t T%d) F%d() int { return t.A + %d }\n", i, i, i)
		fmt.Fprintf(&b, "var X%d = T%d{A: %d, B: \"b\"}\n", i, i, i)
	}
	b.WriteString("func Render(p string) string { return p }\n")
	return b.String()
}

// newVMEvalApp boots the real gnoland app with the realm deployed at genesis
// and one block committed, so handleQueryCustom takes the immutable-snapshot
// path rather than the pre-block-1 fallback. It returns the app and the
// consensus client, leaving the caller to build whichever query client the
// case under test needs.
func newVMEvalApp(t *testing.T) (abci.Application, proxy.ClientCreator) {
	t.Helper()

	const chainID = "dev"
	deployer := ed25519.GenPrivKey()

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

	// MemPackage.Files must be sorted or AddPackage rejects the package.
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(vmEvalRealmPath)},
		{Name: "zzparallel.gno", Body: vmEvalRealmSource(vmEvalPairs)},
	}
	deployTx := std.Tx{
		Msgs: []std.Msg{vmm.NewMsgAddPackage(deployer.PubKey().Address(), vmEvalRealmPath, files)},
		Fee:  std.NewFee(500_000_000, std.NewCoin("ugnot", 10_000_000)),
	}
	// A genesis tx carries a real signature even under SkipSigVerification;
	// an empty Signatures slice fails decoding with std.NoSignaturesError.
	signBytes, err := deployTx.GetSignBytes(chainID, 0, 0)
	require.NoError(t, err)
	sig, err := deployer.Sign(signBytes)
	require.NoError(t, err)
	deployTx.Signatures = []std.Signature{{PubKey: deployer.PubKey(), Signature: sig}}

	genState := DefaultGenState()
	genState.Balances = append(genState.Balances, Balance{
		Address: deployer.PubKey().Address(),
		Amount:  std.Coins{std.NewCoin("ugnot", 100_000_000_000)},
	})
	genState.Txs = append(genState.Txs, TxWithMetadata{Tx: deployTx})

	creator := proxy.NewLocalClientCreator(app)
	cons, err := creator.NewABCIClient()
	require.NoError(t, err)
	require.NoError(t, cons.Start())
	t.Cleanup(func() { cons.Stop() })

	_, err = cons.InitChainSync(abci.RequestInitChain{
		ChainID: chainID,
		Time:    time.Now(),
		ConsensusParams: &abci.ConsensusParams{
			Block: &abci.BlockParams{MaxGas: 1_000_000_000},
		},
		AppState: genState,
	})
	require.NoError(t, err)
	_, err = cons.CommitSync()
	require.NoError(t, err)

	// One empty block, so getLastBlockHeader() reports height >= 1 and the
	// query path is the snapshot one, not the shared-checkState fallback.
	h := app.(*sdk.BaseApp).LastBlockHeight() + 1
	_, err = cons.BeginBlockSync(abci.RequestBeginBlock{
		Header: &bft.Header{ChainID: chainID, Height: h, Time: time.Now()},
	})
	require.NoError(t, err)
	_, err = cons.EndBlockSync(abci.RequestEndBlock{})
	require.NoError(t, err)
	_, err = cons.CommitSync()
	require.NoError(t, err)
	require.GreaterOrEqual(t, app.(*sdk.BaseApp).LastBlockHeight(), int64(1))

	return app, creator
}

// hammerColdPairs fires vmEvalQueriers goroutines at each cold pair in turn,
// all released from one barrier, through the supplied query client.
func hammerColdPairs(t *testing.T, query abcicli.Client) {
	t.Helper()

	for i := range vmEvalPairs {
		req := abci.RequestQuery{
			Path: "vm/qeval",
			Data: []byte(fmt.Sprintf("%s.(func(x I%d) int { return x.F%d() })(X%d)",
				vmEvalRealmPath, i, i, i)),
		}
		var start, wg sync.WaitGroup
		start.Add(1)
		errs := make([]error, vmEvalQueriers)
		out := make([]string, vmEvalQueriers)
		for g := range vmEvalQueriers {
			wg.Go(func() {
				start.Wait()
				res, err := query.QuerySync(req)
				if err != nil {
					errs[g] = err
					return
				}
				if res.IsErr() {
					errs[g] = fmt.Errorf("qeval: %w (log: %q)", res.Error, res.Log)
					return
				}
				out[g] = string(res.Data)
			})
		}
		start.Done()
		wg.Wait()
		for g := range vmEvalQueriers {
			require.NoError(t, errs[g], "pair %d querier %d", i, g)
			// Every querier evaluated the same expression: same answer, always.
			require.Equal(t, out[0], out[g], "pair %d: queriers disagreed", i)
		}
	}
}

// The read-only connection this PR ships: no lock, so the queriers reach the
// cold type memos together. Fails under -race at 5c2227c96.
func TestParallelVMEval_ReadOnlyConn(t *testing.T) {
	_, creator := newVMEvalApp(t)
	query, err := creator.NewReadOnlyABCIClient()
	require.NoError(t, err)
	require.NoError(t, query.Start())
	defer query.Stop()

	hammerColdPairs(t, query)
}

// The pre-6082 shape: the same app and the same query path, reached through a
// client holding a real mutex. Clean under -race, which is what makes the
// failure above attributable to the diff rather than to the VM.
func TestParallelVMEval_SerialisedConn(t *testing.T) {
	app, _ := newVMEvalApp(t)
	query := abcicli.NewLocalClient(new(sync.Mutex), app)
	require.NoError(t, query.Start())
	defer query.Stop()

	hammerColdPairs(t, query)
}

var _ = amino.MustMarshal
