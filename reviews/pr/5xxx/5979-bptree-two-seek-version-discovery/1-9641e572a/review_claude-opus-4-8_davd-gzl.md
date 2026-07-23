# PR [#5979](https://github.com/gnolang/gno/pull/5979): perf(tm2/bptree): find the oldest and newest version with two seeks

URL: https://github.com/gnolang/gno/pull/5979
Author: davd-gzl | Base: master | Files: 3 | +133 -22
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: 9641e572a (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5979 9641e572a`

**TL;DR:** Opening the state tree used to read every version number the node keeps just to learn the smallest and the largest. This replaces that read with two jumps straight to the two ends of the list.

**Verdict: REQUEST CHANGES** — the production path is a verified, very large win, but the same change makes the in-memory backend that gnodev and the integration harness run on about twice as slow at the identical spot (1 Warning, 1 Missing test, 1 Suggestion, 2 Nits).

## Summary

`discoverVersions` scanned every root key to take the min and max. It sits on the per-query path: [`handleQueryCustom`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/sdk/baseapp.go#L533) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/sdk/baseapp.go#L533) and [`Simulate`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/sdk/helpers.go#L71) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/sdk/helpers.go#L71) both open an immutable multistore at a height, and the bptree store's immutable branch calls [`mtree.Load()`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/store/bptree/store.go#L189) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/store/bptree/store.go#L189). With the default `syncable` strategy the node retains [705,600 versions](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/store/types/options.go#L42) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/store/types/options.go#L42), so the scan was 181 ms per query on pebbledb. The fix seeks each end of the `PrefixRoot` range instead, taking it to 54 µs.

Equivalence with the old scan holds. I ran the replaced scan against the new code over 14 key shapes on all four backends, raw and behind the `PrefixDB` the rootmulti store wraps each sub-store in, and they agree everywhere reachable; details in Verified.

## Glossary
- bptree (B+32): the versioned, copy-on-write B+tree backing gno.land's `mainKey` state store.
- fast index: the bptree's optional unauthenticated key-to-value side map.

## Fix

Before, [`discoverVersions`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/nodedb.go#L472-L487) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L472-L487) walked one forward iterator over `[PrefixRoot, PrefixRoot+1)` and tracked the running min and max. After, it calls the new [`edgeRootVersion`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/nodedb.go#L496-L524) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L496-L524) once per direction and returns the first 9-byte key each one lands on. The load-bearing constraint is that [`rootDBKey`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/nodedb.go#L94-L99) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L94-L99) is the only writer under that prefix and always emits `'R'` plus 8 big-endian bytes, so byte order is version order.

## Benchmarks / Numbers

Per-query immutable tree open, A/B by swapping `nodedb.go` between the merge-base and 9641e572a. That file's only delta is `discoverVersions`, confirmed by diff.

pebbledb, the production backend:

| retained versions | merge-base | 9641e572a | change |
|---|---|---|---|
| 1,000 | 141.3 µs | 14.6 µs | 9.7x faster |
| 100,000 | 25.88 ms | 54.3 µs | 477x faster |
| 705,600 | 181.52 ms | 54.1 µs | 3,355x faster |

memdb, the in-memory node and integration harness, 20,000-key tree over 200 versions:

| | merge-base | 9641e572a | change |
|---|---|---|---|
| per-query open | 0.81 ms | 1.68 ms | 2.1x slower |

Discovery alone, three runs of 200 iterations per side:

| backend / size | merge-base | 9641e572a |
|---|---|---|
| pebbledb, 705,600 roots | 138.0 ms | 13.5 µs |
| memdb, 1,000 roots + 50,000 other keys | 0.91 ms | 2.05 ms |
| memdb, 1,000 roots + 200,000 other keys | 9.38 ms | 18.41 ms |

The memdb rows hold the root count fixed at 1,000 and vary everything else, so the cost tracks total store size rather than retention depth.

## Critical (must fix)

None.

## Warnings (should fix)

- **[the same open gets twice as slow on the dev backend]** `tm2/pkg/bptree/nodedb.go:472-487` — memdb has no seek, so the two iterator opens cost two full-map walks where the scan cost one, and the per-query tree open on the in-memory node goes from 0.81 ms to 1.68 ms on a 20,000-key tree.
  <details><summary>details</summary>

  memdb's [`Iterator` and `ReverseIterator` both go through `getSortedKeys`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/db/memdb/mem_db.go#L197-L215) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/db/memdb/mem_db.go#L197-L215), which walks the entire map and sorts the in-domain keys, so an edge seek is a whole-DB materialisation and the doc comment's ["each end is one seek"](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/nodedb.go#L491) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L491) does not hold there. The backend is not a corner case: [`NewInMemoryNode`](https://github.com/gnolang/gno/blob/9641e572a/gno.land/pkg/gnoland/node_inmemory.go#L105) · [↗](../../../../../.worktrees/gno-review-5979/gno.land/pkg/gnoland/node_inmemory.go#L105) is what [gnodev boots](https://github.com/gnolang/gno/blob/9641e572a/contribs/gnodev/pkg/dev/node.go#L649) · [↗](../../../../../.worktrees/gno-review-5979/contribs/gnodev/pkg/dev/node.go#L649) and what the txtar harness runs on, and holding the root count at 1,000 while varying the rest of the store takes discovery from 0.91 ms to 2.05 ms at 50,000 other keys and from 9.38 ms to 18.41 ms at 200,000, so the cost tracks the whole store rather than retention depth.

  Discovery runs twice per open, because [`Load`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/mutable_tree.go#L488) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/mutable_tree.go#L488) calls it and then [`LoadVersion` calls it again](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/mutable_tree.go#L527) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/mutable_tree.go#L527) on the version `Load` just discovered, so the open goes from two map walks to four. Dropping that second call is enough on its own: patched behind a toggle and re-measured at this head, the 20,000-key open comes back to 0.78 ms, level with the merge-base, and pebbledb's discovery halves too. Fix: have `Load` reach a variant of `LoadVersion` that skips the re-discovery, so the open pays for discovery once.
  </details>

## Nits

- **[two spellings of the same key range]** `tm2/pkg/bptree/nodedb.go:497-500` — the `PrefixRoot` range and the 9-byte decode are now written out twice, here and in [`AvailableVersions`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/nodedb.go#L444-L461) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L444-L461); a shared `rootKeyRange()` and decode helper would keep the two from drifting. Not posted, no change needed.
- **[unlabelled boolean at the call site]** `tm2/pkg/bptree/nodedb.go:473` — `edgeRootVersion(false)` and `edgeRootVersion(true)` read as direction only because of the variable they are assigned to. No enabled linter flags it ([`.github/golangci.yml`](https://github.com/gnolang/gno/blob/9641e572a/.github/golangci.yml#L13) · [↗](../../../../../.worktrees/gno-review-5979/.github/golangci.yml#L13) runs `default: none`). Not posted, no change needed.

## Missing Tests

- **[the guard that makes a stray key harmless is unexercised]** `tm2/pkg/bptree/nodedb.go:516-522` — nothing covers the branch that skips a non-9-byte key, so the property that a stray key at either edge cannot stop discovery is asserted nowhere.
  <details><summary>details</summary>

  A coverage run over the package at this head leaves block `518.20,519.12` at zero. The skip is what keeps a short key at the low edge or a long one at the high edge from being decoded, and it is now the difference between reading the right edge and reading whatever sits next to it, so it is worth a case. The equivalence matrix I ran covers it; the four shapes worth keeping are a short key below the first root, a long key above the last, both at once, and a bare `'R'`. Ready-to-add file: [`tests/discover_versions_equivalence_test.go`](tests/discover_versions_equivalence_test.go).
  </details>

## Suggestions

- **[the two ends are read at two different instants]** `tm2/pkg/bptree/nodedb.go:473-477` — the pair now comes from two iterator opens, so on a backend without snapshots a commit and a prune landing between them yield a first and a latest that never coexisted.
  <details><summary>details</summary>

  goleveldb and boltdb both answer `NewSnapshot` with ["snapshots not supported"](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/db/goleveldb/go_level_db.go#L140-L142) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/db/goleveldb/go_level_db.go#L140-L142), so rootmulti falls back to [`ImmutableDB` over the live DB](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/store/rootmulti/store.go#L375-L377) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/store/rootmulti/store.go#L375-L377) and the query connection runs concurrently with block production. With roots 3..100 and a prune of 3..10 plus a commit of 101 landing between the two opens, this head reports first=3 latest=101 while the merge-base reports first=3 latest=100; see [`tests/discover_versions_split_read_test.go`](tests/discover_versions_split_read_test.go).

  Nothing consumes the stale end today: `firstVersion` is read only by [`PruneVersionsTo`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/prune.go#L13) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/prune.go#L13) and [`SaveVersion`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/mutable_tree.go#L425) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/mutable_tree.go#L425), neither of which runs on an immutable store, and the live tree's `Load` happens once at startup. `latest` can only be the fresher of the two, and prune [refuses to delete the latest version](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/prune.go#L15-L17) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/prune.go#L15-L17), so the subsequent `GetRoot(latest)` cannot miss. Fix: say in the function comment that the two ends are read independently, so the next caller that starts trusting the pair as a coherent range knows it is not one.
  </details>

## Verified

- The seek returns what the scan returned. I replayed the replaced scan next to `discoverVersions` over 14 key shapes on memdb, goleveldb, pebbledb and boltdb, raw and behind a `PrefixDB` carrying two neighbouring sub-store root keys, and both agree on every case: sparse gaps, a pruned low end, a pruned high end, stray keys shorter and longer than 9 bytes at each edge, a bare `'R'` key, `'Q'` and `'S'` keys either side of the range, and a DB holding only the other five record prefixes. [`tests/discover_versions_equivalence_test.go`](tests/discover_versions_equivalence_test.go).
- The seek cannot leave the version keyspace. `PrefixRoot` is `'R'` (0x52) so the range ends at `'S'` (0x53), which no other bptree prefix uses ([`const.go:30-39`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/const.go#L30-L39) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/const.go#L30-L39)), and behind the store's `s/k:<name>/` prefix the reverse open still stops before a sibling store's root keys. Both directions confirmed against the four backends above.
- The three inputs where seek and scan disagree are unreachable. A root at version 0 (scan reports the second-smallest, seek reports 0) cannot be written: [`WorkingVersion`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/mutable_tree.go#L712-L717) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/mutable_tree.go#L712-L717) never returns 0 and [`Import`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/import.go#L91-L93) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/import.go#L91-L93) rejects a version at or below the latest. The other two need a version with the high bit set, where the scan's signed comparison and the seek's byte order part ways; that is not a reachable block height.
- The claimed cost is real and the measured shape matches. See Benchmarks; the pebbledb A/B swaps only `nodedb.go`, whose sole delta against the merge-base is this function.
- `go test ./tm2/pkg/bptree/ ./tm2/pkg/store/...` green at 9641e572a.

## Open questions

- [`AvailableVersions`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/bptree/nodedb.go#L443) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/bptree/nodedb.go#L443) still scans. It has no production caller at all, only tests, so nothing on the query path pays for it. Not posted: no action follows.
- `go test -race ./tm2/pkg/db/ -run TestDBIterator` fails with a pebble `mergingIter: lower bound violation`. It comes from the forward `Iterator` at [`pebbledb.go:230`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/db/pebbledb/pebbledb.go#L230) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/db/pebbledb/pebbledb.go#L230) on the inverted domain [`backend_test.go:233`](https://github.com/gnolang/gno/blob/9641e572a/tm2/pkg/db/backend_test.go#L233) · [↗](../../../../../.worktrees/gno-review-5979/tm2/pkg/db/backend_test.go#L233) feeds it. The diff touches no file under `tm2/pkg/db`, and the reverse iterator this PR newly depends on is not on that path. Not posted: unrelated to this diff.
- pebbledb's `ReverseIterator` opens an unbounded `NewIter(nil)` while `Iterator` passes `LowerBound`/`UpperBound`. It costs nothing measurable at 705,600 roots, but it is a free hardening if that file is ever touched. Not posted: separate package, no bearing on this change.
