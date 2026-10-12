package node

// Asserts the persisted genesis doc matches the state record after a reset whose doc write failed.
// Copy to tm2/pkg/bft/node/judge1001_test.go, then:
//   go test ./tm2/pkg/bft/node -run TestJudge1001DroppedDocWrite -v
// Fails at 87f41db34: the reset reports success with InitialHeight 100, the next load returns the refused 1000.

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

// Fails SetSync on genesisDocKey once, when armed.
type failOnceDB struct {
	dbm.DB
	armed bool
}

func (d *failOnceDB) SetSync(k, v []byte) error {
	if d.armed && bytes.Equal(k, genesisDocKey) {
		d.armed = false
		return errors.New("injected I/O error")
	}
	return d.DB.SetSync(k, v)
}

func TestJudge1001DroppedDocWrite(t *testing.T) {
	db := &failOnceDB{DB: memdb.NewMemDB()}
	gt := time.Unix(1700000000, 0).UTC()
	provider := func(h int64) GenesisDocProvider {
		return func() (*types.GenesisDoc, error) {
			return &types.GenesisDoc{ChainID: "judge", GenesisTime: gt, InitialHeight: h, AppState: "app"}, nil
		}
	}

	// Boot 1: InitialHeight 1000, refused by the app (no height-0 key written).
	_, doc, err := LoadStateFromDBOrGenesisDocProvider(db, provider(1000))
	require.NoError(t, err)
	require.Equal(t, int64(1000), doc.InitialHeight)

	// Boot 2: corrected to 100; the doc overwrite fails, the state write lands.
	db.armed = true
	state, doc, err := LoadStateFromDBOrGenesisDocProvider(db, provider(100))
	if err != nil {
		t.Logf("reset surfaced the failed write: %v", err)
		return
	}
	t.Logf("reset: doc.InitialHeight=%d state.InitialHeight=%d", doc.InitialHeight, state.InitialHeight)

	// InitChain succeeds on the corrected doc.
	sm.SaveABCIResponses(db, 0, sm.NewABCIResponsesFromNum(0))

	// Boot 3: the persisted doc is now authoritative.
	state, doc, err = LoadStateFromDBOrGenesisDocProvider(db, provider(100))
	require.NoError(t, err)
	t.Logf("restart: doc.InitialHeight=%d state.InitialHeight=%d", doc.InitialHeight, state.InitialHeight)
	require.Equal(t, int64(100), doc.InitialHeight, "persisted doc must be the corrected one")
}
