# PR #4885: fix(gnovm): correctly reuse/count string in alloc and gc

URL: https://github.com/gnolang/gno/pull/4885
Author: ltzmaxwell | Base: master | Files: 13 | +559 -15
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: NEEDS DISCUSSION** — design is sound and addresses thehowl's two review threads (Fork-clone and slice-undercount via containment lookup), but ships three latent gaps: untracked-StringValue undercount at three direct construction sites ([`uverse.go:171`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/uverse.go#L171), [`values.go:2194`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2194) edge case, [`values.go:2720`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2720)), an O(N²) `slices.Insert` worst case in `TrackString`, and a header double-count on every store-loaded string.

## Summary

GnoVM's allocator was wrong about strings in two ways. First, shared backings were double-counted on every `s1 := s` because `GetShallowSize` charged `header + len(bytes)` for every visit. Second, the original draft fix (a `map[uintptr]int64` keyed by pointer) silently dropped backing bytes when the source string died but a slice with offset > 0 stayed alive — flagged by [@thehowl](https://github.com/gnolang/gno/pull/4885#discussion_r2462322020). This revision replaces the map with a sorted `[]stringRange` slice and looks up by **containment** (rightmost `start <= p` then `p < end`), so a slice's pointer `src+M` resolves to the source's range even when the source is otherwise dead. `Fork()` clones the slice (also flagged by thehowl) so child cleanup can't prune the parent's entries. GC counts each backing once per cycle (`lastCycle` dedup); `CleanupTrackedStrings` drops ranges not visited that cycle, bounding the address-recycling window.

```
NewString("aaa…aaa") ─► stringRanges = [{start=0x1000, end=0x1020, lastCycle=0}]
                              ▲
s2 := s[1:] (ptr=0x1001)  ────┘  lookup finds range via containment
                                  → returns full backing (32B), not len(s2)
GC visitor: vis(s2) → CountStringBytes → (32, true)  // source's range alive
            vis(s)  → CountStringBytes → (0,  false) // dedup, same cycle
end of cycle: range.lastCycle == cycle → preserved
```

## Glossary

- `stringRange` — `{start, end uintptr; lastCycle int64}`, one tracked string-backing extent in `Allocator.stringRanges`.
- `TrackString` — registers a backing extent; idempotent via containment check; called by `NewString` and `fillTypesOfValue`.
- `CountStringBytes` — GC-side lookup; returns `(full-backing-len, true)` on first visit per cycle, `(0, false)` after.
- `CleanupTrackedStrings` — end-of-cycle prune of unvisited ranges.
- `fillTypesOfValue` — store load path that walks a freshly-deserialized object and replaces `RefType` placeholders with real types. Now also re-routes `StringValue` through `NewString` to track its backing.

## Fix

Before: `StringValue.GetShallowSize()` charged `allocString + allocStringByte*len(sv)` on every GC visit, so two `StringValue`s sharing a Go backing (`s1 := s`, `s[low:high]`) were both fully counted. After: `GetShallowSize` returns header only; backing bytes are charged inline in [`GCVisitorFn`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L194-L198) via `alloc.CountStringBytes`, which uses a sorted `[]stringRange` and resolves slice pointers to their containing source via `sort.Search` + bounds check. `NewString` registers ranges; `fillTypesOfValue` re-routes loaded `StringValue`s through `NewString` so persisted strings get tracked; `GetSlice` charges only a header (the slice's pointer falls inside the source's range and inherits its bytes). `GarbageCollect` calls `CleanupTrackedStrings(m.GCCycle)` at the end of each cycle to prune dead backings.

## Benchmarks / Numbers

The two integration testdata gas updates capture the net allocation-cost delta from re-routing loaded strings through `NewString`:

| txtar | before | after | delta | note |
|---|---|---|---|---|
| `gnokey_gasfee.txtar:39` | 1 269 716 | 1 269 748 | +32 | `gnokey maketx call -simulate` of `Hello()`. |
| `stdlib_restart_compare.txtar:7` | 1 973 588 | 1 974 073 | +485 | exact gas constant after stdlib restart. |

