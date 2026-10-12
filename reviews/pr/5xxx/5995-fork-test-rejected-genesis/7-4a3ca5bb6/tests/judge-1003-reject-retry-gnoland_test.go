// judge-1003: gno.land's InitChainer refuses a genesis (InitialHeight
// mismatch), the operator corrects it and boots again on the same DB; the
// first block's AppHash must equal a fresh-DB boot of the corrected genesis.
//
// From a gnolang/gno clone at the PR head:
//   cp <this file> gno.land/pkg/gnoland/zz_judge1003_test.go
//   go test ./gno.land/pkg/gnoland -run TestJudge1003_RejectThenCorrectSharedDB -v -count=1
//   rm gno.land/pkg/gnoland/zz_judge1003_test.go
package gnoland

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	tmcfg "github.com/gnolang/gno/tm2/pkg/bft/config"
	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestJudge1003_RejectThenCorrectSharedDB(t *testing.T) {
	stdlibDir, err := filepath.Abs("../../../gnovm/stdlibs")
	require.NoError(t, err)
	pv := bft.NewMockPV()
	pk := pv.PubKey()
	genTime := time.Unix(1700000000, 0).UTC()

	genesis := func(appInitialHeight int64) *bft.GenesisDoc {
		st := DefaultGenState()
		st.Txs = nil
		st.InitialHeight = appInitialHeight
		return &bft.GenesisDoc{
			GenesisTime:     genTime,
			ChainID:         "judge-1003",
			InitialHeight:   100,
			ConsensusParams: abci.ConsensusParams{Block: defaultBlockParams()},
			Validators:      []bft.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "v"}},
			AppState:        st,
		}
	}

	// boot returns the first block's AppHash, or the boot error.
	boot := func(db dbm.DB, gen *bft.GenesisDoc) ([]byte, error) {
		tmc := tmcfg.TestConfig().SetRootDir(t.TempDir())
		tmc.Consensus.WALDisabled = true
		tmc.Consensus.SkipTimeoutCommit = true
		tmc.Consensus.CreateEmptyBlocks = false
		tmc.RPC.ListenAddress = "tcp://127.0.0.1:0"
		tmc.P2P.ListenAddress = "tcp://127.0.0.1:0"
		n, err := NewInMemoryNode(log.NewNoopLogger(), &InMemoryNodeConfig{
			PrivValidator:              pv,
			Genesis:                    gen,
			TMConfig:                   tmc,
			DB:                         db,
			SkipGenesisSigVerification: true,
			InitChainerConfig: InitChainerConfig{
				GenesisTxResultHandler: func(sdk.Context, std.Tx, sdk.Result) {},
				StdlibDir:              stdlibDir,
				CacheStdlibLoad:        true,
			},
		})
		if err != nil {
			return nil, err
		}
		if err := n.Start(); err != nil {
			return nil, err
		}
		defer n.Stop()
		select {
		case <-n.Ready():
		case <-time.After(90 * time.Second):
			t.Fatal("timeout waiting for the first block")
		}
		// The first block's header carries the genesis AppHash; the state
		// saved after its commit carries the hash InitChain's writes produce.
		for range 100 {
			if st := sm.LoadState(db); st.LastBlockHeight >= 100 {
				return st.AppHash, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatal("state never reached height 100")
		return nil, nil
	}

	shared := memdb.NewMemDB()
	_, err = boot(shared, genesis(999))
	t.Logf("refused boot: err=%v", err)
	require.ErrorContains(t, err, "InitialHeight mismatch")

	retried, err := boot(shared, genesis(100))
	require.NoError(t, err, "the corrected genesis must boot on the same DB")

	fresh, err := boot(memdb.NewMemDB(), genesis(100))
	require.NoError(t, err)

	t.Logf("retry AppHash=%X fresh AppHash=%X", retried, fresh)
	require.Equal(t, fresh, retried, "a refused boot must leave nothing that changes the corrected chain")
}
