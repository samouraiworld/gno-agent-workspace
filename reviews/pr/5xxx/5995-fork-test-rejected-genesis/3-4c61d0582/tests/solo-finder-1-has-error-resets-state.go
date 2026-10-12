// solo-finder-1: genesisApplied swallows a Has error, and resetToGenesisDoc then
// rewrites the state record of a chain that has committed blocks.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/5995/head && git checkout --detach 4c61d0582649ececdaa0ffa5c4ebdcabdeacc702
//	cp <this file> tm2/pkg/bft/node/zz_solofinder1_test.go
//	go test ./tm2/pkg/bft/node/ -run TestSoloFinder1 -v -count=1
//
// Observed at 4c61d0582:
//
//	returned err=<nil> LastBlockHeight=0
//	persisted after restart: LastBlockHeight=0 AppHash=""
//	len(NewABCIResponsesFromNum(0).Bytes()) = 0 (nil=true)
//	memdb/pebbledb/goleveldb/boltdb/lmdbdb/mdbxdb: Has=true genesisApplied=true
//
// Merge base cfcbaf484, same Has-error test (node.go swapped to the base copy,
// the genesisApplied call in the second test replaced by false so it compiles):
//
//	git show cfcbaf484cd489131bc24e95de3632dd1a2eb8b7:tm2/pkg/bft/node/node.go > tm2/pkg/bft/node/node.go
//	sed -i 's/genesisApplied(db)/false/' tm2/pkg/bft/node/zz_solofinder1_test.go
//	go test ./tm2/pkg/bft/node/ -run TestSoloFinder1_HasError -v -count=1
//
//	returned err=<nil> LastBlockHeight=500
//	persisted after restart: LastBlockHeight=500 AppHash="committed-app-hash"
//
// The second test settles that an empty genesis-responses blob (a genesis with
// no tx responses) still reads as applied on every backend: that suspicion is dropped.
package node

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/boltdb"
	_ "github.com/gnolang/gno/tm2/pkg/db/goleveldb"
	"github.com/gnolang/gno/tm2/pkg/db/lmdbdb"
	"github.com/gnolang/gno/tm2/pkg/db/mdbxdb"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	_ "github.com/gnolang/gno/tm2/pkg/db/pebbledb"
)

// hasErrDB fails every Has, as a disk read error would.
type hasErrDB struct{ dbm.DB }

func (hasErrDB) Has([]byte) (bool, error) { return false, errors.New("injected read error") }

// A running chain (genesis applied, 500 blocks) restarted while Has fails:
// genesisApplied swallows the error and resetToGenesisDoc rewrites the state record.
func TestSoloFinder1_HasErrorClobbersCommittedState(t *testing.T) {
	db := memdb.NewMemDB()
	pk := ed25519.GenPrivKey().PubKey()

	state, _, err := LoadStateFromDBOrGenesisDocProvider(db, probeGenesisDocProvider(pk, 1))
	require.NoError(t, err)
	sm.SaveABCIResponses(db, 0, sm.NewABCIResponsesFromNum(3))
	state.LastBlockHeight = 500
	state.AppHash = []byte("committed-app-hash")
	sm.SaveState(db, state)

	got, _, err := LoadStateFromDBOrGenesisDocProvider(hasErrDB{db}, probeGenesisDocProvider(pk, 1))
	t.Logf("returned err=%v LastBlockHeight=%d", err, got.LastBlockHeight)
	persisted := sm.LoadState(db)
	t.Logf("persisted after restart: LastBlockHeight=%d AppHash=%q", persisted.LastBlockHeight, persisted.AppHash)
}

// The genesis ABCI responses blob for a genesis with no tx responses, and
// whether each backend's Has sees it once stored.
func TestSoloFinder1_EmptyGenesisResponsesHas(t *testing.T) {
	bz := sm.NewABCIResponsesFromNum(0).Bytes()
	t.Logf("len(NewABCIResponsesFromNum(0).Bytes()) = %d (nil=%v)", len(bz), bz == nil)

	for _, backend := range []dbm.BackendType{
		dbm.MemDBBackend, dbm.PebbleDBBackend, dbm.GoLevelDBBackend, dbm.BoltDBBackend,
		lmdbdb.LMDBBackend, mdbxdb.MDBXBackend,
	} {
		db, err := dbm.NewDB("state", backend, t.TempDir())
		if err != nil {
			t.Logf("%-10s open error: %v", backend, err)
			continue
		}
		sm.SaveABCIResponses(db, 0, sm.NewABCIResponsesFromNum(0))
		ok, herr := db.Has(sm.CalcABCIResponsesKey(0))
		_, lerr := sm.LoadABCIResponses(db, 0)
		t.Logf("%-10s Has=%v err=%v genesisApplied=%v LoadABCIResponses err=%v", backend, ok, herr, genesisApplied(db), lerr)
		db.Close()
	}
}
