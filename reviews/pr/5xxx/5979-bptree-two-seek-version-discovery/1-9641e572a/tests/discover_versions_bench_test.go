/* Run: from a gno checkout:
gh pr checkout 5979 -R gnolang/gno && git checkout 9641e572a
curl -fsSL -o tm2/pkg/bptree/discover_versions_bench_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5979-bptree-two-seek-version-discovery/1-9641e572a/tests/discover_versions_bench_test.go
go test -run XXX -bench 'BenchmarkImmutableOpen' -benchtime 100x ./tm2/pkg/bptree/
git checkout HEAD~8 -- tm2/pkg/bptree/nodedb.go   # merge-base version, for the A/B
go test -run XXX -bench 'BenchmarkImmutableOpen' -benchtime 100x ./tm2/pkg/bptree/
git checkout 9641e572a -- tm2/pkg/bptree/nodedb.go
rm tm2/pkg/bptree/discover_versions_bench_test.go
*/

// Both benchmarks time the per-query tree open rootmulti performs for every
// custom ABCI query. Swapping nodedb.go between the merge-base and 9641e572a
// isolates the change, since that file's only delta is discoverVersions.
// Pebble gets faster with the retained-version count; memdb gets ~2x slower,
// because its Iterator and ReverseIterator each walk the whole map.

package bptree

import (
	"encoding/binary"
	"fmt"
	"testing"

	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/db/pebbledb"
)

func benchRootKey(v uint64) []byte {
	k := make([]byte, 9)
	k[0] = PrefixRoot
	binary.BigEndian.PutUint64(k[1:], v)
	return k
}

// buildReviewOpenTree writes versions x perVersion entries and returns the tree.
func buildReviewOpenTree(tb testing.TB, d dbm.DB, versions, perVersion int) *MutableTree {
	tb.Helper()
	tree := NewMutableTreeWithDB(d, 10000, NewNopLogger())
	key := make([]byte, 8)
	n := 0
	for range versions {
		for range perVersion {
			binary.BigEndian.PutUint64(key, uint64(n))
			n++
			if _, err := tree.Set(append([]byte(nil), key...), make([]byte, 64)); err != nil {
				tb.Fatal(err)
			}
		}
		if _, _, err := tree.SaveVersion(); err != nil {
			tb.Fatal(err)
		}
	}
	return tree
}

// BenchmarkImmutableOpenPebble pads the root keyspace up to gno.land's default
// retention, reusing the real root record so the load still resolves a live
// root node.
func BenchmarkImmutableOpenPebble(b *testing.B) {
	for _, retained := range []int{1_000, 100_000, 705_600} {
		b.Run(fmt.Sprintf("retained=%d", retained), func(b *testing.B) {
			d, err := pebbledb.NewPebbleDB("bench", b.TempDir())
			if err != nil {
				b.Fatal(err)
			}
			defer d.Close()
			buildReviewOpenTree(b, d, 20, 100)

			ref, err := d.Get(benchRootKey(20))
			if err != nil || ref == nil {
				b.Fatalf("no root record at version 20: %v", err)
			}
			batch := d.NewBatch()
			for v := 21; v <= retained; v++ {
				if err := batch.Set(benchRootKey(uint64(v)), ref); err != nil {
					b.Fatal(err)
				}
				if v%10000 == 0 {
					if err := batch.Write(); err != nil {
						b.Fatal(err)
					}
					batch.Close()
					batch = d.NewBatch()
				}
			}
			if err := batch.Write(); err != nil {
				b.Fatal(err)
			}
			batch.Close()

			probe, err := NewMutableTreeWithDB(d, 10000, NewNopLogger()).Load()
			if err != nil {
				b.Fatal(err)
			}
			if probe != int64(retained) {
				b.Fatalf("Load returned %d, want %d", probe, retained)
			}

			b.ResetTimer()
			for b.Loop() {
				if _, err := NewMutableTreeWithDB(d, 10000, NewNopLogger()).Load(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkImmutableOpenMem runs the same open against a memdb snapshot, the
// backend the in-memory node and the integration harness use.
func BenchmarkImmutableOpenMem(b *testing.B) {
	for _, perVersion := range []int{100, 500} {
		b.Run(fmt.Sprintf("keys=%d", perVersion*200), func(b *testing.B) {
			d := memdb.NewMemDB()
			buildReviewOpenTree(b, d, 200, perVersion)
			snap, err := d.NewSnapshot()
			if err != nil {
				b.Fatal(err)
			}
			sdb := dbm.NewSnapshotDB(snap)

			b.ResetTimer()
			for b.Loop() {
				if _, err := NewMutableTreeWithDB(sdb, 10000, NewNopLogger()).Load(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDiscoverMem isolates discoverVersions on memdb, with the root count
// held fixed and the rest of the store varied, so the cost is visibly driven by
// total key count rather than by the retained-version count.
func BenchmarkDiscoverMem(b *testing.B) {
	for _, other := range []int{50_000, 200_000} {
		b.Run(fmt.Sprintf("roots=1000/other=%d", other), func(b *testing.B) {
			d := memdb.NewMemDB()
			pdb := dbm.NewPrefixDB(d, []byte("s/k:main/"))
			val := make([]byte, NodeKeySize+HashSize)
			for v := 1; v <= 1000; v++ {
				if err := pdb.Set(benchRootKey(uint64(v)), val); err != nil {
					b.Fatal(err)
				}
			}
			nk := make([]byte, 1+NodeKeySize)
			nk[0] = PrefixNode
			big := make([]byte, 200)
			for i := range other {
				binary.BigEndian.PutUint64(nk[1:], uint64(i))
				if err := pdb.Set(append([]byte(nil), nk...), big); err != nil {
					b.Fatal(err)
				}
			}
			ndb := &nodeDB{db: pdb, logger: NewNopLogger()}
			b.ResetTimer()
			for b.Loop() {
				if err := ndb.discoverVersions(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
