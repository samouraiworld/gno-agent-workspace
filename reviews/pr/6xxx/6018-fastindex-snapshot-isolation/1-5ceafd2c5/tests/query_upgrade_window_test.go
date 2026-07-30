/* Run: from a gno checkout:
gh pr checkout 6018 -R gnolang/gno && git checkout 5ceafd2c5
curl -fsSL -o tm2/pkg/store/rootmulti/query_upgrade_window_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6018-fastindex-snapshot-isolation/1-5ceafd2c5/tests/query_upgrade_window_test.go
go test -v -run 'TestQueryImmutable_DuringFastIndexUpgradeWindow$' ./tm2/pkg/store/rootmulti/
rm tm2/pkg/store/rootmulti/query_upgrade_window_test.go
*/

package rootmulti_test

// A restart that turns the fast index on over an existing indexed-but-stale DB
// stages the whole rebuild in the shared BatchCollector, which stays undrained
// until the next block commit, while LoadVersion has already seeded the query
// snapshot from the pre-rebuild disk state. A query arriving in that window
// reads a snapshot whose stamp is behind its own version and whose 'F' entries
// are stale.
//
// At 5ceafd2c5 this passes. Replacing loadImmutableView's LoadReadonly with
// Load panics ("readonlyNoopBatch: unexpected Write on read-only DB"), and
// dropping getImmutable's stamp gate returns the version-1 value for a
// version-2 query.

import (
	"testing"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/pebbledb"
	storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"
	"github.com/gnolang/gno/tm2/pkg/store/rootmulti"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestQueryImmutable_DuringFastIndexUpgradeWindow(t *testing.T) {
	db, err := dbm.NewDB("gnolang", dbm.PebbleDBBackend, t.TempDir())
	if err != nil {
		t.Fatalf("pebble: %v", err)
	}
	defer db.Close()

	mainKey := types.NewStoreKey("main")
	key := []byte("acct")
	opts := types.StoreOptions{PruningOptions: types.NewPruningOptions(0, 1)}

	// Version 1 with the fast index on: stamp=1 and one 'F' entry persist.
	on := rootmulti.NewMultiStore(db)
	on.SetStoreOptions(opts)
	on.MountStoreWithDB(mainKey, storebptree.FastStoreConstructor, db)
	if err := on.LoadLatestVersion(); err != nil {
		t.Fatalf("load with the index on: %v", err)
	}
	cms := on.MultiCacheWrap()
	cms.GetStore(mainKey).Set(nil, key, []byte("a1"))
	cms.MultiWrite()
	on.Commit()
	on.Close()

	// Version 2 with the fast index off: the tree advances to a2, the stamp
	// stays at 1, and the 'F' entry keeps the version-1 value.
	off := rootmulti.NewMultiStore(db)
	off.SetStoreOptions(opts)
	off.MountStoreWithDB(mainKey, storebptree.StoreConstructor, db)
	if err := off.LoadLatestVersion(); err != nil {
		t.Fatalf("load with the index off: %v", err)
	}
	cms = off.MultiCacheWrap()
	cms.GetStore(mainKey).Set(nil, key, []byte("a2"))
	cms.MultiWrite()
	off.Commit()
	off.Close()

	// Restart with the fast index on: this is the upgrade window.
	up := rootmulti.NewMultiStore(db)
	up.SetStoreOptions(opts)
	up.MountStoreWithDB(mainKey, storebptree.FastStoreConstructor, db)
	if err := up.LoadLatestVersion(); err != nil {
		t.Fatalf("upgrade load: %v", err)
	}
	defer up.Close()

	res, err := up.QueryImmutable(abci.RequestQuery{Path: "/main/key", Data: key, Height: 2})
	if err != nil {
		t.Fatalf("QueryImmutable: %v", err)
	}
	if got := string(res.Value); got != "a2" {
		t.Fatalf("QueryImmutable = %q, want a2 (the stale 'F' entry must not be trusted)", got)
	}
}
