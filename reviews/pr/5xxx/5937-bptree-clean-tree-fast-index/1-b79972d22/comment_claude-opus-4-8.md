# Review: PR [#5937](https://github.com/gnolang/gno/pull/5937)
Event: REQUEST_CHANGES

## Body
Most findings here land in the mount and its query path rather than the index gate, which is exact in the safe direction: every published mutation replaces the root with a clone, so a Set writing an identical value still closes the gate and costs only a wasted walk.

Verified on b79972d22: deleting the index stamp by hand and issuing one query-height open restored the stamp and rewrote a doctored index entry, so the read-only query path writes live state. No committed test covers that direction.

One finding has no line in this diff to sit on. [`ImmutableDB.NewBatch`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/db/immutable.go#L58-L65) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/db/immutable.go#L58-L65) returns a bare `nil` where every other mutating method on the type panics with a message naming the misuse, so a caller that batches against it nil-derefs somewhere unrelated instead of failing at its cause.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5937-bptree-clean-tree-fast-index/1-b79972d22/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoland/app.go:106 [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/gnoland/app.go#L106)
This mount passes an explicit db, and [`constructStore` prefers `params.db`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/rootmulti/store.go#L378-L382) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/rootmulti/store.go#L378-L382) over the [`ImmutableDB`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/db/immutable.go#L11) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/db/immutable.go#L11) wrapper that [`MultiImmutableCacheWrapWithVersion`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/rootmulti/store.go#L250-L257) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/rootmulti/store.go#L250-L257) installs, so the query-height view of this store gets the raw writable database. [`ensureFastIndex`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/fast_index.go#L212-L224) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/fast_index.go#L212-L224) then runs on that open with a live batch and ignores `opts.Immutable`, so both fail-safes are inert and a stamp desync turns a query into a full index rebuild that writes live state while holding the ABCI mutex.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5937 -R gnolang/gno

cat > tm2/pkg/store/rootmulti/zz_immutwrite_test.go <<'EOF'
package rootmulti

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestImmutableQueryOpenWrites(t *testing.T) {
	db := memdb.NewMemDB()
	key := types.NewStoreKey("main")
	ms := NewMultiStore(db)
	ms.MountStoreWithDB(key, storebptree.FastStoreConstructor, db) // as app.go does
	if err := ms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	st := ms.GetCommitStore(key)
	st.Set(nil, []byte("k"), []byte("v1"))
	cid := ms.Commit()

	var stampKey, fKey []byte
	itr, _ := db.Iterator(nil, nil)
	for ; itr.Valid(); itr.Next() {
		k := string(itr.Key())
		if len(k) > 7 && k[len(k)-7:] == "fastidx" {
			stampKey = append([]byte{}, itr.Key()...)
		}
		if len(k) >= 5 && k[:4] == "s/_/" && k[4] == 'F' {
			fKey = append([]byte{}, itr.Key()...)
		}
	}
	itr.Close()

	// The documented operator escape hatch: delete the stamp by hand.
	db.Delete(stampKey)
	db.Set(fKey, []byte("junk")) // doctor the entry so a rebuild is observable

	ms.MultiImmutableCacheWrapWithVersion(cid.Version) // supposed to be read-only

	stampAfter, _ := db.Get(stampKey)
	fAfter, _ := db.Get(fKey)
	t.Logf("stamp restored=%v  F entry=%q", stampAfter != nil, fAfter)
	if stampAfter != nil || string(fAfter) != "junk" {
		t.Fatal("the read-only query-height open WROTE to the live DB")
	}
}
EOF
go test -run TestImmutableQueryOpenWrites -v ./tm2/pkg/store/rootmulti/
rm tm2/pkg/store/rootmulti/zz_immutwrite_test.go
```

```
=== RUN   TestImmutableQueryOpenWrites
    zz_immutwrite_test.go:44: stamp restored=true  F entry="\x00\x00\x00\x00\x00\x00\x00\x01v1R\x82-\x12"
    zz_immutwrite_test.go:46: the read-only query-height open WROTE to the live DB
--- FAIL: TestImmutableQueryOpenWrites (0.00s)
```
</details>

## tm2/pkg/store/bptree/store.go:32-41 [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/bptree/store.go#L32)
Mounting this constructor also puts a full scan of every retained root record on the production query path: [`Store.LoadVersion`'s immutable branch](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/bptree/store.go#L188-L199) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/bptree/store.go#L188-L199) calls [`Load`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/mutable_tree.go#L488) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/mutable_tree.go#L488), which runs [`discoverVersions`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/nodedb.go#L473-L499) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/nodedb.go#L473-L499), and discards the result. At [gno.land's `PruneSyncable` default of 705,600 retained versions](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/types/options.go#L42) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/types/options.go#L42) that extrapolates to ~220 ms per ABCI query, held under the mutex [every ABCI connection shares](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bft/proxy/client.go#L26-L34) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bft/proxy/client.go#L26-L34) and unbounded on archive nodes, where [the IAVL mount this replaces](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/iavl/store.go#L175-L182) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/iavl/store.go#L175-L182) went straight to `GetImmutable(ver)`. Nothing on this path reads the discovered counters: [`getImmutable`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/mutable_tree.go#L632-L637) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/mutable_tree.go#L632-L637) looks the root up by key and the adapter's [`VersionExists`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/bptree/tree.go#L69-L71) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/bptree/tree.go#L69-L71) compares against the snapshot's own version.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5937 -R gnolang/gno

cat > tm2/pkg/store/bptree/zz_queryscan_test.go <<'EOF'
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
EOF
go test -run TestQueryOpenScanPebble -v -timeout 3000s ./tm2/pkg/store/bptree/
rm tm2/pkg/store/bptree/zz_queryscan_test.go
```

```
=== RUN   TestQueryOpenScanPebble
    zz_queryscan_test.go:46: retained versions= 50000  per query-path store open: 15.095782ms
    zz_queryscan_test.go:46: retained versions=100000  per query-path store open: 29.788198ms
    zz_queryscan_test.go:46: retained versions=200000  per query-path store open: 61.974838ms
    zz_queryscan_test.go:46: retained versions=400000  per query-path store open: 124.275046ms
--- PASS: TestQueryOpenScanPebble (52.75s)
```
</details>

## gno.land/pkg/sdk/vm/params.go:40 [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/sdk/vm/params.go#L40)
The index never answers absence, so an absent-key GET pays the index probe and then the full descent while being charged this same 1.0: [`effectiveGetReadDepth100`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/cache/store.go#L91-L99) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/cache/store.go#L91-L99) returns the Fixed pin unconditionally and [`cacheStore.Get`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/cache/store.go#L139-L151) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/cache/store.go#L139-L151) charges it before the fetch. The pre-mount pin charged 3.0 for that same walk, so the gap between charged and real widens 3x, and [`cacheStore.Has`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/cache/store.go#L208-L211) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/cache/store.go#L208-L211) inherits it as `Get(key) != nil`, making an existence check on a missing key the cheapest read primitive.

## contribs/gnogenesis/internal/fork/generate.go:676 [↗](../../../../../.worktrees/gno-review-5937/contribs/gnogenesis/internal/fork/generate.go#L676)
Missing test: nothing fails when `vm.DefaultParams()` moves but no new era is appended here, so the next defaults change silently stops repricing forked chains. The rule lives only as [a comment on the defaults](https://github.com/gnolang/gno/blob/b79972d22/gno.land/pkg/sdk/vm/params.go#L36-L38) · [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/sdk/vm/params.go#L36-L38).

<details><summary>test cases</summary>

```go
// TestUntunedFingerprintsCoverCurrentDefaults fails when the vm depth defaults
// change without a matching era appended to untunedDepthFingerprints. A source
// genesis carrying today's defaults is untuned by definition, so no fingerprint
// may equal the live defaults.
func TestUntunedFingerprintsCoverCurrentDefaults(t *testing.T) {
	t.Parallel()

	defaults := vm.DefaultParams()
	for i, fp := range untunedDepthFingerprints {
		require.False(t, depthParamsMatch(defaults, fp),
			"vm.DefaultParams() equals untuned era fingerprint %d; when changing the "+
				"defaults, append a new era carrying the previous values", i)
	}
}
```
</details>

## gnovm/pkg/gnolang/files_test.go:119 [↗](../../../../../.worktrees/gno-review-5937/gnovm/pkg/gnolang/files_test.go#L119)
Nit: `subTestName` is the walk-relative path, so this prefix match only covers allocator goldens sitting at the top level of `gnovm/tests/files/`. One added under a subdirectory would rejoin the pool and flake for the reason this condition exists. The `MAXALLOC:` directive these tests already carry identifies them without depending on the filename.
