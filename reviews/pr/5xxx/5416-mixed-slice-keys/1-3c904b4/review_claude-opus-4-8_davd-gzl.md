# PR #5416: fix(gnovm): Support mixed key and unkeyed elements in a slice

URL: https://github.com/gnolang/gno/pull/5416
Author: aronpark1007 | Base: master | Files: 10 | +213 -191
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `3c904b4` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5416 3c904b4`

**TL;DR:** Gno crashed on `[]int{4: 14, 25, 1: 33}` (a slice literal mixing explicit indices and bare values), which Go accepts. This routes every slice literal through one opcode that handles both forms, and tightens the preprocessor to reject non-constant indices uniformly.

**Verdict: REQUEST CHANGES** — semantics are correct and the opcode unification is clean, but `main / lint` is red on a stray double blank line, and the unified opcode inherited the sparse-path gas base (966) so the common dense literal `[]int{1,2,3}` now costs 2.5x what it did on master.

## Summary

Closes [#3464](https://github.com/gnolang/gno/issues/3464). Gno panicked on mixed keyed/unkeyed slice literals because `doOpCompositeLit` inspected only `Elts[0].Key` to pick between two opcodes — dense `OpSliceLit` (no keys) and sparse `OpSliceLit2` (all keyed) — and each handler panicked when the loop hit the other kind. The fix deletes the dense opcode and reworks the surviving one to walk the AST twice: first pass computes `length = maxIdx + 1` (unkeyed elements take `prev + 1`), second pass fills, gaps zero-filled. As cleanup it folds the preprocessor's scattered non-const-key guards into one whitelist in `CompositeLitExpr TRANS_LEAVE` that rejects any non-`*ConstExpr` slice/array key.

## Glossary

- `OpSliceLit` / `OpSliceLit2` — VM opcodes for slice composite literals; master had a dense one (342 base) and a sparse keyed one (966 base). PR keeps a single opcode under the `OpSliceLit` name.
- `CompositeLitExpr` — AST node for `T{...}`; each element is a `KeyValueExpr` with an optional `.Key`.
- `TRANS_LEAVE` — preprocessor visitor phase that runs after a node's children are processed.
- `incrCPU(n)` — charges `n` against the CPU gas meter; panics on out-of-gas.

## Fix

Before: `doOpCompositeLit` branched on `x.Elts[0].Key != nil`, and `OpSliceLit2` additionally pushed keys onto the value stack and popped them back in its handler. After: one `doOpSliceLit` pushes values only, reads keys straight from the AST via `kx.(*ConstExpr).ConvertGetInt()`, and mirrors the long-standing [`doOpArrayLit`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/op_expressions.go#L418-L475) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L418-L475). `maxIdx` and `idx` start at -1 so an empty literal yields `length = 0`. The non-const-key rejection moved to a single whitelist at [`preprocess.go:2230-2257`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/preprocess.go#L2230-L2257) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/preprocess.go#L2230-L2257).

## Benchmarks / Numbers

Charged CPU gas vs measured cost of the new unified handler (pure ns/op, this hardware, 20000x×3):

| N | master charge (dense) | PR charge | PR measured (ns) | PR over/under |
|---|---|---|---|---|
| 1 | 370 | 997 | ~240 | 4.1x over |
| 10 | 622 | 1276 | ~550 | 2.3x over |
| 100 | 3142 | 4066 | ~3800 | ~matched |
| 1000 | 28342 | 31966 | ~33000 | ~matched (slope accurate) |

master charge = `342 + 28*N`; PR charge = `966 + 31*N`. The slope (31) tracks the measured per-element cost; the inflated base (966, carried from the deleted sparse opcode) is what overcharges, and the overcharge fades as N grows.

## Critical (must fix)

- **[lint is red on CI]** [`gnovm/pkg/gnolang/op_expressions.go:476-477`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/op_expressions.go#L474-L478) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L474-L478) — two consecutive blank lines between `doOpArrayLit` and `doOpSliceLit` fail `gofmt`; `main / lint` is already failing. Fix: delete one blank line.
  <details><summary>details</summary>

  `gofmt -d gnovm/pkg/gnolang/op_expressions.go` reports a one-line diff removing the blank line at 477. Confirmed on `3c904b4`. The `main / build` failure is a separate environmental toolchain-cache issue, but `main / lint` is a real defect in this diff.
  </details>

## Warnings (should fix)

- **[common dense literal now costs 2.5x gas]** [`gnovm/pkg/gnolang/machine.go:1330,1388`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/machine.go#L1327-L1390) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/machine.go#L1327-L1390) — the unified opcode kept the deleted sparse `OpSliceLit2` base (966) instead of the dense `OpSliceLit` base (342), so every plain `[]T{...}` literal is overcharged. Fix: refit the base for the unified handler, or keep the dense base of 342.
  <details><summary>details</summary>

  master had two constants: dense `OpCPUSliceLit = 342` / slope 28, and sparse `OpCPUSliceLit2 = 966` / slope 31. The PR deletes the dense opcode and assigns the sparse constants (966/31) to the surviving `OpSliceLit`. For the most common shape `[]int{1, 2, 3}` the charge goes from `342 + 28*3 = 426` to `966 + 31*3 = 1059`, a 2.5x increase applied to every dense slice literal in every realm. The slope is fine: benchmarking the new handler (`BenchmarkOpSliceLit_{1,10,100,1000}`) gives ~33000 ns/op at N=1000 against a charge of 31966, so the per-element rate is accurate and large literals are charged correctly; the overcharge is entirely in the fixed base (966 vs a measured fixed cost near 220 ns, and vs the dense fit's 342). Overcharging is not a security risk, but it is a 2.5x regression on the hottest literal shape. A fresh fit lives at `gnovm/cmd/calibrate/`.
  </details>

## Nits

- [`gnovm/tests/files/slice6.gno:9`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/tests/files/slice6.gno#L9) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice6.gno#L9) — no trailing newline; [@notJoon flagged this](https://github.com/gnolang/gno/pull/5416#discussion_r2046693249) and the suggested edit didn't land.
- [`gnovm/tests/files/slice7.gno:20`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/tests/files/slice7.gno#L20) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice7.gno#L20) — same; missing trailing newline. (slice8.gno has it.)
- [`gnovm/tests/files/composite16.gno`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/tests/files/composite16.gno) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/composite16.gno) — the PR makes the interpreter reject non-const slice keys that the Go typechecker already rejected (the `// TypeCheckError:` line predates this PR). On master `composite16.gno` only *executed* because the filetest interpreter path skips typecheck; on-chain (`MsgAddPackage`/`MsgRun` both typecheck) such code could never deploy, so deployed realms are unaffected. Worth one line in the PR body noting the interpreter now matches the typechecker.

