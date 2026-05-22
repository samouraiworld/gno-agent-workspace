# PR #5571: feat(bptree): leaf-format v2/v3 + latest-view cache + 13-fix perf pass

**URL:** https://github.com/gnolang/gno/pull/5571
**Author:** clockworkgr | **Base:** master | **Files:** 34 | **+2633 -436**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR introduces three feature areas to the `tm2/pkg/bptree` package on top of PR #5570:

1. **Latest-view fast-node cache** — `MutableTree.fastNodes` is an LRU (default 10K entries) that short-circuits `Get`/`Has` tree walks. Populated on `Set`/`Get` hits, invalidated per-key on `Remove`, purged wholesale on `Rollback`/`LoadVersion`. An adaptive-suspend layer disables the cache when `tree.size > capacity × 4`, avoiding LRU thrash on large working sets. `ImmutableTree` snapshots deliberately bypass the cache.

2. **Inline small-value storage (leaf v2)** — Values ≤ `InlineValueThreshold` (default 64 B, max 4 KiB) are stored directly in the leaf payload via a per-leaf `inlineMask` bitmap, eliminating the external `ValueKey` indirection. External-value slots remain for oversized values. This is the source of the 25–500× iteration/proof wins on pebbledb.

3. **Prefix-compressed leaf keys (leaf v3)** — Leaf keys share a common byte prefix emitted once, with per-slot suffixes. Writers always emit v3; readers accept v1/v2/v3. The cached `prefixLenCached` field avoids recomputation on every `Serialize`.

Additional changes: pooled `bytes.Buffer` serialization (`sync.Pool`), incremental mini-merkle rebuild via `slotsDirty` bitmap + `slotHashes` per-slot cache, bound-once `valueResolver` to avoid per-call closure allocation, all-external leaf decode bulk-read optimization, `getNodeUncontended` singleflight bypass for `LoadVersion`, and a suite of correctness fixes from a self-review pass (leaf-read budget, defensive cache copies, split dirty-marking, Clone miniTreeDirty preservation).

The store wrapper (`tm2/pkg/store/bptree`) is updated but inline-value storage is **disabled by default** — `StoreConstructor` does not pass `InlineValueThresholdOption`. Callers must opt in.

**Areas by file:**
- Core logic: `node.go`, `mutable_tree.go`, `immutable_tree.go`, `nodedb.go`, `insert.go`, `remove.go`, `split.go`, `iterator.go`, `proof.go`, `export.go`, `import.go`
- Configuration: `options.go`, `const.go`, `errors.go`
- Tests: `bptree_test.go`, `dirty_invariants_test.go`, `fastnode_test.go`, `inline_test.go`, `inline_threshold_clamp_test.go`, `leaf_read_budget_test.go`, `leafv3_prefix_test.go`, `valuekey_nonce_test.go`, `robustness_test.go`, `stress_test.go`, `prune_test.go`
- Benchmarks: `benchmarks/` (8 files: bench test, raw data, 2 markdown reports)
- Store: `store/bptree/store.go`

## Test Results

- **Existing tests:** PASS — `go test ./tm2/pkg/bptree/...` completes in ~52s, all green.
- **CI race test:** PASS — `Go test -race (bptree)` is green.
- **CI lint:** FAIL — 6 bptree-specific lint findings (2 unused functions, 3 unused variables, 1 unnecessary conversion). See Warnings.
- **Codecov:** PASS — "All modified and coverable lines are covered by tests."
- **Other CI failures:** `proto not updated`, `go.mod not tidied`, generated code stale — base-branch drift, not this PR's fault.

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `tm2/pkg/bptree/immutable_tree.go:73` — `(*ImmutableTree).resolveValue` is unused. All callers use `t.valueResolver` directly via `leaf.valueAt(slot, t.valueResolver)`. The method is a leftover from before the resolver was stored as a field. Should be removed; if it's intended for future external callers, it should delegate to `t.valueResolver` rather than duplicating the nil-check logic.

- [ ] `tm2/pkg/bptree/node.go:118` — `(*LeafNode).valueKeyAt` is unused. The doc says "Used by orphan tracking paths" but no caller exists in the codebase. Orphan tracking uses `captureSlotPayload` / direct `leaf.valueKeys[i]` reads. Remove or document if reserved for future use.

- [ ] `tm2/pkg/bptree/node.go:823` — `uint64(prefixLen)` unnecessary conversion flagged by `unconvert` linter. The linter is correct; `prefixLen` is already `int` and `binary.ReadUvarint` returns `uint64`. The line is `keyLen := uint64(prefixLen) + suffixLen` where `suffixLen` is `uint64` — the conversion is implicit in Go, the explicit cast is redundant.

- [ ] `tm2/pkg/bptree/iavl_tree_test.go:192` — `hash` assigned but never used (`SA4006`). Dead variable in test code.

- [ ] `tm2/pkg/bptree/iavl_tree_test.go:330` — `version` assigned but never used (`SA4006`). Dead variable in test code.

- [ ] `tm2/pkg/bptree/prune_test.go:908` — `v` assigned but never used (`SA4006`). Dead variable in test code.

- [ ] `tm2/pkg/store/bptree/store.go:38` — `StoreConstructor` does not enable inline-value storage. The PR description and benchmark docs present inline-value wins as the headline result, but the production code path (the store wrapper) won't see them. Consider adding `bp.InlineValueThresholdOption(bp.DefaultInlineValueThreshold)` to the `NewMutableTreeWithDB` call, or at minimum add a TODO/comment explaining that this is a deliberate opt-in requiring a chain-format decision.

