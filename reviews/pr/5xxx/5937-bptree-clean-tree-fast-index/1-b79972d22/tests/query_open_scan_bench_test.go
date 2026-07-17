/* Run: from a gno checkout:
gh pr checkout 5937 -R gnolang/gno && git checkout b79972d22
curl -fsSL -o tm2/pkg/store/bptree/zz_queryscan_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5937-bptree-clean-tree-fast-index/1-b79972d22/tests/query_open_scan_bench_test.go
go test -v -run TestQueryOpenScanPebble -timeout 3000s ./tm2/pkg/store/bptree/
rm tm2/pkg/store/bptree/zz_queryscan_test.go
*/

// Store.LoadVersion's immutable branch calls MutableTree.Load, which runs
// nodeDB.discoverVersions -- a full iteration of every retained root record --
// so each ABCI query at a height rescans the whole retained set. Measured at
// b79972d22 on PebbleDB: 15.1/29.8/62.0/124.3 ms at 50K/100K/200K/400K
// retained versions, linear at ~0.31 us per version. It goes flat once the
// immutable open stops discovering versions it never reads.

package bptree

import (
	"fmt"
	"testing"
	"time"

	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/pebbledb"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestQueryOpenScanPebble(t *testing.T) {
	// PruneSyncable is gno.land's default, so nothing prunes at these sizes.
	opts := types.StoreOptions{PruningOptions: types.PruneSyncable}
	for _, nver := range []int{50000, 100000, 200000, 400000} {
		dir := t.TempDir()
		db, err := dbm.NewDB("scan", dbm.PebbleDBBackend, dir)
		if err != nil {
			t.Fatal(err)
		}
		st := FastStoreConstructor(db, opts).(*Store)
		if err := st.LoadLatestVersion(); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < nver; i++ {
			st.Set(nil, []byte(fmt.Sprintf("k%06d", i)), []byte("v"))
			st.Commit()
		}

		// The shape rootmulti builds per ABCI query at a height.
		qopts := types.StoreOptions{PruningOptions: types.PruneSyncable, Immutable: true}
		idb := dbm.NewImmutableDB(db)
		for r := 0; r < 3; r++ { // warm the block cache
			qst := FastStoreConstructor(idb, qopts).(*Store)
			if err := qst.LoadVersion(int64(nver)); err != nil {
				t.Fatal(err)
			}
		}
		start := time.Now()
		const reps = 20
		for r := 0; r < reps; r++ {
			qst := FastStoreConstructor(idb, qopts).(*Store)
			if err := qst.LoadVersion(int64(nver)); err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("retained versions=%6d  per query-path store open: %v", nver, time.Since(start)/reps)
		db.Close()
	}
}
