// solo-finder-1: the InitialHeight mismatch that `gnogenesis fork test` now
// fails on (in-memory GnoGenesisState path) is accepted by the streaming
// *GenesisStateRef path that `gnoland start` uses.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/5995/head && git checkout 4a3ca5bb6ae549bc5da177ae7e6c96e53756d8eb
//   cp <this file> gno.land/pkg/gnoland/zz_solo_finder1_test.go
//   go test ./gno.land/pkg/gnoland -run TestSoloFinder1_InitialHeightMismatch -v -count=1
//
// Expect if the paths agree: both responses IsErr. Observed: in-memory IsErr,
// streaming IsOK.
package gnoland

import (
	"testing"
	"time"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/sdk"
	"github.com/stretchr/testify/require"
)

func TestSoloFinder1_InitialHeightMismatch(t *testing.T) {
	state := DefaultGenState()
	state.Txs = nil
	state.InitialHeight = 999 // GenesisDoc / RequestInitChain says 100

	run := func(appState any) abci.ResponseInitChain {
		app, err := NewAppWithOptions(TestAppOptions(memdb.NewMemDB()))
		require.NoError(t, err)
		return app.(*sdk.BaseApp).InitChain(abci.RequestInitChain{
			Time:            time.Now(),
			ChainID:         "dev",
			ConsensusParams: &abci.ConsensusParams{Block: defaultBlockParams()},
			Validators:      []abci.ValidatorUpdate{},
			AppState:        appState,
			InitialHeight:   100,
		})
	}

	inMem := run(state)
	t.Logf("in-memory (fork test path):  IsErr=%v err=%v", inMem.IsErr(), inMem.Error)

	src := writeMinimalGenesisFile(t, "dev", state)
	doc, err := LoadStreamingGenesisDoc(src, t.TempDir(), nil)
	require.NoError(t, err)
	ref, ok := doc.AppState.(*GenesisStateRef)
	require.True(t, ok)
	streamed := run(ref)
	t.Logf("streaming (gnoland start path): IsErr=%v err=%v", streamed.IsErr(), streamed.Error)

	require.True(t, inMem.IsErr(), "in-memory path must refuse the mismatch")
	require.True(t, streamed.IsErr(), "streaming path must refuse the same genesis")
}
