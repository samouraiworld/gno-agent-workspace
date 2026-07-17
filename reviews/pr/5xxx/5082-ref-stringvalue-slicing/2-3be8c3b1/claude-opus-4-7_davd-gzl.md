# PR #5082: feat(gnovm): Reduce string slicing allocation cost with reference-based StringValue

URL: https://github.com/gnolang/gno/pull/5082
Author: notJoon | Base: master | Files: 13 | +226 -24
Reviewed by: davd-gzl | Model: claude-opus-4.7, rechecked with claude-opus-4.8 | Commit: `3be8c3b1` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5082 3be8c3b1`

Verdict: NEEDS DISCUSSION — the landed integration test [`stdlib_restart_compare`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) is red because the PR's flat slice charge left `EXACT_GAS` one unit stale, [`gnovm/cmd/benchstore`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/cmd/benchstore/values.go#L44) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/cmd/benchstore/values.go#L44) no longer builds under `-tags=genproto2`, and the size constant for the new struct is 8 bytes stale with no drift guard, so string metering undercharges. The slicing optimization itself is sound and more accurate than master.

## Summary

Replaces `type StringValue string` with `struct{data string; ref bool}` so string slicing charges a fixed `allocStringRef = 48` instead of `48 + len`. Owner mode (literals, concatenation, conversions) keeps the old `48 + len` cost; reference mode is set only by `GetSlice` for primitive strings ([`values.go:2233-2240`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240)). Amino persistence collapses both modes back to owner mode via new `MarshalAmino`/`UnmarshalAmino` plus a manual edit to the generated [`pb3_gen.go`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/pb3_gen.go#L245-L297) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go#L245-L297). Per-call savings cap at `len(slice)` bytes (`allocStringByte = 1`), so realistic gas wins are small; the win is bounded but real for hot paths that slice often.

Since the [previous review](../1-534cee5b/claude-opus-4.6_davd-gzl.md) at `534cee5b` the only author commit is `3be8c3b1 "fix"`, which (a) adds the missing `allocStringRef` constant and (b) routes the generated amino marshal/unmarshal through the new `MarshalAmino`/`UnmarshalAmino` so the build/serialisation hole flagged in round 1 is closed. Of the deeper round-1 concerns, struct-size growth survives as a Warning: the size constant went stale and the drift guard was not extended to the new struct. The GC-accounting concern and the round-trip-drift concern are both retracted: a master baseline shows the retention is identical on master, and a trace shows the ref/owner size difference never reaches a gas path. See Verified.

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

  This is the test that guards the determinism invariant the PR touches: gas identical for restart vs no-restart. The branch reports `GAS USED: 1974481` against the pinned `1974482`, which is the merge-base value, unchanged by the diff. So the constant is stale because the PR's flat slice charge shifted the runtime gas by one unit, not because of a merge fumble. Both halves of the test agree at 1974481, so the determinism it checks holds; only the pinned number is stale. Fix: update `EXACT_GAS` to whatever the branch produces after rebasing on master, and re-run.

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

## Warnings (should fix)

- **[stale alloc constant undercharges string metering; drift guard missing]** [`alloc.go:83-90`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L83-L90) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L83-L90) — `allocString`/`allocStringRef` are still the pre-PR `_allocHeap + 16`, but the struct is 24 bytes now, so both are 8 bytes low; the metering is wrong for every string.
  <details><summary>details</summary>

  The undercharge is uniform and deterministic (gas by 0-3 units per string through the `allocGas` log2 table, the memory-cap counter by a flat 8), so it does not diverge across nodes, but the fee constant is simply wrong against the codebase's own rule that it equal `unsafe.Sizeof`. The [`init()` self-check](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L132-L151) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L132-L151) exists to catch exactly this drift and guards every other struct-backed value type; `StringValue` has no line, so both the current gap and the next size change go uncaught. Fix: define `_allocStringValue = unsafe.Sizeof(StringValue{})`, use it for both constants, and add a `check(...)` line. (The remaining unchecked constants, Bigint/Bigdec/DataByte, are documented estimates, not struct layouts.)
  </details>

- **[latent build break behind the genproto2 tag]** [`gnovm/cmd/benchstore/values.go:44`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/cmd/benchstore/values.go#L44) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/cmd/benchstore/values.go#L44) — `gno.StringValue(s)` is no longer a valid conversion now that `StringValue` is a struct; the file is gated by `//go:build genproto2` so the default build and CI never compile it.
  <details><summary>details</summary>

  The struct migration was applied to every ungated call site (`convert.go`, `uverse.go`, `native.go`, `testing_runtime.go`, `context_testing.go`, `machine_test.go`) but missed this one because the default build excludes it. No CI job builds with `-tags=genproto2` (only `genproto` v1 is exercised, in `ci-codegen-verify.yml` and `ci-dir-misc.yml`), so the break surfaces only when someone builds the benchmark/proto tooling by hand. Fix: `V: gno.NewStringValue(s)`, mirroring the other migrated sites.

  **Repro**:
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5082 -R gnolang/gno
  go build -tags=genproto2 ./gnovm/cmd/benchstore/
  # observe: gnovm/cmd/benchstore/values.go:44:62: cannot convert s (variable of type string) to type gnolang.StringValue
  ```
  </details>

## Nits

- [`values.go:97-133`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L97-L133) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L97-L133) — the seven new doc comments (`NewStringValue` through `UnmarshalAmino`) lack terminating periods. Not posted, no change needed: `godot` is absent from the enabled linters in [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/3be8c3b1/.github/golangci.yml?plain=1#L12-L34) · [↗](../../../../../.worktrees/gno-review-5082/.github/golangci.yml#L12-L34), which runs `default: none`, so nothing enforces it and the prose reads the same either way. Recorded only because 87% of the doc comments in `gnovm/pkg/gnolang` do carry one.
- [`bench_test.go:35`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L35) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L35) — `benchmarkSliceSink` is written at [`:63`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L63) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L63) but never read, so it holds the result away from dead-code elimination without the did-it-run guard the existing `sink` gets at [`:29-32`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L29-L32) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L29-L32) (`if sink == nil { b.Fatal(...) }` plus a reset). Mirroring that guard would catch a benchmark that silently never ran.
- [`alloc_test.go:87`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc_test.go#L87) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L87), [`:113`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc_test.go#L113) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc_test.go#L113) — `_ = result.GetString()` / `_ = s3.GetString()` are no-op reads. Replace with a content assertion (`if got := result.GetString(); got != "hello"…`) or drop them.
- [`bench_test.go:46-58`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/bench_test.go#L46-L58) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/bench_test.go#L46-L58) — `NewAllocator(1024*1024)` lives inside the `b.N` loop; allocator construction is part of the measured signal. Hoist + `alloc.Reset()` per iteration for a cleaner number.

## Missing Tests

- **[no Gno-level filetest for string slicing accounting]** `gnovm/tests/files/` — every check in this PR is at the Go layer (`alloc_test.go`, `bench_test.go`). A `.gno` filetest that slices a string and inspects the allocator delta would close the loop and run on every CI cycle.
  <details><summary>details</summary>

  The behavior change is observable from Gno: a contract that does `s[low:high]` ends up with a TypedValue whose `V.IsRef()` is true. Worth a filetest that pins the gas/alloc cost so any future regression (e.g. someone reverting `GetSlice` back to `NewString`) is caught at the contract layer, not just at the internal Go API.
  </details>

## Suggestions

- **[NewStringRef trusts an unenforced shared-backing precondition]** [`alloc.go:392-398`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L392-L398) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L392-L398) — reference mode charges a flat 48 and skips the per-byte cost because a slice shares its parent's backing; a fresh string passed in would be under-charged by its whole length.
  <details><summary>details</summary>

  Verified: `NewStringRef` on a fresh 1 MiB string charges 48 against `NewString`'s 1,048,624. [`GetSlice`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2233-L2240) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2233-L2240) is the only producer today and always passes a reslice of an existing string, so nothing is under-metered now; this is latent. The doc comments describe ref mode as "for slicing" but nothing marks "only `GetSlice` may produce it" as load-bearing. A one-line contract on the type, or a producer guard, keeps a future ref producer from silently under-metering.
  </details>

- **[every string carries the ref bool]** [`values.go:92-95`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L92-L95) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L92-L95) — the `ref` bool grows `StringValue` from 16 to 24 bytes (1 byte plus 7 padding), so every literal, concatenation, and conversion result pays for a field only `GetSlice` sets.
  <details><summary>details</summary>

  A separate reference type behind the `Value` interface would keep the owner path at 16 bytes, at the cost of type-switching everywhere `StringValue` is handled (amino, `GetShallowSize`, printing, comparison) for a saving dwarfed by data bytes on any non-tiny string. A bit in `TypedValue.N` does not work: `GetShallowSize` is a `Value`-interface method called during GC with no `TypedValue` in reach, and it must read the flag to decide the byte-vs-flat charge, so the flag has to live on the value. Legitimate trade to weigh, not an obvious win.
  </details>

- [`alloc.go:90`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/alloc.go#L90) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/alloc.go#L90) — `allocStringRef = _allocHeap + 16` is byte-identical to `allocString`. Alias them (`allocStringRef = allocString`) so a future struct-size fix hits both at once.

## Verified

Rechecked at `3be8c3b1` with claude-opus-4.8, every causal claim run on both the PR head and a fresh master worktree (master moved from `27b5b8e2` to `959cefd9` mid-review, which is why no absolute master value is cited below). Consequences traced through the code, not asserted.

Holds:
- `unsafe.Sizeof(StringValue{})` is 16 on master, where `allocString = _allocHeap + 16 = 48` is exactly right, and 24 here against the same 48, so the 8-byte struct-overhead gap is this diff's. The gas effect is 0-3 units per string (the `allocGas` log2 table), the cap effect a flat 8 bytes, and both are identical on every node.
- `NewStringRef` on a fresh 1 MiB string charges 48 against `NewString`'s 1,048,624; `GetSlice` at [`values.go:2239`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/values.go#L2239) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/values.go#L2239) is its only producer (one non-test caller in the whole repo).
- `go build -tags=genproto2 ./gnovm/cmd/benchstore/` succeeds on master and fails here at `values.go:44`. No workflow builds with that tag.
- [`stdlib_restart_compare`](https://github.com/gnolang/gno/blob/3be8c3b1/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) · [↗](../../../../../.worktrees/gno-review-5082/gno.land/pkg/integration/testdata/stdlib_restart_compare.txtar) is red because the PR produces 1974481 against the pinned `EXACT_GAS=1974482`, which is the merge-base value; the txtar is untouched by the diff. Flipping the constant makes both the no-restart and after-restart calls emit 1974481, so the determinism the test guards is intact. Master recalibrates this constant most releases, so no absolute value is cited; the author re-derives it on rebase.
- Regenerating [`pb3_gen.go`](https://github.com/gnolang/gno/blob/3be8c3b1/gnovm/pkg/gnolang/pb3_gen.go) · [↗](../../../../../.worktrees/gno-review-5082/gnovm/pkg/gnolang/pb3_gen.go) via [`misc/genproto2`](https://github.com/gnolang/gno/blob/3be8c3b1/misc/genproto2/genproto2.go) · [↗](../../../../../.worktrees/gno-review-5082/misc/genproto2/genproto2.go) leaves `git status` clean, so the hand edit to the generated file is a byte-for-byte no-op.

Retracted over the course of this recheck, all the same failure mode (a consequence asserted without a baseline or a trace):
- The GC under-count as a PR defect. Post-GC a 1-byte ref slice of a 64 MiB parent accounts 49 bytes on master and 48 here, with 70 MiB held on both, so the retention is pre-existing. Master's `NewString` returns `StringValue(s)` with no copy, sharing the parent's backing exactly as a ref does.
- Its reframe as a lost retention bound. Retention is `len(backing)`, the charge tracks `len(slice)`, and the two are independent: master charges 49 / 1,072 / 33,554,480 for 1 B / 1 KiB / 32 MiB slices while retention stays 64 MiB, so the charge is 49 bytes against 64 MiB in the case that matters. The flat 48 here is more accurate, not less: the parent already carries its own `48 + len`, and master double-counts those bytes for a copy it never made.
- The round-trip gas-drift concern. The 48-as-ref / 54-as-owner size difference is real, but nothing charges gas from it. `GetShallowSize` is read by the GC recount (`Recount` adds to the byte counter without charging gas) and by the store's Object-allocation path (`store.go:477`), which does charge gas but only ever receives an `Object`, never a `StringValue`; the reload paths that do touch strings (`store.go:594`, `realm.go:1659`) are `case StringValue: // do nothing`. So no gas path reads the ref/owner size. Within one execution all nodes hold a value in the same mode, so it does not diverge either. The gap is the ref optimization working as designed: ref shares its parent's bytes, the reloaded owner owns its own.
- The `pb3_gen.go` regeneration ask (the no-op above).

## Questions for Author

- How does this PR sit relative to [#4885](https://github.com/gnolang/gno/pull/4885)? `ltzmaxwell` [asked this on 2026-04-15](https://github.com/gnolang/gno/pull/5082#issuecomment-4251302565) and named the missing half exactly: "correctly tracking allocation/GC recount is necessary". It is the last comment on the PR and went unanswered; the author's last push is `3be8c3b1` on 2026-04-23. Meanwhile #4885 is live (`ltzmaxwell` active 2026-07-12, `thehowl` chasing a second review on 2026-06-22) and implements that accounting: it tracks each string backing as a sorted `[]stringRange`, resolves interior pointers to their containing range, and charges each backing once per GC cycle, so a slice inherits its source's bytes. Not superseded on master (`type StringValue string` is unchanged there), and the two PRs are not the same change: this one buys a gas discount on slicing, #4885 makes the counting correct. But #4885's range tracking would make the `ref` flag unnecessary, so the merge-direction call belongs to the author and a maintainer, not to this review.
