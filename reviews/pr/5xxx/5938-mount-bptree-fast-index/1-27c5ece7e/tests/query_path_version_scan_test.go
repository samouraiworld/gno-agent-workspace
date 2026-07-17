/* Run: from a gno checkout:
gh pr checkout 5938 -R gnolang/gno && git checkout 27c5ece7e
curl -fsSL -o tm2/pkg/store/rootmulti/zz_query_path_version_scan_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5938-mount-bptree-fast-index/1-27c5ece7e/tests/query_path_version_scan_test.go
go test -v -run 'TestQueryPathVersionScan' -timeout 1800s ./tm2/pkg/store/rootmulti/
rm tm2/pkg/store/rootmulti/zz_query_path_version_scan_test.go
*/

// MultiImmutableCacheWrapWithVersion is the call baseapp's handleQueryCustom
// makes for every custom ABCI query; on the bptree store it reaches
// MutableTree.Load, which scans every retained root record.
// At 27c5ece7e each backend opens the store per query in: iavl ~14us flat at
// every retained-version count, bptree+fastindex 209us at 1K rising to 101ms
// at 100K. gno.land's default PruneSyncable strategy retains 705,600 versions.
// When fixed, the bptree row is flat in retained, like the iavl row.

package rootmulti

import (
	"fmt"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"
	"github.com/gnolang/gno/tm2/pkg/store/iavl"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestQueryPathVersionScan(t *testing.T) {
	backends := []struct {
		name string
		ctor types.CommitStoreConstructor
	}{
		{"iavl", iavl.StoreConstructor},
		{"bptree+fastindex", storebptree.FastStoreConstructor},
	}
	retained := []int{1000, 20000, 100000}

	// Per-query store-open cost must not grow with the number of retained
	// versions. Ratio of the largest to the smallest retained count, per
	// backend; a version scan shows up as a large ratio.
	const maxGrowth = 5.0

	for _, b := range backends {
		var first, last time.Duration
		for i, versions := range retained {
			d := timeStoreOpen(t, b.ctor, versions)
			t.Logf("%-18s retained=%6d  per-query store open = %v", b.name, versions, d)
			if i == 0 {
				first = d
			}
			last = d
		}
		if growth := float64(last) / float64(first); growth > maxGrowth {
			t.Errorf("%s: per-query store open grew %.0fx between %d and %d retained versions "+
				"(%v -> %v); the query path scans retained roots",
				b.name, growth, retained[0], retained[len(retained)-1], first, last)
		}
	}
}

// timeStoreOpen commits `versions` single-key versions with no pruning, then
// times the store open baseapp performs per custom ABCI query.
func timeStoreOpen(t *testing.T, ctor types.CommitStoreConstructor, versions int) time.Duration {
	t.Helper()

	db := memdb.NewMemDB()
	ms := NewMultiStore(db)
	key := types.NewStoreKey("main")
	ms.MountStoreWithDB(key, ctor, db)
	if err := ms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	for i := range versions {
		ms.GetStore(key).Set(nil, []byte(fmt.Sprintf("k%06d", i)), []byte("v"))
		ms.Commit()
	}
	last := ms.LastCommitID().Version

	const iters = 10
	start := time.Now()
	for range iters {
		if _, err := ms.MultiImmutableCacheWrapWithVersion(last); err != nil {
			t.Fatal(err)
		}
	}
	return time.Since(start) / iters
}
