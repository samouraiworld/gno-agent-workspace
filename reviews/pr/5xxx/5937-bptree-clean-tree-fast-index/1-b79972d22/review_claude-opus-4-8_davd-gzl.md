# PR [#5937](https://github.com/gnolang/gno/pull/5937): feat(tm2/bptree): serve clean-working-tree reads from the fast index

URL: https://github.com/gnolang/gno/pull/5937
Author: jaekwon | Base: master | Files: 49 | +1490 -295
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: b79972d22 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5937 b79972d22`

Round 1 (deep mode). Merged as squash `dc305b6d6` on master; reviewed on its merits regardless. The branch carries two riders the title does not name: `1e2e00e2f` mounts this store on the live chain and reprices consensus gas (ADR filed as PR 5938), and `410137ac9` changes `BaseApp.InitChain`'s checkState isolation. Findings below are written as follow-up work against master as it stands today.

**TL;DR:** The bptree keeps a side lookup table that maps a key straight to its value, so reading a key takes one disk read instead of walking down the tree. Until now that table was only used for reads of already-committed history. This change lets the live read path use it too, during the window when the working tree has no pending writes, which is every read a transaction makes.

**Verdict: REQUEST CHANGES** — the fast-index gate itself is correct and unusually well tested, but the mount rider puts two real regressions on the live chain: every ABCI query at a height now costs a linear scan of all retained versions (~220 ms at the default retention, holding the mutex that block production needs), and the "immutable" query-height store open can write to the live database because the read-only wrapper never reaches the mounted store.

## Summary

The fast index is an optional `'F'‖key → version‖value‖crc` map outside the Merkle commitment. Before this change only `ImmutableTree.Get` consulted it, so the consensus execution path (`Store.Get → MutableTree.Get`) always walked the tree. The insight is that tm2 buffers every block write in cache layers above the tree, so during DeliverTx/CheckTx/Simulate the working tree is byte-identical to the committed snapshot. `MutableTree.Get` now consults the index when `fastReadable()` holds: feature on, and `t.root == t.lastSaved` by pointer identity. That predicate is exact, because both mutation entry points clone the root unconditionally before doing anything else. The PR also fixes a real stale-index hazard on the import path, and replaces the `FastIndexEnabled` package global with a per-mount constructor.

The riders are where the risk is. `1e2e00e2f` switches gno.land's `main` store from IAVL to this store and reprices GET from 3.0 to 1.0 read-ops, writes from 4.4 to 5.4. That is consensus-breaking by design (fresh chains and forks only) and lands the bptree store's per-query version-discovery scan on the production query path, where IAVL's had no scan at all.

## Glossary

- fast index: the bptree's optional key-to-value side map, unauthenticated and outside the Merkle commitment, so it never moves the app hash or charged gas.
- bptree (B+32): the versioned copy-on-write B+tree with 32-way fanout that replaced IAVL as gno.land's `mainKey` store; swapping backends moves every app hash.
- app hash: the per-block commitment to application state; two honest nodes computing different ones halts the chain.
- depth-metered store: a store whose cache wrap charges tree-depth-scaled I/O gas instead of a flat cost.
- copy-on-write: writing a new node and rewriting the path to the root rather than mutating in place.

## Fix

Reads previously always descended the tree. `MutableTree.Get` now probes the index first when the working root is pointer-identical to the committed root, falling back to the walk on a miss, a too-new entry, or a corrupt record. The load-bearing constraint is that pointer identity must be exact in the conservative direction: a dirty tree must never look clean. It holds because every *published* mutation replaces the root with a clone. [`treeInsert`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/insert.go#L25) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/insert.go#L25) clones at entry and [`Set` publishes unconditionally](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/mutable_tree.go#L150) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/mutable_tree.go#L150), so a Set that writes an identical value still closes the gate: a wasted walk, never a wrong read. [`treeRemove`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/remove.go#L21) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/remove.go#L21) also clones at entry, but [`Remove` returns at `!found`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/mutable_tree.go#L261-L263) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/mutable_tree.go#L261-L263) before publishing, discarding the clone and staging nothing, so removing an absent key leaves the session genuinely clean and the gate correctly open.

## Benchmarks / Numbers

Query-height store open, PebbleDB (the production backend), idle machine, warm block cache, `PruneSyncable` retention. Measured via `rootmulti`'s query path (`Store.LoadVersion` with `opts.Immutable`):

| retained versions | per query-path open |
|---|---|
| 50,000 | 15.1 ms |
| 100,000 | 29.8 ms |
| 200,000 | 62.0 ms |
| 400,000 | 124.3 ms |
| 705,600 (gno.land default) | ~219 ms (extrapolated, 0.31 µs/version) |

Gas goldens, identical workloads, price params only:

| workload | before | after | delta |
|---|---|---|---|
| addpkg (`restart_gas`) | 2,780,212 | 2,476,212 | −10.9% |
| alloc loop (`gc`) | 151,321,803 | 151,133,803 | −0.12% |

## Critical (must fix)

None.

## Warnings (should fix)

- **[read path writes production state]** `tm2/pkg/store/rootmulti/store.go:378-382` — the ABCI query-height store open can rebuild and rewrite the live fast index, because the `ImmutableDB` wrapper never reaches any store mounted with an explicit db.
  <details><summary>details</summary>

  [`MultiImmutableCacheWrapWithVersion`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/rootmulti/store.go#L250-L257) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/rootmulti/store.go#L250-L257) wraps `ms.db` in a read-only `ImmutableDB` and sets `storeOpts.Immutable`. But [`constructStore`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/rootmulti/store.go#L378-L382) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/rootmulti/store.go#L378-L382) prefers `params.db` whenever it is non-nil, and [gno.land passes `cfg.DB` explicitly](https://github.com/gnolang/gno/blob/b79972d22/gno.land/pkg/gnoland/app.go#L106) · [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/gnoland/app.go#L106), so the query view is handed the raw writable database and the wrapper is dead for every gno.land store. [`ensureFastIndex`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/fast_index.go#L212-L224) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/fast_index.go#L212-L224) then runs on that open with a live batch and ignores `opts.Immutable` entirely. Both intended fail-safes are inert: the wrapper that would panic on a write, and the nil batch that would fail fast. In-contract this is latent, since the stamp is maintained transactionally and the shared ABCI mutex serializes queries against Commit, but any future stamp desync turns a read into a full rebuild that writes live state and holds the ABCI mutex for the duration. I proved the write end-to-end: deleting the stamp by hand (the documented operator escape hatch) and then issuing one query-height open restored the stamp and rewrote a doctored `'F'` entry. Fix: have `ensureFastIndex` return early when `opts.Immutable` is set, and make `constructStore` wrap `params.db` when `storeOpts.Immutable` holds so the read-only guarantee is real rather than nominal.
  </details>

- **[query cost grows without bound]** `tm2/pkg/bptree/nodedb.go:473` — every ABCI query at a height rescans every retained root record, ~220 ms at gno.land's default retention, under the mutex block production needs.
  <details><summary>details</summary>

  [`discoverVersions`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/nodedb.go#L473-L499) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/nodedb.go#L473-L499) iterates the whole `PrefixRoot` range to find first/latest. [`MutableTree.Load`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/mutable_tree.go#L488) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/mutable_tree.go#L488) calls it, then `LoadVersion` calls it again, and the store's immutable branch calls `Load()` and discards the result before reaching [`GetImmutableUnregistered`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/bptree/store.go#L189-L194) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/bptree/store.go#L189-L194). The IAVL mount this replaces went [straight to `GetImmutable(ver)`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/iavl/store.go#L175-L182) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/iavl/store.go#L175-L182) with no scan, so mounting this store is what puts the scan on the production query path. gno.land defaults to [`PruneSyncableStrategy`](https://github.com/gnolang/gno/blob/b79972d22/gno.land/pkg/gnoland/app.go#L63) · [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/gnoland/app.go#L63), and [`PruneSyncable` keeps 705,600 versions](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/types/options.go#L42) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/types/options.go#L42); all three ABCI connections share one mutex via [`localClientCreator`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bft/proxy/client.go#L26-L34) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bft/proxy/client.go#L26-L34), and [`QuerySync` holds it for the whole query](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bft/abci/client/local_client.go#L176-L182) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bft/abci/client/local_client.go#L176-L182), so a query blocks Commit. Nothing is cached: [`handleQueryCustom`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/sdk/baseapp.go#L514) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/sdk/baseapp.go#L514) builds the immutable multistore per query, and it is the [default route](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/sdk/baseapp.go#L420-L431) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/sdk/baseapp.go#L420-L431) for all four of gno.land's [registered handlers](https://github.com/gnolang/gno/blob/b79972d22/gno.land/pkg/gnoland/app.go#L217-L220) · [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/gnoland/app.go#L217-L220), so every `vm/qeval`, `vm/qrender`, `auth/accounts` and `bank/balances` read pays it: essentially all gnoweb and gnokey read traffic. Measured on PebbleDB: 15.1 ms at 50K retained versions rising cleanly linearly to 124.3 ms at 400K, extrapolating to ~219 ms at 705,600. That is a floor: measuring through `MultiImmutableCacheWrapWithVersion` itself rather than the store open alone gives 0.44-0.58 µs/version against this test's 0.31, or ~310-410 ms at the default retention. Archive nodes (`KeepRecent=0`, keep-all) have no bound at all. The scan is also unnecessary on this path: [`getImmutable`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/bptree/mutable_tree.go#L632-L637) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/bptree/mutable_tree.go#L632-L637) resolves the root by direct key lookup, and the immutable adapter's [`VersionExists`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/bptree/tree.go#L69-L71) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/bptree/tree.go#L69-L71) compares against the snapshot's own version, so nothing on the immutable path reads the discovered first/latest counters. Fix: drop the `Load()` call from the `opts.Immutable` branch of `bptree.Store.LoadVersion`, whose return value is already discarded. Separately, `discoverVersions` should seek to the first and last key of the `PrefixRoot` range rather than iterate it, which also fixes the non-immutable opens.
  </details>

- **[absent-key reads underpriced 3x more than before]** `gno.land/pkg/sdk/vm/params.go:40` — GET is now pinned at one flat read, but a miss walks the whole tree and is charged the same.
  <details><summary>details</summary>

  [`minGetReadDepth100Default` drops 300 → 100](https://github.com/gnolang/gno/blob/b79972d22/gno.land/pkg/sdk/vm/params.go#L40) · [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/sdk/vm/params.go#L40) on the reasoning that a present-key GET is one flat index read. The index never answers absence, so an absent-key GET pays the index probe and then the full descent. [`cacheStore.Get`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/cache/store.go#L139-L151) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/cache/store.go#L139-L151) charges depth gas before the fetch and [`effectiveGetReadDepth100`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/cache/store.go#L91-L99) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/cache/store.go#L91-L99) returns the Fixed pin unconditionally, so present and absent cost the same 1.0. The pre-mount pin charged 3.0 for the same walk, so the gap between charged and real widens 3x. It is not a regression against the IAVL status quo in absolute terms, since IAVL charged 3.0 for a ~15-read descent, but it is a new asymmetry a contract can bias toward at will. `Has` inherits it: [`cacheStore.Has`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/store/cache/store.go#L208-L211) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/store/cache/store.go#L208-L211) is `Get(key) != nil`, so an existence check on a missing key is the cheap primitive. Fix: charge the miss what it costs, by adding a nil-result surcharge in `cacheStore.Get` after the parent fetch, where the nil result is already in hand.
  </details>

## Nits

- `gnovm/pkg/gnolang/files_test.go:119` — the fresh-store condition is `strings.HasPrefix(subTestName, "alloc")`, and `subTestName` is the walk-relative path, so an allocator-golden test added under a subdirectory (`storage/alloc_x.gno`) would silently rejoin the pool and flake. Keying off the `MAXALLOC:` directive the tests already carry would not depend on the filename.

## Missing Tests

- **[silent regression risk]** `contribs/gnogenesis/internal/fork/generate.go:676` — nothing fails when `vm.DefaultParams()` moves but no new era fingerprint is appended to `untunedDepthFingerprints`.
  <details><summary>details</summary>

  [`untunedDepthFingerprints`](https://github.com/gnolang/gno/blob/b79972d22/contribs/gnogenesis/internal/fork/generate.go#L676-L700) · [↗](../../../../../.worktrees/gno-review-5937/contribs/gnogenesis/internal/fork/generate.go#L676-L700) identifies an untuned source genesis by exact match against a per-era fingerprint, and [`params.go` carries the rule as a comment](https://github.com/gnolang/gno/blob/b79972d22/gno.land/pkg/sdk/vm/params.go#L36-L38) · [↗](../../../../../.worktrees/gno-review-5937/gno.land/pkg/sdk/vm/params.go#L36-L38) ("Changing these defaults requires a new legacy fingerprint"). A comment is the only thing enforcing it. The next defaults change without an appended era silently stops repricing forked chains, which is exactly the failure this mechanism exists to prevent, and it fails open and quiet. A guard test asserting that the current `vm.DefaultParams()` is itself one entry behind the newest fingerprint, or simply that the fingerprint list's newest entry differs from the live defaults, turns the comment into a check.
  </details>

## Suggestions

- **[dead safety net]** `tm2/pkg/db/immutable.go:58-65` — `NewBatch` returning a bare `nil` makes every misuse a nil-interface panic at an unrelated call site rather than an error at the point of misuse.
  <details><summary>details</summary>

  [`ImmutableDB.NewBatch`](https://github.com/gnolang/gno/blob/b79972d22/tm2/pkg/db/immutable.go#L58-L65) · [↗](../../../../../.worktrees/gno-review-5937/tm2/pkg/db/immutable.go#L58-L65) returns `nil` with an `// XXX` marker, while every other mutating method on the type panics with a clear message. Returning a batch whose methods panic the same way would make the Warning above surface at its cause instead of deep inside a rebuild.
  </details>

