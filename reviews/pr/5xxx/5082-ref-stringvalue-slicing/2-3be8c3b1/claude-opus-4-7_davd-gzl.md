# PR #5082: feat(gnovm): Reduce string slicing allocation cost with reference-based StringValue

URL: https://github.com/gnolang/gno/pull/5082
Author: notJoon | Base: master | Files: 13 | +226 -24
Reviewed by: davd-gzl | Model: claude-opus-4.7, rechecked with claude-opus-4.8 | Commit: `3be8c3b1` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5082 3be8c3b1`

Verdict: REQUEST CHANGES — landed integration test [`stdlib_restart_compare`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) fails on this branch (reproduced locally), `gnovm/cmd/benchstore` no longer builds with `-tags=genproto2`, the `init()` size self-check in [`alloc.go:132-151`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L132-L151) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L132-L151) misses the new struct so both `allocString` and `allocStringRef` under-charge by 8 bytes, and dropping the per-byte slice charge removes the only bound on how much a transaction can retain by slicing.

## Summary

Replaces `type StringValue string` with `struct{data string; ref bool}` so string slicing charges a fixed `allocStringRef = 48` instead of `48 + len`. Owner mode (literals, concatenation, conversions) keeps the old `48 + len` cost; reference mode is set only by `GetSlice` for primitive strings ([`values.go:2233-2240`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240)). Amino persistence collapses both modes back to owner mode via new `MarshalAmino`/`UnmarshalAmino` plus a manual edit to the generated [`pb3_gen.go`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/pb3_gen.go#L245-L297) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go#L245-L297). Per-call savings cap at `len(slice)` bytes (`allocStringByte = 1`), so realistic gas wins are small; the win is bounded but real for hot paths that slice often.

Since the [previous review](../1-534cee5b/claude-opus-4.6_davd-gzl.md) at `534cee5b` the only author commit is `3be8c3b1 "fix"`, which (a) adds the missing `allocStringRef` constant and (b) routes the generated amino marshal/unmarshal through the new `MarshalAmino`/`UnmarshalAmino` so the build/serialisation hole flagged in round 1 is closed. Of the deeper round-1 concerns, persistence round-trip identity and struct-size growth survive; the GC-accounting one does not, and is retracted: a master baseline shows the retention it describes is identical on master and never belonged to this PR.

## Glossary

- StringValue — VM-level wrapper around a Go string. Was a type alias; now a struct with `data` + `ref` fields.
- Owner mode — `ref=false`. Allocator charges `allocString = 48 + len(data)` bytes.
- Reference mode — `ref=true`. Only set in `GetSlice`. Allocator charges flat `allocStringRef = 48` bytes regardless of length.
- `GetShallowSize()` — value's reported size for the GC recount pass ([`garbage_collector.go:177-188`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/garbage_collector.go#L177-L188) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/garbage_collector.go#L177-L188)). Ref returns 48; owner returns `48 + len`.
- `Recount` — adds to allocator `bytes` without charging gas ([`alloc.go:253-258`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L253-L258) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L253-L258)). Called per live value during GC re-walk.
- `maxAllocTx = 500_000_000` — per-transaction allocator cap ([`keeper.go:50`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/sdk/vm/keeper.go#L50) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/sdk/vm/keeper.go#L50)).

## Fix

Before: `StringValue("abc"[0:1])` flows through `alloc.NewString(...)` which charges `48 + len(slice)`. After: `GetSlice` calls `alloc.NewStringRef(...)` which charges a flat 48. The load-bearing invariant is that ref-mode is only ever entered by `GetSlice`; every other producer (`NewString`, `op_binary` concat, `values_conversions`, gonative, uverse) stays in owner mode. Persistence flattens both modes back to owner via `MarshalAmino`/`UnmarshalAmino`, so the on-disk format is unchanged.

## Critical (must fix)

- **[gas drift in landed integration test]** [`gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar:7`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar#L7) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar#L7) — `EXACT_GAS=1974482` is off by 1 from actual (`1974481`); test fails on this branch and was also red in CI run [24844592434](https://github.com/gnolang/gno/actions/runs/24844592434/job/72727940545).
  <details><summary>details</summary>

  This is the very test that exists to guard the determinism invariant the PR touches: "gas is identical for restart vs no-restart". On the PR branch the second `Convert` call reports `GAS USED: 1974481`, but the txtar asserts `1974482`. The 1-unit gap is consistent with the PR's optimization shaving one `allocStringByte` off some slice path used by `strings.NewReplacer`. The restart half of the test, which is the actual determinism check, passes once `EXACT_GAS` is updated (verified locally — see Repro). The author seems to have updated the constant during the master merge but landed one off. Fix: update `EXACT_GAS=1974481` (or whatever the recalibrated value is at the time of merge) and re-run.

  **Repro** (run inside the gno worktree):
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5082 -R gnolang/gno
  go test -v -run 'TestTestdata/stdlib_restart_compare' ./gno.land/pkg/integration/
  # observe: FAIL with "no match for `GAS USED:\s+1974482`" and "GAS USED: 1974481"

  # flip EXACT_GAS and confirm restart parity holds:
  sed -i 's/EXACT_GAS=1974482/EXACT_GAS=1974481/' gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar
  go test -v -run 'TestTestdata/stdlib_restart_compare' ./gno.land/pkg/integration/
  # observe: PASS — restart and no-restart both produce 1974481

  git checkout HEAD -- gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar
  ```
  </details>

