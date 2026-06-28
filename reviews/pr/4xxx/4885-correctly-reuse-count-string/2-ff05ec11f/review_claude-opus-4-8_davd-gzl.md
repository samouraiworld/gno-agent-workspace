# PR #4885: fix(gnovm): correctly reuse/count string in alloc and gc

URL: https://github.com/gnolang/gno/pull/4885
Author: ltzmaxwell | Base: master | Files: 13 | +559 -15
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `ff05ec11f` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4885 ff05ec11f`

**TL;DR:** Fixes how the GnoVM allocator counts string memory so shared backings (`s1 := s`, `s[1:]`) aren't counted twice and a sliced substring isn't dropped from the budget when its source string dies. It tracks each string backing's address range and charges its bytes once per garbage-collection cycle.

**Verdict: REQUEST CHANGES** - the string-tracking logic is unchanged from round 1 and still correct (both thehowl threads closed, revert-proofs pass), but all six `alloc_*.gno` filetests now fail at the PR head: the merge with master left their `MemStats` byte assertions stale, and CI's green check is from an earlier merge ref against an older master, so it no longer reflects the current head. The round-1 untracked-`StringValue` Warning still stands.

## Summary

GnoVM meters allocation to bound a transaction's memory, and strings were mis-counted two ways. Shared Go backings were charged in full on every GC visit, so `s1 := s` counted the bytes twice; and the first-draft fix (a `map[uintptr]int64` keyed by pointer) silently dropped backing bytes when a slice `s[M:N]` (M>0) outlived its source, because the slice's pointer `src+M` was never a key. The revision tracks each backing as a sorted `[]stringRange` and resolves any interior pointer to its containing range, so a slice inherits its source's bytes and the full backing is charged exactly once per cycle. `Fork()` clones the slice; `CleanupTrackedStrings` prunes ranges not seen in the cycle.

The Go source (`alloc.go`, `garbage_collector.go`, `realm.go`, `values.go`, `alloc_test.go`) is byte-for-byte identical to the round-1 commit `e9199dc9e`; only line numbers shifted from a master merge that pulled in interrealm Phase 3. The merge changed allocation sizes, and the six `alloc_*.gno` byte-count fixtures were re-pinned to numbers that match neither the committed merge head `ff05ec11f` nor a fresh merge into current master. The two txtar gas fixtures were re-derived correctly and pass.

## Glossary

- `stringRange` - `{start, end uintptr; lastCycle int64}`, one tracked string-backing extent in `Allocator.stringRanges`, sorted by `start`.
- `TrackString` - registers a backing extent; idempotent (interior pointer of an existing range adds nothing); called by `NewString` and `fillTypesOfValue`.
- `CountStringBytes` - GC-side lookup; returns `(full-backing-len, true)` on first visit per cycle, `(0, false)` after, for untracked pointers, and for empty strings.
- `CleanupTrackedStrings` - end-of-cycle prune of ranges whose `lastCycle` isn't the current cycle.
- `fillTypesOfValue` - store-load walk that replaces `RefType` placeholders with real types; now also re-routes each loaded `StringValue` through `NewString` so its backing is tracked.

## Fix

Before, [`StringValue.GetShallowSize()`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/alloc.go#L897) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L897) returned `allocString + allocStringByte*len`, charged on every GC visit, so two `StringValue`s over one backing were both fully counted. After, it returns the header only and the bytes are charged inline in [`GCVisitorFn`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/garbage_collector.go#L220-L223) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L220-L223) via `CountStringBytes`, which resolves a slice's pointer to its source range by containment and charges the full backing once per cycle. `GarbageCollect` calls [`CleanupTrackedStrings`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/garbage_collector.go#L151) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L151) at the end of each cycle.

The load path charges each loaded string exactly once, not twice. A container's `GetShallowSize` counts only its own slots ([`StructValue`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/alloc.go#L849-L851) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L849-L851) returns `allocStruct + len(Fields)*allocStructField`, independent of string content), and `internalRefSize` counts only `RefValue` slots, so the `Allocate(ss + rs)` in [`store.go:553`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/store.go#L548-L553) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/store.go#L548-L553) charges nothing for the contained string; the sole charge is [`fillTypesOfValue`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/realm.go#L1875-L1878) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/realm.go#L1875-L1878) → `NewString` (= `allocString + len`).

## Benchmarks / Numbers

txtar gas fixtures, current PR head, both pass:

| txtar | before (round 1) | after (this head) | delta | what it is |
|---|---|---|---|---|
| `gnokey_gasfee:39` | 1 271 011 | 1 271 043 | +32 | `gnokey maketx call -simulate` of `Hello()` |
| `stdlib_restart_compare:7` | 2 235 646 | 2 236 131 | +485 | exact gas after stdlib restart |

`alloc_*.gno` filetests, asserted vs actual at `ff05ec11f` (all fail):

| fixture | asserted | actual | delta |
|---|---|---|---|
| `alloc_0` | 7536 | 8144 | +608 |
| `alloc_1` | 8656 | 9264 | +608 |
| `alloc_7` | 6288 | 6896 | +608 |
| `alloc_7a` | 8280 | 8888 | +608 |
| `alloc_13` before / after | 4456 / 7392 | 7004 / 7740 | +2548 / +348 |
| `alloc_13a` before / after | 4456 / 7392 | 12004 / 8444 | +7548 / +1052 |

## Critical (must fix)

None.

## Warnings (should fix)

- **[the six string filetests fail at the PR head; green CI is from a stale merge ref]** [`alloc_7.gno:16`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_7.gno#L16) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_7.gno#L16) (also [`alloc_0.gno:25`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_0.gno#L25) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_0.gno#L25), [`alloc_1.gno:24`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_1.gno#L24) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_1.gno#L24), [`alloc_7a.gno:21`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_7a.gno#L21) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_7a.gno#L21), [`alloc_13.gno:46-47`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_13.gno#L46-L47) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13.gno#L46-L47), [`alloc_13a.gno:55-56`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_13a.gno#L55-L56) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno#L55-L56)) - `TestFiles/alloc_*` all fail on the current head; the asserted `MemStats` bytes are stale after the master merge.
  <details><summary>details</summary>

  The merge `ff05ec11f` pulled interrealm Phase 3 into the base, which changed allocation sizes. The conflict resolution re-pinned the `alloc_*.gno` byte assertions to numbers that match neither the committed merge head nor a fresh merge into current master: running `TestFiles/alloc_*` against `ff05ec11f` fails all six. The four simple fixtures are each off by a uniform +608, pointing at one structural allocation that grew in master after the numbers were set. `alloc_13.gno` and `alloc_13a.gno` are worse: both assert an identical `before 4456 / after 7392` despite running materially different programs (a 20-byte string vs a 1052-byte string), so their "before" was never re-measured against the merged tree (real before-values are 7004 and 12004). CI is green because the gnovm test job ran against `refs/pull/4885/merge` resolved at an older master (run head `e2f2dba38`); the current merge ref `86afe4210` also fails. Verified by reverting the fix in alloc_test.go is not needed here - the proof is the test suite itself failing on the head. Fix: re-run `TestFiles/alloc_*` against the current head, then update each `// Output:` line to the produced value; rebase onto current master first so the numbers don't drift again before merge.
  </details>

