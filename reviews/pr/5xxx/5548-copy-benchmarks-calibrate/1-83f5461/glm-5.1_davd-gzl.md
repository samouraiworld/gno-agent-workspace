# PR #5548: chore(gnovm): wire Copy/UnrefCopy benchmarks into calibrate pipeline

**URL:** https://github.com/gnolang/gno/pull/5548
**Author:** jaekwon | **Base:** master | **Files:** 5 | **+112 -22**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

Rewrites three existing copy/unrefCopy benchmark functions (`BenchmarkCopyDataToList_*`, `BenchmarkCopyListToData_*`, `BenchmarkUnrefCopy_Int_*`) to use the standard `benchMachine()` + `bm.InitMeasure/SwitchOpCode` + `reportBenchops()` wiring so they emit `ns/op(pure)` and `alloc-gas/op` in the format the calibrate pipeline scripts expect. Renames with `BenchmarkOp*` prefix to match the parse regex in `plot_fits.py` / `gen_analysis.py`.

Adds corresponding family entries in both Python scripts and the `PARAM_GO_NAMES` mapping, so `gen_analysis.py` will emit `OpCPUSlopeCopyPrimitive` (from CopyDataToList, the slower helper) and `OpCPUSlopeCopyElement` (from UnrefCopy on IntType). Appends local M2 arm64 benchmark data. Regenerates `op_gas_fits.png` (visually unchanged since DO data doesn't yet contain new benchmarks). Does **not** change `machine.go` constants — those remain placeholder values per project policy (refreshed only from DO data).

## Test Results

- **CI:** 1 failure in `gno.land/pkg/integration` — `panic: txDispatcher subscription unexpectedly closed` in `tm2/pkg/bft/rpc/core/mempool.go:413`. **Unrelated** to this PR (flaky integration test). All other checks pass.
- **Codecov:** All modified lines covered.
- **Benchmark data** (M2 arm64): Results look consistent and linear:
  - CopyDataToList: ~1.0 ns/elem (1k→1m scales linearly)
  - CopyListToData: ~0.66 ns/elem (1k→1m scales linearly)
  - UnrefCopy_Int: ~6.0 ns/elem (1k→100k scales linearly)
  - All show 0 alloc-gas/op, as expected for these copy paths.

## Critical (must fix)

None.

## Warnings (should fix)

1. **machine.go:1413 comment is stale vs M2 data** — `gnovm/pkg/gnolang/machine.go:1416` says `~20 ns/elem M2` for `OpCPUSlopeCopyElement`, but the new M2 benchmark data shows ~6 ns/elem. While this PR correctly does not change constants (per project policy), the comment is misleading. Consider updating the comment to reflect the measured ~6 ns/elem M2 value, or note it as a placeholder estimate pending DO re-benching.

## Nits

1. **UnrefCopy lacks a 1m variant** — `gnovm/pkg/gnolang/bench_ops_test.go:5731` only has 1k/10k/100k, while CopyDataToList and CopyListToData include 1m. Three points is sufficient for a linear fit, but consistency with the other copy families would be nice (assuming the ~6s/iter runtime is acceptable, or could be gated by a `-benchtime` flag).

## Missing Tests

None — these are benchmarks, not production code. The benchmarks themselves are the tests for the calibrate pipeline wiring.

## Suggestions

1. The `m` variable in `benchOpCopyDataToList` and `benchOpCopyListToData` (`bench_ops_test.go:5662`, `bench_ops_test.go:5686`) is only used for `defer m.Release()`. The side effect of calling `benchMachine()` (setting `benchAllocMeter`) is what matters. This matches the existing pattern of all other benchmarks in the file, so it's fine, but a brief comment like `// sets benchAllocMeter for reportBenchops` could clarify intent.

## Questions for Author

1. The `PARAM_GO_NAMES` entry for `CopyListToData` is `(None, None)` — meaning it generates no Go constant and is only plotted for comparison. Should the plot legend or `gen_analysis.py` output explicitly note that CopyListToData shares the CopyPrimitive slope from CopyDataToList? This is documented in the code comments but not in the generated report.
2. Is the ~6 ns/elem M2 result for UnrefCopy (vs the ~20 ns/elem in the machine.go comment) expected? The comment says "~20 ns/elem M2 × 2", but actual measurement is ~6. Is this a known discrepancy or does the old benchmark measure something different?

## Verdict

**Approve with minor suggestions.** Clean, well-structured PR that correctly wires existing benchmarks into the calibrate pipeline following established patterns. The Python script entries are consistent with the benchmark names and correctly map to the existing `OpCPUSlopeCopyPrimitive`/`OpCPUSlopeCopyElement` constant names. The only substantive concern is the stale comment in machine.go, which should be updated to avoid confusion. The CI failure is unrelated.
