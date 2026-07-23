# Review: PR [#5979](https://github.com/gnolang/gno/pull/5979)
Event: COMMENT

## Body
Timed the per-query tree open on 9641e572a with only `nodedb.go` swapped against the merge-base: at the default 705,600 retained versions pebbledb goes from 181.52 ms to 54.1 µs.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5979-bptree-two-seek-version-discovery/1-9641e572a/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bptree/nodedb.go:472-487 [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L472-L487)
memdb has no seek: [`Iterator` and `ReverseIterator` both walk the whole map](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/db/memdb/mem_db.go#L197-L215) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/db/memdb/mem_db.go#L197-L215), so two opens cost two full-map materialisations where the scan cost one. The per-query tree open on the in-memory node [gnodev boots](https://github.com/gnolang/gno/blob/9641e572a/contribs/gnodev/pkg/dev/node.go#L649) · [↗](../../../../../.worktrees/gno-review-5979/contribs/gnodev/pkg/dev/node.go#L649) goes from 0.81 ms to 1.68 ms on a 20,000-key tree, and the cost tracks the whole store rather than retention depth. [`Load` discovers versions](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/mutable_tree.go#L488) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/mutable_tree.go#L488) and then [`LoadVersion` discovers them again](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/mutable_tree.go#L527) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/mutable_tree.go#L527), so paying for discovery once brings that open back to 0.78 ms.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5979 -R gnolang/gno
cat > tm2/pkg/bptree/zz_open_bench_test.go <<'EOF'
package bptree

import (
	"encoding/binary"
	"testing"

	dbm "github.com/gnolang/gno/tm2/pkg/db"
	"github.com/gnolang/gno/tm2/pkg/db/memdb"
)

func BenchmarkQueryOpen(b *testing.B) {
	d := memdb.NewMemDB()
	tree := NewMutableTreeWithDB(d, 10000, NewNopLogger())
	key := make([]byte, 8)
	n := 0
	for range 200 {
		for range 100 {
			binary.BigEndian.PutUint64(key, uint64(n))
			n++
			if _, err := tree.Set(append([]byte(nil), key...), make([]byte, 64)); err != nil {
				b.Fatal(err)
			}
		}
		if _, _, err := tree.SaveVersion(); err != nil {
			b.Fatal(err)
		}
	}
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
}
EOF
echo "== this head"
go test -run XXX -bench BenchmarkQueryOpen -benchtime 200x -count=3 ./tm2/pkg/bptree/ | grep QueryOpen
git checkout "$(git merge-base origin/master HEAD)" -- tm2/pkg/bptree/nodedb.go
echo "== merge-base discoverVersions"
go test -run XXX -bench BenchmarkQueryOpen -benchtime 200x -count=3 ./tm2/pkg/bptree/ | grep QueryOpen
git checkout HEAD -- tm2/pkg/bptree/nodedb.go
rm tm2/pkg/bptree/zz_open_bench_test.go
```

```
== this head
BenchmarkQueryOpen-16    	     200	   1725706 ns/op
BenchmarkQueryOpen-16    	     200	   1699594 ns/op
BenchmarkQueryOpen-16    	     200	   1606367 ns/op
== merge-base discoverVersions
BenchmarkQueryOpen-16    	     200	    820647 ns/op
BenchmarkQueryOpen-16    	     200	    806226 ns/op
BenchmarkQueryOpen-16    	     200	    806105 ns/op
```
</details>

## tm2/pkg/bptree/nodedb.go:516-522 [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L516-L522)
Missing test: a root-prefixed key that is not 9 bytes long, sitting at either edge of the range. Nothing exercises the skip, so the property that a stray key cannot stop discovery is asserted nowhere, and the coverage block at line 518 stays at zero across the package suite.

<details><summary>test cases</summary>

```go
func TestEdgeRootVersionSkipsStrayKeys(t *testing.T) {
	t.Parallel()

	rk := func(v uint64) []byte {
		k := make([]byte, 9)
		k[0] = PrefixRoot
		binary.BigEndian.PutUint64(k[1:], v)
		return k
	}
	for name, strays := range map[string][][]byte{
		"short key below the first root": {{PrefixRoot, 0, 0, 0, 0, 0, 0, 1}},
		"long key above the last root":   {{PrefixRoot, 0, 0, 0, 0, 0, 0, 0, 0, 7}},
		"bare prefix key":                {{PrefixRoot}},
		"strays at both edges": {
			{PrefixRoot, 0, 0, 0, 0, 0, 0, 1},
			{PrefixRoot, 0, 0, 0, 0, 0, 0, 0, 0, 7},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := memdb.NewMemDB()
			for _, v := range []uint64{3, 4, 9} {
				require.NoError(t, db.Set(rk(v), []byte{1}))
			}
			for _, k := range strays {
				require.NoError(t, db.Set(k, []byte{1}))
			}

			ndb := &nodeDB{db: db, logger: NewNopLogger()}
			require.NoError(t, ndb.discoverVersions())
			require.Equal(t, int64(3), ndb.getFirstVersion())
			require.Equal(t, int64(9), ndb.getLatestVersion())
		})
	}
}
```
</details>

## tm2/pkg/bptree/nodedb.go:473-477 [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L473-L477)
Suggestion: the two ends come from two iterator opens, so on goleveldb and boltdb, which [report no snapshot support](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/db/goleveldb/go_level_db.go#L140-L142) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/db/goleveldb/go_level_db.go#L140-L142) and so leave the immutable query store [reading the live DB](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/store/rootmulti/store.go#L375-L377) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/store/rootmulti/store.go#L375-L377), a commit and a prune landing between the two opens return a first and a latest that never coexisted. Confirmed behaviourally with roots 3 to 100 and a prune of 3 to 10 plus a commit of 101 in between: this reports first=3 latest=101 where the single scan reported first=3 latest=100. Nothing reads the stale end today, so it is latent.