## Verified

- The clean/dirty gate is exact in the safe direction. `treeInsert` and `treeRemove` clone the root before inspecting the key, so a no-op Set (same value) and a Remove of an absent key both close the gate, costing a walk but never serving stale. The committed-empty case (`root == lastSaved == nil`) is caught by `Get`'s nil-root return before the gate is consulted.
- `Has` stays walk-only on both trees, and consensus existence checks resolve through `cacheStore.Get`, matching what the PR claims.
- The index stamp rides the same atomic batch as the root and nodes in `SaveVersion`, so a crash cannot leave the stamp disagreeing with the committed tree.
- A read-only query-height store open rewrote the live database: after deleting the stamp by hand and doctoring an `'F'` entry, one `MultiImmutableCacheWrapWithVersion` call restored the stamp and replaced the doctored entry with a rebuilt one.
- Query-path open cost is linear in retained versions on PebbleDB: 15.1 / 29.8 / 62.0 / 124.3 ms at 50K / 100K / 200K / 400K, on an idle machine with a warm block cache.
- The gas goldens match the repricing claim: addpkg 2,780,212 → 2,476,212 is −10.9%, against a claimed −11%.
- The allocator golden set is complete: all 9 alloc filetests that print a byte count moved, and no byte-count test was left stale. `alloc_2` did not move because it errors on the allocation limit before printing, and the three `_long` variants did not move because they already ran on their own store. The shift is not uniform, so it is not a single pool-contamination constant: 8 of the 9 moved by exactly +608, while [`alloc_4`](https://github.com/gnolang/gno/blob/b79972d22/gnovm/tests/files/alloc_4.gno#L25) · [↗](../../../../../.worktrees/gno-review-5937/gnovm/tests/files/alloc_4.gno#L25) moved +3879 on its first-GC count (16,868 → 20,747) and 0 on its second-GC count. The 10th changed file, [`alloc_10c`](https://github.com/gnolang/gno/blob/b79972d22/gnovm/tests/files/alloc_10c.gno#L35) · [↗](../../../../../.worktrees/gno-review-5937/gnovm/tests/files/alloc_10c.gno#L35), prints no byte count at all: its golden error gained a source location and `(no GC)`, so the limit now trips earlier, during import. The outcome class is still an allocation-limit error, so the goldens are right, but the fresh store moves where the limit trips rather than only shifting counts.
- The CodeQL alert on `fast_index.go:65` (`make([]byte, 8+len(value)+checksumSize)` may overflow) is a false positive on 64-bit: `int` is 64-bit, so the sum can only overflow for a value within 12 bytes of `MaxInt64`, which is unallocatable, and tx-borne values are bounded by `MaxTxBytes` well below that.

## Open questions

- The `+fast` column in `PERFORMANCE.md` is marked as derived rather than measured at 101M, and the GET pin of 100 rests on it. The reads-per-op claim is structural (one flat `db.Get` is one read by construction) so the pin is defensible on its own terms; the unmeasured part is latency, which gas does not price. Not posted: the doc already carries the caveat and the pin does not depend on the unmeasured quantity.
- Both riders (the mount plus repricing, and the `InitChain` checkState isolation) are consensus-relevant changes landing under a title that names neither. The mount has its own ADR; the checkState fix is described only in a commit message and an ADR addendum. Not posted: the PR has merged and the packaging question no longer changes what anyone does.
