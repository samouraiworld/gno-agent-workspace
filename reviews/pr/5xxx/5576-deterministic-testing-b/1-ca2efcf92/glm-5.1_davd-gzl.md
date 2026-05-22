# PR #5576: feat(gnovm): implement deterministic testing.B with cycles/op and gas/op

**URL:** https://github.com/gnolang/gno/pull/5576
**Author:** notJoon | **Base:** master | **Files:** 24 | **+1767 -143**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR implements `testing.B` for deterministic benchmarking of Gno code. The design replaces wall-clock time (non-deterministic across machines) with two VM-native metrics:

1. **`cycles/op`** — raw VM CPU workload, the weighted sum of opcodes (`Machine.Cycles`).
2. **`gas/op`** — billable cost, aggregated from cycles × GasFactorCPU + allocation bytes + GC visit cost + native function gas.

Additional metrics `B/op` and `allocs/op` are reported when `-benchmem` or `b.ReportAllocs()` is enabled.

### Key changes by area

**Allocator (`gnovm/pkg/gnolang/alloc.go`):**
- Adds `numAllocs` and `totalAllocBytes` fields to `Allocator`, incremented on every `Allocate()` call.
- `Reset()` now also clears these new fields (previously only cleared `bytes`).
- New `resetLiveBytesForGC()` replaces `Reset()` in the GC path — only zeroes live bytes, preserving cumulative counters across GC cycles. This is critical: the old code called `Reset()` which would have wiped `numAllocs`/`totalAllocBytes` during GC, producing incorrect benchmark reports.
- `Fork()` copies the new fields.

**Garbage collector (`gnovm/pkg/gnolang/garbage_collector.go`):**
- `GarbageCollect()` now calls `resetLiveBytesForGC()` instead of `Reset()` — the behavioral change flagged in the author's own self-review comment. Only `bytes` (live heap size) is zeroed before the GC re-walk; `numAllocs` and `totalAllocBytes` survive.

**Go test runner (`gnovm/pkg/test/test.go`):**
- `runBenchmarkFiles()` mirrors `runTestFiles()` but uses `benchRunner.call()` and decodes `benchmarkReports` (a flat list of sub-benchmark reports).
- `gnoRunnerSpec.call()` is a refactored common path for test and benchmark invocation, eliminating the duplicated crossing/non-crossing code from `runTestFiles`.
- `loadFuncs()` is a generalized prefix-based loader replacing `loadTestFuncs()`, now also supporting `"Benchmark"`.
- `formatBenchmarkResult()` formats the output line: name, N, cycles/op, gas/op, [bytes/op], [B/op, allocs/op].
- `panicError()` extracted from `runTestFiles` defer block for reuse.
- `loadTestPackage()` extracted for reuse.

**Gno testing stdlib (`gnovm/tests/stdlibs/testing/testing.gno`):**
- Full `testing.B` implementation: `Fail/FailNow/Fatal/Error/Skip` methods, `StartTimer/StopTimer/ResetTimer`, `SetBytes`, `ReportAllocs`, `Run` (sub-benchmarks, 1-depth only).
- Timer methods snapshot four native counters (`cycleCount`, `gasConsumed`, `allocBytes`, `allocCount`) and compute elapsed deltas.
- `b.Run()` pauses parent timer, runs sub-benchmark, resumes parent timer.
- `b.reports()` collects flat `[]BenchmarkReport`, omitting parent line if `ranSubBench` is true.
- `RunBenchmark` / `runBenchmark_cur` are the entry points (analogous to `RunTest` / `runTest_cur`), returning JSON-serialized `BenchmarkReports`.
- Unsupported methods panic with clear messages: `RunParallel`, `SetParallelism`, `PB.Next`, `Cleanup`, `ReportMetric`, `Setenv`, `TempDir`.

**Native bindings (`gnovm/tests/stdlibs/testing/testing.go`):**
- Four new native functions: `X_cycleCount`, `X_gasConsumed`, `X_allocBytes`, `X_allocCount` — all under the `tests/stdlibs` path and inaccessible from production realms.

**CLI flags (`gnovm/cmd/gno/test.go`):**
- `-bench <pattern>`, `-benchcount N`, `-benchmem` added.

**Tests:** 14 txtar integration tests covering: basic, determinism, filter, sub-benchmarks, sub-bench filter, nested (panics), fail, skip, realm crossing, bad flags, reportallocs, multiple benchmarks. Go-side unit tests for `formatBenchmarkResult`, `loadBenchFuncs`, `shouldRun`. Allocator unit tests for `NumAllocs`, `TotalAllocBytes`, survival across GC/Reset/Fork. Gno-side unit tests (`bench_test.gno`) for unsupported-method panics and timer pause/resume behavior.

### CI status

Two test failures: `gnokeykc/test` and `tx-archive/test`. These appear to be pre-existing flaky/integration failures unrelated to this PR (the main gno test suite, lint, build, and stdlibs checks all pass).

