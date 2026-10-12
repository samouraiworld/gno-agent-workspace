package node

// Judge probes for gnolang/gno#5995 round 9, candidates 1 and 2.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

func judgeProvider(pk crypto.PubKey, chainID, appState string) GenesisDocProvider {
	return func() (*types.GenesisDoc, error) {
		return &types.GenesisDoc{
			GenesisTime: time.Unix(0, 0), ChainID: chainID, InitialHeight: 100,
			ConsensusParams: types.DefaultConsensusParams(),
			Validators:      []types.GenesisValidator{{Address: pk.Address(), PubKey: pk, Power: 10, Name: "t"}},
			AppState:        appState,
		}, nil
	}
}

// Candidate 1: once the height-0 key exists, a file with another validator set is mixed silently.
func TestJudge1AppliedMix(t *testing.T) {
	db := memdb.NewMemDB()
	pk1, pk2 := ed25519.GenPrivKey().PubKey(), ed25519.GenPrivKey().PubKey()
	_, _, err := LoadStateFromDBOrGenesisDocProvider(db, judgeProvider(pk1, "c", "v1"))
	require.NoError(t, err)
	sm.SaveABCIResponses(db, 0, sm.NewABCIResponsesFromNum(0))
	_, doc, err := LoadStateFromDBOrGenesisDocProvider(db, judgeProvider(pk2, "c", "v2"))
	if doc == nil {
		t.Logf("JUDGE1 err=%v doc=nil", err)
		return
	}
	t.Logf("JUDGE1 err=%v validators_from_persisted_pk1=%v appstate=%v", err, doc.Validators[0].PubKey.Equals(pk1), doc.AppState)
}

// Candidate 2: an invalid chain_id is persisted on first boot, then the corrected file is refused.
func TestJudge2EmptyChainIDPinned(t *testing.T) {
	db := memdb.NewMemDB()
	pk := ed25519.GenPrivKey().PubKey()
	_, _, err := LoadStateFromDBOrGenesisDocProvider(db, judgeProvider(pk, "", "a"))
	t.Logf("JUDGE2 first err=%v", err)
	_, _, err = LoadStateFromDBOrGenesisDocProvider(db, judgeProvider(pk, "fixed", "a"))
	t.Logf("JUDGE2 second err=%v", err)
}
