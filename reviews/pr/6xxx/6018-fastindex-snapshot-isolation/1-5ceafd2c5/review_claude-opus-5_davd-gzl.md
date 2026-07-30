# PR [#6018](https://github.com/gnolang/gno/pull/6018): fix(tm2): snapshot-isolate query paths and write-proof bptree fast-index maintenance

URL: https://github.com/gnolang/gno/pull/6018
Author: jaekwon | Base: master | Files: 26 | +2200 -155
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 5ceafd2c5 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-6018 5ceafd2c5`

**TL;DR:** A testnet node computed a different app hash than the rest of the chain because a lookup cache for account values held a balance from 3,000 blocks earlier. This PR traces that to read-only query traffic being allowed to rewrite the cache from an out-of-date view of the database, and closes the hole in four independent places so query traffic can no longer write to the store at all.

**Verdict: NEEDS DISCUSSION** — the fix is correct and each layer has a working regression guard, but two items need an author decision before merge: `.store` queries now rebuild a whole store per request, and the boot-refusal this PR introduces prints a recovery instruction an operator cannot follow (2 Warnings, 1 Missing test, 4 Nits, 2 Suggestions).

## Verify first

- [`tm2/pkg/store/rootmulti/store.go:610-612`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L610-L612) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L610-L612) — the whole fix rests on this three-line reroute. Confirm it is the only thing standing between query views and the live DB: replace `ms.db` with `raw` and run `go test ./gno.land/pkg/gnoland/ -run TestQueryRace_FastIndexParity` — it must fail on the `hooked.fired` assertion.
- [`tm2/pkg/bptree/fast_index.go:249-253`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/fast_index.go#L249-L253) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/fast_index.go#L249-L253) — this turns a stamp ahead of the loaded version into a refusal to start. Confirm no in-repo startup path can reach it benignly: the stamp is written by [`SaveVersion`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/mutable_tree.go#L415) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/mutable_tree.go#L415) in the same batch as the version's records, so equality at rest is the only in-contract state.
- [`tm2/pkg/store/rootmulti/store.go:145-146`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L145-L146) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L145-L146) — a new startup panic on any mount whose DB is not the root DB. Confirm every mount site passes `nil` or the multistore's own DB: `grep -rn '\.MountStoreWithDB(' --include='*.go'` lists 40 call sites, and the only non-test ones outside gnolang/gno's own app are the two in [`gnogenesis`](https://github.com/gnolang/gno/blob/5ceafd2c5/contribs/gnogenesis/internal/fork/source_txs_data_dir.go#L101-L102) · [↗](../../../../../.worktrees/gno-review-6018/contribs/gnogenesis/internal/fork/source_txs_data_dir.go#L101-L102), which pass `s.appDB`.

## Summary

A topaz-1 node rejected the proposal for block 227783 with an app-hash mismatch. Its persisted bptree fast-index entry for one account held the block-224859 value while the authoritative tree carried the block-227773 update, under a stamp claiming currency at 227782, and [`MutableTree.Get`'s clean-working-tree fast path](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/mutable_tree.go#L207-L211) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/mutable_tree.go#L207-L211) served the stale value during block execution. The cause is four gaps that compose: dedicated-db mounts were routed to the live DB even for the immutable query multistore, the query ABCI connection runs on its own mutex, immutable query stores held a real writable batch, and `Load()` read "latest" through an iterator and the stamp through a later `Get`, so a commit landing between the two made a query conclude the index was stale and rebuild the entire live index from the previous block's root. The fix stacks four independent stops: read-only loads that skip index maintenance, a fail-loud guard when the stamp is ahead, immutable multistores reading the frozen post-commit snapshot, and `.store` queries served from that snapshot.

Reading order: [`tm2/pkg/db/collecting.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/collecting.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/collecting.go) (batch semantics everything else assumes), then [`tm2/pkg/db/immutable.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/immutable.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/immutable.go) and [`snapshot_db.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/snapshot_db.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/snapshot_db.go), then [`tm2/pkg/bptree/mutable_tree.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/mutable_tree.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/mutable_tree.go) and [`fast_index.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/fast_index.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/fast_index.go), then [`tm2/pkg/store/bptree/store.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/bptree/store.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/bptree/store.go), then [`tm2/pkg/store/rootmulti/store.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go), then [`tm2/pkg/sdk/baseapp.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/sdk/baseapp.go) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/sdk/baseapp.go), then the two lifecycle callers and the tests.

## Diagram

```
BEFORE — one live DB handle reachable from both connections
  consensus conn ──► multiStore.Commit ──► CollectingDB ──► cfg.DB (live)
                                                              ▲   ▲
  query conn ──► MultiImmutableCacheWrapWithVersion ──────────┘   │
                   constructStore: params.db != nil → params.db ───┘
                   bptree Store.LoadVersion → Load() → ensureFastIndex()  ← WRITES

AFTER — query conn cannot reach the live handle
  consensus conn ──► multiStore.Commit ──► CollectingDB ──► cfg.DB (live)
                          └─ refreshQuerySnapshot ──► SnapshotDB (frozen at N)
                                                          ▲
  query conn ──► immutableAtVersion ──► constructStore: Immutable → ms.db
                   bptree Store.LoadVersion → LoadReadonly()   (no maintenance)
                   getImmutable → fast reads only if stamp >= version
  .store query ──► QueryImmutable ─┘   (error → legacy live path)
```

## Fix

Query-path store loads split away from the maintenance-capable one: [`MutableTree.LoadReadonly`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/mutable_tree.go#L519-L528) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/mutable_tree.go#L519-L528) does discovery and a version load with no `ensureFastIndex`, and [`Store.loadImmutableView`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/bptree/store.go#L171-L183) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/bptree/store.go#L171-L183) is the only entry point immutable stores use. `constructStore` then reroutes immutable multistores off `params.db` onto the frozen snapshot in `ms.db`, which is only sound because a dedicated mount DB must now be the root DB, enforced at mount time. `ensureFastIndex` keeps rebuilding on a missing or older stamp but [errors on a newer one](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/fast_index.go#L249-L253) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/fast_index.go#L249-L253), and `getImmutable` gates fast reads on the stamp covering the snapshot's version so a read-only load that cannot refresh the stamp still cannot trust a too-old index. The load-bearing constraint throughout is that a read path must never produce a write: the batch handed to an immutable store is a [no-op batch whose `Write` panics](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/snapshot_db.go#L47-L49) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/snapshot_db.go#L47-L49), so any surviving maintenance path fails loudly rather than silently poisoning.

## Benchmarks / Numbers

`.store/main/key` at a fixed height, 20 iterations, pebble, one key rewritten per block, `PruningOptions(0, 1)`:

| Retained versions | Legacy live path | Snapshot path (this PR) | Ratio |
|---|---:|---:|---:|
| 1,000 | 7.4 µs | 0.50 ms | 68x |
| 5,000 | 16.3 µs | 8.27 ms | 507x |
| 20,000 | 94.7 µs | 25.3 ms | 268x |
| 60,000 | 19.1 µs | 57.6 ms | 3013x |

The snapshot column is linear in retained versions past 5,000 (about 1 µs per retained root); the live column is flat and dominated by pebble read amplification, hence its noise.

Package test time, `tm2/pkg/store/rootmulti`:

| | Time |
|---|---:|
| merge base d1a33f574 | 0.17 s |
| 5ceafd2c5 | 305.6 s |
| of which `TestFastIndex_RootmultiPipelineFuzz` | 203.5 s |
| of which `TestFastIndex_NaturalQueryCommitRace` | 78.8 s |
| of which `TestFastIndex_ConcurrentQueryCommit` | 15.2 s |
| of which `TestFastIndex_RebuildOverCollectingDB` | 7.4 s |

## Critical (must fix)

None.

## Warnings (should fix)

- **[recovery instruction cannot be followed]** `tm2/pkg/bptree/fast_index.go:250-253` — the only documented recovery for a node this change refuses to start names a Go identifier instead of a key an operator can delete.
  <details><summary>details</summary>

  The [error text](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/fast_index.go#L250-L253) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/fast_index.go#L250-L253) renders as `delete the fast-index stamp (PrefixMeta"fastidx")`. `PrefixMeta` is [an unexplained byte constant](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/const.go#L33) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/const.go#L33) whose value the message does not print, and the real key also carries the store's `s/_/` prefix, which the message omits entirely. This error is reached from `Load` through [`Store.LoadLatestVersion`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/bptree/store.go#L187-L197) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/bptree/store.go#L187-L197) up to `baseApp.LoadLatestVersion`, so it is a node that will not boot, and the operator's only path forward is a resync unless they read bptree's source. Fix: print the literal key bytes the operator must delete, `s/_/Mfastidx`, rather than the constant name.
  </details>

- **[public query path becomes O(retained versions)]** `tm2/pkg/sdk/baseapp.go:507-514` — every `.store` query now builds a fresh immutable multistore, which scans every retained root; measured 268x to 3013x slower than the path it replaces.
  <details><summary>details</summary>

  [`handleQueryStore`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/sdk/baseapp.go#L507-L514) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/sdk/baseapp.go#L507-L514) prefers [`QueryImmutable`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L469-L481) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L469-L481), which calls `immutableAtVersion` → `ims.LoadVersion` → one `constructStore` and one `LoadReadonly` per mount. `LoadReadonly` runs [`discoverVersions`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/nodedb.go#L473-L509) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/nodedb.go#L473-L509), a full iterator scan of the `R` keyspace, so the cost is linear in retained versions; gno.land's default retention is [705,600](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/types/options.go#L42) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/types/options.go#L42). Before this PR the same query resolved through the live tree in microseconds; see the Benchmarks table for the measured points. The query ABCI connection serializes on [one mutex](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bft/proxy/client.go#L26) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bft/proxy/client.go#L26), so this cost caps total query throughput for every path, not just `.store`. The snapshot changes once per block while the store is rebuilt once per request, so a single cached immutable multistore per snapshot generation would remove the term for the latest-height case that dominates real traffic. Fix: decide whether `.store` should take this cost now or wait for the O(1) immutable load, and say which in the PR.
  </details>

## Nits

- **[guard names a layer it does not detect]** `gno.land/pkg/gnoland/app_query_race_test.go:281-282` — the assertion message claims to catch query-side index maintenance, but replacing `LoadReadonly` with `Load` leaves this test green.
  <details><summary>details</summary>

  [The assertion](https://github.com/gnolang/gno/blob/5ceafd2c5/gno.land/pkg/gnoland/app_query_race_test.go#L281-L282) · [↗](../../../../../.worktrees/gno-review-6018/gno.land/pkg/gnoland/app_query_race_test.go#L281-L282) fires only on a stamp read through the live handle. With the `constructStore` reroute in place, restoring maintenance on the immutable load reads the stamp through the snapshot, so the hook never fires. I ran both reverts: replacing `st.mtree.LoadReadonly()` with `st.mtree.Load()` at [`store.go:172`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/bptree/store.go#L172) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/bptree/store.go#L172) keeps the test passing in 10.65 s, while disabling the `ms.storeOpts.Immutable` reroute fails it on this exact line. What the test pins is snapshot routing, and `TestFastIndex_NoStaleRebuildOnRacingQueryLoad` is what pins the read-only load. Fix: reword the message to name live-DB routing.
  </details>

- **[test skips the lifecycle rule the PR adds]** `tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go:128-133` — each simulated restart drops the previous multistore without releasing its query snapshot, so the fuzz never exercises the release-before-close rule this PR introduces.
  <details><summary>details</summary>

  [`newMS`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L62-L83) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L62-L83) seeds a snapshot on every load and asserts it is non-nil, and the [three restart sites](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L124-L133) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L124-L133) reassign `ms` without calling `Close`. On memdb a snapshot is [a full map clone](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/memdb/mem_db.go#L222-L227) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/memdb/mem_db.go#L222-L227) and closing is a no-op, so nothing fails, but the same shape on pebble is precisely what forced the two companion fixes in [`app_test.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/gno.land/pkg/gnoland/app_test.go#L1760-L1764) · [↗](../../../../../.worktrees/gno-review-6018/gno.land/pkg/gnoland/app_test.go#L1760-L1764) and [`source_txs_data_dir.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/contribs/gnogenesis/internal/fork/source_txs_data_dir.go#L125-L131) · [↗](../../../../../.worktrees/gno-review-6018/contribs/gnogenesis/internal/fork/source_txs_data_dir.go#L125-L131). Fix: close the outgoing multistore at each restart site.
  </details>

- **[unreachable panic]** `tm2/pkg/store/rootmulti/store.go:296-298` — the metadata-write panic can never fire, since [`batchHandle.Write`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/collecting.go#L244-L249) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/collecting.go#L244-L249) unconditionally returns nil and [`ms.collector.NewBatch`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/collecting.go#L93-L95) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/collecting.go#L93-L95) is the only source of that batch. Harmless as defensive code against a future collector change; no change needed, not posted.

- **[modernizer applied unevenly]** `tm2/pkg/store/rootmulti/fastindex_concurrent_repro_test.go:57-58` — this file keeps [`wg.Add(1)` plus `go func(g int)`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/fastindex_concurrent_repro_test.go#L57-L58) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_concurrent_repro_test.go#L57-L58) while its sibling was rewritten to [`wg.Go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/fastindex_natural_race_test.go#L73) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_natural_race_test.go#L73). No enabled linter covers it ([`.github/golangci.yml`](https://github.com/gnolang/gno/blob/5ceafd2c5/.github/golangci.yml#L13) · [↗](../../../../../.worktrees/gno-review-6018/.github/golangci.yml#L13) runs `default: none`); not posted, no change needed.

## Missing Tests

- **[the reachable trigger is untested]** `tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go:122-126` — no test issues a query during the fast-index upgrade window, the one window where the `getImmutable` stamp gate is load-bearing rather than hardening.
  <details><summary>details</summary>

  The upgrade restart stages the whole rebuild in the shared collector, which stays undrained until the next block commit, while [`LoadVersion` has already seeded the query snapshot](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L241-L243) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L241-L243) from the pre-rebuild disk state. A query in that window therefore reads a snapshot whose stamp is behind its own version and whose `F` entries are stale. The fuzz reaches the window at [line 124](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L124) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L124) and only checks that a snapshot exists; [`TestGetImmutable_StampGate`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/fast_index_test.go#L595-L639) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/fast_index_test.go#L595-L639) covers the gate at the tree level with a hand-doctored entry rather than through a query. The added test at [`tests/query_upgrade_window_test.go`](tests/query_upgrade_window_test.go) closes both directions at the store layer: it passes at 5ceafd2c5, returns the version-1 value when the stamp gate is dropped, and panics with `readonlyNoopBatch: unexpected Write on read-only DB` when `loadImmutableView` calls `Load` instead of `LoadReadonly`. Fix: add it to `tm2/pkg/store/rootmulti/`.
  </details>

## Suggestions

- **[isolation can switch off with no signal]** `tm2/pkg/store/rootmulti/store.go:331-344` — a snapshot-creation failure and a snapshot-query failure are both silent, so a node can serve every query from the live DB with nothing in the logs.
  <details><summary>details</summary>

  [`refreshQuerySnapshot`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L331-L344) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L331-L344) drops the `NewSnapshot` error with `if snap, err := ...; err == nil` and leaves `querySnapshot` at whatever it was, and `immutableAtVersion` then [falls back to `ImmutableDB` over the live DB](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L425-L430) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L425-L430). On the query side, `handleQueryStore` reports the snapshot path's failure at [Debug](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/sdk/baseapp.go#L513-L514) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/sdk/baseapp.go#L513-L514), which production nodes do not enable. Reads stay write-proof either way, so this is not a correctness risk today; the gap is that the property the PR is built to guarantee is unobservable at runtime. A one-shot warn on the first snapshot failure would make it visible without per-query noise.
  </details>

