# PR #5591: fix(bptree): deep-dive: correctness, safety, and performance fixes

**URL:** https://github.com/gnolang/gno/pull/5591
**Author:** @clockworkgr | **Base:** feat/jae/bp32tree | **Files:** 22 | **+1313 -107**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR is a structured deep-dive review and fix pass over `tm2/pkg/bptree`, delivered as 15 commits on branch `feat/alex/bp32tree-deep-dive`, one per finding. Every behavioral fix ships with a dedicated regression test. The PR targets the `feat/jae/bp32tree` feature branch, not `master`.

### Areas changed

**Correctness fixes (6)**
- `mutable_tree.go` / `nodedb.go`: `SaveVersion` previously called `VersionExists` which swallowed DB errors, treating them as "does not exist" and proceeding to overwrite the existing version with new (potentially different) data. Fixed by adding `versionExistsE` — an error-propagating variant — and wiring it in `SaveVersion`.
- `mutable_tree.go`: Idempotent `SaveVersion` (same version, same hash — typical deterministic block replay) left `sessionValues`, `versionOrphans`, and `nextValueNonce` intact. A subsequent `Rollback` would call `DeleteValueDirect` on session ValueKeys that collide with the already-persisted ones, wiping live values from the DB. Fixed by clearing session state on the idempotent path.
- `prune.go`: `pruneVersion(v, nextV)` only processed `orphans[nextV]` (values displaced when `nextV` was created). When pruning the very first version in a batch, `orphans[first]` (values displaced when `first` was created, from older sessions) were never consumed — permanent value leak. Fixed by iterating `{v, nextV}` in the orphan cleanup loop. The old `deleteAllNodesForVersion` fallback was removed: it walked nodes without processing value-orphan lists, so any version without a successor (which by design cannot happen given the `toVersion < latest` guard) would have leaked every leaf value in that version.
- `nodedb.go` / `mutable_tree.go`: `GetValue` returned `(nil, nil)` when the DB stored an empty value (`[]byte{}`), making a stored empty value indistinguishable from "key not found" on backends that return nil for empty values. Fixed by adding an explicit `Has()` disambiguation path and returning `[]byte{}` for confirmed-present-but-empty keys; missing keys now return a wrapped `ErrKeyDoesNotExist`.
- `node.go`: `LeafNode.Serialize` previously wrote 12 zero bytes as a placeholder for nil `valueKey` slots, producing a `{version:0, nonce:0}` NodeKey on round-trip that silently maps to "value not found" on every subsequent `Get`. Fixed by failing fast with an error instead.
- `node.go`: `ReadNode` did not verify that all bytes were consumed after decoding. Trailing bytes from on-disk corruption would be silently ignored, returning a partially-decoded node. Fixed by checking `r.Len() != 0` after decode.

**Safety fixes (3)**
- `mutable_tree.go` / `nodedb.go`: `Load()` now calls `cleanupCrashedSessionValues`, which scans for ValueKeys with version greater than the latest persisted version and deletes them. These are values written eagerly by `SaveValue` (which bypasses the batch for pre-commit `Get` support) in a session that crashed before `SaveVersion` or `Rollback`. The scan is O(leaked values) via a prefix iterator seeded at `PrefixVal||(latestVersion+1)`, effectively a no-op on clean shutdowns.
- `iterator.go` / `immutable_tree.go`: Iterators created from an `ImmutableTree` never incremented `versionReaders`, so `PruneVersionsTo` could delete the nodes being iterated. Fixed by wiring `ndb.incrVersionReaders` in `ImmutableTree.Iterator` and `NewIteratorWithNDB`, with corresponding `decrVersionReaders` on `Close`. `GetImmutable` now stores the `ndb` reference on the returned `ImmutableTree` so version tracking can target the exact snapshot version.
- `insert.go`, `remove.go`: Key slices stored across different inner nodes during redistribute operations were aliasing each other (the same slice was both in `parent.keys[idx]` and in the child node). Fixed by adding `copyKey` calls at the two transfer sites in `redistributeRight` and `redistributeLeft` for the inner-node branch. The leaf branch already copied correctly.

