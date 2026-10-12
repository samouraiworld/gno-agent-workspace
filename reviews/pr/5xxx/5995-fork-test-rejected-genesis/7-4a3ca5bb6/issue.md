Target: https://github.com/gnolang/gno/issues/new
Status: draft from round 7 of gnolang/gno#5995, reproduced on master at 02606904e. Searched gnolang/gno issues and pull requests, open and closed, for initial_height, InitialHeight with gnoland start, validateSignerInfo, applyStreamingAppState, GenesisStateRef and hardfork genesis start: no issue covers it. Nearest: gnolang/gno#6133, open, on the speed of applyStreamingAppState.

## Title

gnoland start ignores a genesis file's initial_height and skips the checks the fork test applies

## Body

## Description

`gnoland start` boots a hardfork genesis from height 1 whatever its `initial_height` says, and it accepts genesis files that `gnogenesis fork test` and the in-memory node refuse.

`gnoland start` reads the genesis file through [`StreamingGenesisProvider`](https://github.com/gnolang/gno/blob/02606904e30753f7775246624bfec96cb4749891/gno.land/cmd/gnoland/start.go#L304). Its [top-level field switch](https://github.com/gnolang/gno/blob/02606904e30753f7775246624bfec96cb4749891/gno.land/pkg/gnoland/genesis_state_ref.go#L477-L490) reads `genesis_time`, `chain_id`, `consensus_params`, `validators` and `app_hash`, and logs `initial_height` as an unknown field. The same file read by [`GenesisDocFromFile`](https://github.com/gnolang/gno/blob/02606904e30753f7775246624bfec96cb4749891/tm2/pkg/bft/types/genesis.go#L186), the loader the fork test uses, keeps it:

```
WARN ignoring unknown top-level genesis field field=initial_height
GenesisDocFromFile.InitialHeight=100 LoadStreamingGenesisDoc.InitialHeight=0
```

The app state it streams goes to [`applyStreamingAppState`](https://github.com/gnolang/gno/blob/02606904e30753f7775246624bfec96cb4749891/gno.land/pkg/gnoland/app.go#L563), which is called without the requested `InitialHeight`. It runs neither of the two checks the in-memory path runs: the [`InitialHeight` comparison](https://github.com/gnolang/gno/blob/02606904e30753f7775246624bfec96cb4749891/gno.land/pkg/gnoland/app.go#L574-L578) and [`validateSignerInfo`](https://github.com/gnolang/gno/blob/02606904e30753f7775246624bfec96cb4749891/gno.land/pkg/gnoland/app.go#L591). One genesis whose `app_state.initial_height` is 999 and whose `initial_height` is 100 gets two answers from `InitChain`:

```
in-memory (fork test path):  IsErr=true err=InitialHeight mismatch: GnoGenesisState.InitialHeight=999, GenesisDoc.InitialHeight=100
streaming (gnoland start path): IsErr=false err=<nil>
```

<details><summary>Repro, two tests in gno.land/pkg/gnoland</summary>

From a clone of gnolang/gno at 02606904e, save as `gno.land/pkg/gnoland/zz_streaming_test.go` and run `go test ./gno.land/pkg/gnoland/ -run 'TestStreamingDropsInitialHeight|TestStreamingSkipsInitialHeightCheck' -v -count=1`. Both fail on master.

```go
package gnoland

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/sdk"
)

func TestStreamingDropsInitialHeight(t *testing.T) {
	pk := bft.NewMockPV().PubKey()
	doc := bft.GenesisDoc{
		GenesisTime:     time.Unix(1700000000, 0).UTC(),
		ChainID:         "ih-test",
		InitialHeight:   100,
		ConsensusParams: abci.ConsensusParams{Block: defaultBlockParams()},
		Validators:      []bft.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "v"}},
		AppState:        DefaultGenState(),
	}
	data, err := amino.MarshalJSONIndent(doc, "", "  ")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "genesis.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	plain, err := bft.GenesisDocFromFile(path)
	require.NoError(t, err)
	streamed, err := LoadStreamingGenesisDoc(path, t.TempDir(), nil)
	require.NoError(t, err)
	t.Logf("GenesisDocFromFile.InitialHeight=%d LoadStreamingGenesisDoc.InitialHeight=%d", plain.InitialHeight, streamed.InitialHeight)
	require.Equal(t, plain.InitialHeight, streamed.InitialHeight)
}

func TestStreamingSkipsInitialHeightCheck(t *testing.T) {
	state := DefaultGenState()
	state.Txs = nil
	state.InitialHeight = 999

	initChain := func(appState any) abci.ResponseInitChain {
		app, err := NewAppWithOptions(TestAppOptions(memdb.NewMemDB()))
		require.NoError(t, err)
		return app.(*sdk.BaseApp).InitChain(abci.RequestInitChain{
			Time:            time.Now(),
			ChainID:         "dev",
			ConsensusParams: &abci.ConsensusParams{Block: defaultBlockParams()},
			AppState:        appState,
			InitialHeight:   100,
		})
	}

	inMem := initChain(state)
	t.Logf("in-memory (fork test path):  IsErr=%v err=%v", inMem.IsErr(), inMem.Error)

	doc, err := LoadStreamingGenesisDoc(writeMinimalGenesisFile(t, "dev", state), t.TempDir(), nil)
	require.NoError(t, err)
	streamed := initChain(doc.AppState)
	t.Logf("streaming (gnoland start path): IsErr=%v err=%v", streamed.IsErr(), streamed.Error)

	require.True(t, inMem.IsErr())
	require.True(t, streamed.IsErr(), "the streaming path accepts a genesis the in-memory path refuses")
}
```

</details>

Both come from gnolang/gno#5684, which moved `gnoland start` onto the streaming loader. gnolang/gno#5511 had added `initial_height` to the genesis doc and the two checks to the in-memory path, and the streaming path was written without them.
