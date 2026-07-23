/* Run: from a gno checkout:
gh pr checkout 5979 -R gnolang/gno && git checkout 9641e572a
curl -fsSL -o tm2/pkg/bptree/discover_versions_equivalence_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5979-bptree-two-seek-version-discovery/1-9641e572a/tests/discover_versions_equivalence_test.go
go test -v -run 'TestEdgeRootVersionMatchesScan' ./tm2/pkg/bptree/
rm tm2/pkg/bptree/discover_versions_equivalence_test.go
*/

// The seek-based discoverVersions must return what the scan it replaced
// returned, on every backend and behind the PrefixDB the rootmulti store wraps
// each sub-store in. Green at 9641e572a on memdb, goleveldb, pebbledb, boltdb.
// Version 0 and versions with the high bit set are excluded: they diverge, and
// neither is reachable.

package bptree

import (
	"encoding/binary"
	"fmt"
	"testing"

	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/boltdb"
	"github.com/gnolang/gno/tm2/pkg/db/goleveldb"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/db/pebbledb"
	"github.com/stretchr/testify/require"
)

// scanRootVersions is the discoverVersions body this PR replaced, verbatim.
func scanRootVersions(db dbm.DB) (int64, int64, error) {
	prefix := []byte{PrefixRoot}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	end[0]++

	itr, err := db.Iterator(prefix, end)
	if err != nil {
		return 0, 0, err
	}
	defer itr.Close()

	first := int64(0)
	latest := int64(0)
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		if len(key) != 9 {
			continue
		}
		v := int64(binary.BigEndian.Uint64(key[1:]))
		if first == 0 || v < first {
			first = v
		}
		if v > latest {
			latest = v
		}
	}
	if err := itr.Error(); err != nil {
		return 0, 0, err
	}
	return first, latest, nil
}

func edgeTestRootKey(v uint64) []byte {
	k := make([]byte, 9)
	k[0] = PrefixRoot
	binary.BigEndian.PutUint64(k[1:], v)
	return k
}

func edgeScenarios() []struct {
	name string
	keys [][]byte
} {
	shortR := []byte{PrefixRoot, 0, 0, 0, 0, 0, 0, 1}              // 8 bytes
	longR := []byte{PrefixRoot, 0, 0, 0, 0, 0, 0, 0, 0, 7}         // 10 bytes
	bareR := []byte{PrefixRoot}                                    // 1 byte
	lowNeighbour := []byte{PrefixRoot - 1, 0xff, 0xff, 0xff, 0xff} // 'Q'
	highNeighbour := []byte{PrefixRoot + 1, 0, 0, 0, 0}            // 'S'

	rk := edgeTestRootKey
	return []struct {
		name string
		keys [][]byte
	}{
		{"empty", nil},
		{"single", [][]byte{rk(1)}},
		{"dense", [][]byte{rk(1), rk(2), rk(3), rk(4), rk(5)}},
		{"sparse-gaps", [][]byte{rk(3), rk(7), rk(11), rk(400)}},
		{"pruned-low-end", [][]byte{rk(3), rk(4), rk(5)}},
		{"pruned-high-end", [][]byte{rk(1), rk(2)}},
		{"stray-short-key-at-low-edge", [][]byte{shortR, rk(3), rk(9)}},
		{"stray-long-key-at-high-edge", [][]byte{rk(3), rk(9), longR}},
		{"stray-keys-at-both-edges", [][]byte{shortR, rk(3), rk(9), longR}},
		{"bare-prefix-key", [][]byte{bareR, rk(3), rk(9)}},
		{"only-stray-keys", [][]byte{shortR, longR, bareR}},
		{"neighbour-prefixes", [][]byte{lowNeighbour, highNeighbour, rk(4), rk(6)}},
		{"only-neighbour-prefixes", [][]byte{lowNeighbour, highNeighbour}},
		{"every-other-record-prefix", [][]byte{
			append([]byte{PrefixNode}, make([]byte, NodeKeySize)...),
			append([]byte{PrefixVal}, make([]byte, NodeKeySize)...),
			append([]byte{PrefixMeta}, []byte("fastidx")...),
			append([]byte{PrefixOrphan}, make([]byte, 8)...),
			append([]byte{PrefixFast}, []byte("k")...),
		}},
	}
}

func TestEdgeRootVersionMatchesScan(t *testing.T) {
	backends := []struct {
		name string
		open func(t *testing.T) dbm.DB
	}{
		{"memdb", func(t *testing.T) dbm.DB { return memdb.NewMemDB() }},
		{"goleveldb", func(t *testing.T) dbm.DB {
			d, err := goleveldb.NewGoLevelDB("t", t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { d.Close() })
			return d
		}},
		{"pebbledb", func(t *testing.T) dbm.DB {
			d, err := pebbledb.NewPebbleDB("t", t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { d.Close() })
			return d
		}},
		{"boltdb", func(t *testing.T) dbm.DB {
			d, err := boltdb.New("t", t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { d.Close() })
			return d
		}},
	}

	for _, bf := range backends {
		for _, wrap := range []string{"raw", "prefixdb"} {
			for _, sc := range edgeScenarios() {
				t.Run(fmt.Sprintf("%s/%s/%s", bf.name, wrap, sc.name), func(t *testing.T) {
					base := bf.open(t)
					var db dbm.DB = base
					if wrap == "prefixdb" {
						db = dbm.NewPrefixDB(base, []byte("s/k:main/"))
						// Root keys of neighbouring sub-stores, which the seek
						// must not reach.
						require.NoError(t, base.Set([]byte("s/k:aaa/R\x00\x00\x00\x00\x00\x00\x00\x01"), []byte{1}))
						require.NoError(t, base.Set([]byte("s/k:other/R\xff\xff\xff\xff\xff\xff\xff\xff"), []byte{1}))
					}
					for _, k := range sc.keys {
						require.NoError(t, db.Set(k, []byte{1}))
					}

					wantFirst, wantLatest, err := scanRootVersions(db)
					require.NoError(t, err)

					ndb := &nodeDB{db: db, logger: NewNopLogger()}
					require.NoError(t, ndb.discoverVersions())

					require.Equal(t, wantFirst, ndb.getFirstVersion(), "first version")
					require.Equal(t, wantLatest, ndb.getLatestVersion(), "latest version")
				})
			}
		}
	}
}
