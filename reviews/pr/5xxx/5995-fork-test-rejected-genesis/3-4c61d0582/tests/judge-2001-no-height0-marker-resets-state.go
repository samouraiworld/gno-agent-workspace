// Asserts a committed chain whose state DB has no height-0 ABCI responses key,
// a data dir whose genesis was applied before gnolang/gno#1941 (3aa0f0b7d, 2024-07-23),
// keeps its state record across a boot. Fails at 4c61d0582: LastBlockHeight
// 500 -> 0 persisted. Passes at the merge base cfcbaf484.
//
//   cp <this file> tm2/pkg/bft/node/zz_no_marker_test.go
//   go test ./tm2/pkg/bft/node -run TestJudgeNoHeight0MarkerKeepsState -v -count=1

package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

func TestJudgeNoHeight0MarkerKeepsState(t *testing.T) {
	db := memdb.NewMemDB()
	pk := ed25519.GenPrivKey().PubKey()
	provider := func() (*types.GenesisDoc, error) {
		return &types.GenesisDoc{
			GenesisTime:     time.Unix(0, 0),
			ChainID:         "genesis-doc-test",
			ConsensusParams: types.DefaultConsensusParams(),
			Validators: []types.GenesisValidator{
				{Address: pk.Address(), PubKey: pk, Power: 10, Name: "test"},
			},
			AppState: "app-state",
		}, nil
	}

	state, _, err := LoadStateFromDBOrGenesisDocProvider(db, provider)
	require.NoError(t, err)

	// 500 blocks committed, no abciResponsesKey:0 written.
	state.LastBlockHeight = 500
	state.AppHash = []byte("committed-app-hash")
	sm.SaveState(db, state)

	_, _, err = LoadStateFromDBOrGenesisDocProvider(db, provider)
	after := sm.LoadState(db)
	t.Logf("boot err=%v; persisted LastBlockHeight=%d AppHash=%q", err, after.LastBlockHeight, after.AppHash)
	assert.Equal(t, int64(500), after.LastBlockHeight)
	assert.Equal(t, []byte("committed-app-hash"), after.AppHash)
}