- **[new "counted only if tracked" rule is undefended; future direct `StringValue()` sites silently undercount]** [`uverse.go:217`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/uverse.go#L217) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/uverse.go#L217) (also [`values.go:2278`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/values.go#L2278) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2278), [`values.go:2816`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/values.go#L2816) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values.go#L2816)) - a `StringValue` that never passes through `TrackString` has its backing bytes counted as zero at GC.
  <details><summary>details</summary>

  After this PR `GetShallowSize` is header-only and the bytes are supplied by `CountStringBytes`, which returns `(0, false)` for any pointer not inside a tracked range. So a `StringValue` constructed directly, without `NewString`/`TrackString`, contributes only its header to the GC budget regardless of length. Three such sites exist: `NewConcreteRealm` (`StringValue(pkgPath)`, reached on every cross-call), `GetSlice` (the slice is built directly and relies on the *source* being tracked), and `typedString` (runtime-panic values).

  Magnitude today is bounded and not a consensus risk: every user-controllable string path - literals via [`op_eval.go:224`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/op_eval.go#L224) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/op_eval.go#L224), concatenation [`op_binary.go:808`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/op_binary.go#L808) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/op_binary.go#L808), and `string(...)` conversions in [`values_conversions.go`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/values_conversions.go#L1062) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/values_conversions.go#L1062) - routes through `NewString` and is tracked, so a `GetSlice` source is tracked and containment covers the slice. The three untracked sites carry only bounded internal strings (a realm path, fixed panic messages), and the result is deterministic across nodes. The real cost is decay: the invariant "the bytes live in a tracked range" is implicit, nothing enforces it, and no test fails if a future direct `StringValue(x)` site is added - it would just undercount. Fix: centralize the three sites behind a `safeStringValue(alloc, s)` helper that tracks then constructs, and add a regression test that builds a `StringValue` directly, makes it a GC root, and asserts the byte count includes the backing. Or consciously accept the bounded undercount and document the invariant on `StringValue.GetShallowSize`.
  </details>

## Nits

- [`alloc_13a.gno:9-18`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_13a.gno#L9-L18) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno#L9-L18) - the header prose cites old-design numbers ("Observed: after = 8396", "11956") that match neither the asserted `// Output:` (7392) nor the real output (8444). When the output line is re-pinned, refresh the prose so the worked example matches.
- [`alloc.go:394-414`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/alloc.go#L394-L414) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L394-L414) - `TrackString`'s `slices.Insert` is O(N) per call, O(N²) over N distinct live backings in one cycle, and unmetered. Measured cost is small: ~11ms at N=100K, ~21ms at N=200K distinct live strings (Go `memmove` of 24-byte entries), so it isn't a practical DoS at gas-reachable N, but a one-line note that the structure is insert-heavy would help the next reader. Confirmed behaviorally: timed `TrackString` over N distinct strings in the worktree.
- [`alloc_13.gno`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_13.gno) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13.gno), [`alloc_13a.gno`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/tests/files/alloc_13a.gno) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/tests/files/alloc_13a.gno) - the asserted `before/after` bytes fold in the whole preamble (package values, stdlib blocks), so any unrelated allocator change flips them and the number can't be derived from the comment. The stale fixtures above are exactly that maintenance cost realized. Consider asserting invariants (after < before; backing counted once) alongside the exact byte line.
- [`garbage_collector.go:207`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/garbage_collector.go#L207) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/garbage_collector.go#L207) - "GetShallowSize returns header-only for strings" is true only for `StringValue`; restate as "for `StringValue`, header only; bytes charged inline below" so the asymmetry is clear without grepping.

## Missing Tests

- **[regression test for an untracked `StringValue` at a GC root]** [`alloc_test.go:187`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/alloc_test.go#L187) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc_test.go#L187) - no test asserts that a directly-constructed `StringValue` (the `NewConcreteRealm` / `typedString` shape) is counted with its backing under GC.
  <details><summary>details</summary>

  Every byte-counting test in `alloc_test.go` builds the string through `alloc.NewString`, so all of them are tracked by construction and none exercises the untracked path. A test that constructs `StringValue("…")` directly, makes it reachable from a GC root, runs `GCVisitorFn`, and asserts `allocString + len` would turn the Warning above from latent into a caught regression and would lock the invariant.
  </details>

## Suggestions

- [`alloc.go:618-622`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/alloc.go#L618-L622) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L618-L622) - add `safeStringValue(alloc *Allocator, s string) StringValue` next to `NewString` that calls `TrackString` then returns `StringValue(s)`, and route the three direct sites through it, so the tracking invariant is locally enforceable and the next direct-construction site is a one-liner that can't forget.
- [`alloc.go:326-340`](https://github.com/gnolang/gno/blob/ff05ec11f/gnovm/pkg/gnolang/alloc.go#L326-L340) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/alloc.go#L326-L340) - `Fork()` clones via `slices.Clone`, which is safe only while no goroutine mutates the parent's slice header concurrently. Today the parent is only `NewString`-mutated during init and read-only under ABCI traffic, so there's no race, but that's an undocumented, load-bearing assumption. Note "parent must be quiescent during Fork" on the `Allocator` struct or freeze the root after init.

## Open questions

- The `lastCycle` dedup field is `int64` and `m.GCCycle` only ever increments; the overflow horizon is astronomically far, but a one-word comment would lock that reasoning in. Not posted: no practical risk.
- thehowl's two review threads (Fork alias-vs-clone, slice-undercount) are anchored to the old `map[uintptr]int64` commit `6638a4168` and are closed by the current `[]stringRange` redesign; round 1 reacted to them. Not re-posting.
