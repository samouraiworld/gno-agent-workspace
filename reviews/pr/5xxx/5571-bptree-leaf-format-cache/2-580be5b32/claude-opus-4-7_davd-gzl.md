# PR #5571: feat(bptree): leaf-format v2/v3 + latest-view cache + 13-fix perf pass

URL: https://github.com/gnolang/gno/pull/5571
Author: clockworkgr | Base: feat/alex/bp32tree-second-pass | Files: 45 | +4544 -530
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — round-2 sweep at HEAD `580be5b32` confirms every blocking item from round 1 and from [@ajnavarro](https://github.com/gnolang/gno/pull/5571#pullrequestreview-4155092448) is resolved; only the pre-existing `proof.go` mini-merkle-tearing hazard remains explicitly deferred (Finding #9), and a memdb-only `ScalingSet 1K` +98% self-acknowledged regression deserves a follow-up issue, not a block.

## Summary

This PR sits on top of PR #5570 and adds three orthogonal features to the `tm2/pkg/bptree` package — a latest-view LRU fast-node cache, inline small-value storage (leaf format v2), and common-prefix-compressed leaf keys (leaf format v3) — plus a 13-commit perf sweep and a self-review correctness pass. Round 1 reviewed commit `68b70d94a`; the only delta to HEAD `580be5b32` is one chore commit (`chore(bptree): clear golangci-lint findings under .github/golangci.yml`) that closes every lint finding from round 1 and the lint subset of @ajnavarro's review. Tree semantics, hashing, versioning, and proof format are unchanged; v1/v2/v3 readers coexist so existing chains mount cleanly.

Headline numbers (B+32 vs IAVL on pebbledb at 100K, count=3 -benchtime=2s, from [`benchmarks/PERFORMANCE-5571.md`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/benchmarks/PERFORMANCE-5571.md)):

| Metric | IAVL | B+32 | Win |
|---|---|---|---|
| `IterationFull` / `Descending` | 17–19 ms | 681 µs | 25–29x |
| `Has` | 45 µs | 200 ns | 224x |
| `MembershipProof` | 110 µs | 1.65 µs | 67x |
| `Prune` | 656 ms | 52 ms | 12.6x |
| `LoadVersion` | 29.9 µs | 8.50 µs | 3.5x |
| `SaveVersion` | 8.77 ms | 6.14 ms | 1.43x |
| `DiskSpace` | 27.6 MB | 17.85 MB | -35% |

## Glossary

- **`InlineThreshold`** — named `int` type at [`const.go:72`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/const.go#L72) that gates inline-value storage; `InlineDisabled` (-1) disables, positive values are byte cutoffs clamped to `MaxInlineValueThreshold` (4 KiB).
- **`slotsDirty` / `miniTreeDirty`** — per-leaf bitmap and per-node flag that defer mini-merkle rebuilds until the next `Hash()` call (saves O(N) rebuilds across a burst of writes to the same node).
- **`fastNodes`** — LRU keyed by `string(key)` that short-circuits `Get`/`Has` tree walks for the working version; suspended when `tree.size > capacity * 4`.
- **`valueResolver`** — `func(vk []byte) ([]byte, error)` bound once at tree construction so per-call closures don't escape to the heap.
- **leaf format v1/v2/v3** — on-disk leaf encoding versions; readers accept all three, writers emit v3 only.

## What Changed Since Round 1

Single commit (`580be5b32`):

- `(*ImmutableTree).resolveValue` and `(*LeafNode).valueKeyAt` are now annotated `//nolint:unused` rather than removed — author's deliberate choice to preserve them as API-surface parity. Acceptable; the prior round-1 warning is closed.
- `uint64(prefixLen)` cast in `readLeafNodeV3` dropped; my round-1 review was wrong about the underlying type (`prefixLen` is `uint64` from `binary.ReadUvarint`, not `int`), but the cleanup is correct either way.
- Three dead `hash` / `version` / `v` variables in test files replaced with `_`; the `errorlint` (`%v` → `%w`, `==` → `errors.Is`) and `predeclared` (`make`, `max` shadowing) cleanups land.
- `gosec G602` false positive on indexed array writes resolved via composite-literal initialisation in `SaveValue` / `valueDBKey`.

Lint now reports `0 issues` against `.github/golangci.yml`. Full bptree test suite passes locally (`go test ./tm2/pkg/bptree/...` → 44.9s, all green).

## Round-1 Items: Resolution Status

| Round-1 item | Status at HEAD |
|---|---|
| `resolveValue` / `valueKeyAt` unused (W) | Closed — annotated `//nolint:unused` |
| `uint64(prefixLen)` redundant cast (W) | Closed — dropped |
| Three `SA4006` dead vars in tests (W) | Closed |
| `store.go:38` no opt-in for inline values (W) | Closed — now has a 12-line doc block explaining the on-disk-format gate ([`store.go:27-37`](../../../../../.worktrees/gno-review-5571/tm2/pkg/store/bptree/store.go#L27-L37)) |
| Cross-format round-trip missing test (MT) | Wrong claim — [`inline_test.go:61-100`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/inline_test.go#L61-L100) already covers save→reload→verify on a DB-backed tree |
| Fast-node cache off→on re-activation missing test (MT) | Still missing, but low-risk; keep as a follow-up |
| SaveVersion 100K pebbledb +51% regression (MT) | Documented in [`BENCHMARKS-5571.md`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/benchmarks/BENCHMARKS-5571.md) as a known follow-up; see Warnings below |
| Adaptive cache with inline values missing test (MT) | Still missing |

## @ajnavarro Round-1 Items: Resolution Status

| @ajnavarro item | Status at HEAD |
|---|---|
| `slotsDirty` not shifted in `leafInsert` shift loop | Closed — [`insert.go:127-149`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/insert.go#L127-L149) no longer shifts `slotHashes` either, and `markLeafSlotsDirtyRange(pos, numKeys)` dirties the entire affected range |
| `InnerNode.Clone()` doesn't copy `miniTreeDirty` | Closed — [`node.go:387`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/node.go#L387) explicitly copies it; `LeafNode.Clone()` uses `*n` which copies the field |
| Comment names wrong function (RebuildMiniMerkle vs `rebuildMiniMerkleIncremental`) | Closed — [`node.go:75-80`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/node.go#L75-L80) now names both correctly |
| Doc says zero→default, resolver maps zero→disabled | Closed — both the option doc ([`options.go:21-35`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/options.go#L21-L35)) and the resolver doc ([`mutable_tree.go:116-129`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/mutable_tree.go#L116-L129)) now consistently document "disabled by default; opt in via `InlineValueThresholdOption`" |
| Cache aliases leaf inline storage on Get-hit (no defensive copy) | Closed — [`mutable_tree.go:387-390`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/mutable_tree.go#L387-L390) now copies before `fastNodes.Add` |
| `ndb.GetValue` `(nil, nil)` corruption passthrough | Closed — `GetValue` now returns `ErrValueMissing` on `data == nil` ([`nodedb.go:358-360`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/nodedb.go#L358-L360)); callers no longer have to interpret a nil-nil sentinel |
| `Remove` swallows `resolveValue` error silently | Closed — [`mutable_tree.go:492-505`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/mutable_tree.go#L492-L505) captures and logs the error with the explicit reason why surface-via-return is intentionally avoided |
| `nodedb.go:132` pool comment inverted | Closed — [`nodedb.go:128-138`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/nodedb.go#L128-L138) now correctly describes the cap-and-drop policy |
| `ADVANCED_PLAN.md` development scaffolding | Closed — file removed; `PLAN.md`, `PERFORMANCE.md`, `README.md` retain design content only |
| `InlineValueThresholdOption` no upper-bound clamp | Closed — `MaxInlineValueThreshold = 4 KiB` cap applied at both the option-builder layer ([`options.go:74-79`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/options.go#L74-L79)) and the resolver ([`mutable_tree.go:130-138`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/mutable_tree.go#L130-L138)), with a third re-check at the per-Set call site ([`mutable_tree.go:271-279`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/mutable_tree.go#L271-L279)) |
| Sentinel-heavy bare-int `InlineValueThreshold` | Closed — replaced with named `InlineThreshold` type and named `InlineDisabled` / `DefaultInlineValueThreshold` / `MaxInlineValueThreshold` constants |
| Benchmarks don't cover pebbledb (production backend) | Closed — `backends := []string{"memdb", "pebbledb"}` at [`bench_test.go:901`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/benchmarks/bench_test.go#L901) and the default-flag help string updated to `memdb, pebbledb`. The headline numbers in the PR description are measured on pebbledb. |

## Critical (must fix)

None.

## Warnings (should fix)

- **[pebbledb-only regression]** [`benchmarks/BENCHMARKS-5571.md`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/benchmarks/BENCHMARKS-5571.md) — `SaveVersion 100K` is **+51% slower** on pebbledb (6.74 ms → 10.18 ms) vs PR #5570
  <details><summary>details</summary>

  The bytes/op drop (1.18 MB → 458 KB, -61%) and allocs/op drop (9 173 → 1 232, -87%) are large wins, but per-call ns rises. The doc hypothesises "flush-timing interaction with pebbledb's internal write batching at the larger pool buffer size." Pebbledb is the gno.land production backend, so this is the only headline number that goes the wrong direction. B+32 is still 1.43x faster than IAVL at the same size, but the direction concerns me because future PRs that add overhead will compound on top. Fix: open a follow-up issue tracking the regression with a target investigation date, and either (a) confirm the variance hypothesis with multi-run statistics (count >= 10), or (b) pin the buffer-cap (`saveBufCapCap` at [`nodedb.go:157`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/nodedb.go#L157)) back to 8 KiB on the pebbledb path and re-measure.
  </details>

- **[self-acknowledged regression]** [`benchmarks/BENCHMARKS-5571.md`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/benchmarks/BENCHMARKS-5571.md) — `ScalingSet 1K` memdb is **+98%** (2.54 µs → 5.03 µs), `SET update 1K` memdb is **+31%** (2.52 µs → 3.31 µs)
  <details><summary>details</summary>

  Memdb-only, small-tree, cache-active path. The doc attributes it to the cached `payload.inline` slice now being shared with the LRU rather than copied — long-lived cache entries shift GC pressure on the fast-Set path. The 100K case (cache suspended) is faster, so the regression sits squarely in the workload where the cache is supposed to help. It's flagged in the benchmark doc but no follow-up tracked. Fix: either revert the share-with-cache optimisation for the inline-payload path (call `cacheValueForKey` with `owned=nil` and let it copy), or open a follow-up issue.
  </details>

- **[concurrent-proof tearing]** [`proof.go:282-284`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/proof.go#L282-L284) — `immutableForProof` shares `t.root` in memory; concurrent `Set`/`Remove` on the MutableTree can tear mini-merkle state
  <details><summary>details</summary>

  This is the pre-existing Finding #9 the PR explicitly defers via a Note comment. The hazard is real: a proof reader walking the shared root while a writer flips a `miniTreeDirty` bit or rebuilds a mini-merkle can observe an inconsistent intermediate. The current "caller must not mutate concurrently" contract works for sequential ABCI flows but won't hold up under any future call site that proves from a goroutine while the working tree continues to mutate. Fix: clone the root (cheap; just a struct copy at the head — children stay shared via the COW invariant) before constructing `imm`, or add a load-only mode where mini-merkle reads atomically skip the rebuild and fall back to `RebuildMiniMerkle` on a fresh local copy.
  </details>

## Nits

- [`mutable_tree.go:308-309`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/mutable_tree.go#L308-L309) — `reconcileFastNodeState()` is called immediately before `cacheValueForKey` on the new-root path (size just went from 0 to 1, well below any threshold); the reconcile is a no-op here.

- [`node.go:833-842`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/node.go#L833-L842) — the zero-suffix alias path is correct and the three-index slice cap prevents the obvious foot-gun, but the comment could note that all such zero-suffix keys within the same leaf share the same backing array — a future refactor that drops the `:prefixLen` cap would silently couple them.

- [`split.go:22`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/split.go#L22) — `splitLeaf` takes `inlineMask` as `uint64` (the B+1 overflow case needs bit 31 to survive a left-shift) but the two produced `LeafNode`s store it as `uint32`. The widening/narrowing is correct and documented, but a one-line note at the caller in `leafInsert` would help a future maintainer.

- [`iterator.go:346-362`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/iterator.go#L346-L362) — `loadLeafValues` only resolves the `leafVisitWindow` range; slots outside `[lo, hi)` remain `nil` in `it.leafValues`. If a future iterator change reads backward past `lo`, the nil surfaces silently. Today's `Next` logic preserves the invariant.

## Missing Tests

- **[low-risk]** [`mutable_tree.go:184-186`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/mutable_tree.go#L184-L186) — Fast-node cache off→on re-activation untested
  <details><summary>details</summary>

  `reconcileFastNodeState` transitions back to active when `t.size` drops below the threshold (`!over && t.fastNodesOff`). No test inserts above threshold, removes back below, then verifies a `Get`-hit re-populates and serves correctly. The on→off purge path is implicitly tested. Adversarial test would: set 4*cap+1 keys, confirm `fastNodesOff`, remove until below threshold, confirm a fresh `Set`+`Get` cycle returns a value via the cache (not the tree).
  </details>

- **[low-risk]** [`fastnode_test.go`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/fastnode_test.go) — Adaptive cache with inline values not covered
  <details><summary>details</summary>

  Existing tests cover the external-value path. An inline-slot `Get`-hit on a cached key after a `Set` that crossed the inline threshold (e.g., wrote 70 B with threshold 64 — goes external; then writes 30 B — goes inline; cache must surface the inline copy, not the stale external one). The cache invalidates on `Remove` not on `Set`, and `Set` rewrites the cache entry; this path looks correct but is untested.
  </details>

## Suggestions

- [`const.go:86-95`](../../../../../.worktrees/gno-review-5571/tm2/pkg/bptree/const.go#L86-L95) — Make the `MaxInlineValueThreshold * B + headers + keys <= maxLeafReadBytes` invariant compile-time-checked
  <details><summary>details</summary>

  The doc on `MaxInlineValueThreshold` explains the relationship in prose ("32 * 4 KiB ~= 128 KiB plus keys + headers stays safely below the read cap"). A `var _ = maxLeafReadBytes - B*int(MaxInlineValueThreshold) - leafHeaderBytes` (or similar) at package scope would make any future change to either constant trigger a visible compile error rather than a silent oversize-leaf panic at runtime.
  </details>

## Questions for Author

- The PR's base is `feat/alex/bp32tree-second-pass`, not master. What's the merge plan — does #5570 land first and this rebase onto master, or are both merged as a stack via squash? The 26 commits in this PR include a merge of the base branch (`dd30b0aa0`), which suggests a chain.

- The `proof.go:282` Note explicitly defers the mini-merkle-tearing hazard to "caller must not mutate concurrently." Given how cheap the root clone is (the children stay shared via COW), is there a reason not to close the hazard inside this PR rather than carry it forward?

- The pebbledb `SaveVersion 100K` +51% regression is attributed to flush-timing variance. Was the bench re-run with `count >= 10` to confirm? The same doc cites 8.59 ms vs 11.76 ms across two measurements (38% spread), which suggests the underlying signal isn't clean at count=3.
