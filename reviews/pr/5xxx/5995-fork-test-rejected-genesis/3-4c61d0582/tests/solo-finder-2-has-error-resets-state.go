// Repro: a failed read of the height-0 ABCI responses key at boot makes
// LoadStateFromDBOrGenesisDocProvider rebuild a committed chain's state record
// from the genesis file and persist it (genesisApplied returns false on err).
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/5995/head && git checkout 4c61d0582649ececdaa0ffa5c4ebdcabdeacc702
//   cp <this file> tm2/pkg/bft/node/zz_has_error_test.go
//   go test ./tm2/pkg/bft/node -run TestZZHasErrorResetsCommittedChain -v -count=1
//   -> FAIL at head: LastBlockHeight 500 -> 0, AppHash replaced, state record persisted.
// Merge base cfcbaf484cd489131bc24e95de3632dd1a2eb8b7: same commands -> PASS
// (the base never probes Has; LoadState returns the stored record).

package node

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

type zzHasErrDB struct {
	*memdb.MemDB
	fail bool
}

func (d *zzHasErrDB) Has(key []byte) (bool, error) {
	if d.fail {
		return false, errors.New("injected read error")
	}
	return d.MemDB.Has(key)
}

func zzProvider(pk crypto.PubKey) GenesisDocProvider {
	return func() (*types.GenesisDoc, error) {
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
}

func TestZZHasErrorResetsCommittedChain(t *testing.T) {
	d := &zzHasErrDB{MemDB: memdb.NewMemDB()}
	db := dbm.DB(d)
	pk := ed25519.GenPrivKey().PubKey()

	state, _, err := LoadStateFromDBOrGenesisDocProvider(db, zzProvider(pk))
	require.NoError(t, err)

	// A chain that has run: genesis applied, 500 blocks committed.
	sm.SaveABCIResponses(db, 0, sm.NewABCIResponsesFromNum(0))
	state.LastBlockHeight = 500
	state.AppHash = []byte("committed-app-hash")
	sm.SaveState(db, state)

	// One failed read at boot.
	d.fail = true
	_, _, err = LoadStateFromDBOrGenesisDocProvider(db, zzProvider(pk))
	d.fail = false

	after := sm.LoadState(db) // what the next boot reads, read error gone
	t.Logf("boot err=%v; persisted LastBlockHeight=%d AppHash=%q", err, after.LastBlockHeight, after.AppHash)
	assert.Equal(t, int64(500), after.LastBlockHeight, "a read error must not rewrite a committed chain's state record")
	assert.Equal(t, []byte("committed-app-hash"), after.AppHash)
}