**Performance fix (1)**
- `mutable_tree.go`: `saveNode` previously called `inner.getChild(i)` for every child of every COW'd inner node, which eagerly loaded all unloaded siblings from the DB just to early-return "already has a NodeKey". For a path-length-H insert with branching B=32, that is H×(B-1) = ~62 unnecessary DB reads per `SaveVersion`. Fixed by iterating only `inner.childNodes[i]` (in-memory) and skipping nil entries (unloaded = unchanged).
- `prune.go`: `PruneVersionsTo` accumulated all deletions into a single batch regardless of prune size. Fixed by flushing the batch whenever it exceeds `FlushThreshold` (from `Options`, default 100 KiB via `DefaultOptions()`), bounding working memory to O(threshold) even during large catch-up prunes.

**Defensive input validation (1)**
- `errors.go`, `mutable_tree.go`, `import.go`: `MaxKeyLen = 1 MiB` constant added. `Set` and `Import.Add` now reject keys exceeding this cap. Without the cap, a key longer than the 1 MiB read limit in `readBytes` would serialize successfully but fail to deserialize on the next load, permanently wedging that version.

**Documentation (1)**
- `README.md`: Corrected the out-of-line-values section to reflect the `ValueKey` scheme (per-allocation `(version, nonce)` identifier) rather than the content-addressed scheme previously described.

**New test files**: `crash_recovery_test.go`, `insert_update_alloc_test.go`, `key_len_test.go`, `node_framing_test.go`, `orphan_first_test.go`, `prune_flush_test.go`, `save_sibling_load_test.go`, `value_missing_test.go`, `version_exists_error_test.go`.

## Test Results

- **Existing tests:** PASS (`go test -count=1 ./tm2/pkg/bptree/...` — all 13.1s, full suite including stress tests)
- **Store integration:** PASS (`go test -count=1 ./tm2/pkg/store/bptree/...`)
- **Edge-case tests:** All 9 new test files pass; tests directly exercise each bug path

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `prune.go:37-38` — The inline 4 MiB fallback (`if flushThreshold <= 0 { flushThreshold = 4 * 1024 * 1024 }`) is dead code: `DefaultOptions()` always sets `FlushThreshold` to 100 KiB, so `opts.FlushThreshold` is never zero when a tree is constructed via `NewMutableTreeWithDB`. The comment in `TestPruneVersionsTo_ZeroFlushThresholdUsesDefault` reads "At the default 4 MiB threshold" which contradicts `options.go:16`. The fallback value (4 MiB) and the real default (100 KiB) are inconsistent — either the fallback should match the default or the fallback should be removed. As written, a caller who somehow passes `FlushThreshold=0` gets a far larger batch than expected from the documented default.

- [ ] `mutable_tree.go:404` — `cleanupCrashedSessionValues` is called only from `Load()`, which only runs on a "load latest" path. `LoadVersion(specificVersion)` (direct, without first calling `Load()`) skips the cleanup. In the current store layer (`store/bptree/store.go:162`) `Load()` is always called before `LoadVersion()`, so the invariant holds in practice. However this dependency is implicit and fragile: a future caller that uses `LoadVersion` directly (e.g., in tests or future tooling) would silently skip crash recovery. Suggest adding a comment on `LoadVersion` documenting that crash cleanup is the caller's responsibility, or add an internal guard.

- [ ] `import.go:100-103` — Separator keys from an untrusted import stream are validated for `> MaxKeyLen` but not for empty (`len == 0`). An empty separator key in an inner node would produce a structurally corrupt tree (inner nodes should never have empty separators — they are the smallest key in the right child subtree, which by the `ErrEmptyKey` guard on `Set` can never be empty for leaf data). Consistent with the leaf-key check added at line 39-43, separator keys should also reject `len == 0`.

## Nits

- [ ] `prune_flush_test.go:99` — Comment says "At the default 4 MiB threshold, a small prune should fit in one batch" but the actual default is 100 KiB (options.go:16). The assertion `flushes > 2` still passes for the right reason (100 KiB threshold is large enough for 10 trivial versions), but the comment is misleading.

