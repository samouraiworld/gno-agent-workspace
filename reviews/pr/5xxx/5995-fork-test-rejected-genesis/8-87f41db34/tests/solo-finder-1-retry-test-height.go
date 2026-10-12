// Repro (dropped row): from a plain clone of gnolang/gno at 87f41db34,
//   cp this file gno.land/pkg/gnoland/zz_probe_test.go
//   go test ./gno.land/pkg/gnoland -run TestZZProbeStopsAt999 -v -count=1
// Observed: first observed height=1000, app hash 38F9015A... unchanged for 8s; the chain stops at the proof block.
package gnoland

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/log"
)

// Probe: does the chain stop at block 999 with CreateEmptyBlocks=false, or do
// proof blocks keep coming (changing the app hash the retry test compares)?
func TestZZProbeStopsAt999(t *testing.T) {
	pv := bft.NewMockPV()
	pk := pv.PubKey()
	state := DefaultGenState()
	state.InitialHeight = 999
	gen := &bft.GenesisDoc{
		GenesisTime: time.Now(), ChainID: "retry-test", InitialHeight: 999,
		ConsensusParams: abci.ConsensusParams{Block: defaultBlockParams()},
		Validators:      []bft.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "self"}},
		AppState:        state,
	}
	db := memdb.NewMemDB()
	tmcfg := NewDefaultTMConfig(t.TempDir())
	tmcfg.Consensus.CreateEmptyBlocks = false
	n, err := NewInMemoryNode(log.NewNoopLogger(), &InMemoryNodeConfig{
		PrivValidator: pv, Genesis: gen, TMConfig: tmcfg, DB: db,
		InitChainerConfig: InitChainerConfig{
			GenesisTxResultHandler: PanicOnFailingTxResultHandler,
			StdlibDir:              filepath.Join(gnoenv.RootDir(), "gnovm", "stdlibs"),
			CacheStdlibLoad:        true,
		},
	})
	require.NoError(t, err)
	require.NoError(t, n.Start())
	defer n.Stop()
	require.Eventually(t, func() bool { return sm.LoadState(db).LastBlockHeight >= 999 }, 60*time.Second, 100*time.Millisecond)
	s := sm.LoadState(db)
	t.Logf("first observed: height=%d apphash=%X", s.LastBlockHeight, s.AppHash)
	for i := 0; i < 4; i++ {
		time.Sleep(2 * time.Second)
		s = sm.LoadState(db)
		t.Logf("+%ds: height=%d apphash=%X", 2*(i+1), s.LastBlockHeight, s.AppHash)
	}
}