- **[build break behind genproto2 tag]** [`gnovm/cmd/benchstore/values.go:44`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/cmd/benchstore/values.go#L44) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/cmd/benchstore/values.go#L44) — `gno.StringValue(s)` is no longer a valid type conversion; this file only compiles under `-tags=genproto2` so plain `go build ./...` doesn't catch it.
  <details><summary>details</summary>

  The file is gated by `//go:build genproto2`. `StringValue` is now a struct with two unexported fields, so the old type-conversion call no longer compiles. The migration was applied everywhere else in the repo (`convert.go`, `uverse.go`, `native.go`, `testing_runtime.go`, `context_testing.go`, `machine_test.go`) but missed here because the default build excludes the file. Fix: `V: gno.NewStringValue(s)` (mirrors the other migrated call sites).

  **Repro**:
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5082 -R gnolang/gno
  go build -tags=genproto2 ./gnovm/cmd/benchstore/
  # observe: gnovm/cmd/benchstore/values.go:44:62: cannot convert s (variable of type string) to type gnolang.StringValue
  ```
  </details>

## Warnings (should fix)

- **[dropping the per-byte slice charge removes the only bound on retention]** [@ltzmaxwell](https://github.com/gnolang/gno/pull/5082#issuecomment-4251302565) (theme only: "correctly tracking allocation/GC recount is necessary", never anchored or demonstrated on this diff) [`values.go:2234-2239`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2234-L2239) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2234-L2239) — a 32 MiB slice of a 64 MiB parent is charged 48 bytes on this branch against 33,554,480 on master, and the slice pins the parent's whole backing array on both.
  <details><summary>details</summary>

  Baselined against master `27b5b8e24` before attributing anything to the diff. Master's `GetSlice` calls `alloc.NewString(tv.GetString()[low:high])`, and `NewString` returns `StringValue(s)` with no copy, so a master substring shares the parent's backing exactly as a ref does. The retention is therefore pre-existing, not introduced here: the identical test post-GC accounts 49 bytes on master and 48 on this branch, and `runtime.ReadMemStats` reports 70 MiB held on both. Round 2 called that a PR defect; it was wrong, and the claim is retracted.

  What this PR does change is the charge at allocation time. Master charges `48 + len` for a slice it never copied, which over-charges in the copy sense but happens to bound how much a transaction can retain by slicing; this branch charges a flat 48. Neither revision tracks the retained backing, so that accidental per-byte bound is the only one there is, and removing it before backing-range tracking lands leaves retention unbounded. [#4885](https://github.com/gnolang/gno/pull/4885) tracks each backing as a sorted `[]stringRange` and charges it once per GC cycle, which is the principled replacement.

  This is the PR's stated optimization, not a bug in it, so the question for the author is sequencing rather than correctness. Repro in [comment_claude-opus-4-7.md](comment_claude-opus-4-7.md).
  </details>

- **[`allocString`/`allocStringRef` are stale: struct is 24B, constants assume 16B]** [`alloc.go:83-90`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L83-L90) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L83-L90) — `unsafe.Sizeof(StringValue{}) == 24` now (string header 16 + bool 1 + padding 7) but both constants are still `_allocHeap + 16 = 48`. Both modes under-charge by 8 bytes, and the [`init()` self-check](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L132-L151) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L132-L151) has no entry for `StringValue` to catch it.
  <details><summary>details</summary>

  Every other escaping value type has a `check("_alloc<Type>", _alloc<Type>, unsafe.Sizeof(<Type>{}))` line in the init that panics on drift. There's no such line for `StringValue`, so the 8-byte gap goes unnoticed. The under-charge is symmetric across owner and ref so it doesn't change relative costs, but it does mean every StringValue on the VM pays 48 when it should pay 56. Fix: bump both to `_allocHeap + 24` (or define `_allocStringValue = 24` and reuse), and add `check("_allocStringValue", _allocStringValue, unsafe.Sizeof(StringValue{}))` to the init.
  </details>

- **[round-trip changes the value's reported size]** [`values.go:123-133`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L123-L133) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L123-L133) — `MarshalAmino` writes only `data`; `UnmarshalAmino` always sets `ref=false`. A ref-mode StringValue with `data="abcdef"` reports `GetShallowSize()=48` before persistence and `54` after — same bytes, two accounted sizes.
  <details><summary>details</summary>

  The PR doc says this is fine because the on-wire format is identical and post-restart all strings are owner-mode. That's true for byte-equality, but the GC's recount pass uses `GetShallowSize` to rebuild the allocator's `bytes` counter — so on the second tx after restart the same string contributes more to the budget than on the first tx before restart. The integration test we already have ([`stdlib_restart_compare`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar)) doesn't exercise GC mid-tx so it passes once `EXACT_GAS` is corrected — but a longer-running workload that crosses the GC threshold could see drift between "no-restart" and "after-restart" gas. The cleanest fix is the same as the GC concern above (don't keep refs alive across persistence at all), with the same trade-off. If the PR keeps ref mode in memory, document the asymmetry explicitly. Adversarial test: [`tests/string_ref_undercount_test.go:TestStringRefMarshalRoundTripChangesSize`](tests/string_ref_undercount_test.go).
  </details>

- **[`Allocator.NewString` still alive but no longer charges for sliced bytes]** [`alloc.go:387-390`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L387-L390) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L387-L390) — every old caller of `NewString` (concat, conversions, gonative, uverse) still pays `48 + len`, but slicing now silently switches to `NewStringRef`. Two parallel allocator entry points with different cost models, no comment at the call sites explaining when each is appropriate.
  <details><summary>details</summary>

  Future contributors adding a new string-producing op need to know: copy-producing path → `NewString`; slice-of-existing-data path → `NewStringRef`. The doc comments on both functions are decent but the call-site discipline isn't documented anywhere ("only `GetSlice` may use `NewStringRef`"). A one-liner in the `StringValue` type doc — e.g. "Reference mode is exclusively produced by `Allocator.NewStringRef` from string-slicing primitives; do not introduce new ref producers without auditing GC accounting" — would lock the contract in place. Without it, the next `NewStringRef` call site is an easy way to expand the under-counting blast radius.
  </details>

## Nits

- [`values.go:97`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L97) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L97) — doc comment `// NewStringValue creates a new StringValue in owner mode` missing terminating period (carry-over from round 1).
- [`values.go:102-133`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L102-L133) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L102-L133) — same for `NewStringValueRef`, `Value`, `IsRef`, `Len`, `MarshalAmino`, `UnmarshalAmino`.
- [`bench_test.go:35`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L35) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L35) — `benchmarkSliceSink` is written at [`:63`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L63) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L63) but never read, so it holds the result away from dead-code elimination without the did-it-run guard the existing `sink` gets at [`:29-32`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L29-L32) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L29-L32) (`if sink == nil { b.Fatal(...) }` plus a reset). Mirroring that guard would catch a benchmark that silently never ran.
- [`alloc_test.go:87`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc_test.go#L87) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L87), [`:113`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc_test.go#L113) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L113) — `_ = result.GetString()` / `_ = s3.GetString()` are no-op reads. Replace with a content assertion (`if got := result.GetString(); got != "hello"…`) or drop them.
- [`bench_test.go:46-58`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L46-L58) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L46-L58) — `NewAllocator(1024*1024)` lives inside the `b.N` loop; allocator construction is part of the measured signal. Hoist + `alloc.Reset()` per iteration for a cleaner number.