- [ ] `nodedb.go:379-419` — `cleanupCrashedSessionValues` calls `itr.Close()` explicitly at line 405 after the `defer itr.Close()` at line 365. The explicit close is intentional (to satisfy the "no writes during iteration" requirement before the delete loop), but `defer itr.Close()` will then call `Close()` a second time. Double-close behavior depends on the DB implementation. Most implementations are idempotent on this, but a comment explaining why the explicit close is needed and why double-close is safe (or suppressing the defer) would reduce confusion.

- [ ] `mutable_tree.go:252-254` — In the idempotent path, `existingEmpty != newEmpty` is checked before the hash comparison. If both are non-empty, the hash comparison catches divergence. If one is empty and the other is not, this returns an error. But `existingHash` could be `nil` if `GetRoot` returns an old-format legacy root (see `nodedb.go:283-285`: `len(data) == 0` returns `nil, nil, nil`). `bytes.Equal(nil, newHash)` evaluates to false only if `newHash` is non-empty, which is correct. This is fine but worth noting that `existingHash` can legitimately be nil for legacy empty roots.

- [ ] `crash_recovery_test.go:130` — `t2 = nil //nolint:wastedassign` followed by `_ = t2` is an unusual pattern. The intent is to make the GC eligible to collect `t2` and signal "crash" to the reader, but `t2 = nil` and the subsequent `_ = t2` don't actually guarantee the object is collected before `t3` opens. The test is still correct because the crash simulation doesn't need actual GC — the point is that `Rollback` was never called. The comment is clear, but the `_ = t2` line is slightly noisy.

## Missing Tests

- [ ] `import.go:100` — No test for an empty (`len == 0`) separator key in an untrusted import stream. Given the test already covers over-max keys, an `imp.Add(&ExportNode{Height: 1, NumKeys: 1, SeparatorKeys: [][]byte{{}}, ...})` case should be added if empty-separator validation is added.

- [ ] `mutable_tree.go:404` — No test for `LoadVersion(specificVersion)` directly (bypassing `Load()`): ensure crash-recovery values are not present (i.e., document that this path does NOT clean up). If the behavior is intentional, a comment-only fix suffices; if it should clean up, a test guards the regression.

## Suggestions

- The `cleanupCrashedSessionValues` implementation iterates the entire `[PrefixVal||(latest+1), PrefixVal+1)` range and materializes all keys into `toDelete` before deleting. For the common case (clean shutdown, zero orphans) this is a no-op range query, which is cheap. However if a process crashed after writing thousands of values, the `toDelete` slice grows unboundedly. Applying the same `FlushThreshold` pattern used in `PruneVersionsTo` would cap memory usage during recovery. This is low priority since crash recovery is a rare, startup-time operation.

- The README update correctly explains `ValueKey` semantics and the no-dedup design. Consider also noting the `MaxKeyLen` constraint in the README since it is now an externally observable limit with a dedicated sentinel error (`ErrKeyTooLong`).

## Questions for Author

- The idempotent-save caveat (line 274-281) says that if replay's allocation order diverges, session VKs don't collide and leak as DB duplicates. Is there a scenario in gno.land's block execution where this divergence can actually happen (e.g., a failed tx that was re-executed)? If so, it might be worth tracking the expected VK count and warning in the log when the leak is detected.

- `deleteAllNodesForVersion` is removed as a "value-leak timebomb." Were there any real-world versions created via this path (e.g., genesis or snapshot import without a next version), or is it guaranteed that `PruneVersionsTo` has always been called only with `toVersion < latest`?

- The `FlushThreshold` default in `DefaultOptions()` (100 KiB) vs. the inline fallback in `PruneVersionsTo` (4 MiB) — was the 4 MiB value chosen intentionally to differ from the default, or is it a leftover from an earlier iteration where `DefaultOptions()` didn't set `FlushThreshold`?

## Verdict

APPROVE — The PR fixes multiple real bugs (DB-error-swallowing in SaveVersion, Rollback-corrupts-DB after idempotent save, iterator-races-with-pruner, value-orphan leak on first-version prune) with well-targeted, well-documented patches and matching regression tests; all tests pass. The warnings are style/documentation issues that do not affect correctness of this PR. The most actionable issue is the 4 MiB vs 100 KiB FlushThreshold inconsistency, which the author should clarify before merge.
