# PR #5827: fix(gnovm): ignore make(map, n) size hint

URL: https://github.com/gnolang/gno/pull/5827
Author: thehowl | Base: master | Files: 8 | +84 -24
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 96c9a8ce0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5827 96c9a8ce0`

**TL;DR:** In the GnoVM, `make(map[K]V, n)` used to charge gas for `n` map items up front, but each item also pays again when actually inserted, and a giant `n` crashed the whole transaction with an internal overflow panic. This PR stops reading the size hint at all: the map is created empty and you pay per item only as you fill it. The hint never affected stored data anyway (it is recomputed from the real item count when state reloads), so nothing of value is lost.

**Verdict: APPROVE** — clean, well-reasoned simplification that fixes the double-charge, the overflow panic, and an unmetered Go-level preallocation in one stroke; Go parity on the edge contract is exact. One pre-merge snag: the new filetest `make19.gno` collides with the still-open, independent [#5723](https://github.com/gnolang/gno/pull/5723).

## Summary
`make(map[K]V, n)` routed the hint through `NewMap` → `AllocateMap`, charging `allocMap + allocMapItem*n` before a single item exists. Each item is then charged a second `allocMapItem` on real insertion via [`MapList.Append`](https://github.com/gnolang/gno/blob/96c9a8ce0/gnovm/pkg/gnolang/values.go#L726) · [↗](../../../../../.worktrees/gno-review-5827/gnovm/pkg/gnolang/values.go#L726), so a hinted-then-filled map double-pays the per-item cost; and a hint large enough to overflow `allocMapItem*n` panicked with an unrecoverable `"multiplication overflow"`. The fix ignores the hint end-to-end: [`AllocateMap()`](https://github.com/gnolang/gno/blob/96c9a8ce0/gnovm/pkg/gnolang/alloc.go#L369-L374) · [↗](../../../../../.worktrees/gno-review-5827/gnovm/pkg/gnolang/alloc.go#L369) charges only the header, `NewMap`/`MakeMap` drop their size params, and the [`make()` map builtin](https://github.com/gnolang/gno/blob/96c9a8ce0/gnovm/pkg/gnolang/uverse.go#L1140-L1158) · [↗](../../../../../.worktrees/gno-review-5827/gnovm/pkg/gnolang/uverse.go#L1140) no longer reads the hint. This is a strictly better alternative to #5770 (which clamped only the int64-overflow corner and left the double-charge and the sub-pivot Go preallocation in place).

## Glossary
- **hint** — the optional size argument to `make(map, n)`; advisory in Go, never affects `len`.
- **allocMap / allocMapItem** — alloc-gas constants: the map header charge (`_allocHeap + 168`) and the per-item charge (`2 * _allocTypedValue` = 80 bytes).
- **double-charge** — the bug: items billed once up front (per hint) and again on insertion.

## Fix
Before: [`NewMap(t, size)`](https://github.com/gnolang/gno/blob/96c9a8ce0/gnovm/pkg/gnolang/alloc.go#L601) · [↗](../../../../../.worktrees/gno-review-5827/gnovm/pkg/gnolang/alloc.go#L601) fed the hint into `AllocateMap(int64(size))`, which ran `Addp(allocMap, Mulp(allocMapItem, size))` — overflowing for huge hints — and into `MakeMap(size)` → `make(map[MapKey]*MapListItem, size)`, preallocating Go buckets sized to a hint that gets thrown away on the next state reload. After: every map charges exactly `allocMap` at creation and one `allocMapItem` per real insert; the hint argument is still evaluated upstream (so any side effects survive) but its value is discarded. The load-bearing fact is that the hint is never persisted — on recovery the map is rebuilt from its actual item count at [`realm.go:1895`](https://github.com/gnolang/gno/blob/96c9a8ce0/gnovm/pkg/gnolang/realm.go#L1895) · [↗](../../../../../.worktrees/gno-review-5827/gnovm/pkg/gnolang/realm.go#L1895) (`make(..., cv.List.Size)`), so honoring it bought nothing durable while costing a double-charge and an overflow surface.

## Benchmarks / Numbers
`make(map, n)` then fill with `m` items:

| | up-front | per-insert | total |
|---|---|---|---|
| old (master) | `allocMap + allocMapItem*n` | `allocMapItem*m` | `allocMap + allocMapItem*(n+m)` |
| new (PR) | `allocMap` | `allocMapItem*m` | `allocMap + allocMapItem*m` |

Edge hints, single `make()`:

| hint | old (master) | new (PR) |
|---|---|---|
| `0` / none (incl. all map literals) | `allocMap` | `allocMap` (unchanged) |
| `MaxInt` | panic `"multiplication overflow"` (unrecoverable) | `allocMap`, usable empty map |
| `-1` | `allocMap - allocMapItem` (under-charge) | `allocMap` |

## Critical (must fix)
None.

## Warnings (should fix)
- **[two open PRs add the same new filetest with different contents]** `gnovm/tests/files/make19.gno:1` — The still-open, independent [#5723](https://github.com/gnolang/gno/pull/5723) (ltzmaxwell, slice-overflow `recover()` test) also adds `gnovm/tests/files/make19.gno`, with different contents; whichever lands second hits an add/add conflict. Fix: rename this filetest to the next free slot (`make20.gno`).
  <details><summary>details</summary>

  Master tops out at `make18.gno` (confirmed: `git cat-file -e origin/master:gnovm/tests/files/make19.gno` reports the path absent). Three open PRs — #5723, #5770, and this one — each independently claim `make19.gno`; #5770 is the one #5827 replaces, so the live collision is #5827 vs the cross-author #5723. Both PRs also edit `alloc.go`, but in different functions (`#5723` near `Fork`/`AllocatePointer`, this PR at `AllocateMap`/`NewMap`), >3 lines apart, so that file auto-merges; only the `make19.gno` add/add is a hard conflict. Renaming this one to `make20.gno` clears it.
  </details>