## Missing Tests

- **[no Gno-level filetest for string slicing accounting]** `gnovm/tests/files/` — every check in this PR is at the Go layer (`alloc_test.go`, `bench_test.go`). A `.gno` filetest that slices a string and inspects the allocator delta would close the loop and run on every CI cycle.
  <details><summary>details</summary>

  The behavior change is observable from Gno: a contract that does `s[low:high]` ends up with a TypedValue whose `V.IsRef()` is true. Worth a filetest that pins the gas/alloc cost so any future regression (e.g. someone reverting `GetSlice` back to `NewString`) is caught at the contract layer, not just at the internal Go API.
  </details>

- **[nothing pins the ref/owner size asymmetry across persistence]** `gnovm/pkg/gnolang/` — a ref reports 48 going into amino and 54 coming back out as an owner, and no in-tree test asserts it either way; see [`tests/string_ref_undercount_test.go`](tests/string_ref_undercount_test.go).
  <details><summary>details</summary>

  The asymmetry is introduced by this diff: master has no ref mode, so the same value reports 54 on both sides. Whether the PR keeps it or removes it, an in-tree test makes the choice explicit and forces the next change to decide. Scoped to the round-trip only: the GC-recount half of the round-2 version rested on the retracted under-count claim and is dropped.
  </details>