## Nits

- [ ] `tm2/pkg/bptree/node.go:839` — In `readLeafNodeV3`, when `suffixLen == 0`, the key aliases the `prefix` slice: `n.keys[i] = prefix[:prefixLen:prefixLen]`. This is correct under the key-ownership invariant, but the comment could note that *all* such zero-suffix keys within the same leaf share the same backing array, meaning a caller holding two such keys and accidentally `append`ing to one could corrupt the other (the three-index slice prevents this, but a future refactor that drops the `:prefixLen` cap would silently break).

- [ ] `tm2/pkg/bptree/mutable_tree.go:360-370` — `Get` makes a defensive copy on the fast-node cache hit path. This is correct and documented, but the allocation (1 per hit) is now the dominant cost for the cached `Get` path. If the caller contract ever allows returning shared slices (e.g., the store wrapper already copies), this copy could be made conditional.

- [ ] `tm2/pkg/bptree/mutable_tree.go:309` — `cacheValueForKey` is called on the new-root path (tree was empty, first Set). At that point `t.size == 1` and the cache is always active, so the call is useful but the `reconcileFastNodeState()` call immediately before is redundant (size just went from 0 to 1, well below any threshold). Minor.

- [ ] `tm2/pkg/bptree/split.go:22` — `splitLeaf` takes `inlineMask` as `uint64` (for the B+1 overflow case where bit 31 needs to survive a left-shift), but the two produced `LeafNode`s store it as `uint32`. The widening/narrowing is correct and documented, but a future maintainer might not immediately understand why the parameter type differs from the field type. A one-line comment at the call site in `leafInsert` would help.

- [ ] `tm2/pkg/bptree/iterator.go:346-362` — `loadLeafValues` resolves only the `leafVisitWindow` range. This is a meaningful optimization, but `it.leafValues` is a `[B][]byte` array — slots outside `[lo, hi)` remain `nil`. If a future caller iterates backward past `lo` (currently impossible given the current `Next` logic), the nil would surface. The invariant is maintained today; just flagging for awareness.

## Missing Tests

- [ ] **Cross-format round-trip**: No test verifies that a tree built with inline values (v3), saved, then loaded, produces a v3 leaf that is correctly deserialized and that `Get`/`Has`/iteration still return correct values. `inline_test.go` and `leafv3_prefix_test.go` cover individual components but not the full save→load→verify path through a DB-backed tree. This is the critical migration scenario for existing chains.

- [ ] **Fast-node cache re-activation**: `reconcileFastNodeState` transitions from off→on when `t.size` drops below the threshold. No test exercises the off→on path (e.g., insert above threshold, then remove back below, then verify cache hit). Only the on→off (purge) path is implicitly tested.

- [ ] **SaveVersion 100K pebbledb regression**: The benchmark data shows a +51% regression on `SaveVersion 100K pebbledb`. No test or follow-up issue is referenced. This should at minimum be tracked as a known issue or a TODO in the code.

- [ ] **Adaptive cache with inline values**: No test verifies that when the cache is active, `Get` returns the correct inline value (not a stale external value from a previous `Set`). The `fastnode_test.go` tests only cover external values.

## Suggestions

- The 6 lint findings (2 unused functions, 3 unused variables, 1 unnecessary conversion) are trivially fixable and would turn the lint CI green for the bptree package. Removing the two unused methods (`resolveValue` on `ImmutableTree`, `valueKeyAt` on `LeafNode`) also reduces the public API surface before it becomes depended on.

- Consider adding `bp.InlineValueThresholdOption(bp.DefaultInlineValueThreshold)` to `StoreConstructor` in `store.go:39` now that the three-layer clamp makes it safe, or at minimum add a `// TODO(clockworkgr): enable inline-value storage after chain-format decision` comment so the gap between benchmark claims and production behavior is documented in code.

- The `MaxInlineValueThreshold = 4 KiB` cap is well-chosen (32 × 4 KiB + keys + headers ≈ 144 KiB, under the 256 KiB `maxLeafReadBytes`). Consider making this relationship explicit with a compile-time assertion, e.g. `var _ = maxLeafReadBytes - B*MaxInlineValueThreshold` in a test file, so a future change to either constant triggers a visible failure.

## Questions for Author

- The PR stacks on top of PR #5570 and includes its merge. What is the merge plan — should #5570 merge first and this PR rebase, or is the intent to merge this as a standalone superset?

- The `SaveVersion 100K pebbledb` regression (+51% vs PR #5570) is acknowledged in the benchmark doc as "likely a flush-timing interaction." Is there a plan to investigate this before merge, or is it deferred to a follow-up? The regression is partially masked by the fact that B+32 is still 1.43× faster than IAVL at the same size, but the direction is concerning.

- `immutableForProof` at `proof.go:285` shares `t.root` in memory with the MutableTree and documents (Finding #9) that concurrent mutations can tear mini-merkle state. Is there a plan to close this hazard (e.g., by cloning the root for proof generation), or is the current "caller must not mutate concurrently" contract considered sufficient?

## Verdict

APPROVE — The PR is well-structured with thorough self-review, correct on-disk format evolution (v1→v2→v3 with full backward compatibility), sensible safety clamps on inline-value thresholds, and impressive performance gains. The lint findings are trivially fixable and don't affect correctness. The two unused methods should be removed before merge to prevent API surface lock-in. The store wrapper's inline-value opt-in gap should be documented with a TODO. The `SaveVersion 100K pebbledb` regression deserves a follow-up tracking issue.
