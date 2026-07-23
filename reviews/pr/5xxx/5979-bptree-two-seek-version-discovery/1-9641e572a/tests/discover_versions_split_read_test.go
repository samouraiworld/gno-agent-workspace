/* Run: from a gno checkout:
gh pr checkout 5979 -R gnolang/gno && git checkout 9641e572a
curl -fsSL -o tm2/pkg/bptree/discover_versions_split_read_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5979-bptree-two-seek-version-discovery/1-9641e572a/tests/discover_versions_split_read_test.go
go test -v -run 'TestDiscoverVersionsPairCoexists' ./tm2/pkg/bptree/
rm tm2/pkg/bptree/discover_versions_split_read_test.go
*/

// discoverVersions opens two iterators, so first and latest come from two
// points in time. goleveldb and boltdb report "snapshots not supported", so
// rootmulti's immutable query store reads the live DB and a writer can land
// between the two opens. At 9641e572a this reports first=3 latest=101, a pair
// that never coexisted; the same wrapper against the single-iterator scan
// reports first=3 latest=100. The test asserts the coherent pair.

package bptree

import (
	"encoding/binary"
	"testing"

	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/stretchr/testify/require"
)

// writeOnFirstIterator lands a commit and a prune right after the first
// iterator is opened, modelling a live backend under block production.
type writeOnFirstIterator struct {
	dbm.DB
	fired bool
	write func(dbm.DB)
}

func (r *writeOnFirstIterator) Iterator(start, end []byte) (dbm.Iterator, error) {
	itr, err := r.DB.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	if !r.fired {
		r.fired = true
		r.write(r.DB)
	}
	return itr, nil
}

func splitReadRootKey(v uint64) []byte {
	k := make([]byte, 9)
	k[0] = PrefixRoot
	binary.BigEndian.PutUint64(k[1:], v)
	return k
}

func TestDiscoverVersionsPairCoexists(t *testing.T) {
	base := memdb.NewMemDB()
	for v := 3; v <= 100; v++ {
		require.NoError(t, base.Set(splitReadRootKey(uint64(v)), []byte{1}))
	}

	live := &writeOnFirstIterator{DB: base, write: func(d dbm.DB) {
		for v := 3; v <= 10; v++ {
			require.NoError(t, d.Delete(splitReadRootKey(uint64(v))))
		}
		require.NoError(t, d.Set(splitReadRootKey(101), []byte{1}))
	}}

	ndb := &nodeDB{db: live, logger: NewNopLogger()}
	require.NoError(t, ndb.discoverVersions())
	first, latest := ndb.getFirstVersion(), ndb.getLatestVersion()

	// Only two pairs ever existed: (3, 100) before the write and (11, 101)
	// after it. (3, 101) never did.
	pair := [2]int64{first, latest}
	require.Contains(t, [][2]int64{{3, 100}, {11, 101}}, pair,
		"first=%d latest=%d is a pair that never coexisted", first, latest)
}
