# PR #4885: fix(gnovm): correctly reuse/count string in alloc and gc

URL: https://github.com/gnolang/gno/pull/4885
Author: ltzmaxwell | Base: master | Files: 13 | +559 -15
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `e9199dc9e` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4885 e9199dc9e`

**TL;DR:** Fixes how the GnoVM allocator counts string memory so shared backings (`s1 := s`, `s[1:]`) aren't counted twice and a sliced substring isn't dropped from the budget when its source string dies. It tracks each string backing's address range and charges its bytes once per garbage-collection cycle.

**Verdict: NEEDS DISCUSSION** — design is sound and genuinely closes both of thehowl's threads (Fork-aliasing and slice-undercount, both verified by reverting); the load path charges each string once (no double-count); the open item is the new "bytes are counted only when tracked" invariant, undefended at three direct `StringValue(...)` construction sites that bypass tracking.

## Summary

GnoVM meters allocation to bound a transaction's memory, and strings were mis-counted two ways. Shared Go backings were charged in full on every GC visit, so `s1 := s` counted the bytes twice; and the first-draft fix (a `map[uintptr]int64` keyed by pointer) silently dropped backing bytes when a slice `s[M:N]` (M>0) outlived its source, because the slice's pointer `src+M` was never a key. The revision tracks each backing as a sorted `[]stringRange` and resolves any interior pointer to its containing range, so a slice inherits its source's bytes and the full backing is charged exactly once per cycle. `Fork()` clones the slice; `CleanupTrackedStrings` prunes ranges not seen in the cycle.

```
NewString("abc…")  ─► stringRanges = [{start=0x1000, end=0x1020, lastCycle=0}]
s2 := s[1:] (0x1001) ──┘  containment: 0x1000 ≤ 0x1001 < 0x1020 → same range
GC: vis(s2) → CountStringBytes → (32, true)   // full backing, source dead OK
    vis(s)  → CountStringBytes → (0,  false)  // dedup, same cycle
