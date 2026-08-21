# Review: [#6082](https://github.com/gnolang/gno/pull/6082)
Event: REQUEST_CHANGES

## Body
Swapping the query client for `abcicli.NewLocalClient(new(sync.Mutex), app)` takes the same app and the same query paths from 61 reported races under `-race` to zero.

## tm2/pkg/bft/proxy/client.go:40-43 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L40-L43) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L40)
Critical: two concurrent `vm/qeval` calls race on the [`FuncType.BoundType`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1149-L1156) and [`FuncType.TypeID`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1281-L1293) memos that every query's store fork shares through the [`cacheNodes` parent](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/store.go#L255), and [`VerifyImplementedBy`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1029-L1040) decides interface satisfaction off `TypeID`. [#5811](https://github.com/gnolang/gno/pull/5811) sealed uverse types only, and [`uverse.go:1930`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/uverse.go#L1930) says `Per-store types are unaffected (each is preprocessed by a single goroutine)`, so make those memo fields write-once rather than check-then-set.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6082 -R gnolang/gno
cat > gno.land/pkg/gnoland/app_parallel_vmeval_race_test.go <<'EOF'
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
EOF
go test -race -count=1 -timeout 25m -run 'TestParallelVMEval' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/app_parallel_vmeval_race_test.go
```

The read-only connection races; the same app behind a real mutex does not.

```
WARNING: DATA RACE
Read at 0x00c0133f57b0 by goroutine 292:
  gnolang.(*FuncType).BoundType()              types.go:1150
  gnolang.buildEmbeddedTrail()                 types.go:3210
  gnolang.(*InterfaceType).VerifyImplementedBy() types.go:1029
  gnolang.checkAssignableTo()                  type_check.go:427
  gnolang.Preprocess() / (*Machine).Eval()     machine.go:1045
  vm.(*VMKeeper).withQueryEvalMachine()        keeper.go:1445
  vm.vmHandler.queryEval()                     handler.go:203
  sdk.handleQueryCustom() / (*BaseApp).Query()
  abcicli.(*localClient).QuerySync()           local_client.go:187
Previous write at 0x00c0133f57b0 by goroutine 294:
  gnolang.(*FuncType).BoundType()              types.go:1151
# … 3 more races, TypeID reading Params/Results while BoundType writes them
--- FAIL: TestParallelVMEval_ReadOnlyConn (89.62s)
--- PASS: TestParallelVMEval_SerialisedConn (83.88s)
```
</details>

## tm2/adr/pr6082_lock_free_query_connection.md:78-84 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/adr/pr6082_lock_free_query_connection.md#L78-L84) · [↗](../../../../../.worktrees/gno-review-6082/tm2/adr/pr6082_lock_free_query_connection.md#L78)
Critical: this fallback stays live past genesis and concurrent simulates there share one `infiniteGasMeter` through the by-value `Context` copy, since `Commit()` republishes the header it was handed and after `InitChain` that header carries no `Height` ([`baseapp.go:356`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/sdk/baseapp.go#L356), [`:1026`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/sdk/baseapp.go#L1026)). Give this branch its own context and meter rather than a copy of `app.checkState.ctx`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6082 -R gnolang/gno
cat > gno.land/pkg/gnoland/app_parallel_prefirstblock_race_test.go <<'EOF'
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
EOF
go test -race -count=1 -timeout 25m -run 'TestPreFirstBlockSimulate' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/app_parallel_prefirstblock_race_test.go
```

