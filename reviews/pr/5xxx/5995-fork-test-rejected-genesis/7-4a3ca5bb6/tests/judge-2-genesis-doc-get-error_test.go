// judge-2: a read error on the persisted genesis doc lets an edited genesis
// file replace it with no chain-id guard. The control (Get working) refuses
// the same file. Fails at 4a3ca5bb6: the Get-error boot returns nil.
//
// From a gnolang/gno clone at the PR head:
//   cp <this file> tm2/pkg/bft/node/zz_judge2_test.go
//   go test ./tm2/pkg/bft/node -run TestJudge2_GetErrorReplacesPersistedDoc -v -count=1
//   rm tm2/pkg/bft/node/zz_judge2_test.go
package node

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

// DB whose next Get of the genesis doc key fails once.
type getErrDB struct {
	*memdb.MemDB
	failNext bool
}

func (d *getErrDB) Get(key []byte) ([]byte, error) {
	if d.failNext && bytes.Equal(key, genesisDocKey) {
		d.failNext = false
		return nil, errors.New("injected read error")
	}
	return d.MemDB.Get(key)
}

func TestJudge2_GetErrorReplacesPersistedDoc(t *testing.T) {
	pk := ed25519.GenPrivKey().PubKey()
	edited := func() (*types.GenesisDoc, error) {
		d, _ := probeGenesisDocProvider(pk, 1)()
		d.ChainID = "edited-chain"
		return d, nil
	}

	ctrl := dbm.DB(memdb.NewMemDB())
	seedCommittedChain(t, ctrl, pk)
	_, _, cerr := LoadStateFromDBOrGenesisDocProvider(ctrl, edited)
	t.Logf("control, Get ok: err=%v", cerr)

	d := &getErrDB{MemDB: memdb.NewMemDB()}
	db := dbm.DB(d)
	seedCommittedChain(t, db, pk)
	d.failNext = true
	state, _, err := LoadStateFromDBOrGenesisDocProvider(db, edited)
	persisted, _ := loadGenesisDoc(db)
	t.Logf("Get error once: err=%v persisted.ChainID=%s state.ChainID=%s state.LastBlockHeight=%d",
		err, persisted.ChainID, state.ChainID, state.LastBlockHeight)

	require.Error(t, cerr, "control must refuse the edited file")
	require.Error(t, err, "a read error on the persisted genesis doc must fail the boot")
}
