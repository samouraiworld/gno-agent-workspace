# Review: PR [#5995](https://github.com/gnolang/gno/pull/5995)
Event: COMMENT

## Body
Two things outside the diff break once every `loadAppState` rejection is fatal. Verified on d1b51746a: nothing from a rejected genesis survives the abort, and removing the three-line block in [`replay.go`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L350-L352) makes the fork test print PASS again.

- [`NewDefaultGenesisConfig`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/node_inmemory.go#L43) now hands out a genesis that cannot boot: [`loadAppState`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L498-L505) rejects its `*GnoGenesisState` with `invalid AppState of type *gnoland.GnoGenesisState`. The only in-tree caller overwrites the field [one line later](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnodev/pkg/dev/node.go#L717), so nothing in the tree notices.
- An operator who hits the new abort cannot fix it from the genesis file in the case the message points at. [`LoadStateFromDBOrGenesisDocProvider`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/node/node.go#L1058-L1085) persists the doc before the handshake and on the next boot re-reads only `AppState` from the file, so a corrected doc-level `InitialHeight` is discarded and the node re-aborts.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5995 -R gnolang/gno

cat > gno.land/pkg/gnoland/zz_probe_test.go <<'EOF'
package gnoland

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	tmcfg "github.com/gnolang/gno/tm2/pkg/bft/config"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
)

func TestProbeDefaultGenesisConfigBoots(t *testing.T) {
	pv := bft.NewMockPV()
	pk := pv.PubKey()
	genesis := NewDefaultGenesisConfig("probe-chain", "gno.land")
	genesis.Validators = []bft.GenesisValidator{
		{Address: pk.Address(), PubKey: pk, Power: 10, Name: "probe"},
	}

	tmc := tmcfg.TestConfig().SetRootDir(gnoenv.RootDir())
	tmc.Consensus.WALDisabled = true
	tmc.RPC.ListenAddress = "tcp://127.0.0.1:0"
	tmc.P2P.ListenAddress = "tcp://127.0.0.1:0"

	n, err := NewInMemoryNode(log.NewNoopLogger(), &InMemoryNodeConfig{
		PrivValidator: pv,
		Genesis:       genesis,
		TMConfig:      tmc,
		DB:            memdb.NewMemDB(),
		InitChainerConfig: InitChainerConfig{
			GenesisTxResultHandler: NoopGenesisTxResultHandler,
			StdlibDir:              gnoenv.RootDir() + "/gnovm/stdlibs",
		},
	})
	t.Logf("AppState type = %T", genesis.AppState)
	t.Logf("NewInMemoryNode err = %v", err)
	if err == nil {
		n.Stop()
	}
}
EOF

cat > tm2/pkg/bft/node/zz_probe_test.go <<'EOF'
package node

import (
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/bft/types"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

// One data dir across a restart: boot 1 persists the doc, boot 2 reads it back
// while genesis.json has been corrected.
func TestProbeStaleGenesisDocWins(t *testing.T) {
	db := dbm.DB(memdb.NewMemDB())
	pk := types.NewMockPV().PubKey()
	mk := func(h int64) func() (*types.GenesisDoc, error) {
		return func() (*types.GenesisDoc, error) {
			return &types.GenesisDoc{
				GenesisTime:     time.Unix(0, 0),
				ChainID:         "probe-chain",
				InitialHeight:   h,
				ConsensusParams: types.DefaultConsensusParams(),
				Validators: []types.GenesisValidator{
					{Address: pk.Address(), PubKey: pk, Power: 10, Name: "probe"},
				},
				AppState: "some-app-state",
			}, nil
		}
	}

	_, doc1, err := LoadStateFromDBOrGenesisDocProvider(db, mk(999))
	t.Logf("boot1 err=%v InitialHeight=%d", err, doc1.InitialHeight)

	_, doc2, err := LoadStateFromDBOrGenesisDocProvider(db, mk(100))
	t.Logf("boot2 (file corrected to 100) err=%v InitialHeight=%d", err, doc2.InitialHeight)
}
EOF

go test -count=1 -run TestProbeDefaultGenesisConfigBoots -v ./gno.land/pkg/gnoland/
go test -count=1 -run TestProbeStaleGenesisDocWins -v ./tm2/pkg/bft/node/

rm gno.land/pkg/gnoland/zz_probe_test.go tm2/pkg/bft/node/zz_probe_test.go
```

```
=== RUN   TestProbeDefaultGenesisConfigBoots
    AppState type = *gnoland.GnoGenesisState
    NewInMemoryNode err = error during handshake: error on replay: InitChain rejected the genesis: invalid AppState of type *gnoland.GnoGenesisState
--- PASS: TestProbeDefaultGenesisConfigBoots

=== RUN   TestProbeStaleGenesisDocWins
    boot1 err=<nil> InitialHeight=999
    boot2 (file corrected to 100) err=<nil> InitialHeight=999
--- PASS: TestProbeStaleGenesisDocWins
```
On master both boots of the first probe return `err = <nil>`.
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5995-fork-test-rejected-genesis/1-d1b51746a/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/consensus/replay_test.go:1239-1241 [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay_test.go#L1239-L1241)
Missing test: the abort is never exercised on a hardfork genesis, where [`ReplayBlocks` persists the height alignment](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/consensus/replay.go#L311-L320) before `InitChain` runs. The comment says nothing from the rejected genesis may be persisted, but the check under it covers only the ABCI responses and the alignment write survives.

<details><summary>test cases</summary>

Passes on d1b51746a.

```go
// TestHandshakeInitChainErrorHardfork pins what the abort does and does not
// roll back on a hardfork genesis, where ReplayBlocks has already persisted the
// InitialHeight alignment before InitChain runs: the genesis ABCI responses are
// never saved, and the alignment write is left in place.
func TestHandshakeInitChainErrorHardfork(t *testing.T) {
	t.Parallel()

	app := initChainApp{
		initChain: func(req abci.RequestInitChain) abci.ResponseInitChain {
			return abci.ResponseInitChain{
				ResponseBase: abci.ResponseBase{
					Error: abci.StringError("InitialHeight mismatch"),
				},
			}
		},
	}
	clientCreator := proxy.NewLocalClientCreator(app)

	config, genesisFile := ResetConfig("handshake_test_")
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(config.RootDir)) })
	stateDB, state, store := makeStateAndStore(config, genesisFile, "v0.0.0-test")

	genDoc, _ := sm.MakeGenesisDocFromFile(genesisFile)
	genDoc.InitialHeight = 100

	handshaker := NewHandshaker(stateDB, state, store, genDoc)
	proxyApp := appconn.NewAppConns(clientCreator)
	require.NoError(t, proxyApp.Start(), "Error starting proxy app connections")
	t.Cleanup(func() { require.NoError(t, proxyApp.Stop()) })

	require.ErrorContains(t, handshaker.Handshake(proxyApp), "InitialHeight mismatch")

	_, loadErr := sm.LoadABCIResponses(stateDB, 0)
	require.Error(t, loadErr, "genesis ABCI responses must not be saved after a rejected InitChain")

	// The alignment write precedes InitChain and is not rolled back; a retry
	// with the same genesis re-runs InitChain and converges.
	after := sm.LoadState(stateDB)
	assert.Equal(t, int64(100), after.InitialHeight)
	assert.Equal(t, int64(99), after.LastBlockHeight)
}
```
</details>

## contribs/gnogenesis/internal/fork/test.go:265-269 [↗](../../../../../.worktrees/gno-review-5995/contribs/gnogenesis/internal/fork/test.go#L265-L269)
Nit: nothing can reach `processed < expected` any more, so the comment describes a failure mode that cannot occur. Every tx the [in-memory loop](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L557-L560) delivers [reaches the result handler](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L866) except the [`metadata.Failed`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app.go#L833) ones, which [`countDeliverableTxs`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/internal/fork/test.go#L342-L350) already subtracts. Reword it as an assertion that is unreachable today.

## gno.land/pkg/gnoland/app.go:400-403 [↗](../../../../../.worktrees/gno-review-5995/gno.land/pkg/gnoland/app.go#L400-L403)
Nit: the comment still frames the log line as what names the cause for the operator. The returned error now carries the same string, and it is what [`TestExecTest_InitialHeightMismatch`](https://github.com/gnolang/gno/blob/d1b51746a/contribs/gnogenesis/internal/fork/test_test.go#L286) asserts on.

## tm2/pkg/bft/consensus/replay.go:343-349 [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L343-L349)
Nit: the comment's `StrictReplay` example is not reachable: nothing sets it outside [`TestInitChainer_StrictReplay`](https://github.com/gnolang/gno/blob/d1b51746a/gno.land/pkg/gnoland/app_test.go#L2434), which bypasses the handshake, and no flag exposes it. The comment also says nothing used to read the field, but [`baseapp.InitChain`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/sdk/baseapp.go#L377-L381) does; the handshake was the one ignoring it.

## tm2/pkg/bft/consensus/replay.go:350-351 [↗](../../../../../.worktrees/gno-review-5995/tm2/pkg/bft/consensus/replay.go#L350-L351)
Nit: [`ResponseBase.IsErr`](https://github.com/gnolang/gno/blob/d1b51746a/tm2/pkg/bft/abci/types/types.go#L120-L122) is this exact check, and `%v` on the value replaces `%s` with `.Error()`.
