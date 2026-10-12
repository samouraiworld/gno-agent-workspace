package node

// Judge artifact, gnolang/gno#5995 round 10, candidate 2.
// Copy to tm2/pkg/bft/node/ and run:
//   go test ./tm2/pkg/bft/node -run TestJudgeEmptyChainIDPersistedByOlderBinary -v -count=1

import (
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

func TestJudgeEmptyChainIDPersistedByOlderBinary(t *testing.T) {
	pk := ed25519.GenPrivKey().PubKey()
	db := memdb.NewMemDB()

	// What a pre-PR binary left: the doc saved before MakeGenesisState
	// validated it, so ChainID is "", and no height-0 ABCI responses key.
	bad := &types.GenesisDoc{
		GenesisTime:     time.Unix(0, 0),
		ChainID:         "",
		ConsensusParams: types.DefaultConsensusParams(),
		Validators: []types.GenesisValidator{
			{Address: pk.Address(), PubKey: pk, Power: 10, Name: "test"},
		},
	}
	if err := saveGenesisDoc(db, bad); err != nil {
		t.Fatal(err)
	}

	// The operator fixes chain_id in the file and restarts on the new binary.
	_, _, err := LoadStateFromDBOrGenesisDocProvider(db, probeGenesisDocProvider(pk, 1))
	t.Logf("boot error: %v", err)
	if err == nil {
		t.Fatal("expected the persisted empty chain id to block the boot")
	}
}