## Missing Tests

- **[dup-index on the slice-specific path is untested]** [`gnovm/tests/files/slice8.gno`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/tests/files/slice8.gno) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/tests/files/slice8.gno) — no filetest for `[]int{1, 0: 2}`, where explicit key 0 collides with the implicit index of the leading unkeyed element. Fix: add a panicking filetest asserting `duplicate index 0 in array or slice literal`.
  <details><summary>details</summary>

  Confirmed behaviorally on `3c904b4`: `[]int{1, 0: 2}` panics with `duplicate index 0 in array or slice literal`. The collision detection lives in the new `doOpSliceLit` (`es[idx].IsDefined()` at [op_expressions.go:510](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/op_expressions.go#L510-L512) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L510-L512)) but no filetest exercises it through the slice path, so a future refactor could drop dup-detection silently.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/op_expressions.go:497`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/op_expressions.go#L496-L499) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L496-L499) — `incrCPU(OpCPUSlopeSliceLit * length)` fires before `NewListArray`, so a sparse literal like `[]int{1<<40: 1}` is gas-bounded before it can allocate. Correct and deliberate; a one-line comment would stop a future reorder from moving the allocation above the charge.
- [`gnovm/pkg/gnolang/op_expressions.go:499`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/op_expressions.go#L499) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L499) — `int(length)` truncates if `length > 2^31` on a 32-bit build; production is amd64 so this is theoretical, and the gas charge above caps `length` well below 2^31 anyway. A short comment would preempt a confused reader.
- [`gnovm/pkg/gnolang/op_expressions.go:478-535`](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/op_expressions.go#L478-L535) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/op_expressions.go#L478-L535) — the handler walks `x.Elts` twice (length, then fill). A single pass tracking `length = max(length, idx+1)` while filling would avoid the second walk; minor, only matters at large N.

## Open questions

- The negative-const-key path (`const k = -3; []int{k: 1}`) is caught by the typechecker (`index k (constant -3 of type int) must not be negative`) before the preprocessor's `Sign()` guard at [preprocess.go:2251](https://github.com/gnolang/gno/blob/3c904b4/gnovm/pkg/gnolang/preprocess.go#L2251) · [↗](../../../../../.worktrees/gno-review-5416/gnovm/pkg/gnolang/preprocess.go#L2251) runs, so that guard is defense-in-depth. Not posted: a filetest there would pin the typechecker message, not the guard, so it adds little.
</content>
</invoke>
