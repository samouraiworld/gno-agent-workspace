# PR #5438: feat(tm2): immutable B+32 tree — drop-in replacement for IAVL

URL: https://github.com/gnolang/gno/pull/5438
Author: jaekwon | Base: master | Files: 243 | +37938 -2463
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5438 33f5dc340` (then `gh -R gnolang/gno pr checkout 5438` inside it)

**Verdict: REQUEST CHANGES** — bptree library is solid and well-tested in isolation, but the PR bundles ~5 unrelated major refactors (gas model overhaul, MDBX/LMDB backends, stdlib byte cache, PkgID flag nibble, MinDepth governance param) under a misleading title, and ships with 27 failing CI checks plus known data-loss bugs already fixed in stacked follow-ups (#5570, #5591). Should be split, rebased on current master, and CI greened before merge.

## Summary

The PR introduces `tm2/pkg/bptree` (an immutable B+32 tree as drop-in IAVL replacement) and its `tm2/pkg/store/bptree` CommitStore wrapper. Design highlights: B=32 branching factor (~6 levels for 100M items vs IAVL's ~28); out-of-line values addressed by per-allocation `ValueKey{version, nonce}`; binary mini-merkle per node with RFC 6962 domain separators (0x00/0x01/0x02); sentinel short-circuit so empty subtrees collide at every depth (required for ICS23 `EmptyChild`); 90/10 leaf split for append-only keys; stack-based iteration (no leaf sibling pointers, avoiding COW cascades); dual-tree-walk pruning by child-hash set comparison. The bptree code is registered as an alternative proof decoder in `rootmulti` ([`proof.go:135`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/rootmulti/proof.go#L135) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/rootmulti/proof.go#L135)) but production stores remain on IAVL — this PR adds the library, it does **not** switch the chain.

Beyond bptree, the same branch bundles: (1) a complete gas-charging refactor moving I/O metering to the `cache.Store` boundary via a new `GasContext` parameter threaded through every Store method ([`tm2/pkg/store/types/store.go`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/types/store.go) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/types/store.go), [`tm2/pkg/store/cache/store.go`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/cache/store.go) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/cache/store.go)); (2) MDBX + LMDB DB backends ([`tm2/pkg/db/mdbxdb/`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/db/mdbxdb) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/db/mdbxdb), [`tm2/pkg/db/lmdbdb/`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/db/lmdbdb) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/db/lmdbdb)); (3) a stdlib byte cache for gas-free stdlib reads ([`gno.land/pkg/sdk/vm/keeper.go`](https://github.com/gnolang/gno/blob/33f5dc340/gno.land/pkg/sdk/vm/keeper.go) · [↗](../../../../../.worktrees/gno-review-5438/gno.land/pkg/sdk/vm/keeper.go)); (4) PkgID flag nibble distinguishing stdlib/immutable/internal packages ([`gnovm/pkg/gnolang/ownership.go`](https://github.com/gnolang/gno/blob/33f5dc340/gnovm/pkg/gnolang/ownership.go) · [↗](../../../../../.worktrees/gno-review-5438/gnovm/pkg/gnolang/ownership.go)); (5) `MinDepth` governance parameter ([`gno.land/pkg/sdk/vm/params.go`](https://github.com/gnolang/gno/blob/33f5dc340/gno.land/pkg/sdk/vm/params.go) · [↗](../../../../../.worktrees/gno-review-5438/gno.land/pkg/sdk/vm/params.go)); (6) op-handler gas calibration via dedicated benchmarks ([`gnovm/cmd/calibrate/`](https://github.com/gnolang/gno/blob/33f5dc340/gnovm/cmd/calibrate) · [↗](../../../../../.worktrees/gno-review-5438/gnovm/cmd/calibrate)); (7) `KeepEvery > 1` deprecation across IAVL and bptree ([`tm2/pkg/store/types/options.go:25-27`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/types/options.go#L25-L27) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/types/options.go#L25-L27)). The PR title and description discuss only the tree.

Gas costs for `TestAddPkgDeliverTx` jumped from `226_738` to `18_651_076` ([`gno.land/pkg/sdk/vm/gas_test.go:75`](https://github.com/gnolang/gno/blob/33f5dc340/gno.land/pkg/sdk/vm/gas_test.go#L75) · [↗](../../../../../.worktrees/gno-review-5438/gno.land/pkg/sdk/vm/gas_test.go#L75)) — an 82× increase driven by the gas refactor, not the tree. Anchor: existing testnet `MaxGasPerBlock` budgets need re-calibration before this can be merged.

## Glossary

- **ValueKey (VK)**: 12-byte `{version:8, nonce:4}` identifier for an out-of-line value. Leaves store keys + value hashes + VKs; raw values live under `V<vk>` in the DB.
- **NodeKey (NK)**: same shape as ValueKey, identifies a stored inner/leaf node under `B<nk>`.
- **Mini merkle**: in-memory binary merkle tree of size `2*B` over a node's slot hashes. Root is the node's `Hash()`. Not serialized — recomputed on load.
- **Sentinel hash**: `SHA256(0x02)`. Domain-separated value for empty mini-merkle slots; short-circuits so empty subtrees at any depth produce the same hash (ICS23 `EmptyChild` compatibility).
- **Tier 1 / Tier 2 orphans**: intra-version VKs are deleted eagerly on overwrite/remove; cross-version VKs are persisted to an orphan list at `O<version>` and deleted by `PruneVersionsTo`.
- **Dual-tree-walk pruning**: comparing version `v` against the next-existing version `nextV` by descending both roots and skipping subtrees whose child-hash sets match.

## Fix

The PR adds `bptree` + `store/bptree` packages and registers `BptreeCommitmentOpDecoder` in `rootmulti`'s proof runtime ([`tm2/pkg/store/rootmulti/proof.go:135`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/rootmulti/proof.go#L135) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/rootmulti/proof.go#L135)). Production mounts ([`gno.land/pkg/gnoland/app.go`](https://github.com/gnolang/gno/blob/33f5dc340/gno.land/pkg/gnoland/app.go) · [↗](../../../../../.worktrees/gno-review-5438/gno.land/pkg/gnoland/app.go), [`tm2/pkg/sdk/bank/common_test.go`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/sdk/bank/common_test.go) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/sdk/bank/common_test.go), all `_test.go`) continue to use `iavl.StoreConstructor` — `bptree.StoreConstructor` is exposed but never wired. The library is therefore a parallel implementation, not a migration. Separately the PR rewrites cache/gas accounting at [`tm2/pkg/store/cache/store.go`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/cache/store.go) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/cache/store.go), threads a new `GasContext` through every Store method ([`tm2/pkg/store/types/store.go`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/types/store.go) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/types/store.go)), and recalibrates per-op gas in `gnovm/pkg/gnolang/op_*.go` based on the new calibration benchmarks.

## Benchmarks / Numbers

| Operation (100M items, 10K node cache) | B+32 | IAVL | IAVL+fast |
|---|---|---|---|
| GET reads | 3 | 15 | 1 |
| SET reads | 2 | 15 | 15 |
| SET writes | 7 | 28 | 29 |
| SET total ops | 9 | 43 | 44 |

| Gas test (gas_test.go) | Before | After | Δ |
|---|---|---|---|
| `TestAddPkgDeliverTxInsuffGas` charge | 3,462 | 3,404,387 | 983× |
| `TestAddPkgDeliverTx` deliver | 226,738 | 18,651,076 | 82× |
| `TestAddPkgDeliverTxFailed` deliver | 1,231 | 1,240,309 | 1,007× |
| `gnokey_gasfee.txtar` addpkg estimate | 269,942 | 269,648 | -0.1% |
| `restart_gas.txtar` addpkg | 593,591 | 586,837 | -1.1% |

The 82–1000× shift is the gas refactor (boundary metering + amino-encode/decode cost model + alloc-gas calibration), not the tree. Integration `.txtar` numbers stay flat because production stores still use IAVL.

## Critical (must fix)

- **[PR scope is misleading]** PR title / body describe a tree; diff contains 5+ unrelated major refactors
  <details><summary>details</summary>

  The PR header says "drop-in replacement for IAVL", but [the commit list](../../../../../.worktrees/gno-review-5438) includes `feat(gnovm): stdlib byte cache for gas-free stdlib object reads`, `feat(gnovm): skip refcount mutations on immutable package objects`, `feat(gnovm): add PkgID flag nibble for stdlib/immutable/internal`, `feat(vm): add MinDepth governance parameter`, `feat(store): calibrate gas constants from LMDB benchmarks`, `refactor(store): delete gas.Store package`, `feat(gnovm): replace per-operation gas constants with amino encode/decode`, `feat(gnovm): thread GasContext through Gno store operations`, `feat(store): wire up gas charging at cache.Store boundary`, `refactor(store): add GasContext parameter to Store interface methods`, `add MDBX database backend`, and `add LMDB backend`. None of these are bptree changes. Reviewing them as a unit is intractable; each affects a different invariant. The gas-refactor changes alone (82× to 1000× gas-cost shifts in existing tests) need an isolated debate about consensus-breaking economic impact. Fix: split into separate PRs — (1) bptree library, (2) store-boundary gas refactor, (3) DB backends, (4) stdlib cache + PkgID flag, (5) governance params. Each can land independently.
  </details>

- **[CI is red across 27 jobs]** `gh pr checks 5438` shows 27 failing jobs including TM2 Test, GnoVM Test, gno.land Test, lint, proto-gen, go.mod-tidy
  <details><summary>details</summary>

  `Run TM2 suite / Go Test`, `Run gno.land suite / Go Test`, `Run GnoVM suite / Go Test`, `Run Main (gnodev|gnobro|gnogenesis|gnokeykc|gnomigrate|tx-archive) Go Test/Lint`, `Ensure .proto files are updated`, `Ensure go.mods are tidied`, `stdlibs Run tests/fmt/lint`, `benchops`, `Run GnoVM Go Build / generated` are all failing on the current HEAD. Some of these are likely pre-existing on the long-lived `feat/jae/bp32tree` branch (master moved 186 commits ahead of merge-base); others are introduced. There is no signal until CI is green. Fix: rebase on current master, regenerate proto, tidy go.mod, address lint, then re-evaluate test failures.
  </details>

- **[VersionExists swallows DB errors → overwrites previous version]** [`tm2/pkg/bptree/nodedb.go:303-306`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/nodedb.go#L303-L306) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/nodedb.go#L303-L306) — already fixed in stacked PR #5591
  <details><summary>details</summary>

  ```go
  func (ndb *nodeDB) VersionExists(version int64) bool {
      has, _ := ndb.db.Has(rootDBKey(version))
      return has
  }
  ```
  A DB read error returns `(false, err)`; `err` is discarded and `VersionExists` returns `false`. `SaveVersion` ([`mutable_tree.go:227`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L227) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L227)) then concludes the version does not exist and proceeds to write — silently overwriting the existing version's nodes with a different root. Block replay or any deterministic re-execution can produce DB corruption. PR #5591 ships `versionExistsE` (error-returning variant) plus an explicit fail-on-error path in `SaveVersion`; the same fix must be in this PR before merge. Fix: thread `error` through `VersionExists` (or add a sibling `VersionExistsE`) and treat any DB error as a hard fail in `SaveVersion`, not as "version does not exist".
  </details>

- **[Idempotent SaveVersion leaks session state into next Rollback]** [`tm2/pkg/bptree/mutable_tree.go:246-250`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L246-L250) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L246-L250) — already fixed in #5591
  <details><summary>details</summary>

  On the idempotent path (same version saved again with same hash, the typical deterministic block-replay case), the code returns early without clearing `sessionValues`, `versionOrphans`, or `nextValueNonce`. A subsequent `Rollback` ([`mutable_tree.go:563-566`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L563-L566) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L563-L566)) iterates `sessionValues` and `DeleteValueDirect`-deletes them from the DB. Because session VKs are allocated from `WorkingVersion()`, they collide bit-for-bit with the already-persisted VKs of the just-replayed version — wiping live values from the DB. The persisted tree still references VKs whose value bytes no longer exist. Fix: clear `sessionValues`, `versionOrphans`, `nextValueNonce` on every `SaveVersion` exit path, including the idempotent early-return.
  </details>

- **[deleteAllNodesForVersion leaks all leaf values when there is no successor]** [`tm2/pkg/bptree/prune.go:33-36`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/prune.go#L33-L36) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/prune.go#L33-L36) — already removed in #5591
  <details><summary>details</summary>

  When pruning a version with no later version, `PruneVersionsTo` falls back to `deleteAllNodesForVersion` ([`prune.go:255-292`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/prune.go#L255-L292) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/prune.go#L255-L292)), which walks the node tree and deletes nodes but never touches the orphan-value list. Every leaf value referenced by that version becomes permanent dead bytes in the value DB. The current `toVersion >= latest` guard at [`prune.go:16-18`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/prune.go#L16-L18) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/prune.go#L16-L18) makes the "no successor" case theoretically unreachable from the public API, but `findNextVersion` returning 0 ([`prune.go:57-64`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/prune.go#L57-L64) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/prune.go#L57-L64)) sets up the same fallback if a gap appears mid-range. PR #5591 removes `deleteAllNodesForVersion` and processes orphans for the first pruned version even when its predecessor is absent. Fix: adopt #5591's approach.
  </details>

- **[Iterate silently swallows value-resolution errors]** [`tm2/pkg/bptree/mutable_tree.go:630-641`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L630-L641) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L630-L641) + [`immutable_tree.go:114-128`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/immutable_tree.go#L114-L128) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/immutable_tree.go#L114-L128) — raised in PR thread by clockworkgr
  <details><summary>details</summary>

  Both `MutableTree.Iterate(fn)` and `ImmutableTree.Iterate(fn)` resolve each value via `t.resolveValue(vk)` inside a closure. If the resolver returns an error, the closure returns `true` (stop iteration), and the outer call returns `(true, nil)`. From the caller's perspective, iteration completed successfully or `fn` signalled stop — exactly the same as a normal terminate. A missing-value DB error is indistinguishable from "user stopped iterating". For a node verifying integrity, this is silent data loss. The low-level `bp.Iterator` does propagate `it.err` via `Error()` — so callers that go through `Store.Iterator` are safe — but anyone calling `MutableTree.Iterate` directly is not. Fix: change the high-level signatures to `(bool, error)` returning the resolver error, or stop iteration with a panic on resolver error.
  </details>

- **[`(*InnerNode).getChild` panics on any GetNode error including legitimate "not found"]** [`tm2/pkg/bptree/node.go:72-76`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/node.go#L72-L76) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/node.go#L72-L76)
  <details><summary>details</summary>

  The lazy-load slow path treats every `ndb.GetNode` error as an unrecoverable corruption and crashes the process: `panic(fmt.Sprintf("bptree: failed to load child node %x: %v", n.children[idx], err))`. `GetNode` ([`nodedb.go:128-134`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/nodedb.go#L128-L134) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/nodedb.go#L128-L134)) returns `fmt.Errorf("node not found: %x", nkBytes)` for any DB-miss — including a stale ref to a node that was just pruned, a child not yet committed, or a transient backend error. In a validator running both reads and pruning concurrently, a single race here kills the node. PR #5570 narrowed the equivalent panic to exclude `ErrNodeNotFound`; this PR's `getChild` is the same shape pre-narrowing. Fix: distinguish "node not found" (return nil, let caller decide) from "DB error" (panic) and surface the difference via a typed error in `GetNode`.
  </details>

## Warnings (should fix)

- **[`splitInner` separator aliases the source key slice]** [@notJoon](https://github.com/gnolang/gno/pull/5438#discussion_r2058225789) [`tm2/pkg/bptree/split.go:69`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/split.go#L69) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/split.go#L69) — `sep := keys[splitPoint]` returns a shared slice header
  <details><summary>details</summary>

  `splitLeaf` (line 41-43) explicitly copies the separator: `sep := make([]byte, len(right.keys[0])); copy(sep, right.keys[0])`. `splitInner` skips the copy: `sep := keys[splitPoint]`. The `keys[]` argument was assembled in `innerInsert` ([`insert.go:168-170`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/insert.go#L168-L170) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/insert.go#L168-L170)) by `copy(allKeys[:childIdx], inner.keys[:childIdx])` — copying slice headers, not backing arrays. The promoted separator therefore still aliases one of the parent's existing `keys[i]` (or the just-promoted leaf-split separator). Keys are not mutated in-place under the current code, so this is benign today, but it's a class of footgun a future refactor will trip on. Fix: copy the separator unconditionally, matching `splitLeaf`.
  </details>

- **[Inner-redistribute paths leave promoted-key aliasing across siblings]** [`tm2/pkg/bptree/remove.go:210-216`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/remove.go#L210-L216) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/remove.go#L210-L216) + [`remove.go:271-278`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/remove.go#L271-L278) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/remove.go#L271-L278)
  <details><summary>details</summary>

  In the InnerNode branches of `redistributeRight`/`redistributeLeft`, the demoted parent separator becomes the new first/last key of a sibling without an explicit `copyKey()`: `r.keys[0] = parent.keys[idx]; ...; parent.keys[idx] = l.keys[lastKeyIdx]`. After the operation, `l.keys[lastKeyIdx]` is `nil`'d ([line 217](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/remove.go#L217) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/remove.go#L217)) and `parent.keys[idx]` holds the new promoted key — but the slice header taken out of `parent.keys[idx]` before the reassignment is now living inside `r.keys[0]`, sharing the parent's backing array. The leaf branches above ([line 184](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/remove.go#L184) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/remove.go#L184), [line 258](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/remove.go#L258) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/remove.go#L258)) do `parent.keys[idx] = copyKey(r.keys[0])`. PR #5591 added `copyKey` at exactly these inner-branch sites as one of its safety fixes. Fix: copy keys on every parent↔child key transfer.
  </details>

- **[`FlushThreshold` and `AsyncPruning` Option fields exist but are never consumed]** [@clockworkgr](https://github.com/gnolang/gno/pull/5438#discussion_r2057848211) [`tm2/pkg/bptree/options.go:7-8`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/options.go#L7-L8) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/options.go#L7-L8)
  <details><summary>details</summary>

  Only `opts.Sync` (used in [`nodedb.go:407`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/nodedb.go#L407) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/nodedb.go#L407)) and `opts.InitialVersion` (used in [`mutable_tree.go:66`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L66) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L66)) are read. `FlushThreshold` is set to `100 * 1024` in `DefaultOptions()` but no code path inspects it (a follow-up PR #5591 then *uses* it in `PruneVersionsTo`). `AsyncPruning` is never used. Exposing dead options in the public API is a documentation hazard and locks the field names. Fix: drop the unused fields, add them back when their consumers land. The same comment was raised by `clockworkgr` and is unresolved.
  </details>

- **[Open `notJoon`/`clockworkgr` GetNode error-handling thread unresolved across 7 review comments]** [PR review thread](https://github.com/gnolang/gno/pull/5438#pullrequestreview-3304192800)
  <details><summary>details</summary>

  clockworkgr filed 7 separate `same`/`Again` comments asking the author to handle (not silently swallow) `GetNode` errors in `prune.go`, `nodedb.go`, and `node.go`. The author replied to a single instance suggesting `ErrNodeNotFound` as a typed error; no follow-up commits address the broader pattern. PR #5591 ships the structural fix (split `data == nil` → `ErrNodeNotFound`, DB error → panic). Until that's in this PR, every silent skip during pruning is a potential lost orphan-cleanup. Fix: either merge the #5591 error-semantics change before this PR, or sequence the merge so #5591 lands first and this PR rebases on top.
  </details>

- **[`KeepEvery > 1` panic in `NewPruningOptions` is bypassable via struct literal]** [`tm2/pkg/store/types/options.go:25-27`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/types/options.go#L25-L27) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/types/options.go#L25-L27)
  <details><summary>details</summary>

  ```go
  func NewPruningOptions(keepRecent, keepEvery int64) PruningOptions {
      if keepEvery > 1 { panic("...") }
      ...
  }
  ```
  But `PruningOptions{KeepEvery: 5}` constructs a valid struct that bypasses the panic and is then consumed by [`tm2/pkg/store/iavl/store.go:105`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/iavl/store.go#L105) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/iavl/store.go#L105) / [`bptree/store.go:106`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/bptree/store.go#L106) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L106): `if st.opts.KeepEvery == 0 || toRelease%st.opts.KeepEvery != 0 { /* skip pruning */ }` — meaning the broken waypoint logic still runs for any caller that constructs the struct directly. [`tm2/pkg/store/prefix/store_test.go:96`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/prefix/store_test.go#L96) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/prefix/store_test.go#L96) and [`tm2/pkg/store/iavl/store_test.go:614`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/iavl/store_test.go#L614) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/iavl/store_test.go#L614) already do this. Fix: validate at the `Store.Commit()` call site, not only the constructor; or make `PruningOptions` unexported with the constructor as the only entry point.
  </details>

- **[`SetCommitting`/`UnsetCommitting` are dead code]** [`tm2/pkg/bptree/mutable_tree.go:546-557`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L546-L557) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L546-L557) + [`nodedb.go:388-398`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/nodedb.go#L388-L398) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/nodedb.go#L388-L398)
  <details><summary>details</summary>

  `isCommitting` is set and unset but never read by any code path. PR #5570 removes it. Until then it's surface-area lock-in. Fix: delete the methods and the field.
  </details>

- **[Stdlib byte cache `PopulateStdlibCache` runs after `MultiWrite` with no documented gas accounting]** [`gno.land/pkg/gnoland/app.go:350-354`](https://github.com/gnolang/gno/blob/33f5dc340/gno.land/pkg/gnoland/app.go#L350-L354) · [↗](../../../../../.worktrees/gno-review-5438/gno.land/pkg/gnoland/app.go#L350-L354)
  <details><summary>details</summary>

  The new `cfg.vmk.PopulateStdlibCache()` call is added after `msCache.MultiWrite()` in `loadStdlibs`. Cached objects are then read gas-free per the commit message. No test verifies that the same objects, when read at runtime, charge zero — or that the cache survives a chain restart. If the cache is process-local, a node restart re-charges gas for stdlib reads inconsistently with peers that have warm caches. Fix: confirm cache is deterministic/genesis-bound, add a test that proves zero-gas after warmup AND after restart.
  </details>

- **[Gas-test deltas point at consensus-breaking economic change with no migration plan]** [`gno.land/pkg/sdk/vm/gas_test.go:75`](https://github.com/gnolang/gno/blob/33f5dc340/gno.land/pkg/sdk/vm/gas_test.go#L75) · [↗](../../../../../.worktrees/gno-review-5438/gno.land/pkg/sdk/vm/gas_test.go#L75) and surrounding tests
  <details><summary>details</summary>

  `TestAddPkgDeliverTx` went from 226,738 gas to 18,651,076 gas — an 82× jump. `TestAddPkgDeliverTxInsuffGas` jumped 983×. The integration `.txtar` numbers stayed nearly flat (`gnokey_gasfee` 269,942 → 269,648; `restart_gas` 593,591 → 586,837) because they exercise the IAVL store and the .txtar pipeline doesn't yet route through the new boundary. The 82× shift is therefore latent — it will materialize whenever production switches to the new cache.Store gas charging. Anchor: testnet `MaxGasPerBlock` is currently 10,000,000 ([gno.land/genesis](https://github.com/gnolang/gno/blob/master/gno.land/genesis/genesis_balances.txt)); an 18M tx exceeds the entire block budget. This needs an explicit migration: either re-calibrate gas constants downward, or coordinate a chain-halt + governance vote to raise the limits. Fix: add a clear migration note in the PR body, ideally a runnable upgrade path.
  </details>

## Nits

- [`tm2/pkg/bptree/node.go:215`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/node.go#L215) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/node.go#L215) — `LeafNode.Serialize` writes 12 zero bytes when `valueKeys[i] == nil`; round-trip gives a `NodeKey{0,0}` that silently maps to "value not found" on `Get`. PR #5591 fails fast with an error instead. Should match.

- [`tm2/pkg/bptree/node.go:223-237`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/node.go#L223-L237) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/node.go#L223-L237) — `ReadNode` does not assert `r.Len() == 0` after deserialization; trailing on-disk garbage from a partial write is silently accepted. PR #5591 adds the check.

- [`tm2/pkg/bptree/mini_merkle.go:80-88`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mini_merkle.go#L80-L88) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mini_merkle.go#L80-L88) — `miniMerkleDepth()` recomputes `log₂(B)` at runtime even though B is a compile-time constant; `const MiniMerkleDepth = 5` would suffice. Raised by clockworkgr.

- [`tm2/pkg/bptree/mutable_tree.go:76-78`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L76-L78) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L76-L78) — `Set` rejects nil value with `fmt.Errorf("value must not be nil")`; should be a sentinel error matching `ErrEmptyKey`.

- [`tm2/pkg/bptree/iterator.go:271-276`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/iterator.go#L271-L276) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/iterator.go#L271-L276) — `Iterator.Key()` returns the raw `leaf.keys[]` slice; caller can corrupt the tree by `append`-extending past the cap. IAVL's iterator copies. Document or copy.

- [`tm2/pkg/bptree/prune.go:170-172`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/prune.go#L170-L172) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/prune.go#L170-L172) — `if oldInner.children[i] == nil { continue }` comment says "shouldn't happen for saved nodes" but takes no action to verify. A `panic` here on a saved-tree path would catch corruption fast.

- [`tm2/pkg/store/bptree/store.go:99`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/store/bptree/store.go#L99) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L99) — `Store.Commit()` panics on `SaveVersion` error. Matches IAVL convention but worth a comment that the panic crashes the validator.

- `keys[mid]` slice indexing in `searchLeaf`/`searchInner` ([`tm2/pkg/bptree/search.go:12`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/search.go#L12) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/search.go#L12)) without bounds annotation reads cleanly for B=32, but loops over keys-array-of-fixed-size relying on `numKeys` upper-bound; an assertion at function entry would document the precondition.

## Missing Tests

- **[idempotent SaveVersion + Rollback]** [`tm2/pkg/bptree/mutable_tree.go:246`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/mutable_tree.go#L246) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L246) — no test exercises `SaveVersion(v) → SaveVersion(v) → Rollback()` and verifies that values from the persisted version survive. This is exactly the silent-DB-corruption scenario PR #5591 fixes.

- **[concurrent prune + reader]** no `-race` test verifies that pruning version `v-2` while an `ImmutableTree` reader at `v-1` is walking nodes is safe. The `versionReaders` map ([`nodedb.go:363-378`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/nodedb.go#L363-L378) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/nodedb.go#L363-L378)) tracks counts but `ImmutableTree.Iterator` does not `incrVersionReaders` (only `bp.Iterator` does, via the `version` parameter at [`iterator.go:329`](https://github.com/gnolang/gno/blob/33f5dc340/tm2/pkg/bptree/iterator.go#L329) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/iterator.go#L329) where `version` is `0` — the no-op path). An `ImmutableTree.Iterator` thus does NOT register, and a concurrent prune can race-free delete its nodes.

- **[`Iterate` resolver error]** no test ensures that an iterator over a version with missing values surfaces the error to the caller — current code returns `(true, nil)` indistinguishable from clean stop.

- **[`KeepEvery` struct-literal bypass]** no test confirms `Store.Commit` panics or errors when `opts.KeepEvery > 1` is provided directly (bypassing `NewPruningOptions`).

- **[gas migration impact]** no end-to-end test demonstrates a real chain at block N committing under the new gas model and querying state with the IAVL store — to surface the latent 82× cost shift before the production switch.

## Suggestions

- Sequence-merge PR #5591 (the deep-dive audit fixes) into this branch *before* this PR is considered. The audit's 15 commits each ship a regression test; landing them on top of a still-buggy base is harder to bisect than baking them in.

- The B=32 design depends on `B == 1<<MiniMerkleDepth` (asserted at runtime in `init()`); replace with `var _ [B - 1<<5]byte` for a compile-time check, eliminating a panic path.

- `DefaultOptions().FlushThreshold = 100 * 1024` is dead until #5591 lands. Either remove the field now or wire it up to `nodedb.Commit`'s batch threshold in this PR.

- The bptree `Store` wrapper duplicates a lot of iavl `Store` boilerplate. After both implementations stabilize, a `CommitStore[T Tree]` generic would reduce the divergence surface. Out of scope but worth noting.

- Document in `PLAN.md` whether the on-disk format is considered frozen; the value-key + orphan-list layout has changed several times across the 113 commits in this branch and an early adopter would benefit from a "format version" entry.

## Questions for Author

- Why bundle the gas refactor, MDBX/LMDB backends, stdlib cache, and PkgID flag into the bptree PR? Each is a substantial standalone change. Is the intent to land the whole stack atomically with a single migration?

- Production stores still use IAVL ([`gno.land/pkg/gnoland/app.go`](https://github.com/gnolang/gno/blob/33f5dc340/gno.land/pkg/gnoland/app.go) · [↗](../../../../../.worktrees/gno-review-5438/gno.land/pkg/gnoland/app.go) mounts via `iavl.StoreConstructor`). What's the cutover plan for bptree — a separate PR, a genesis flag, or a runtime config?

- The 82× gas-cost jump in `TestAddPkgDeliverTx` (226K → 18M) exceeds the current 10M `MaxGasPerBlock` on testnet. Is this expected to coincide with a governance bump? Without one, the chain halts on first AddPkg after deploy.

- PRs #5570 / #5591 ship 60+ correctness fixes for `bptree`. Is the merge order PR #5438 → #5591 → #5570, or is the plan to squash all three into one merge to master? If the former, the issues in this review (VersionExists, idempotent-save, deleteAllNodesForVersion) live in master between merges; if the latter, the audit trail collapses.

- `getChild` panics on every `GetNode` error including stale refs. Was the prune-reader invariant (active-readers check before delete) designed to make that race unreachable in practice? If so, a comment + test would document the guarantee.