end of cycle: lastCycle == cycle → kept
```

## Glossary

- `stringRange` — `{start, end uintptr; lastCycle int64}`, one tracked string-backing extent in `Allocator.stringRanges`, sorted by `start`.
- `TrackString` — registers a backing extent; idempotent (interior pointer of an existing range adds nothing); called by `NewString` and `fillTypesOfValue`.
- `CountStringBytes` — GC-side lookup; returns `(full-backing-len, true)` on first visit per cycle, `(0, false)` after, for untracked pointers, and for empty strings.
- `CleanupTrackedStrings` — end-of-cycle prune of ranges whose `lastCycle` isn't the current cycle.
- `fillTypesOfValue` — store-load walk that replaces `RefType` placeholders with real types; now also re-routes each loaded `StringValue` through `NewString` so its backing is tracked.

## Fix

Before, [`StringValue.GetShallowSize()`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/alloc.go#L739) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L739) returned `allocString + allocStringByte*len`, charged on every GC visit, so two `StringValue`s over one backing were both fully counted. After, it returns the header only and the bytes are charged inline in [`GCVisitorFn`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/garbage_collector.go#L194-L196) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L194-L196) via `CountStringBytes`, which resolves a slice's pointer to its source range by containment and charges the full backing once per cycle. `GarbageCollect` calls [`CleanupTrackedStrings`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/garbage_collector.go#L151) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L151) at the end of each cycle.

The load path charges each loaded string exactly once, not twice. A container's `GetShallowSize` counts only its own slots ([`StructValue`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/alloc.go#L691-L693) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L691-L693) returns `allocStruct + len(Fields)*allocStructField`, independent of string content), and `internalRefSize` counts only `RefValue` slots, so the `Allocate(ss + rs)` in [`store.go:484`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/store.go#L480-L484) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/store.go#L480-L484) charges nothing for the contained string; the sole charge is [`fillTypesOfValue`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/realm.go#L1661-L1663) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/realm.go#L1661-L1663) → `NewString` (= `allocString + len`). The +32 / +485 txtar gas deltas are that single, previously-absent charge, not an overcount.

## Benchmarks / Numbers

| txtar | before | after | delta | what it is |
|---|---|---|---|---|
| `gnokey_gasfee:39` | 1 269 716 | 1 269 748 | +32 | `gnokey maketx call -simulate` of `Hello()` |
| `stdlib_restart_compare:7` | 1 973 588 | 1 974 073 | +485 | exact gas after stdlib restart |

Both integration tests pass on `e9199dc9e` with the updated numbers.

## Critical (must fix)

None.

## Warnings (should fix)

- **[new "counted only if tracked" rule is undefended; future direct `StringValue()` sites silently undercount]** [`uverse.go:171`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/uverse.go#L171) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/uverse.go#L171) (also [`values.go:2194`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/values.go#L2194) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2194), [`values.go:2720`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/values.go#L2720) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2720)) — a `StringValue` that never passes through `TrackString` has its backing bytes counted as zero at GC.
  <details><summary>details</summary>

  After this PR `GetShallowSize` is header-only and the bytes are supplied by `CountStringBytes`, which returns `(0, false)` for any pointer not inside a tracked range. So a `StringValue` constructed directly, without `NewString`/`TrackString`, contributes only its 48-byte header to the GC budget regardless of length. Three such sites exist: `NewConcreteRealm` (`StringValue(pkgPath)`, reached on every cross-call), `GetSlice` (the slice is built directly and relies on the *source* being tracked), and `typedString` (runtime-panic values).

  Magnitude today is bounded and not a consensus risk: every user-controllable string path — literals via [`op_eval.go:224`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/op_eval.go#L224) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/op_eval.go#L224), concatenation [`op_binary.go:756`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/op_binary.go#L756) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/op_binary.go#L756), and `string(...)` conversions in [`values_conversions.go`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/values_conversions.go#L1062) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values_conversions.go#L1062) — routes through `NewString` and is tracked, so a `GetSlice` source is tracked and containment covers the slice. The three untracked sites carry only bounded internal strings (a realm path, fixed panic messages), and the result is deterministic across nodes. The real cost is decay: the invariant "the bytes live in a tracked range" is implicit, nothing enforces it, and no test fails if a future direct `StringValue(x)` site is added — it would just undercount. Fix: centralize the three sites behind a `safeStringValue(alloc, s)` helper that tracks then constructs, and add a regression test that builds a `StringValue` directly, makes it a GC root, and asserts the byte count includes the backing. Or consciously accept the bounded undercount and document the invariant on `StringValue.GetShallowSize`.
  </details>

## Nits

- [`alloc.go:343-362`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/alloc.go#L343-L362) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L343-L362) — `TrackString`'s `slices.Insert` is O(N) per call, O(N²) over N distinct live backings in one cycle, and unmetered. Measured cost is small: 37ms at N=100K, 71ms at N=200K distinct live strings (Go `memmove` of 24-byte entries), so it isn't a practical DoS at gas-reachable N, but a one-line note that the structure is insert-heavy would help the next reader. Confirmed behaviorally: timed `TrackString` over N distinct strings in the worktree.
- [`gnovm/tests/files/alloc_13.gno`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/tests/files/alloc_13.gno) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13.gno), [`alloc_13a.gno`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/tests/files/alloc_13a.gno) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno) — the asserted `bytes:6956` / `bytes:11956` fold in the whole preamble (package values, stdlib blocks), so any unrelated allocator change flips them and the number can't be derived from the comment. The three exact-number bumps to `alloc_0/1/7/7a.gno` in this PR show the maintenance cost. Consider asserting invariants (after < before; backing counted once) alongside the exact byte line.
- [`garbage_collector.go:181`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/garbage_collector.go#L181) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L181) — "GetShallowSize returns header-only for strings" is true only for `StringValue`; restate as "for `StringValue`, header only; bytes charged inline below" so the asymmetry is clear without grepping.

## Missing Tests

- **[regression test for an untracked `StringValue` at a GC root]** — no test asserts that a directly-constructed `StringValue` (the `NewConcreteRealm` / `typedString` shape) is counted with its backing under GC.
  <details><summary>details</summary>

  Every byte-counting test in `alloc_test.go` builds the string through `alloc.NewString`, so all of them are tracked by construction and none exercises the untracked path. A test that constructs `StringValue("…")` directly, makes it reachable from a GC root, runs `GCVisitorFn`, and asserts `allocString + len` would turn the Warning above from latent into a caught regression and would lock the invariant.
  </details>

## Suggestions

- [`alloc.go:343-364`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/alloc.go#L343-L364) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L343-L364) — add `safeStringValue(alloc *Allocator, s string) StringValue` that calls `TrackString` then returns `StringValue(s)`, and route the three direct sites through it, so the tracking invariant is locally enforceable and the next direct-construction site is a one-liner that can't forget.
- [`alloc.go:286-300`](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/alloc.go#L286-L300) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L286-L300) — `Fork()` clones via `slices.Clone`, which is safe only while no goroutine mutates the parent's slice header concurrently. Today the parent is only `NewString`-mutated during init and read-only under ABCI traffic, so there's no race, but that's an undocumented, load-bearing assumption. Note "parent must be quiescent during Fork" on the `Allocator` struct or freeze the root after init.

## Open questions

- The `lastCycle` dedup field is `int64` and `m.GCCycle` only ever increments; the overflow horizon is astronomically far, but a one-word comment would lock that reasoning in. Not posted: no practical risk.
