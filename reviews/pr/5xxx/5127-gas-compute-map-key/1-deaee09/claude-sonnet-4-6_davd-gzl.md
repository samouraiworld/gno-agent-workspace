# PR #5127: fix: consume gas on ComputeMapKey

**URL:** https://github.com/gnolang/gno/pull/5127
**Author:** Villaquiranm | **Base:** master | **Files:** 10 | **+205 -7**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

Before this PR, every map key computation — triggered by map indexing (`m[k]`), map assignment (`m[k] = v`), and `delete(m, k)` — ran the `ComputeMapKey` function at zero gas cost. For complex key types (large byte arrays, deeply nested structs), this function walks the value tree and allocates a string representation proportional to the key's size. Without gas, an attacker could encode an unbounded amount of work by using large composite keys without exhausting gas.

The fix has two components:

1. **Flat CPU charge**: A constant `OpCPUComputeMapKey = 10` (10 ns equivalent) is charged via `store.ConsumeGas(OpCPUComputeMapKey, GasComputeMapKeyDesc)` at the entry of `ComputeMapKey`. This charge fires for every recursive call (once per node in the key's type tree). The constant is explicitly marked as needing calibration with a `// TODO` comment.

2. **Allocation charge**: A deferred closure calls `store.GetAllocator().Allocate(int64(len(bz) - childrenLength))` after the function body, charging allocation gas for the bytes produced at the current level only. `childrenLength` accumulates the sizes of recursively-produced child keys so the parent does not double-count bytes already charged by children. This deferred charge runs even on early-return NaN paths.

**Interface change**: `ConsumeGas(gas int64, descriptor string)` is added to the `Store` interface. The existing private `consumeGas` method on `defaultStore` is now exposed as a public `ConsumeGas` wrapper. The new method on the interface requires no changes to other implementations because `defaultStore` (and its `transactionStore` wrapper) is the only concrete `Store` implementation in the codebase.

**Tests**: Five new filetests under `gnovm/tests/files/gas/` cover small and large byte-array keys, small and large struct keys, and the allocation-limit exceeded case. A new `values_bench_test.go` adds benchmarks for string, int, byte-array (at various sizes), and int-array keys. Existing `TestComputeMapKey` and `TestComputeMapKey_collisions` tests are updated to pass a real `Store` (previously `nil`), since the nil path would now panic at the new `store.ConsumeGas` call.

## Test Results

- **Existing tests:** FAIL — `gnovm/pkg/gnolang` fails on 4 gas golden file tests introduced by this PR. All other packages pass.
- **Edge-case tests:** skipped

Failures:
```
TestFiles/gas/compute_map_key_big_bytes.gno:   expected 134224443, got 19959108
TestFiles/gas/compute_map_key_big_struct.gno:  expected 218272173, got 126726778
TestFiles/gas/compute_map_key_small_bytes.gno: expected 6687,       got 7854
TestFiles/gas/compute_map_key_small_struct.gno: expected 6143,      got 5505
```

Root cause: the last merge from master (commit `deaee09`, a merge of `master` into the PR branch) pulled in gas-model changes from upstream (e.g. `#5291` "parameterize and calibrate gas model", `#5587` "fix regression introduced by #5291") that changed the absolute gas numbers, but the golden values in the test files were not updated to match. A prior commit `9d098312c` ("fix(gnovm): update gas golden values") had already updated them once after a rebase, but the subsequent merge invalidated them again. The `compute_map_key_exceed_alloc.gno` test passes.

## Critical (must fix)

- [ ] `gnovm/tests/files/gas/compute_map_key_big_bytes.gno:13`, `compute_map_key_big_struct.gno:24`, `compute_map_key_small_bytes.gno:13`, `compute_map_key_small_struct.gno:20` — Gas golden values are stale and do not match actual execution. All four new gas filetests fail. The author must re-run the tests after a final rebase onto master and update the `// Gas:` values to match the current gas model. Use `go test ./gnovm/pkg/gnolang/... -run TestFiles/gas/compute_map_key -update` (or equivalent) once merged/rebased.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/machine.go:1469-1470` — `OpCPUComputeMapKey = 10` is not calibrated from benchmarks. The comment says "TODO: fix an accurate value with benchmarks." The benchmarks exist (`values_bench_test.go`) but were not used to derive this value before opening the PR. The flat 10-ns charge fires once per node in the key's type tree, not once per byte. For a `[1<<25]byte` key the flat charge is still 10 ns, while the allocation gas appropriately scales with size. This means the CPU overhead of iterating the byte data is not represented. The benchmarks should be run and the constant should be calibrated before merge.

- [ ] `gnovm/pkg/gnolang/values.go:1566` — `store.ConsumeGas(OpCPUComputeMapKey, ...)` is routed through the store's gas meter, not `m.incrCPU`. Every other CPU gas charge in `machine.go`, `op_expressions.go`, `uverse.go`, etc. goes through `m.GasMeter.ConsumeGas` (via `m.incrCPU`). In production both meters are the same object (`ctx.GasMeter()`), so the charge lands in the right place. But this is an asymmetry: map key CPU gas bypasses the `m.Cycles` counter (updated in `incrCPU`), which is used for telemetry/debugging. Consider routing through the allocator or a helper that also increments `m.Cycles`.

- [ ] `gnovm/pkg/gnolang/uverse.go:766,771` — The `delete(m, k)` builtin calls both `GetValueForKey` and `DeleteForKey`, both of which call `ComputeMapKey`. This double-charges the flat CPU gas for a single delete. This is likely acceptable but the author should confirm this is intentional.

## Nits

- [ ] `gnovm/tests/files/gas/compute_map_key_big_bytes.gno:6-7` and `compute_map_key_small_bytes.gno:6-7` — The comment says "We should have different gas depending on the size of the array" which is accurate, but the trailing blank line before the `// Gas:` comment is inconsistent with `compute_map_key_exceed_alloc.gno` (no blank line). Minor style inconsistency across the new test files.

- [ ] `gnovm/pkg/gnolang/store.go:1110-1112` — `ConsumeGas` is a one-line wrapper around `consumeGas`. The private `consumeGas` was intentional (all internal callers use the unexported version). Now that `ConsumeGas` is on the interface, the private wrapper is redundant — consider making `consumeGas` call through `ConsumeGas` or just inlining the nil-check.

## Missing Tests

- [ ] No test for map-key gas with pointer types (`*T` as map key). `*PointerType` is a valid key and has its own encoding path in `ComputeMapKey`; it should get a filetest.
- [ ] No gas test for `delete(m, k)` to verify the double-charge is intentional and measurable.
- [ ] No integration test verifying that a map-heavy program now exhausts gas where it previously did not — the pre-fix DoS vector is not demonstrated by any test.

## Suggestions

- The `childrenLength` variable name is slightly misleading: it represents the total byte-length of child keys already charged by recursive calls, not a count of children. A name like `alreadyChargedBytes` or `childKeyBytes` would clarify intent. (`gnovm/pkg/gnolang/values.go:1578`)
- The benchmarks in `values_bench_test.go` accumulate gas across all iterations and divide by `b.N` for reporting. Because the GasMeter is shared across iterations, later iterations have a higher base. This does not affect correctness of `gas/op` (the total is divided by count), but if `b.N` varies across benchmark runs (as it does for `-benchtime`), the results will be accurate. However, if the GasMeter overflows (unlikely with `1 << 62` limit), the benchmark silently panics mid-run. Resetting or recreating the store per iteration would make individual iterations independent and also enable `b.ReportAllocs()`. (`gnovm/pkg/gnolang/values_bench_test.go:10-17`)

## Questions for Author

- Why is the flat `OpCPUComputeMapKey = 10` applied at the top of the recursive function rather than once at the call site in `GetPointerForKey` / `GetValueForKey` / `DeleteForKey`? A top-level call from the machine charges once for a primitive key (correct) but also once per struct field or array element (because the recursive child calls also hit the top of the function). This means a 1000-element array of ints charges 1001 × 10 ns of flat overhead. Is that intentional, or should the flat charge only apply to the root call?
- Was ADR considered for this change? The AGENTS.md requires an ADR for non-trivial AI-assisted PRs. While this may not be AI-assisted, the design decision (store-level gas vs machine-level gas, flat vs proportional) is worth documenting for future calibration work.

## Verdict

REQUEST CHANGES — The four new gas golden file tests fail (stale values after a master merge), making CI red; the flat gas constant is explicitly unfinished (TODO comment) and uncalibrated; and the asymmetry between store-level and machine-level gas charging warrants discussion before the pattern spreads.