## Suggestions

- [`values.go:92-95`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L92-L95) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L92-L95) — consider whether the `ref` flag belongs on every `StringValue` (every string in the VM pays 8 bytes of struct overhead for a property only set by `GetSlice`). Alternatives: a separate `StringRefValue` type behind the `Value` interface, or a bit in `TypedValue.N` (the unused word). Either avoids paying the bool on every literal and concatenation result. Worth weighing against the simplicity of the current approach.
- [`alloc.go:90`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L90) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L90) — `allocStringRef = _allocHeap + 16` is literally identical to `allocString`. Either make the relationship explicit (`allocStringRef = allocString`) or — once the constants are bumped to 24 — keep them aliased so future struct-size fixes hit both at once.

## Verified

Round-2 findings rechecked at `3be8c3b1` with claude-opus-4.8. All numbers below come from runs in the PR worktree.

- The hand edit to the generated [`pb3_gen.go`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/pb3_gen.go) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go) is exactly what the generator emits: running [`misc/genproto2`](https://github.com/gnolang/gno/blob/3be8c3b1/misc/genproto2/genproto2.go) · [↗](../../../../../.worktrees/gno-review-5082/misc/genproto2/genproto2.go) in the worktree leaves `git status` clean, so regeneration is a byte-for-byte no-op and asks nothing of the author. This overturns the round-2 Warning that told the author to re-run genproto2; the finding is dropped, not carried. No CI job references genproto2, so nothing else covers this.
- `unsafe.Sizeof(StringValue{}) = 24` and `_allocHeap = 32`, so the sizeof-derived constant is 56 against the committed 48. The 8-byte under-charge is measured, not inferred.
- Every behavioral claim above is baselined against master `27b5b8e24`, running the identical test on both revisions. This is what killed the round-2 GC-undercount Warning: post-GC a 1-byte ref slice of a 64 MiB parent accounts 49 bytes on master and 48 here, and `runtime.ReadMemStats` reports 70 MiB held on both, so the retention is pre-existing and was never this PR's to answer for. Master's `NewString` returns `StringValue(s)` with no copy, so a master substring shares the parent's backing exactly as a ref does.
- What the diff does move is the allocation-time charge: the identical 32 MiB slice of a 64 MiB parent costs 33,554,480 bytes on master and 48 here.
- The struct-size drift is master-clean: `unsafe.Sizeof` is 16 on master where `allocString = _allocHeap + 16 = 48` is exactly right, and 24 here against the same 48, so the 8-byte gap is introduced by this diff.
- The round-trip asymmetry is master-clean too: master has no ref mode, so `GetShallowSize("abcdef")` is 54 before and after. Here a ref reports 48 and its post-restart owner reports 54.
- `go build -tags=genproto2 ./gnovm/cmd/benchstore/` succeeds on master and fails here.
- [`stdlib_restart_compare`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) still fails on this branch, and the cause is the PR, not staleness: `EXACT_GAS=1974482` is the value at the merge base `066f15a79`, the branch produces 1974481, and the file is untouched by this diff. Flipping the constant to 1974481 makes both calls emit 1974481 and the test pass, so restart parity itself holds.
- Master recalibrates `EXACT_GAS` on most releases, so no absolute value for it stays true for long and none is cited here. Round 2 named one and it had already gone stale; this pass named a fresher one and it was stale within a commit. The author re-derives it after merging, and a reviewer should quote the merge base rather than master.
- `go build -tags=genproto2 ./gnovm/cmd/benchstore/` still fails at `values.go:44`.
- Round-2 claims retracted by this pass: the GC under-count as a PR defect (pre-existing, identical on master), and the `pb3_gen.go` regeneration ask (a byte-for-byte no-op).

## Questions for Author

- Is the "in-memory ref, owner after persistence" asymmetry intentional? If so, the PR description should state it explicitly and explain why the resulting `GetShallowSize` asymmetry across restart doesn't cause gas drift in any realistic workload (the integration test only exercises the no-GC fast path).
- How does this PR sit relative to [#4885](https://github.com/gnolang/gno/pull/4885)? `ltzmaxwell` [asked this on 2026-04-15](https://github.com/gnolang/gno/pull/5082#issuecomment-4251302565) and named the missing half exactly: "correctly tracking allocation/GC recount is necessary". It is the last comment on the PR and went unanswered; the author's last push is `3be8c3b1` on 2026-04-23. Meanwhile #4885 is live (`ltzmaxwell` active 2026-07-12, `thehowl` chasing a second review on 2026-06-22) and implements that accounting: it tracks each string backing as a sorted `[]stringRange`, resolves interior pointers to their containing range, and charges each backing once per GC cycle, so a slice inherits its source's bytes. Not superseded on master (`type StringValue string` is unchanged there), and the two PRs are not the same change: this one buys a gas discount on slicing, #4885 makes the counting correct. But #4885's range tracking would make the `ref` flag unnecessary, so the merge-direction call belongs to the author and a maintainer, not to this review.
- Is the `gnovm/cmd/benchstore` path covered by any CI job that exercises `-tags=genproto2`? If not, the missed migration there will only surface when someone manually rebuilds, which is exactly when build breaks hurt most.
