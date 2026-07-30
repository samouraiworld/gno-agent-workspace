# Review: PR [#6018](https://github.com/gnolang/gno/pull/6018)
Event: COMMENT

## Body
Ran all five new query/commit race tests under `-race`, which CI does not: no races on 5ceafd2c5, and replacing the atomic publication of `lastCommitID` with a plain field makes the full-app test report one.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6018-fastindex-snapshot-isolation/1-5ceafd2c5/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## tm2/pkg/bptree/fast_index.go:250-253 [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/fast_index.go#L250-L253)
An operator cannot act on this message, which now blocks boot: it says to delete `PrefixMeta"fastidx"`, and `PrefixMeta` is [a byte constant](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/const.go#L33) whose value the message never prints. The key to delete is `s/_/Mfastidx`, since the real key also carries the store's `s/_/` prefix.

## tm2/pkg/store/rootmulti/store.go:469-481 [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L469-L481)
The snapshot changes once per block, but every `.store` query rebuilds the immutable multistore. Each mount's [`LoadReadonly`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/mutable_tree.go#L519) then runs [`discoverVersions`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/nodedb.go#L473-L509), a full iterator scan of the `R` keyspace that dominates at gno.land's [default retention of 705,600 versions](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/types/options.go#L42). The query connection serializes on [one mutex](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bft/proxy/client.go#L26), so the cost caps throughput on every query path, not just `.store`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6018 -R gnolang/gno
cat > tm2/pkg/store/rootmulti/zz_cost_test.go <<'EOF'
package rootmulti_test

import (
	"fmt"
	"testing"
	"time"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	dbm "github.com/gnolang/gno/tm2/pkg/db"
	_ "github.com/gnolang/gno/tm2/pkg/db/pebbledb"
	storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"
	"github.com/gnolang/gno/tm2/pkg/store/rootmulti"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestStoreQueryCost(t *testing.T) {
	for _, versions := range []int{1000, 20000} {
		t.Run(fmt.Sprintf("versions=%d", versions), func(t *testing.T) {
			db, err := dbm.NewDB("gnolang", dbm.PebbleDBBackend, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			mainKey := types.NewStoreKey("main")
			ms := rootmulti.NewMultiStore(db)
			ms.SetStoreOptions(types.StoreOptions{PruningOptions: types.NewPruningOptions(0, 1)})
			ms.MountStoreWithDB(mainKey, storebptree.FastStoreConstructor, db)
			if err := ms.LoadLatestVersion(); err != nil {
				t.Fatal(err)
			}
			defer ms.Close()

			key := []byte("key0000")
			for blk := 1; blk <= versions; blk++ {
				cms := ms.MultiCacheWrap()
				cms.GetStore(mainKey).Set(nil, key, fmt.Appendf(nil, "v%d", blk))
				cms.MultiWrite()
				ms.Commit()
			}

			req := abci.RequestQuery{Path: "/main/key", Data: key, Height: int64(versions)}
			const iters = 20

			start := time.Now()
			for range iters {
				if res := ms.Query(req); len(res.Value) == 0 {
					t.Fatalf("live query empty: %+v", res)
				}
			}
			live := time.Since(start) / iters

			start = time.Now()
			for range iters {
				res, err := ms.QueryImmutable(req)
				if err != nil || len(res.Value) == 0 {
					t.Fatalf("snapshot query: %v %+v", err, res)
				}
			}
			t.Logf("RESULT versions=%d live=%v snapshot=%v", versions, live, time.Since(start)/iters)
		})
	}
}
EOF
go test -run 'TestStoreQueryCost' -v -timeout 30m ./tm2/pkg/store/rootmulti/
rm tm2/pkg/store/rootmulti/zz_cost_test.go
```

```
=== RUN   TestStoreQueryCost/versions=1000
    zz_cost_test.go:63: RESULT versions=1000 live=7.359µs snapshot=501.681µs
=== RUN   TestStoreQueryCost/versions=20000
    zz_cost_test.go:63: RESULT versions=20000 live=94.683µs snapshot=25.344825ms
--- PASS: TestStoreQueryCost (…)
```

At 60,000 retained versions the same measurement gives 19.1 µs against 57.6 ms.
</details>

## tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go:124 [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L124)
Missing test: a query during the fast-index upgrade restart, with the rebuild still undrained in the collector and the query snapshot on the pre-rebuild stamp. The fuzz reaches that window here and only checks that a snapshot exists, so nothing asserts a query in it returns the authoritative value. That window is also the only place [`loadImmutableView`'s `LoadReadonly`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/bptree/store.go#L171-L172) is load-bearing against a panic rather than against poisoning.

<details><summary>test cases</summary>

Passes on 5ceafd2c5. Dropping the `getImmutable` stamp gate makes it return `a1`; replacing `LoadReadonly` with `Load` in `loadImmutableView` makes it panic with `readonlyNoopBatch: unexpected Write on read-only DB`.

```go
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
```
</details>

## gno.land/pkg/gnoland/app_query_race_test.go:281-282 [↗](../../../../../.worktrees/gno-review-6018/gno.land/pkg/gnoland/app_query_race_test.go#L281-L282)
Nit: this assertion does not cover `LoadReadonly`: replacing [`st.mtree.LoadReadonly()`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/bptree/store.go#L172) with `st.mtree.Load()` leaves the test green, because the restored maintenance reads the stamp through the snapshot rather than the live handle. It catches immutable stores reading the live DB instead, and disabling [the `constructStore` reroute](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L610-L612) does fail here.

## tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go:128-133 [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L128-L133)
Nit: each restart reassigns `ms` without closing the previous multistore, so the query snapshot every load seeds is never released. On memdb it costs only [a map clone](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/memdb/mem_db.go#L222-L227), while the same shape on pebble needed the companion fixes in [`app_test.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/gno.land/pkg/gnoland/app_test.go#L1760-L1764) and [`gnogenesis`](https://github.com/gnolang/gno/blob/5ceafd2c5/contribs/gnogenesis/internal/fork/source_txs_data_dir.go#L125-L131).

## tm2/pkg/sdk/baseapp.go:513-514 [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/sdk/baseapp.go#L513-L514)
Suggestion: a `.store` query that cannot use the snapshot path falls back to the live DB with only a Debug line, and [`refreshQuerySnapshot`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L331-L344) drops a [`NewSnapshot`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/types.go#L60) error entirely. A node can then serve every query without snapshot isolation and log nothing at the default level.

## tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go:39-49 [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L39-L49)
Suggestion: this loop accounts for 203 s of the 305 s [`tm2/pkg/store/rootmulti`](https://github.com/gnolang/gno/tree/5ceafd2c5/tm2/pkg/store/rootmulti) now takes, up from 0.17 s at the merge base. Its own [header](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L11-L17) says it guards batch semantics, snapshot seeding and index parity rather than the race, and the shape varies with the three modes and the restart schedule, not with the thirtieth seed.