- **[five minutes of new CI time in one package]** `tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go:39` — the package goes from 0.17 s to 305.6 s, two thirds of it in a fuzz whose own header says it does not reproduce the bug.
  <details><summary>details</summary>

  The [30-seed by 3-mode loop](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L39-L49) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/fastindex_pipeline_repro_test.go#L39-L49) is 203.5 s of the 305.6 s, and its header states it guards batch semantics, snapshot seeding and index parity rather than the race itself. Three of the six new tests carry a `-short` skip, so even a short run still pays about 16 s. Cutting the seed count is the cheapest lever: the three modes and the restart schedule are what vary the shape, not the thirtieth seed.
  </details>

## Verified

- Reverting the immutable reroute at [`store.go:610-612`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L610-L612) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L610-L612) makes `TestQueryRace_FastIndexParity` fail on `hooked.fired` in 8.06 s; reverting `LoadReadonly` to `Load` at [`store.go:172`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/bptree/store.go#L172) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/bptree/store.go#L172) leaves it passing in 10.65 s. The full-app guard pins snapshot routing, not the read-only load.
- `LoadReadonly` is load-bearing against a panic, not only against poisoning: with it reverted, a query in the fast-index upgrade window dies in [`readonlyNoopBatch.Write`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/snapshot_db.go#L47-L49) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/snapshot_db.go#L47-L49) through `nodeDB.Commit`, so a restart that enables the index would have made every query panic until the first block landed.
- Dropping the `getImmutable` stamp gate at [`mutable_tree.go:700-703`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/bptree/mutable_tree.go#L700-L703) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/bptree/mutable_tree.go#L700-L703) makes a `.store`-shaped query in that same window return the version-1 value while the tree holds version 2, the exact issue [#6011](https://github.com/gnolang/gno/issues/6011) shape reaching a client.
- Values crossing the snapshot boundary are copies, not views into pebble's arena: [`pebbleSnapshot.Get`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/db/pebbledb/pebbledb.go#L347-L365) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/db/pebbledb/pebbledb.go#L347-L365) copies before closing the pebble closer, so `QueryImmutable`'s `defer release()` cannot hand back freed memory.
- Replacing the `atomic.Pointer` publication of `lastCommitID` at [`store.go:58`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L58) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L58) with a plain field makes `TestQueryRace_FastIndexParity` report a data race under `-race`, so this PR does carry a guard for the race [#6014](https://github.com/gnolang/gno/pull/6014) fixes.
- CI runs no `-race`. All five new query/commit concurrency tests are clean under it at 5ceafd2c5: `TestFastIndex_ConcurrentQueryCommit`, `TestFastIndex_NoStaleRebuildOnRacingQueryLoad` and `TestQueryView_SnapshotIsolationUnderPruning` in 83.4 s, `TestFastIndex_NaturalQueryCommitRace` in 93.3 s, and `TestQueryRace_FastIndexParity` in 30.5 s.
- The one red test in the affected suites, `TestContract_ConcurrentSnapshotReadsVsWriter_NoRace`, is a writer-throughput floor that fails only under external load: three runs pass at 5ceafd2c5 on an idle machine and the full package is green at both 5ceafd2c5 and the merge base d1a33f574.
- Green at 5ceafd2c5: `tm2/pkg/bptree`, `tm2/pkg/bptree/benchmarks`, `tm2/pkg/db`, and every package under `tm2/pkg/store/...`.

## Open questions

- The absolute per-query store-construction cost predates this PR on the custom-query path, which `handleQueryCustom` has always paid through [`MultiImmutableCacheWrapWithVersion`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/sdk/baseapp.go#L551) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/sdk/baseapp.go#L551), and `loadVersionDiscovered` halves it. Not posted: the PR already states the number, and only the `.store` delta is this diff's to answer for.
- No in-repo client issues `.store` queries; the only reference is the ABCI route itself. Worth knowing whether external indexers depend on it before accepting the latency change. Not posted: it folds into the Warning's decision.
- [PR #6014](https://github.com/gnolang/gno/pull/6014) is the same `atomic.Pointer[types.CommitID]` change with the same publication points, so this PR subsumes it and the two conflict textually in [`rootmulti/store.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store.go#L58) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store.go#L58) and in [`store_test.go`](https://github.com/gnolang/gno/blob/5ceafd2c5/tm2/pkg/store/rootmulti/store_test.go#L490) · [↗](../../../../../.worktrees/gno-review-6018/tm2/pkg/store/rootmulti/store_test.go#L490). Its `TestStoreQueryConcurrentWithCommit` is the one piece not carried over, and it is redundant given the revert-proof above. Not posted: it is a merge-order call for the maintainers, not a change to this diff.
- `TestFastIndex_NaturalQueryCommitRace` requires 1,900 successful query loads to call the race exercised, and each load now costs a `discoverVersions` scan that grows with the block count. Not posted: the floor cleared comfortably here, so there is no evidence of flake yet.
