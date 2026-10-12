// Repro (dropped row): from a plain clone of gnolang/gno at 87f41db34,
//   cp this file tm2/pkg/bft/node/zz_probe_test.go
//   CGO_ENABLED=1 go test ./tm2/pkg/bft/node -run TestZZProbeEmptyABCIResponses -v -count=1
// Observed: len(Bytes())=0, yet genesisApplied=true on boltdb, goleveldb, lmdbdb, mdbxdb, pebbledb after close+reopen.
package node

import (
	"testing"

	sm "github.com/gnolang/gno/tm2/pkg/bft/state"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/_all"
)

// Probe: does genesisApplied see a height-0 ABCI responses record with zero
// tx responses on every backend?
func TestZZProbeEmptyABCIResponses(t *testing.T) {
	b := sm.NewABCIResponsesFromNum(0).Bytes()
	t.Logf("len(NewABCIResponsesFromNum(0).Bytes()) = %d", len(b))
	for _, be := range dbm.BackendList() {
		dir := t.TempDir()
		db0, err := dbm.NewDB("state", be, dir)
		if err != nil {
			t.Logf("%s: open: %v", be, err)
			continue
		}
		sm.SaveABCIResponses(db0, 0, sm.NewABCIResponsesFromNum(0))
		db0.Close()
		db, err := dbm.NewDB("state", be, dir)
		if err != nil {
			t.Logf("%s: reopen: %v", be, err)
			continue
		}
		v, gerr := db.Get(sm.CalcABCIResponsesKey(0))
		applied, aerr := genesisApplied(db)
		t.Logf("%-10s Get nil=%v len=%d err=%v | genesisApplied=%v err=%v", be, v == nil, len(v), gerr, applied, aerr)
		db.Close()
	}
}