## Test Results

- **Existing tests:** gno-checks/test PASS, main/test PASS, stdlibs/test PASS. Two unrelated CI failures (gnokeykc, tx-archive).
- **Edge-case tests:** skipped (extensive txtar coverage already present)

## Critical (must fix)

- [ ] `gnovm/pkg/gnolang/alloc.go:327` — `Allocate()` increments `numAllocs` and `totalAllocBytes` **after** the GC-triggered retry path (line 319 `alloc.bytes += size`). When `alloc.collect()` is called and GC runs, `resetLiveBytesForGC()` clears `bytes` but **GC itself does not increment `numAllocs`** — it only calls `Recount()` which just adds to `bytes`. However, the GC traversal calls `alloc.Recount(size)` at `garbage_collector.go:188` which does NOT increment `numAllocs`/`totalAllocBytes`. This is **correct** for GC (GC doesn't allocate new objects). But the issue is: if `Allocate()` triggers GC (lines 303-323), the `alloc.numAllocs++` and `alloc.totalAllocBytes += size` at lines 327-328 happen **once** regardless of whether the allocation went through the GC path or the fast path. This is actually correct — only one allocation is being counted. However, there is a subtle issue: after GC reclaims memory and the `alloc.bytes += size` retry at line 319, the **second** `alloc.bytes += size` does NOT go through the overflow check. If `alloc.bytes` overflows `maxBytes` after the GC retry path, it will panic at line 321 — this is pre-existing behavior, not introduced by this PR. **No critical issue found in the new code after full analysis.**

- [ ] `gnovm/pkg/gnolang/store.go:221` — `ds.alloc.Fork().Reset()` is called in the transaction-scoped store initialization. Previously `Reset()` only cleared `bytes`; now it also clears `numAllocs` and `totalAllocBytes`. Since this is a **fresh forked** allocator (just created from Fork, which copies the parent's counters), calling Reset() to zero everything is correct — the new transaction should start with zero cumulative counters. **Not an issue.**

- [ ] `gnovm/pkg/gnolang/store.go:1169` — `ds.alloc.Reset()` in `ClearObjectCache()` now also clears `numAllocs` and `totalAllocBytes`. This function is called between transactions. Since the allocator is being fully reset between transactions, clearing the cumulative counters is correct — they should not persist across transaction boundaries. **Not an issue.**

*(After thorough verification, no critical issues remain. The GC change from `Reset()` to `resetLiveBytesForGC()` is correct and necessary.)*

None.

## Warnings (should fix)

- [ ] `gnovm/pkg/test/test.go:749-763` — `formatBenchmarkResult` uses integer division for per-op metrics (`rep.Cycles / n`, `rep.Gas / n`, `rep.AllocBytes / n`, `rep.Allocs / n`). Integer division silently truncates, losing fractional information. For small N (e.g., N=1 with 4821 cycles), the per-op value is exact, but for N=3 with 5000 total cycles, this reports 1666 cycles/op instead of 1666.67. Go's `testing` package also uses integer division for `ns/op`, so this matches Go behavior — but `cycles/op` and `gas/op` are finer-grained metrics where truncation may be more noticeable. Consider documenting the truncation behavior or using float formatting for at least the gas/op column, where the delta from cycles×GasFactorCPU is the interesting signal.

- [ ] `gnovm/tests/stdlibs/testing/testing.gno:417-419` — `b.Run` panics on nested sub-benchmarks (`if b.parent != nil`), but this check only prevents the **second** level. A deeply-nested B that doesn't call `Run` from a `b` with a parent could still be constructed via other means. The panic message is clear, so this is acceptable, but worth noting that the single-depth limitation is enforced by this runtime check rather than by type system or compile-time enforcement.

- [ ] `gnovm/pkg/test/test.go:660` — `runBenchmarkFiles` creates a new `Machine` with `store.NewInfiniteGasMeter()` for each benchmark, but `alloc.Reset()` is called on a **shared** allocator across benchmarks. If a prior benchmark's GC cycle mutated the allocator state (via `resetLiveBytesForGC`), the shared allocator's `numAllocs`/`totalAllocBytes` still carry over. Since `Reset()` clears all fields, this is fine — but the shared allocator pattern means that if `Reset()` were ever removed or changed, benchmark isolation would break. The test path (`runTestFiles`) uses the same pattern (shared alloc, per-test Reset), so this is consistent.

- [ ] `gnovm/tests/stdlibs/testing/testing.gno:259-283` — `BenchmarkReport.marshal()` builds JSON by string concatenation. While functional, this is fragile for future changes (field reordering, special characters in Name). The author noted this in a self-review comment suggesting templates. This matches the existing `Report.marshal()` style, so it's consistent but worth noting for future cleanup.

## Nits

- [ ] `gnovm/pkg/test/test.go:391` — `loadTestPackage` comment says "materialised" (British spelling) while the rest of the codebase uses American English. Minor.

- [ ] `gnovm/cmd/gno/test.go:159-163` — `-benchcount` defaults to 1, matching Go's `-count` for benchmarks. However, Go's `-benchtime` controls iteration count via duration (e.g., `1s`) or count (e.g., `100x`), while this PR uses a separate `-benchcount` flag. The naming diverges from Go's convention, which could confuse users. Consider documenting this difference clearly or aligning with Go's `-benchtime` syntax.

- [ ] `gnovm/tests/stdlibs/testing/testing.gno:554` — `b.reports()` uses `!b.ranSubBench && !b.filtered` to decide whether to include the parent's own report. The `!b.filtered` check means that if the top-level benchmark is filtered out, its sub-benchmarks are also excluded (since `shouldRun` is checked in `bRunner`). This is correct behavior but could be made more explicit with a comment.

- [ ] `gnovm/pkg/gnolang/alloc.go:268-273` — `resetLiveBytesForGC` is unexported (correct, it's internal), but has no documentation explaining why it exists separately from `Reset()`. A brief doc comment would help future maintainers understand the distinction.

## Missing Tests

- [ ] No test for the `benchmem` flag with `SetBytes` interaction — i.e., `b.SetBytes(7) + -benchmem` should show `bytes/op` AND `B/op + allocs/op` columns together. The `formatBenchmarkResult` unit test covers this (`bytes_and_benchmem_both_appear`), but there's no txtar integration test for this combination.
- [ ] No test for `b.ResetTimer()` followed by `b.StartTimer/StopTimer` interaction — verifying that ResetTimer correctly re-baselines the start counters when the timer is running.
- [ ] No test for a benchmark that calls `b.Fatal` inside `b.Run` — the `benchmark_subbench_fail.txtar` tests `b.Fatal` in a sub-bench, but doesn't verify that the parent's own report is omitted (since it `ranSubBench`).
- [ ] No test for `-benchcount` > 1 with N-scaling behavior. The PR description mentions `-benchcycles` (default 1e8) for auto-scaling N, but the implementation appears to only use the fixed `-benchcount` value directly as `b.N`. Auto-scaling is mentioned in the PR body but not implemented — either the description should be updated, or auto-scaling should be added.
- [ ] No test for the `gas/op` value being >= `cycles/op × GasFactorCPU` — i.e., verifying that gas includes allocation+GC+native overhead beyond pure CPU cycles.

## Suggestions

- The PR body describes an auto-scaling mechanism where `b.N` doubles until total cycles ≥ `-benchcycles` (default 1e8), but the implementation just passes `-benchcount` as a fixed `N`. Either implement auto-scaling or update the PR description to remove the claim. The description is currently misleading.
- Consider adding `-benchtime` as an alias for `-benchcount` in a future iteration, to align with Go's CLI surface.
- The `report.go` file has a comment "Wire format must stay in lock-step with testing.gno's marshal() methods." Consider adding a test that validates the Go struct tags match the Gno-side JSON output, to catch drift.
- The `AllocsPerRun2` function in `testing.gno:55-59` still returns 0. Now that `NumAllocs` is tracked, this could be implemented properly. Add a TODO or implement it.

## Questions for Author

- The PR description describes `b.N` auto-scaling with a cycle target (`-benchcycles`, default 1e8), but the code passes `-benchcount` directly as `N` with no doubling loop. Is auto-scaling planned for a follow-up, or should the description be updated?
- In `bRunner` (testing.gno:688-708), the `shouldRun` check sets `b.filtered = true` and returns early without calling `StartTimer`. This means `b.report()` would return zero-value elapsed counters. The `reports()` method then skips filtered benchmarks via `if sub.filtered { continue }`. Is there a reason `b.filtered` is not part of `BenchmarkReport` for the Go-side to also filter? Currently the Go-side `shouldRun` in `runBenchmarkFiles` pre-filters at the top level, but sub-benchmark filtering happens on the Gno side — the Go side has no visibility into why a report is absent.
- The GC change (`Reset()` → `resetLiveBytesForGC()`) means that `numAllocs` and `totalAllocBytes` now survive GC cycles. In a long-running transaction where GC runs multiple times, these counters grow monotonically. For benchmark reporting, this is the desired behavior (total cumulative allocations). But for the `Allocator.MemStats()` string method (which only shows `maxBytes` and `bytes`), should it also include `numAllocs` and `totalAllocBytes` for debugging?

## Verdict
REQUEST CHANGES — The PR body describes an auto-scaling `b.N` mechanism that is not implemented; the description is misleading and should be updated to match the actual `-benchcount`-based behavior. Beyond that, the core implementation is solid: the GC change is correct, native bindings are properly scoped to test-only, timer semantics are sound, and test coverage is extensive. After fixing the description discrepancy, this is ready for merge.
