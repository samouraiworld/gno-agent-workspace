# PR #5416: fix(gnovm): Support mixed key and unkeyed elements in a slice

URL: https://github.com/gnolang/gno/pull/5416
Author: aronpark1007 | Base: master | Files: 10 | +213 -191
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — correct semantics and clean unification, but ships with a gofmt lint failure (already red on CI), two test files missing trailing newlines, and a gas re-calibration that overcharges small slice literals ~2.5x and undercharges large ones.

## Summary

Closes [#3464](https://github.com/gnolang/gno/issues/3464). Gno panicked on `[]int{4: 14, 25, 1: 33, 34}` (mixed keyed/unkeyed slice literals) — valid Go that the old `doOpCompositeLit` rejected because it inspected only the first element to choose between `OpSliceLit` (unkeyed) and `OpSliceLit2` (keyed). The fix collapses both paths into a single `OpSliceLit` that walks the AST, tracks the running index (`prev + 1` for unkeyed, explicit key otherwise), computes `maxIdx + 1` as the length, and zero-fills gaps. As cleanup, the author also unified the preprocessor's per-type non-const-key guards into a single whitelist in `CompositeLitExpr TRANS_LEAVE` that rejects any non-`*ConstExpr` Key for slice/array types — aligning Gno with Go's "index must be a non-negative integer constant" rule.

## Glossary

- `OpSliceLit` / `OpSliceLit2` — VM opcodes; old code dispatched to one based on first-elt key. PR collapses to single `OpSliceLit`.
- `CompositeLitExpr` — AST node for `T{...}`. Each element is a `KeyValueExpr` with optional `.Key`.
- `TRANS_LEAVE` — preprocessor visitor phase running after children processed.
- `ConvertGetInt` — coerces a `TypedValue` to `IntType` and returns `int64`.
- `incrCPU(n)` — charges `n * GasFactorCPU` against the gas meter; panics on out-of-gas.

## Fix

Before: `doOpCompositeLit` checked `x.Elts[0].Key != nil` to pick `OpSliceLit2` (keyed) vs `OpSliceLit` (unkeyed), and each handler panicked if the loop encountered the other kind — so `[]int{4:14, 25}` and `[]int{25, 4:14}` both blew up. `OpSliceLit2` also pushed keys onto the value stack and popped them back in the handler. After: a single `OpSliceLit` path pushes only values; the handler walks `x.Elts` twice (first to compute `length = maxIdx + 1`, second to fill), reading keys directly from the AST via `kx.(*ConstExpr).ConvertGetInt()` (mirroring how [`doOpArrayLit`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L418-L475) already worked). Initial `maxIdx = -1` and `idx = -1` so that an empty literal yields `length = 0`. The non-const-key check moved from per-NameExpr scattered guards into a single whitelist at [`preprocess.go:2230-2257`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/preprocess.go#L2230-L2257) — and the previously-eager panic at [`preprocess.go:1240`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/preprocess.go#L1234-L1240) was replaced by `return n, TRANS_CONTINUE` so the unified whitelist catches it.

## Benchmarks / Numbers

Empirical per-call cost vs charged gas (charged = `OpCPUSliceLit + OpCPUSlopeSliceLit * length` = `966 + 31*N`):

| N | Pure ns/op | Charged gas | Ratio |
|---|---|---|---|
| 10 | 487 | 1276 | 2.6x overcharged |
| 100 | 4655 | 4066 | 1.1x undercharged |
| 1000 | 72325 | 31966 | 2.3x undercharged |

The base 966 was inherited from old `OpSliceLit2` (which had key-stack-pop overhead the unified handler no longer pays); slope 31 was derived from sparse-fill, not dense-fill. The fit is now linear-but-miscalibrated across the dense range.

## Critical (must fix)

- **[lint red on CI]** [`gnovm/pkg/gnolang/op_expressions.go:477`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L474-L478) — double blank line between `doOpArrayLit` and `doOpSliceLit` fails `gofmt`/`goimports`; `main / lint` is already red.
  <details><summary>details</summary>

  Lines 476-477 are two consecutive blank lines after `doOpArrayLit`'s closing `}`. `gofmt -d` reports a one-line diff; `golangci-lint` on CI flagged both `gofmt` and `goimports`. The `main / build` failure on this PR is environmental (Go toolchain cache tar collision), but `main / lint` is a real PR defect. Fix: delete one of the blank lines.
  </details>

## Warnings (should fix)

- **[gas overcharged 2.6x on small slices, undercharged 2.3x on large]** [`gnovm/pkg/gnolang/machine.go:1330,1388`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/machine.go#L1327-L1390) — `OpCPUSliceLit` base (966) and `OpCPUSlopeSliceLit` slope (31) are the old SliceLit2 constants, fit to a handler that pushed keys onto the value stack and popped them back; the new handler reads keys from the AST and does no such pop.
  <details><summary>details</summary>

  See the Benchmarks table above. The base 966 reflects the cost of an opcode that paid for `PopValues(el * 2)` plus per-key `ConvertGetInt` from the stack. The new path only `PopValues(ne)` and reads keys from immutable AST `*ConstExpr` fields. For `[]int{1, 2, 3}` (the most common shape) the user is charged `966 + 93 = 1059` gas where the old code charged `342 + 84 = 426`. That is a 2.5x gas-cost regression for every dense slice literal in every realm.

  Fix: re-run `BenchmarkOpSliceLit_{1,10,100,1000}` on the unified handler and refit base+slope; the calibration script lives at `gnovm/cmd/calibrate/`. Alternative: keep both opcodes (cheap dense path + general path) and dispatch based on `hasKey` in `doOpCompositeLit` — this is the design [@notJoon suggested earlier in-thread](https://github.com/gnolang/gno/pull/5416#discussion_r2046693283) and was rejected on simplicity grounds, but the gas mis-fit is the cost of that simplification.
  </details>

- **[breaking change unannounced]** [`gnovm/tests/files/composite16.gno`](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/composite16.gno) — code with side-effecting non-const keys (e.g. `[]int{f(): g()}`) used to execute and print; now panics at preprocess. Worth flagging in PR description / changelog.
  <details><summary>details</summary>

  On master, `composite16.gno` (with `x(1): x(2), x(3): x(4)` non-const keys) printed `1\n2\n3\n4\nslice[(0 int),(2 int),(0 int),(4 int)]`. The PR removes those `// Output:` lines and asserts only `// Error: ... slice/array literals may not contain non-const keys`. This is the *correct* semantic (Go rejects it too) but it is a behavior change: any deployed realm with non-const slice/array indices that happened to work on Gno will now refuse to execute. A grep of `examples/` and `gnovm/stdlibs/` finds no current uses, so blast radius is likely zero — but the PR description should call this out as a "no longer accepts non-const keys" tightening so chain operators don't get blindsided. Fix: one sentence in the PR body, e.g. "Also tightens preprocessor: slice/array composite literals with non-const keys now panic uniformly (was inconsistent before)."
  </details>

## Nits

- [`gnovm/tests/files/slice6.gno:9`](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice6.gno#L9) — no trailing newline; [@notJoon flagged this](https://github.com/gnolang/gno/pull/5416#discussion_r2046693249) on round 1 and the suggestion-block edit didn't land.
- [`gnovm/tests/files/slice7.gno:20`](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice7.gno#L20) — same; missing trailing newline. (slice8.gno has it.)
- [`gnovm/pkg/gnolang/op_expressions.go:477`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L474-L478) — same double-blank-line that triggers Critical above; folded here for completeness.
- [`gnovm/pkg/gnolang/op_expressions.go:478-535`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L478-L535) — `doOpSliceLit` walks `x.Elts` twice (once for length, once for fill). For large `ne` with monotonically-increasing keys this is fine, but a single pass tracking both `idx` and `length = max(length, idx+1)` would halve loop overhead. Optional polish; only matters at scale.

## Missing Tests

- **[duplicate-index from unkeyed-after-keyed]** [`gnovm/tests/files/slice8.gno`](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice8.gno) — no test for `[]int{1, 0: 2}` (key 0 collides with implicit unkeyed-first idx 0).
  <details><summary>details</summary>

  I verified locally: `[]int{1, 0: 2}` correctly panics with `duplicate index 0 in array or slice literal`. The PR has duplicate-index *coverage* indirectly via the existing path in `doOpArrayLit`, but no filetest exercises the slice-specific code added here. One four-line filetest pins the contract so a future refactor doesn't regress dup-detection silently. Fix: append a panicking case to slice8.gno or add slice9.gno with `// Error: duplicate index 0 in array or slice literal`.
  </details>

- **[const-folded negative key]** [`gnovm/pkg/gnolang/preprocess.go:2251`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/preprocess.go#L2244-L2257) — covered by `convertType→ConvertGetInt` semantics but no filetest asserts that `const k = -3; []int{k: 1}` panics with the negative-index message.
  <details><summary>details</summary>

  Verified manually: it panics correctly. But the only filetest hitting this branch is for inline literal `-1` keys (which the parser may special-case). A named-const negative path exercises the post-fold sign check at `preprocess.go:2251`. Cheap insurance against a future "I'll just resolve names later" refactor.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/op_expressions.go:497`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L497) — `m.incrCPU(OpCPUSlopeSliceLit * length)` fires before the allocation. Good — guarantees gas-bounded sparse-index DoS. Worth a one-line comment explaining the ordering so a future reorder doesn't move the allocation above the gas charge.
- [`gnovm/pkg/gnolang/op_expressions.go:499`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L499) — `int(length)` truncates on 32-bit platforms if `length > 2^31`. Gno runs on amd64 in production so this is theoretical, but a `// length fits in int because AllocateListArray's gas budget caps it well below 2^31` comment would preempt a confused reader.

## Questions for Author

- Did you re-run `gnovm/cmd/calibrate/op_bench_analysis.txt` for the unified opcode, or are 966/31 carried over verbatim from the old SliceLit2 fit? The benchmark numbers in this review suggest a fresh refit would drop the base substantially.
- The `// TypeCheckError:` line in [`composite16.gno`](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/composite16.gno#L17) still asserts Go's type-checker rejects non-const keys; the new `// Error:` line asserts Gno's preprocessor now rejects them too. Is the parallel-track assertion still wired up, or has it become redundant?
- `OpSliceLit` opcode value moved from `0x4D` to `0x4E` (the old `OpSliceLit2` slot, [`machine.go:1123`](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/machine.go#L1120-L1124)). The `_Op_map` in `string_methods.go` was regenerated. Is there anywhere outside `string_methods.go` and `machine.go` that hard-codes opcode integers (e.g. on-chain serialized state, debug dumps)?
