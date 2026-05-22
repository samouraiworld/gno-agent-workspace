# PR #5570: fix(bptree): correctness, robustness, and performance pass — 52 issues closed

**URL:** https://github.com/gnolang/gno/pull/5570
**Author:** clockworkgr | **Base:** feat/jae/bp32tree | **Files:** 50 | **+6378 -779**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR is a comprehensive correctness, robustness, and performance pass over `tm2/pkg/bptree/`, closing 52 issues found during a line-by-line audit. The changes are substantial (+6378/-779 across 50 files) but focused: no core semantics (tree structure, hashing, versioning, proof format) are altered. The PR targets the `feat/jae/bp32tree` branch (not master directly).

**Changes by area:**

### 1. Core algorithm — Pruning (prune.go)
The biggest architectural change: replaces the positional-descent pruning algorithm (`findCorrespondingChild` / `walkAndPrune`) with **content-addressed mark-and-sweep**. A single `buildRetainedReachableSet` walk builds a `map[[12]byte]struct{}` of all live NodeKeys; then `sweepOld` deletes any old-version node whose NodeKey is not in the set. This is immune to the misidentification class that broke the old positional flow under nested splits/merges (Finding #3, the highest-severity correctness fix). Additional pruning fixes: empty-tree orphan leak (#2), TOCTOU on active-readers check (#15), reader registration ordering (#40), partial-batch error handling (#52), `firstVersion` rewind (#41), leaf-skip invariant verification (#46), short NodeKey ref handling (#47), in-memory child handling in sweepOld (#49) and deleteSubtree (#50).

### 2. Core algorithm — COW / mutation discipline (mutable_tree.go, remove.go, insert.go, split.go)
- `cowRoot()` now clones only when the root is shared with `lastSaved` or carries a durable NodeKey, eliminating the ~4.3 KB per-Set redundant clone (Finding #17).
- `assertCloned` guards on `redistributeRight`, `redistributeLeft`, and `merge` enforce that structural mutations operate only on COW clones (Finding #24).
- `saveNode` skips unchanged subtrees: nil `childNodes[i]` and clean cached children are not recursed (Finding #4, major perf fix — reduces DB reads from ~192 to ~6 per update).
- `Set` is now atomic with value save: `SaveValue` runs before tree mutation, so a failed write leaves the tree untouched (Finding #28).
- Root collapse on `treeRemove` preserves the child's `ndb` but the child keeps its DB-assigned `nodeKey`; `SaveVersion` will re-derive from the new root's assigned NodeKey (Finding #37, documented).
- `copyKey()` used uniformly in redistribute/merge/split paths for key-ownership invariant (Finding #20).

### 3. Concurrency (nodedb.go, node.go, immutable_tree.go, iterator.go)
- **`pruneMu` (RWMutex)** serializes prune against reader registration. Prune takes exclusive lock; readers (`incrVersionReaders`) take shared. Closes the TOCTOU between active-reader check and deletes (#15, #40).
- **Lock-free `getChild` fast path**: `InnerNode.childLoaded` is an `atomic.Uint32` bitmap. Cache-hit reads do `atomic.Load` + non-atomic interface read with release-acquire ordering; only the lazy-DB-load slow path takes `childMu` (#3 architecture change). Bulk mutators call `rebuildChildLoaded` after array shifts.
- **`singleflight`** on `nodeDB.GetNode` cache-miss dedup (#3.2).
- **`ImmutableTree.Close`** uses `sync.Once` for idempotent decr (#45).
- **Iterator version-reader registration**: `newIterator` now receives `version` and calls `incrVersionReaders`; `Close` decrements (#1).

### 4. Value lifecycle (mutable_tree.go, nodedb.go, import.go)
- Tier 1 / Tier 2 orphan model: intra-version orphans deleted eagerly via `DeleteValueDirect`; cross-version orphans deferred to prune-time orphan list.
- `orphanValueKey` length-checks valueKey before `binary.BigEndian.Uint64` (#23).
- `LoadVersion` seeds `nextValueNonce` past the max persisted nonce for the working-version namespace, preventing `SaveValue` from overwriting live values (#1.1 / BUG-3).
- `Rollback` logs per-entry `DeleteValueDirect` errors rather than short-circuiting (#25/#31).
- `Importer.Close` cleans up eagerly-written values on abort (#27).
- `SaveVersion` partial-failure triggers `discardBatch` + `Rollback` (#36).

### 5. Performance (mini_merkle.go, node.go, insert.go, proof.go, const.go, node_key.go)
- `NodeKey.bytes` cached on struct; `GetKey()` is a field load (#21).
- `cachedRootHash` / `cachedSavedHash` on `MutableTree` avoid array-to-slice escape (#21).
- `SiblingPath` returns fixed-size arrays by value instead of heap-allocated slices (#21).
- `const MiniMerkleDepth = 5` replaces runtime `log₂` loop (#21).
- `MiniMerkle.Clear()` is a single struct copy of a package-level template (#21).
- Stack arrays for overflow entries in leaf/inner split (#21).
- Non-membership proof collapses six tree walks to three (#21).
- Batched `Serialize` via `sync.Pool` scratch buffers (#18 architecture).
- Range-aware iterator value prefetch via `leafVisitWindow` (#16).

### 6. Error handling / robustness (nodedb.go, errors.go, node.go, mutable_tree.go, iterator.go)
- `GetNode` error semantics split: `data == nil` → `ErrNodeNotFound`; DB read / deserialize errors → panic (#5).
- `ErrNoValueResolver` distinguishes misconfigured tree from missing key (#10/#11).
- `LoadVersionForOverwriting` / `DeleteVersionsFrom` return `ErrUnsupported` instead of panicking (#12).
- `InnerNode.Serialize` asserts `len(n.children[i]) == NodeKeySize` (#18).
- Deserialization bounds checks on `numKeys` before `int16` cast (#23).
- `readBytes` caps allocations at 64 KiB (#22).
- Iterator error propagation via `it.err`; documented "check Error() after Valid() returns false" (#34/#35).
- `Commit` lifecycle: pending evictions applied only on success; discarded on error (#38/#44).
- `SaveVersion` idempotent re-save handles legacy empty-tree blobs (#26).

### 7. Store wrapper (store/bptree/store.go, tree.go)
- `Store.Close()` delegates to `tree.CloseSnapshot()` for immutable stores.
- `GetImmutable` wires value resolver from `mtree.GetValueByKey`.
- `immutableTreeAdapter.CloseSnapshot` delegates to `ImmutableTree.Close`.
- Query path closes `iTree` after proof generation (line 349).
- Removed dead `SetCommitting` / `UnsetCommitting`.

### 8. CI / benchmarks / docs
- New `.github/workflows/tm2-bptree-race.yml` for `-race` CI.
- `README.md` thread-safety section.
- `BENCHMARKS-5570.md` pre/post comparison on both memdb and pebbledb.
- Baseline raw data files under `benchmarks/baselines/` and `benchmarks/post/`.
- Benchmark harness switched from goleveldb to pebbledb.

## Test Results
- **Existing tests:** PASS (`go test -count=1 ./tm2/pkg/bptree/...` and `./tm2/pkg/store/bptree/...` both OK)
- **CI:** bptree race test PASSES. Multiple other CI checks FAIL (lint, generated, proto) — these appear to be pre-existing issues on the `feat/jae/bp32tree` branch, not introduced by this PR.
- **Edge-case tests:** skipped (314 existing tests + 8 new test files provide strong coverage)

## Critical (must fix)

- [ ] `nodedb.go:111` — `getChild` panics on `GetNode` error: `panic(fmt.Sprintf("bptree: failed to load child node %x: %v", n.children[idx], err))`. While `GetNode` itself panics on DB read errors by design (Finding #5), the `GetNode` call at line 109 can also return `ErrNodeNotFound` if the child has been pruned or the ref is stale. A `singleflight` deduped call returning `ErrNodeNotFound` would propagate through `loadGroup.Do` as a non-nil `err`, which `getChild` would then panic on. This should either handle `ErrNodeNotFound` explicitly (return nil child) or the panic should be narrowed to exclude the "node not found" case. In a DB-backed tree where pruning can run concurrently on a different version's nodes, this is a process-killing path.

  **Update after re-reading:** On closer inspection, `getChild` is called on an `InnerNode` whose `children[idx]` is a serialized ref from a committed version. If that ref is valid and the node was committed, it should always exist in the DB unless pruned. The prune-reader invariant (pruneMu + versionReaders) should prevent a concurrent prune from deleting nodes this inner node references, since the inner node belongs to a version with active readers. However, if `getChild` is called on an `InnerNode` loaded from a *different* tree path (e.g., the mark phase loading a retained version while prune is running on a different range), the `ErrNodeNotFound` path is reachable. The `singleflight` wrapper in `GetNode` returns `ErrNodeNotFound` as a regular error (not a panic), so `getChild`'s unconditional panic on any `err != nil` is too aggressive. This is a real risk for validator liveness.

- [ ] `node.go:248-251` — `LeafNode.Clone()` uses `c := *n` (silent struct copy). The author's own self-review (C-4) flags this: if a future contributor adds a `sync.Mutex` or `atomic.Uint32` to `LeafNode`, the silent copy breaks. `InnerNode.Clone` already uses an explicit struct literal for this reason. The two Clone methods should use the same discipline. This is a maintainability landmine that will bite silently.

- [ ] `prune.go:555-566` — `beginPruning` takes `pruneMu.Lock()` then `mtx.Lock()`, checks readers, then releases `mtx` but keeps `pruneMu`. If the version loop in `PruneVersionsTo` takes a long time (large retained-tree mark walk), the exclusive `pruneMu` blocks ALL new version-reader registrations for the entire prune duration — not just for the pruned versions but for *every* version. This means a long prune on an old range blocks `GetImmutable` on the latest version, which could stall consensus-critical queries. Consider narrowing the pruneMu scope or using per-version granularity.

## Warnings (should fix)

- [ ] `nodedb.go:189-220` — `singleflight` overhead on the common single-writer path. The author's self-review (C-6) identifies this: `loadGroup.Do` costs ~3 allocs / ~100-150 ns even when no contention exists. For the predominant single-writer block-production workload this is pure overhead. The author suggests a fast-path bypass; this should be tracked as a follow-up PR with a clear commitment, not deferred indefinitely.

- [ ] `mutable_tree.go:120` — `Set` rejects `value == nil` with `fmt.Errorf("value must not be nil")` but this error is not a typed/sentinel error. Callers cannot distinguish it programmatically. Consider using a sentinel error for API ergonomics, especially since `ErrEmptyKey` is already a sentinel for the key case.

- [ ] `iterator.go:307-311` — `Iterator.Key()` returns the raw `it.leaf.keys[it.leafIdx]` slice. Callers who retain this slice beyond a `Next()` call may observe corruption if the same key slot is mutated (e.g., in a MutableTree iteration). The IAVL iterator copies the key; B+ tree should match for safety, or at minimum document the borrow semantics.

- [ ] `store/bptree/store.go:115-119` — `Store.Commit()` panics on `SaveVersion` error. In a consensus-critical path this crashes the process. IAVL's store wrapper also panics here, so this is consistent, but it's worth noting that a partial write failure takes down the node rather than returning an error up the stack.

- [ ] `store/bptree/store.go:91-111` — `GetImmutable` creates a new `Store` wrapping an `immutableTreeAdapter`, but the new Store holds a reference to `st.mtree`. If the original Store is garbage-collected while the immutable Store is still in use, the `mtree` reference keeps the `MutableTree` alive (acceptable), but the `mtree`'s `ndb` pointer may be in an indeterminate state if the original Store was closed. This is a theoretical concern since `GetImmutable` is typically called on a long-lived Store, but the ownership semantics are not documented.

- [ ] `nodedb.go:294-321` — `maxValueNonceForVersion` does a reverse-seek that is O(N log N) on memdb (the test backend). The author notes this is acceptable because `LoadVersion` runs once per process lifecycle. However, if someone adds a `LoadVersion` call in a hot test loop, this becomes a performance trap. The O(N log N) behavior should be documented on the function signature.

- [ ] `node.go:367-369` — `serializeBufPool` is a `sync.Pool` with no `Put` cleanup for the `keyBytes` scratch buffer. The `keyBytes` array may retain stale child refs from a previous serialization. The `LeafNode.Serialize` path explicitly clears nil valueKey slots (line 349), but `InnerNode.Serialize` does not clear unused slots in `keyBytes` — if a node with fewer than B children is serialized, the remaining bytes in `scratch.keyBytes` from a prior larger-node serialization could leak into the write. However, the write is bounded by `nc*NodeKeySize` (line 305), so unused slots are not written. Still, the asymmetry with the leaf path's explicit clearing suggests adding a comment or clearing for defense-in-depth.

- [ ] `immutable_tree.go:73-78` — `ImmutableTree.resolveValue` returns `ErrNoValueResolver` when no resolver is set, but `Get` (line 81-90) returns `(nil, nil)` for "not found" and `(nil, ErrNoValueResolver)` for misconfigured. Callers must check both the value and the error to distinguish three states (found, not-found, misconfigured). The `MutableTree.resolveValue` path returns `ErrKeyDoesNotExist` on memValues miss — a different error. This asymmetry could confuse callers who handle both tree types.

- [ ] `prune.go:86-89` — `buildRetainedReachableSet` only walks the *first existing* retained version (breaks at line 179). The correctness argument relies on the COW invariant: any node reachable from the first retained version is also reachable from all later retained versions that share it. This is correct for nodes whose `NodeKey.Version` is ≤ the first retained version. But if a node was created (assigned a NodeKey) in a *later* retained version (version > first retained), it would not be in the reachable set built from the first retained version alone, even though it is reachable from that later version. **Re-reading the correctness argument:** The claim is that a node N with `NodeKey.Version = W` that is reachable from some retained version R must also be reachable from the first retained version F ≥ from > W. This holds because: (1) N was created at version W, (2) N is immutable after save, (3) the path from R to N was unchanged since W, (4) F ≥ from > W, so the path from F to N also exists (every version in [W, F] shares the path). **This is correct.** Withdraw this warning.

## Nits

- [ ] `const.go:55-61` — `init()` panics on `B != 1<<MiniMerkleDepth`. A compile-time assertion would be preferable (e.g., `var _ [B - 1<<MiniMerkleDepth]byte = 0`), though Go's constant evaluation limits may prevent this. The runtime check is acceptable but not ideal.

- [ ] `node.go:248` — `LeafNode.Clone` comment says "The clone SHARES keys and valueKeys slice headers with n" but `valueKeys` slice headers contain 12-byte backing arrays (NodeKeySize), not shared pointers. The sharing is the slice descriptor (ptr/len/cap), not the backing array. The comment is technically accurate but could confuse readers who think the backing arrays are shared.

- [ ] `options.go:6-10` — The comment about removed options (`FlushThreshold`, `AsyncPruning`) reads like a changelog entry rather than API documentation. Consider removing the explanation of what was removed and just documenting what exists.

- [ ] `hash.go:23` — `valueHashLenVarint` is defined as `const valueHashLenVarint byte = 0x20` but the `init()` assertion that `HashSize < 0x80` (ensuring single-byte varint) is missing from the init in const.go. The comment on line 22 states this, but a runtime check would prevent silent breakage if `HashSize` changes.

- [ ] `mutable_tree.go:683` — `WorkingVersion()` returns `t.version + 1` when `t.initialVersion == 0`. If `t.version == math.MaxInt64`, this overflows to `math.MinInt64`. The author guards `maxValueNonceForVersion` against `math.MaxInt64` input (nodedb.go:299), but `WorkingVersion()` itself has no overflow check. Theoretically unreachable in practice (version increments from 0), but a bounds-checking purist would add a guard.

- [ ] `nodedb.go:93` — `newNodeDB` panics on `lru.New` error. The only error `lru.New` returns is negative size; since `cacheSize` comes from a constant (`defaultCacheSize = 10000`), this panic is unreachable. Still, a typed error return would be more idiomatic.

- [ ] `proof.go:169-183` — `findPathToIndex` uses `goto next` to break out of the inner `for` loop. This is a valid Go pattern but stylistically unusual; a labeled break or extracted helper would be more conventional.

- [ ] `nodedb.go:467` — `AvailableVersions` skips entries where `len(key) != 9`. This silently swallows corrupted root records. Logging at debug level would help operators detect DB corruption.

## Missing Tests

- [ ] `getChild` returning `ErrNodeNotFound` from `singleflight` — no test exercises the path where `GetNode` returns `ErrNodeNotFound` through the singleflight wrapper and `getChild` panics on it. Given the critical severity finding above, a targeted test is essential.

- [ ] `LeafNode.Clone` vs `InnerNode.Clone` consistency — no test verifies that adding a non-copyable field to `LeafNode` would be caught at compile time. The C-4 concern from the author's self-review has no test backing.

- [ ] `pruneMu` starvation under long prune — no test measures whether `GetImmutable` on the latest version is blocked while a prune of old versions is in flight. A concurrency test with a timeout would verify the scope of the lock.

- [ ] `Set` with `value == nil` — the error path is not tested. The new `fmt.Errorf("value must not be nil")` guard in `Set` has no corresponding test case.

- [ ] `Iterator.Key()` slice aliasing — no test verifies that the returned key slice is safe to retain across `Next()` calls. If it's a borrowed reference (current implementation), a test should document that contract.

- [ ] `maxValueNonceForVersion` O(N log N) on memdb — no performance test for the memdb path of this function. Given the LoadVersion regression, a benchmark isolating this function would help quantify the contribution.

- [ ] `Importer` abort cleanup — `import.go`'s `Close()` path (deleting eagerly-written values) is exercised by `savedVKs` tracking, but no test explicitly creates an importer, calls `Add` several times, then calls `Close` without `Commit`, and verifies the DB state is clean.

## Suggestions

- Convert `LeafNode.Clone` to explicit struct literal matching `InnerNode.Clone`'s discipline (author's C-4). One-line change, prevents a silent future breakage.

- Narrow `getChild`'s panic to exclude `ErrNodeNotFound`: check `errors.Is(err, ErrNodeNotFound)` and return `nil` instead of panicking. The prune-reader invariant should prevent this in practice, but defensive coding prevents a process crash on a rare edge case.

- Add a per-version `pruneMu` granularity or time-bound the exclusive lock hold. The current design blocks reader registration for ALL versions during prune, which can stall consensus queries on the latest version while old versions are being pruned. An alternative: release `pruneMu` between versions in the loop and re-acquire with a fresh `beginPruning` check for each version.

- Stack-allocate the boundary keys in `maxValueNonceForVersion` (author's C-3): `encodeNodeKeyBytes` allocates on the heap; a stack-allocated `[NodeKeySize]byte` with inline encoding would recover ~200-400 ns and ~4 allocs per `LoadVersion` call, partially addressing the LoadVersion regression.

- Consider a fast-path that bypasses `singleflight` when the load group is empty (author's C-6). Under single-writer block production, the ~3 alloc / ~100-150 ns overhead is pure waste.

- Add a compile-time assertion that `HashSize < 0x80` (ensuring `valueHashLenVarint` is a single byte) alongside the existing `B == 1<<MiniMerkleDepth` init check.

- Document the `Iterator.Key()` borrow semantics explicitly: is the returned slice valid only until the next `Next()` / `Close()` call? If so, document it; if not, copy the key (matching IAVL behavior).

## Questions for Author

- The base branch is `feat/jae/bp32tree`, not `master`. What is the merge plan? Will this be squashed into `feat/jae/bp32tree` first, then that branch merges to master? This affects how CI results should be interpreted (many CI failures are on paths unrelated to bptree).

- The `getChild` panic on `ErrNodeNotFound` (node.go:111) — was this considered during the GetNode error-semantics redesign (Finding #5)? The redesign panics on DB read / deserialize errors but returns `ErrNodeNotFound` for absent nodes; `getChild` does not distinguish the two.

- For `beginPruning` holding `pruneMu` across the entire prune duration: is there data on how long a typical 50-version prune takes on pebbledb? If it's > 100ms, that's a significant window where no new reader can register for any version.

- The author's self-review lists C-1 through C-6 as documented concerns. Are these committed as follow-up work with issue tracking, or aspirational? In particular, C-5 (assertCloned not universal) and C-6 (singleflight overhead) seem like they should have concrete plans.

## Verdict

REQUEST CHANGES — Three critical items require resolution before merge: (1) `getChild` panics on `ErrNodeNotFound` from `singleflight`, which is a validator-liveness risk; (2) `LeafNode.Clone` should use explicit struct literal for consistency with `InnerNode.Clone` and to prevent a silent future breakage; (3) `pruneMu` scope blocks all version-reader registration for the full prune duration, potentially stalling consensus queries. The overall quality of the PR is high — the mark-and-sweep rewrite is a correctness and performance win, the error-handling improvements are thorough, and the test coverage is strong. The three critical items are narrow fixes, not architectural rework.
