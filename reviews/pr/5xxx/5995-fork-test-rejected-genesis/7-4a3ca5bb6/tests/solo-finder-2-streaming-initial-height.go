// Repro: gnoland start's streaming genesis loader drops initial_height.
//
// From a plain clone of gnolang/gno at 4a3ca5bb6ae549bc5da177ae7e6c96e53756d8eb
// (same result at merge base cfcbaf484cd489131bc24e95de3632dd1a2eb8b7):
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 4a3ca5bb6ae549bc5da177ae7e6c96e53756d8eb
//   mkdir -p gno.land/pkg/gnoland/zzrepro
//   cp <this file> gno.land/pkg/gnoland/zzrepro/repro_test.go
//   go test ./gno.land/pkg/gnoland/zzrepro/ -run TestStreamingDropsInitialHeight -v -count=1
//
// Observed at head:
//   WARN ignoring unknown top-level genesis field field=initial_height
//   GenesisDocFromFile.InitialHeight=100 LoadStreamingGenesisDoc.InitialHeight=0
//   --- FAIL: TestStreamingDropsInitialHeight
//
// A FAIL is the defect: the loader gnoland start uses (StreamingGenesisProvider)
// returns InitialHeight 0 for a genesis declaring 100.

package zzrepro

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoland"
	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
)

// gnoland start loads genesis through StreamingGenesisProvider ->
// LoadStreamingGenesisDoc. Compare its InitialHeight with the plain loader.
func TestStreamingDropsInitialHeight(t *testing.T) {
	pk := bft.NewMockPV().PubKey()
	st := gnoland.DefaultGenState()
	st.PastChainIDs = []string{"src-chain"}
	st.InitialHeight = 999 // disagrees with the doc: the in-memory path refuses this
	doc := bft.GenesisDoc{
		GenesisTime:     time.Unix(1700000000, 0).UTC(),
		ChainID:         "ih-test",
		InitialHeight:   100,
		ConsensusParams: abci.ConsensusParams{Block: &abci.BlockParams{MaxTxBytes: 1_000_000, MaxDataBytes: 2_000_000, MaxGas: 3_000_000_000, TimeIotaMS: 100}},
		Validators:      []bft.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "v"}},
		AppState:        st,
	}
	data, err := amino.MarshalJSONIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "genesis.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	plain, err := bft.GenesisDocFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	streamed, err := gnoland.LoadStreamingGenesisDoc(path, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("GenesisDocFromFile.InitialHeight=%d LoadStreamingGenesisDoc.InitialHeight=%d", plain.InitialHeight, streamed.InitialHeight)
	if streamed.InitialHeight != plain.InitialHeight {
		t.Fatalf("streaming loader (gnoland start) dropped initial_height: got %d, file says %d", streamed.InitialHeight, plain.InitialHeight)
	}
}
