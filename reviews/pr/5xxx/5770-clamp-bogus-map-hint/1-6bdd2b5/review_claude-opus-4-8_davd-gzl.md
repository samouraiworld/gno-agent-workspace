# PR #5770: fix(gnovm): clamp bogus make(map, n) hint

URL: https://github.com/gnolang/gno/pull/5770
Author: ltzmaxwell | Base: master | Files: 3 | +71 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `6bdd2b5` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5770 6bdd2b5`

**TL;DR:** In the GnoVM, `make(map[K]V, n)` used to crash the whole transaction with an internal "addition overflow" when the size hint `n` was astronomically large. This PR makes a bogus hint (negative, or large enough to overflow the allocator's byte math) behave exactly like real Go: the hint is quietly ignored and the map is created empty-but-usable, no crash.

**Verdict: APPROVE** — correct, Go-faithful fix with solid boundary tests; one mergeability snag worth fixing before merge: this PR and its still-open sibling [#5723](https://github.com/gnolang/gno/pull/5723) both add a different `gnovm/tests/files/make19.gno`, so the second to land conflicts.

## Summary
A map size hint is advisory in Go: `make(map[K]V, n)` never panics on a bad `n`. The GnoVM violated that. `NewMap` fed the hint straight into `AllocateMap`, which computes `allocMap + allocMapItem*n` through `overflow.Mulp`/`Addp`; once `n` pushed that product past `math.MaxInt64` the overflow helpers panicked with a bare Go string ("addition overflow" / "multiplication overflow") that Gno code cannot `recover()`. The fix clamps `size < 0 || int64(size) > maxMapHint` to 0 inside [`NewMap`](https://github.com/gnolang/gno/blob/6bdd2b5/gnovm/pkg/gnolang/alloc.go#L602-L616) · [↗](../../../../../.worktrees/gno-review-5770/gnovm/pkg/gnolang/alloc.go#L602), where `maxMapHint = (math.MaxInt64 - allocMap) / allocMapItem` (≈1.15e17) is the largest hint whose preallocation cost still fits in int64. Realistic over-budget hints (e.g. 10e9) sit far below the pivot, so they still get charged the full byte cost and fail cleanly with "allocation limit exceeded" at the `make()` line.

## Glossary
- **hint** — the optional size argument to `make(map, n)`; a capacity suggestion, never a hard length.
- **maxMapHint** — pivot constant: hints `≤` it charge full preallocation cost, hints `>` it (and negatives) clamp to 0.
- **clamp** — replace an out-of-range value with an in-range one (here, with 0).

## Fix
Before: a hint above the int64-overflow pivot reached `overflow.Addp`/`Mulp` inside [`AllocateMap`](https://github.com/gnolang/gno/blob/6bdd2b5/gnovm/pkg/gnolang/alloc.go#L369-L371) · [↗](../../../../../.worktrees/gno-review-5770/gnovm/pkg/gnolang/alloc.go#L369) and panicked with an unrecoverable bare string. After: [`NewMap`](https://github.com/gnolang/gno/blob/6bdd2b5/gnovm/pkg/gnolang/alloc.go#L608-L610) · [↗](../../../../../.worktrees/gno-review-5770/gnovm/pkg/gnolang/alloc.go#L608) clamps such hints (and negatives) to 0 before any allocation math, so both the byte charge and the backing `make(map[MapKey]*MapListItem, c)` see 0. The load-bearing constraint is that the clamp only fires above `maxMapHint`; the entire realistic over-budget band stays below it and is still charged in full, preserving the clean "allocation limit exceeded" diagnostic at the `make()` site rather than masking real over-allocation as a no-op.

## Benchmarks / Numbers
| hint | byte charge | outcome |
|------|-------------|---------|
| `-1` | `allocMap` (200) | clamp → usable empty map |
| `maxMapHint - 1` | `allocMap + allocMapItem*(pivot-1)` | full charge |
| `maxMapHint` (115292150460684695) | `allocMap + allocMapItem*pivot` = 9223372036854775800 | full charge, last value that fits int64 |
| `maxMapHint + 1` | `allocMap` (200) | clamp → usable empty map (pre-PR: "addition overflow" panic) |
| `math.MaxInt` | `allocMap` (200) | clamp → usable empty map (pre-PR: "multiplication overflow" panic) |
| `10_000_000_000` (realistic) | full charge | "allocation limit exceeded" at `make()` |

## Critical (must fix)
None.

## Warnings (should fix)
- **[two open PRs add the same new file with different contents]** `gnovm/tests/files/make19.gno:1` — This PR and its still-open sibling [#5723](https://github.com/gnolang/gno/pull/5723) each add a *different* `gnovm/tests/files/make19.gno` (here: the map-hint clamp test; in #5723: the slice-overflow `recover()` test), and both also edit `gnovm/pkg/gnolang/alloc.go`. Whichever merges second hits an add/add conflict on the test file plus an overlap in `alloc.go`. Fix: rename this PR's filetest to the next free slot (`make20.gno`) or coordinate a merge order with #5723 so the second rebases.
  <details><summary>details</summary>

  Master currently tops out at `make18.gno`; both PRs independently claim `make19.gno`. Same author on both, so this is a coordination nit rather than a correctness issue, but left as-is it guarantees a manual conflict resolution on merge. Confirmed: `git cat-file -e origin/master:gnovm/tests/files/make19.gno` reports the path absent on master, and `gh pr view 5723 --json files` lists `gnovm/tests/files/make19.gno` among its additions. Fix: bump this file to `make20.gno`.
  </details>

## Nits
- `gnovm/pkg/gnolang/alloc.go:602` — `NewMap` clamps the hint but `NewListArray`/`NewDataArray` (same file) reject a negative length with a panic, and #5723 makes oversized array sizing a recoverable panic; the divergence is correct (Go's map hint is advisory, slice/array length is not) but undocumented at the call site. The `NewMap` comment already explains the map side; no change needed, noting only that the two sibling PRs deliberately resolve the same overflow class two different ways for two different Go semantics.

## Missing Tests
None. The added [`make19.gno`](https://github.com/gnolang/gno/blob/6bdd2b5/gnovm/tests/files/make19.gno#L1-L25) · [↗](../../../../../.worktrees/gno-review-5770/gnovm/tests/files/make19.gno#L1) filetest covers the negative and MaxInt hints end-to-end (map stays usable), and [`TestNewMapHintBoundary`](https://github.com/gnolang/gno/blob/6bdd2b5/gnovm/pkg/gnolang/alloc_test.go#L57-L83) · [↗](../../../../../.worktrees/gno-review-5770/gnovm/pkg/gnolang/alloc_test.go#L57) pins the exact pivot (full charge at `pivot-1`/`pivot`, clamp to `allocMap` at `pivot+1`/`MaxInt`/`-1`).

## Suggestions
None.

## Open questions
- The clamp leaves the entire `(maxMapHint, MaxInt64]` band charging only `allocMap` (a no-op) instead of failing, so a hint of exactly `maxMapHint+1` is "free" while `maxMapHint` costs the whole budget. This matches Go (the hint is advisory and a too-large hint allocates nothing up front), and any subsequent insert is still metered, so there's no gas-evasion path. Not posted: behavior is correct and matches Go; flagging only as a reasoning checkpoint.

---

Verified on 6bdd2b5:
- Native Go parity is exact: `make(map[int]int, -1)` and `make(map[int]int, math.MaxInt)` both yield `len 0`, both stay usable after insert, neither panics — identical to the filetest's `// Output:` block.
- Reverting the clamp reproduces the pre-PR crash: at `maxMapHint+1` the `Addp(allocMap, Mulp(...))` in `AllocateMap` panics "addition overflow"; at `MaxInt64` the inner `Mulp` panics "multiplication overflow". Both are bare Go strings, not Gno `*Exception`s, so `recover()` in Gno cannot catch them.
- `maxMapHint` computes to 115292150460684695; the pivot byte cost 9223372036854775800 is the last value `≤ MaxInt64`, and `pivot+1` overflows the `Addp` — so the boundary test's `want` values are the true allocator outputs, not assumptions.
- The `make19.gno` add/add collision with #5723 is real: the file is absent on master and listed as an addition in both PRs.
