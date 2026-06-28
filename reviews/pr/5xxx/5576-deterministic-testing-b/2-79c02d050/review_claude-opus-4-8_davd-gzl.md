# PR #5576: feat(gnovm): implement deterministic `testing.B` with `cycles/op` and `gas/op`

URL: https://github.com/gnolang/gno/pull/5576
Author: notJoon | Base: master | Files: 24 | +1827 -143
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: 79c02d050 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5576 79c02d050`

**TL;DR:** Adds `gno test -bench` so Gno code can be benchmarked. Instead of wall-clock time it reports `cycles/op` (VM opcode workload) and `gas/op` (billable cost), both deterministic across machines, plus optional `B/op` and `allocs/op`.

**Verdict: REQUEST CHANGES** — three things block: the PR body promises `b.N` auto-scaling to a `-benchcycles` target that is not in the code, so the default run reports per-op numbers dominated by one-time setup; the new `TestShouldRun` is flaky (parallel subtests race a package-global regex cache and intermittently fail); and `benchmark_determinism.txtar`'s `cmp` compares empty stdout, so it never checks the determinism it claims to. Core implementation otherwise holds: determinism verified byte-identical, native counters scoped to test-only, timer pause/resume sound, GC counter survival correct.

## Summary
Benchmarks run each `Benchmark*` func in an isolated per-iteration `Machine` with a fresh `NewInfiniteGasMeter()`, reading four monotonic VM counters (`Cycles`, `GasConsumed`, `TotalAllocBytes`, `NumAllocs`) through test-only native bindings and reporting deltas per op. `b.N` is the `-benchcount` value (default 1), used directly with no doubling loop. The allocator gains cumulative `numAllocs`/`totalAllocBytes` counters that must survive GC, so the GC path switches from `Reset()` (which would wipe them) to a new `resetLiveBytesForGC()` that zeroes only live bytes. Sub-benchmarks (`b.Run`, one level) pause the parent timer; the parent's own line is omitted once it ran a sub.

Since the last reviewed commit (ca2efcf92), the only PR-content change is `d5011c61c` (refactor benchmark result formatting): `formatBenchmarkResult` gained a `nameWidth` parameter and fixed-width columns (`benchCol*` consts), the runner pre-filters benchmarks to compute the longest name, and the `benchmark.txtar` line assertion relaxed `\t` to `\s+`. No allocator, GC, native-binding, or timer logic changed this round; those carry from round 1's reads.

## Examples
| Command | N | cycles/op | What the number reflects |
|---|---|---|---|
| `gno test -bench .` (default) | 1 | 27159 | one run, dominated by setup/first-call cost |
| `gno test -bench . -benchcount 100` | 100 | 1465 | setup amortized |
| `gno test -bench . -benchcount 1000` | 1000 | 1231 | converged steady-state |

Same benchmark, same machine: per-op cost swings 22x purely from the chosen N. Go hides this by auto-scaling N until the run is long enough; this PR leaves N at the user's flag, default 1.

## Glossary
- cycles/op — weighted opcode sum per iteration (`Machine.Cycles`), the deterministic VM-CPU baseline.
- gas/op — billable cost per iteration: `cycles × GasFactorCPU` plus allocation, GC, and native-call gas.

## Fix
Three changes close the blockers. Either implement the doubling loop the body describes (scale `b.N` until measured cycles ≥ a `-benchcycles` target) or rewrite the PR body to describe the fixed-`-benchcount` behavior the code ships. Make `TestShouldRun` deterministic by dropping `t.Parallel()` on its subtests, since they share the package-global regex cache in [`util_match.go:174-183`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/util_match.go#L174-L183) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/util_match.go#L174-L183). Make `benchmark_determinism.txtar` capture and `cmp` stderr (where the result lines go) instead of the always-empty stdout.

## Critical (must fix)
None.

## Warnings (should fix)
- **[promised feature absent; default run reports misleading numbers]** [`gnovm/cmd/gno/test.go:159-164`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/cmd/gno/test.go#L159-L164) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/cmd/gno/test.go#L159-L164) — PR body describes `b.N` doubling to a `-benchcycles` target (default 1e8); the code has only `-benchcount` and uses it as a fixed `b.N`.
  <details><summary>details</summary>

  The body states `b.N` "scales to a cycle target" following Go's doubling scheme, stopping at `-benchcycles` (default 1e8). No `-benchcycles` flag exists, and no doubling loop exists: `-benchcount` (default 1) flows straight to `b.N` at [`test.go:677`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test.go#L677) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test.go#L677) and the benchmark body loops `for i := 0; i < b.N; i++`. The practical cost: `gno test -bench .` with the default N=1 reports per-op cost polluted by one-time setup. Measured on the same machine, the same loop reports 27159 cycles/op at N=1, 1465 at N=100, 1231 at N=1000 — a 22x swing driven only by N. Auto-scaling is Go's fix for exactly this. Fix: implement the doubling loop, or change the PR body to describe the fixed-`-benchcount` behavior the code ships.
  </details>

- **[new test fails intermittently]** [`gnovm/pkg/test/test_test.go:160-168`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test_test.go#L160-L168) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test_test.go#L160-L168) — `TestShouldRun`'s parallel subtests share the package-global regex cache in `matchString`, so a concurrent subtest can compile its pattern between this subtest's cache check and match, making the wrong pattern win.
  <details><summary>details</summary>

  `matchString` caches one compiled regex in package-level `matchPat`/`matchRe` with no lock ([`util_match.go:168-183`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/util_match.go#L168-L183) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/util_match.go#L168-L183)). `TestShouldRun` runs each pattern case as a `t.Parallel()` subtest, all calling `shouldRun` → `matchString` concurrently. One subtest can overwrite `matchRe` after another passes the `matchPat != pat` guard but before it calls `matchRe.MatchString`, so the second subtest matches against the first's pattern. The global cache is pre-existing and out of scope; the new test is what surfaces it. `go test -race` on this package flags the race, and even without `-race` the value flake reproduces (`anchored_end_matches_exact_suffix` expected true, got false). Repro: [comment_claude-opus-4-8.md](../../../../../reviews/pr/5xxx/5576-deterministic-testing-b/2-79c02d050/comment_claude-opus-4-8.md). Fix: drop `t.Parallel()` from the subtests.
  </details>

- **[determinism test asserts nothing]** [`gnovm/cmd/gno/testdata/test/benchmark_determinism.txtar:7-14`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/cmd/gno/testdata/test/benchmark_determinism.txtar#L7-L14) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/cmd/gno/testdata/test/benchmark_determinism.txtar#L7-L14) — `cmp run1.txt run2.txt` compares stdout snapshots, but benchmark results print to stderr, so it compares two empty files and passes for any output.
  <details><summary>details</summary>

  Benchmark lines go to `opts.Error` (stderr) at [`test.go:715`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test.go#L715) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test.go#L715); stdout stays empty without `-v`. The txtar does `cp stderr run1-all.txt` then greps it only for line existence, then `cp stdout run1.txt` and `cmp run1.txt run2.txt`. The `cmp` is the only equality check, and it runs on stdout, which is empty in both runs. The test passes whether or not the output is deterministic — it does not test the property it is named for. The underlying determinism does hold (verified separately, byte-identical), so this is a test-coverage gap, not a behavior bug. Fix: `cp stderr` into the compared files, or grep-compare the captured result lines.
  </details>

- **[allocs/op rounds to zero below N allocations]** [`gnovm/pkg/test/test.go:794-795`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test.go#L794-L795) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test.go#L794-L795) — integer division `rep.Allocs/n` reports 0 allocs/op whenever total allocations are fewer than N, even when `B/op` is nonzero.
  <details><summary>details</summary>

  Per-op metrics use integer division (`rep.Cycles / n`, `rep.Gas / n`, `rep.AllocBytes/n`, `rep.Allocs/n`). For a benchmark that allocates 15 objects across N=50 iterations, `allocs/op` truncates to 0 while `B/op` shows 154, an inconsistent pair a reader will misread as "allocates bytes but not objects." Observed live: N=1 shows `7728 B/op 15 allocs/op`; the same benchmark at N=50 shows `154 B/op 0 allocs/op`. Go's `ns/op` truncates too, so cycles/op and gas/op matching is defensible, but the allocs/op-to-zero case is the visible footgun. This is closely tied to the auto-scaling gap: with a properly scaled N the divisor dwarfs the counts and the truncation is harmless; with N=1 default it is not. Fix: at minimum, note the truncation, or report allocs/op with one decimal.
  </details>

## Nits
- [`gnovm/pkg/test/test.go:391`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test.go#L391) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test.go#L391) — "materialised" is British spelling; the codebase uses American English elsewhere.
- [`gnovm/pkg/gnolang/alloc.go:268-273`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/gnolang/alloc.go#L268-L273) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/gnolang/alloc.go#L268-L273) — `resetLiveBytesForGC` has no doc comment explaining why it exists apart from `Reset()` (it zeroes only live bytes so the cumulative benchmark counters survive a GC re-walk).
- [`gnovm/cmd/gno/test.go:159-164`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/cmd/gno/test.go#L159-L164) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/cmd/gno/test.go#L159-L164) — `-benchcount` diverges from Go's `-benchtime`/`-count` naming; a one-line help note on what it controls (fixed `b.N`) would prevent confusion.

## Missing Tests
- **[determinism unverified by the suite]** [`gnovm/cmd/gno/testdata/test/benchmark_determinism.txtar:14`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/cmd/gno/testdata/test/benchmark_determinism.txtar#L14) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/cmd/gno/testdata/test/benchmark_determinism.txtar#L14) — see the determinism Warning; the only repeatability assertion in the suite is vacuous.
- **[no in-package race coverage]** [`gnovm/pkg/test/test_test.go:143-170`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/pkg/test/test_test.go#L143-L170) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/pkg/test/test_test.go#L143-L170) — the package now has parallel tests touching the shared regex cache, but CI runs no `-race`, so the race goes unseen until a flake fails a normal run.

## Suggestions
- [`gnovm/tests/stdlibs/testing/testing.gno:259-283`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/tests/stdlibs/testing/testing.gno#L259-L283) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/tests/stdlibs/testing/testing.gno#L259-L283) — `BenchmarkReport.marshal()` builds JSON by hand-concatenated strings; matches the existing `Report.marshal()` style but is fragile under field changes. Author already noted this in a self-review thread.
- [`gnovm/tests/stdlibs/testing/testing.gno:53-60`](https://github.com/gnolang/gno/blob/79c02d050/gnovm/tests/stdlibs/testing/testing.gno#L53-L60) · [↗](../../../../../.worktrees/gno-review-5576/gnovm/tests/stdlibs/testing/testing.gno#L53-L60) — `AllocsPerRun2` still returns 0 with a TODO; `NumAllocs` is now available to implement it for real.

## Open questions
- The body says sub-benchmarks "inherit N" from the parent; with no auto-scaling, every sub runs at the same fixed `-benchcount`. If auto-scaling lands later, per-sub scaling will need its own loop. Not posted: deferred-scope, only matters once scaling exists.
- `runtime.ReadMemStats(*MemStats)` is added as the future single path but exposes only `{Allocs, MaxAllocs}`, not the new `numAllocs`/`totalAllocBytes`. Not posted: design choice for the author, no current defect.