Both [`gnodev`](https://github.com/gnolang/gno/blob/5c2227c96/contribs/gnodev/pkg/dev/node.go#L93) and the [in-memory integration node](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/integration/node_testing.go#L194) set `CreateEmptyBlocks=false`, so they sit in this window until a transaction arrives. 57 races, every one of them the same gas meter.

```
Read at 0x00c005c504b8 by goroutine 46:
  store/types.(*infiniteGasMeter).ConsumeGas()  gas.go:290
  store/cache.(*cacheStore).Get()               cache/store.go:144
  sdk/params.getStructFieldsFromStore()
  vm.(*VMKeeper).GetParams()                    params.go:255
  sdk.(*BaseApp).runTx() / Simulate()           helpers.go:63
  sdk.handleQueryApp() / (*BaseApp).Query()
Previous write at 0x00c005c504b8 by goroutine 51:
  store/types.(*infiniteGasMeter).ConsumeGas()  gas.go:294
--- FAIL: TestPreFirstBlockSimulate_ReadOnlyConn (28.35s)
--- PASS: TestPreFirstBlockSimulate_SerialisedConn (30.39s)
```
</details>

## tm2/pkg/bft/proxy/client.go:52 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L52) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L52)
The mutex was the only aggregate bound on query work: [`maxAllocQuery` and `maxGasQuery`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/sdk/vm/keeper.go#L50-L52) are installed [per call](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/sdk/vm/keeper.go#L1390-L1394) and nothing under `tm2/pkg/bft/rpc/` bounds concurrency, so with it gone the ceiling is [`MaxOpenConnections`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/rpc/config/config.go#L105), 900 by default and in every `misc/deployments/test*/config.toml`. Bound in-flight queries on this connection behind a `GOMAXPROCS`-sized semaphore with a config knob, so operators keep a way back to today's behaviour without a rebuild.

## gno.land/pkg/gnoland/app_parallel_query_test.go:22 [gh](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go#L22) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_parallel_query_test.go#L22)
CI never runs `-race`, so this file's central assertion never executes on merge: [`_ci-go.yml:124`](https://github.com/gnolang/gno/blob/5c2227c96/.github/workflows/_ci-go.yml?plain=1#L124) passes `-covermode=set ./...`, and `go test -covermode=set -race` exits with `-covermode must be "atomic", not "set", when -race is enabled`. Run a `-race -covermode=atomic` step over `gno.land/pkg/gnoland`, `gno.land/pkg/sdk/vm` and `tm2/pkg/bft/proxy` instead of the whole tree.

## tm2/pkg/bft/proxy/client.go:37-38 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L37-L38) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L37)
[`ClientCreator.NewReadOnlyABCIClient`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L16-L18) still promises an independent mutex, as do [`multi_app_conn.go:24-26`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/appconn/multi_app_conn.go#L24-L26) and its call site at [`:74`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/appconn/multi_app_conn.go#L74), so an implementation written against either interface would supply one and silently revert this. Both declarations need the precondition instead, that the application is goroutine-safe for `Query` and `Info`.

## tm2/adr/pr6082_lock_free_query_connection.md:90-92 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/adr/pr6082_lock_free_query_connection.md#L90-L92) · [↗](../../../../../.worktrees/gno-review-6082/tm2/adr/pr6082_lock_free_query_connection.md#L90)
`TestQueryRace_FastIndexParity` runs [four concurrent query goroutines](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L212), not one against one committer; they never overlapped because `queryMtx` serialised them. Update that file's comments at [`:11-13`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L11-L13), [`:134-135`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L134-L135) and [`:179-181`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L179-L181) in the same pass, since the hooked query no longer holds a mutex during that wait.

## tm2/adr/pr6082_lock_free_query_connection.md:64-67 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/adr/pr6082_lock_free_query_connection.md#L64-L67) · [↗](../../../../../.worktrees/gno-review-6082/tm2/adr/pr6082_lock_free_query_connection.md#L64)
Every concurrent query now pays the per-query immutable-multistore rebuild in [`immutableAtVersion`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/rootmulti/store.go#L408-L447), a per-call cost [#6018](https://github.com/gnolang/gno/pull/6018) was signed off on because the serialised query connection capped throughput. Consequences should record that, and say whether that review's deferred fix, one cached immutable multistore per snapshot generation, is a prerequisite here or a follow-up.

## tm2/adr/pr6082_lock_free_query_connection.md:69-71 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/adr/pr6082_lock_free_query_connection.md#L69-L71) · [↗](../../../../../.worktrees/gno-review-6082/tm2/adr/pr6082_lock_free_query_connection.md#L69)
Nit: every method on the read-only client lost its lock, the mutating ones included, and `SetResponseCallback` is the only one named here, while `NewReadOnlyABCIClient` is public and hands back the raw `abcicli.Client` as both tests in this PR do.

## gno.land/pkg/gnoland/app_parallel_query_test.go:75 [gh](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go#L75) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_parallel_query_test.go#L75)
Missing test: a `bank.NewMsgSend` simulate never enters the GnoVM, so nothing here, or anywhere else in the tree, exercises the `gnoStore`, `cacheNodes` and type-graph sharing the change calls load-bearing, while [gnoweb already fans out](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoweb/handler_http.go#L1082) several VM queries per page render.

<details><summary>test cases</summary>

```go
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
```
</details>

## tm2/pkg/bft/proxy/client.go:59-62 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L59-L62) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L59)
Missing test: `tm2/pkg/bft/proxy`, `tm2/pkg/bft/abci/client` and `tm2/pkg/bft/appconn` have no test files between them, so the only guard on this contract is the 342-line [`app_parallel_query_test.go`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go), which needs a database, the VM and `gnoenv`. A cheaper guard pins the property structurally in 1.4s with none of them, using a gate application that deadlocks when any lock returns rather than merely slowing down.

<details><summary>test cases</summary>

```go
package proxy

/* Run: from a gno checkout:
gh pr checkout 6082 -R gnolang/gno && git checkout 5c2227c96
curl -fsSL -o tm2/pkg/bft/proxy/client_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6082-lock-free-query-connection/1-5c2227c96/tests/proxy_client_test.go
go test -race -count=1 ./tm2/pkg/bft/proxy/
rm tm2/pkg/bft/proxy/client_test.go
*/

// The locking contract of the three ABCI connections, pinned structurally: the
// read-only connection must admit every caller at once, the mutating one must
// admit exactly one, and the two must not share a lock. Reintroducing any lock
// on the query connection deadlocks the first test rather than slowing it, so
// the property cannot regress quietly. 1.4s under -race, no DB and no VM.

import (
	"sync"
	"testing"
	"time"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
)

// gateApp reports entry into Query/CheckTx and blocks there until released, so
// a test can prove how many calls the client lets in at once without timing
// heuristics.
type gateApp struct {
	abci.BaseApplication
	entered chan struct{}
	release chan struct{}
}

func (g *gateApp) Query(abci.RequestQuery) abci.ResponseQuery {
	g.entered <- struct{}{}
	<-g.release
	return abci.ResponseQuery{}
}

func (g *gateApp) CheckTx(abci.RequestCheckTx) abci.ResponseCheckTx {
	g.entered <- struct{}{}
	<-g.release
	return abci.ResponseCheckTx{}
}

// The read-only connection must admit every caller at once. Reintroducing any
// per-call lock on it deadlocks this test rather than slowing it down, so the
// invariant cannot regress silently.
func TestReadOnlyABCIClient_AdmitsConcurrentQueries(t *testing.T) {
	t.Parallel()

	const n = 8
	app := &gateApp{entered: make(chan struct{}, n), release: make(chan struct{})}
	cli, err := NewLocalClientCreator(app).NewReadOnlyABCIClient()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			if _, err := cli.QuerySync(abci.RequestQuery{Path: ".app/version"}); err != nil {
				t.Error(err)
			}
		})
	}

	for i := range n {
		select {
		case <-app.entered:
		case <-time.After(30 * time.Second):
			t.Fatalf("only %d of %d queries entered Application.Query: the query "+
				"connection is serialising again", i, n)
		}
	}
	close(app.release)
	wg.Wait()
}

// The mutating connection must still serialise: one caller inside the
// application, the rest waiting.
func TestABCIClient_SerialisesMutatingCalls(t *testing.T) {
	t.Parallel()

	app := &gateApp{entered: make(chan struct{}, 2), release: make(chan struct{})}
	cli, err := NewLocalClientCreator(app).NewABCIClient()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if _, err := cli.CheckTxSync(abci.RequestCheckTx{}); err != nil {
				t.Error(err)
			}
		})
	}

	<-app.entered // first caller is inside
	select {
	case <-app.entered:
		t.Fatal("both CheckTx calls entered the application at once: the mutating " +
			"connection is no longer serialised")
	case <-time.After(200 * time.Millisecond):
	}
	close(app.release)
	wg.Wait()
}

// The consensus and mempool connections share one lock; the query connection
// shares none with them.
func TestReadOnlyClient_DoesNotContendWithConsensus(t *testing.T) {
	t.Parallel()

	app := &gateApp{entered: make(chan struct{}, 2), release: make(chan struct{})}
	creator := NewLocalClientCreator(app)
	cons, err := creator.NewABCIClient()
	if err != nil {
		t.Fatal(err)
	}
	query, err := creator.NewReadOnlyABCIClient()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if _, err := cons.CheckTxSync(abci.RequestCheckTx{}); err != nil {
			t.Error(err)
		}
	})
	<-app.entered // consensus is inside and holding its mutex

	wg.Go(func() {
		if _, err := query.QuerySync(abci.RequestQuery{}); err != nil {
			t.Error(err)
		}
	})
	select {
	case <-app.entered:
	case <-time.After(30 * time.Second):
		t.Fatal("query blocked behind the consensus connection")
	}
	close(app.release)
	wg.Wait()
}

// NewLocalClient's nil guard: an untyped nil gets a working mutex.
func TestNewLocalClient_NilLockerGetsAMutex(t *testing.T) {
	t.Parallel()

	cli, err := (&localClientCreator{app: abci.NewBaseApplication()}).NewABCIClient()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.QuerySync(abci.RequestQuery{}); err != nil {
		t.Fatal(err)
	}
}
```
</details>

## tm2/pkg/bft/abci/client/local_client.go:27-31 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/abci/client/local_client.go#L27-L31) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/abci/client/local_client.go#L27)
Nit: no caller passes nil, since [`client.go:34`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L34) and [`:52`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L52) pass `&l.mtx` and `noLock` and [`common_test.go:284-285`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/consensus/common_test.go#L284-L285) passes `new(sync.Mutex)`, so deleting the guard drops three lines, this four-line caveat and a Consequences entry, and moves a typed nil's panic from first use to the call site.

## tm2/pkg/bft/proxy/client.go:55-57 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L55-L57) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L55)
Nit: `noopMutex{}` is an empty struct, so inlining it at the one call site allocates nothing and removes a package-level mutable `var` from a change whose subject is removing shared mutable state.

## tm2/pkg/bft/proxy/client.go:45-46 [gh](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L45-L46) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L45)
Nit: the `*Async` forms of those three names nil-deref on the unset `Callback` at [`local_client.go:226`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/abci/client/local_client.go#L226), pre-existing and identical on the consensus client, so naming `EchoSync`, `InfoSync` and `QuerySync` keeps this from reading as a safety claim about `QueryAsync`.

## gno.land/pkg/gnoland/app_parallel_query_test.go:335-341 [gh](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go#L335-L341) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_parallel_query_test.go#L335)
Nit: round 0 is itself a parallel round, so a shift caused by concurrency moves every round equally and still passes; capture a serial baseline before the barrier to make this an equality check rather than a stability check.