Delta is small but always positive — the load path now charges one extra `allocString` (48B) per loaded `StringValue` (header counted at `defaultStore.GetObject`'s `Allocate(ss + rs)` then again in `fillTypesOfValue` → `NewString` → `AllocateString`). See the Warnings section.

## Critical (must fix)

None.

## Warnings (should fix)

- **[load path double-charges string header]** [`store.go:480-484`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/store.go#L480-L484) + [`realm.go:1661-1663`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/realm.go#L1661-L1663) — every store-loaded string is charged `allocString` twice.
  <details><summary>details</summary>

  `defaultStore.GetObject` deserializes via amino, then calls `Allocate(ss + rs)` where `ss = oo.GetShallowSize()`. For an object containing a `StringValue`, `GetShallowSize` now returns `allocString` (header) — bytes excluded by design. Immediately after, `fillTypesOfValue` walks the same object and replaces every `StringValue` with `store.GetAllocator().NewString(string(cv))`, which calls `AllocateString(len(s)) = allocString + len(s)`. Net charge for one loaded N-byte string: `allocString + allocString + N = 2·allocString + N`. The correct charge is `allocString + N`. The 32–485 gas delta in the txtar files is exactly this overcount accumulated across all the strings touched by the test. Not a security issue (overcount is conservative) but it makes gas estimates drift upward for every code path that loads realm state. Fix: have `fillTypesOfValue` call a tracking-only variant (`TrackString` alone, no `AllocateString`) for the loaded case — the bytes were already charged by the initial `Allocate(ss + rs)`. Or, conversely, exclude `StringValue` from the initial `ss` computation and let `NewString` charge it.
  </details>

- **[untracked StringValue undercount at three construction sites]** [`uverse.go:171`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/uverse.go#L171), [`values.go:2194`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2194), [`values.go:2720`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2720) — `StringValue(x)` constructions that bypass `NewString`/`TrackString` produce header-only GC counts.
  <details><summary>details</summary>

  After this PR, `GetShallowSize()` returns `allocString` only and the bytes are charged inline by `CountStringBytes` lookup. If a `StringValue` reaches GC roots without ever passing through `TrackString`, `CountStringBytes` returns `(0, false)` for it and its bytes are **never** counted. Three direct sites:

  1. `uverse.go:171` `NewConcreteRealm(pkgPath)` — `StringValue(pkgPath)` embedded in the realm-context struct. Reached at every cross-call ([`op_call.go:41,64`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/op_call.go#L41) and [`preprocess.go:1933`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/preprocess.go#L1933)). The pkgPath string often comes from a loaded `FuncValue.PkgPath` (so its backing is shared with a tracked string, and containment lookup saves us), but on first-deployment or in-memory paths it may not be — undercount.
  2. `values.go:2194` `GetSlice` — when the source string itself was never tracked (e.g. a slice of a `typedString` panic value that escapes into a captured variable), the slice's pointer doesn't fall inside any range and bytes go uncounted.
  3. `values.go:2720` `typedString` panic values — used for runtime panics; can be reified into Exception.Value and survive across frames.

  This is a regression: before this PR `GetShallowSize` always counted bytes regardless of tracking; now it counts them only when tracked. The inverse of the original PR's overcount bug, and the same shape thehowl flagged for slices: an exploitable undercount in a metered VM, just for a smaller set of strings. Fix: either (a) inline a `TrackString` call at each of the three sites (and any future direct-construction site — make a `safeStringValue(alloc, s)` helper to centralize this), or (b) fall back to `allocStringByte * len(sv)` in `GetShallowSize` when the pointer is untracked, accepting the dedup hole that creates for shared backings. (a) is cleaner.
  </details>

- **[TrackString O(N²) in number of distinct backings per tx]** [`alloc.go:354-363`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L354-L363) — `slices.Insert` shifts the tail on every new range; with N distinct string allocations, total cost is O(N²).
  <details><summary>details</summary>

  Each `NewString` for a fresh backing (string concatenation, `string(byteSlice)` cast, etc.) calls `TrackString`, which does `sort.Search` (O(log N)) then `slices.Insert` (O(N) memmove). For N strings in one tx, total `TrackString` cost is O(N²). At N=10K the slice-insert cost dominates; at N=100K it becomes seconds of wall time. The gas table at [`alloc.go:185-218`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L185-L218) charges only for the **Go heap allocation** of the string itself — the slice-insert work is unmetered. An adversarial contract that does `s := s + "x"` in a tight loop spends N alloc-budget for the strings but pays nothing for the O(N²) tracking overhead. Same shape as the historical `Realm.Owner` reflection cost. Fix: keep the sorted-slice for lookups but accumulate inserts in a small append-only buffer and merge-sort on the cleanup boundary; or switch to a Go `map[uintptr]*stringRange` keyed on `start` with a separate sorted index rebuilt lazily; or charge gas proportional to `len(stringRanges)` per insert.
  </details>

- **[parent allocator's `stringRanges` is mutated through `Fork().Reset()`-chain without documented invariant]** [`store.go:221`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/store.go#L221), [`alloc.go:286-300`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L286-L300) — `BeginTransaction` calls `ds.alloc.Fork()` which reads the parent's `stringRanges` via `slices.Clone`. If two transactions ever begin concurrently on the same parent, the clone races with whatever last wrote the parent.
  <details><summary>details</summary>

  thehowl flagged this on round 1 for the map-aliasing case and the author switched to `slices.Clone`. The clone itself is safe-by-construction (no shared backing), but only if no goroutine is concurrently mutating the parent's slice header (`slices.Insert` reallocates the underlying array). In practice, two paths touch the parent: `BeginTransaction` (which only **reads** the parent then writes to the child) and any direct `NewString` on the parent (e.g. preload or stdlib registration). The latter happens during initial setup, not under concurrent ABCI traffic — so today there's no race. But the invariant "parent `stringRanges` is read-only after init" is undocumented and load-bearing. If a future caller decides to `NewString` on the parent's allocator after the chain is up, it races against every `BeginTransaction`. Fix: document the invariant in the `Allocator` struct comment, and either freeze the parent at the end of init or add a `sync.RWMutex` around `stringRanges` access on the parent only.
  </details>

- **[`alloc_13.gno` and `alloc_13a.gno` golden numbers are brittle and uncommented]** [`gnovm/tests/files/alloc_13.gno`](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13.gno), [`gnovm/tests/files/alloc_13a.gno`](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno) — exact byte numbers (`bytes:6956`, `bytes:11956`) depend on the entire preamble and will break on unrelated allocator changes; no derivation shown.
  <details><summary>details</summary>

  The tests' assertion `MemStats: Allocator{maxBytes:10000, bytes:6956}` includes everything the gno preamble allocates — package values, stdlib blocks, the `runtime` import. Any unrelated change to one of those numbers flips the test, and the maintainer has no way to compute the expected value from first principles. The header-comment formula gestures at the shape ("before GC = sum of all live objects after auto-GC") but doesn't reconstruct the number. Make the test resilient by asserting **invariants** (after < before; auto-GC fired; backing counted once) instead of exact bytes, or commit a `// recompute: go test -update ./...` directive plus an explanation of which line moves when stdlib changes. The `alloc_0/1/7/7a.gno` shifts in this PR illustrate the maintenance cost — three exact numbers bumped, none explained.
  </details>

## Nits

- [`alloc.go:339-364`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L339-L364) — `TrackString` is exported but the doc never names a caller outside this package; consider lowercasing or noting that it's exposed for future `unsafe`-using callers.
- [`alloc.go:354`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L354) — `sort.Search` could be replaced with `slices.BinarySearchFunc` for symmetry with `slices.Insert` already imported.
- [`garbage_collector.go:182`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L182) — comment says "GetShallowSize returns header-only for strings"; that's true after this PR but only for `StringValue`; restate as "for `StringValue`, `GetShallowSize` returns header only and bytes are charged inline below" so a reader scanning the GC visitor understands the asymmetry without grepping.
- [`values.go:2186`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2186) — the local `sv := tv.GetString()[low:high]` is used once; inline into the `StringValue(...)` literal to make the slicing visible in the value being constructed.
- [`alloc.go:24-39`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L24-L39) — the `stringRanges` doc-comment is excellent; consider extracting the "Identity by *containment*, not equality" sentence as a top-of-file design note since it's the load-bearing invariant the whole design rests on.

## Missing Tests

- **[concurrency invariant]** [`alloc.go:286-300`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L286-L300) — no test covers `Fork()` concurrent with `TrackString` on the parent.
  <details><summary>details</summary>

  thehowl's round-1 concern was specifically about a query goroutine touching the parent's map concurrently with a tx writer. `TestFork_ClonesStringRanges` confirms the child can be mutated without affecting the parent, but no `-race` test exercises the inverse: parent mutated mid-clone. Either add a `t.Parallel` test with concurrent `Fork()` + `NewString` on the parent under `-race`, or add a comment in `Fork()` that the parent must be quiescent.
  </details>

- **[regression test for the three untracked-StringValue sites]** — no test asserts that strings produced by `NewConcreteRealm`, `typedString`, or `GetSlice` of an untracked source are counted correctly under GC.
  <details><summary>details</summary>

  The unit tests in `alloc_test.go` all go through `alloc.NewString`. Add a regression test that constructs `StringValue("…")` directly, makes it reachable from a GC root, runs `GCVisitorFn`, and asserts the byte count matches `allocString + len`. Today such a test would fail (it would count `allocString` only), which would surface the Warning above as a hard bug rather than a latent one.
  </details>

- **[Reset() does not clear stringRanges]** [`alloc.go:267-273`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L267-L273) — `Reset` zeros `bytes` but leaves `stringRanges` intact; no test pins this behavior.
  <details><summary>details</summary>

  This is intentional (the BeginTransaction chain depends on it: `Fork().Reset()` preserves the inherited ranges so children can see them on first lookup), but it's invisible behavior. A short test `TestReset_PreservesStringRanges` would lock in the contract and prevent a future "looks like a leak, let me clear it" change.
  </details>

## Suggestions

- [`alloc.go:343-364`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L343-L364) — centralize untracked-StringValue protection.
  <details><summary>details</summary>

  Introduce a `safeStringValue(alloc *Allocator, s string) StringValue` that does `alloc.TrackString(s); return StringValue(s)`, and replace the three direct-construction sites with it. Comment in the helper that any new direct `StringValue(x)` site must use this or risk silent undercount. Makes the invariant locally enforceable.
  </details>

- [`garbage_collector.go:194-198`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L194-L198) — extract the StringValue special case to its own visitor method.
  <details><summary>details</summary>

  The inline type assertion in `GCVisitorFn` breaks the symmetry of `VisitAssociated`. Consider adding a `GetGCAdjustedSize(gcCycle int64, alloc *Allocator) int64` method on `Value` with a default forwarding to `GetShallowSize`, and a `StringValue` override that adds the tracked bytes. Centralizes the special-case logic and avoids the type-assert in the hot path.
  </details>

## Questions for Author

- Why does `fillTypesOfValue` allocate fresh tracking via `NewString` for every loaded string instead of just calling `TrackString(string(cv))`? The former double-charges the header (see Warnings); the latter would charge nothing extra and still register the backing.
- Is `typedString` (used for panic values) expected to ever appear in GC roots? If not, document; if yes, it's the third untracked site that needs `TrackString`.
- Have you measured `TrackString`'s `slices.Insert` cost on a contract that allocates many distinct short strings (e.g. JSON parsing, `fmt.Sprint` in a loop)? If the O(N²) cost is real at N=10K, the gas table at [`alloc.go:185-218`](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L185-L218) doesn't capture it.
- The `lastCycle` field is `int64` but `m.GCCycle` is incremented every GC and never reset across the chain's lifetime. Is the implicit overflow horizon (~292 billion years) considered too far to bother, or do you want a comment locking that in?