## Nits
None.

## Missing Tests
None. [`TestNewMapChargesHeaderOnly`](https://github.com/gnolang/gno/blob/96c9a8ce0/gnovm/pkg/gnolang/alloc_test.go#L35) · [↗](../../../../../.worktrees/gno-review-5827/gnovm/pkg/gnolang/alloc_test.go#L35) pins the header-only creation charge plus the per-item insertion charge, and since `NewMap` no longer takes a hint it is now structurally impossible to over-charge on creation. The [`make19.gno`](https://github.com/gnolang/gno/blob/96c9a8ce0/gnovm/tests/files/make19.gno#L1-L26) · [↗](../../../../../.worktrees/gno-review-5827/gnovm/tests/files/make19.gno#L1) filetest covers the negative and `MaxInt` hints end-to-end (no panic, map stays usable).

## Suggestions
None.

## Open questions
- The "only ever lowers map cost" monotonicity claim in the PR description holds for hints `>= 0` but is reversed for negative hints: on master `make(map[int]int, -1)` charged `allocMap - allocMapItem` (an under-charge, 80 bytes below the header), and the PR raises it to `allocMap`. The increase is bounded by one `allocMapItem` and only ever fires on a deliberately negative runtime hint, so no realistic transaction is affected; the new charge is also the more-correct one (never below the header). Not posted: pathological corner, the code is right, only the description's blanket wording is slightly imprecise.

---

Verified on 96c9a8ce0:
- Go parity is exact: a real Go program with `make(map[int]int, -1)` and `make(map[int]int, MaxInt)` prints `neg: 0 / huge: 0 / usable: 10 20 1 1`, byte-identical to the filetest's `// Output:` block.
- The overflow fix removes an unrecoverable crash, not just a panic-message change: on master `make(map[int]int, MaxInt)` panics with the bare Go string `"multiplication overflow"` inside `AllocateMap`'s `Mulp`, which Gno `recover()` cannot catch.
- The lowered gas needs no golden updates: `go test ./gno.land/pkg/integration/ -run TestTestdata` passes (65s) and no gas-golden txtar in `testdata/` exercises a map, so nothing regenerates.
