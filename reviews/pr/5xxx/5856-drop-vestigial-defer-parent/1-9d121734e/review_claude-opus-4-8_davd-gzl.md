# PR #5856: perf(gnovm): drop vestigial Defer.Parent; recycle defer-origin blocks

URL: https://github.com/gnolang/gno/pull/5856
Author: ltzmaxwell | Base: dev/morgan/gnovm-block-pool | Files: 45 | +245 -91
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 9d121734e (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5856 9d121734e`

**TL;DR:** When a function schedules a `defer`, the VM was holding a pointer back to the scope block the defer was written in. This PR shows nothing reads that pointer at defer-execution time, removes it, and lets those blocks be reclaimed sooner. The deferred call rebuilds its scope from the function value itself, so behavior is unchanged.

**Verdict: APPROVE** — clean revert of #5813's `noRecycle` guard once `Defer.Parent` is shown vestigial; defer semantics match Go, all goldens revert to pre-#5813, the new debugAssert is a strictly better recycle-safety guard. Only caveat is shared with the whole stack: the gas-golden shifts are consensus-relevant and land via the stack, not this PR alone.

## Summary
`Defer.Parent` pinned a defer's origin block so the GC recount would visit it; the field was written at [`doOpDefer`](https://github.com/gnolang/gno/blob/9d121734e/gnovm/pkg/gnolang/op_call.go#L673) and read only by the recount walk in [`Frame.Visit`](https://github.com/gnolang/gno/blob/9d121734e/gnovm/pkg/gnolang/garbage_collector.go#L510) plus #5813's `releaseBlock` debugAssert. Defer execution resolves its scope from [`dfr.Func.GetParent` + copied `Captures`](https://github.com/gnolang/gno/blob/9d121734e/gnovm/pkg/gnolang/op_call.go#L584-L594), never from `Defer.Parent`, so the field only ever served as an accounting root. Dropping it (and the `noRecycle`/`setNoRecycle`/`isNoRecycle` machinery #5813 added to keep those pinned blocks out of the pool) lets defer-origin blocks recycle and shrinks `Block` 536→528 bytes; the recount now removes a popped, defer-pinned block it previously counted, which is both deterministic and more correct since the recount only ever drops provably-dead blocks.

## Glossary
- gas: metered CPU+memory cost; consensus-relevant, so any shift is a behavior change.
- filetest: a `.gno` file run by the VM and asserted against `// Output:` / `// Gas:` goldens.
- Allocator: VM component tracking allocation and charging allocation gas; the recount re-derives live bytes each GC.

## Fix
Before, `doOpDefer` called `lb.setNoRecycle()` on the last block and stored it as `Defer.Parent`; `releaseBlock` then skipped any `noRecycle` block and asserted no pooled block was still a pending `Defer.Parent`. After, `doOpDefer` records only `Func`/`Args`/`Source`, `releaseBlock` drops the `noRecycle` check, and the recount in `Frame.Visit` no longer visits `dfr.Parent`. The load-bearing fact is that [`doOpReturnCallDefers`](https://github.com/gnolang/gno/blob/9d121734e/gnovm/pkg/gnolang/op_call.go#L584-L594) reconstructs the defer's scope block from `dfr.Func.GetParent(m.Store)` and re-copies `fv.Captures`, identically to `doOpCall`, so the origin block carries nothing the defer needs.

## Benchmarks / Numbers
| Value | Before (this PR's base, #5813) | After (this PR) |
|---|---|---|
| `sizeof(Block)` | 536 | 528 |
| `gas/const.gno` | 2346 | 2343 |
| `gnokey_gasfee` GAS USED | 2815593 | 2815592 |
| `restart_gas` GAS USED (1st) | 2839193 | 2839188 |
| `alloc_0.gno` bytes | 6974 | 6918 |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. Coverage is unusually thorough for the change size: execution-equivalence under forced pool churn ([`defer_block_recycle.gno`](https://github.com/gnolang/gno/blob/9d121734e/gnovm/tests/files/defer_block_recycle.gno#L1) · [↗](../../../../../.worktrees/gno-review-5856/gnovm/tests/files/defer_block_recycle.gno#L1)), cross-restart realm persistence ([`defer_realm_recycle.txtar`](https://github.com/gnolang/gno/blob/9d121734e/gno.land/pkg/integration/testdata/defer_realm_recycle.txtar#L1) · [↗](../../../../../.worktrees/gno-review-5856/gno.land/pkg/integration/testdata/defer_realm_recycle.txtar#L1)), the recount-exclusion golden ([`alloc_defer_gc.gno`](https://github.com/gnolang/gno/blob/9d121734e/gnovm/tests/files/alloc_defer_gc.gno#L1) · [↗](../../../../../.worktrees/gno-review-5856/gnovm/tests/files/alloc_defer_gc.gno#L1)), and a `debugAssert`-gated recycle-safety invariant in `GarbageCollect`.

## Suggestions
None.

## Notes on verification
Verified on 9d121734e (checks beyond what CI shows):

- **Go parity.** Replicated all four `defer_block_recycle.gno` cases (named-return mutation through a recycled call block, per-iteration defers across pool churn, `continue`-driven block release, nested defer) in a real Go program; output is byte-identical to the filetest `// Output:` block, including `namedRet: 6` and the reverse defer ordering. Dropping `Defer.Parent` changes no defer semantics.
- **`Defer.Parent` is genuinely vestigial.** No reference to the field or to `dfr.Parent` survives anywhere in the tree; `doOpReturnCallDefers` resolves scope from `dfr.Func.GetParent(m.Store)` (op_call.go:584), the same path `doOpCall` uses, and captured vars stay GC-reachable through `dfr.Func` (the recount visits `dfr.Func` and `dfr.Args`, garbage_collector.go:494-512).
- **`sizeof(Block) == 528`** measured directly; the `init()` assert at [alloc.go:159](https://github.com/gnolang/gno/blob/9d121734e/gnovm/pkg/gnolang/alloc.go#L159) (`check("_allocBlock", _allocBlock, unsafe.Sizeof(Block{}))`) enforces it, so a future field add that desyncs the constant fails at startup.
- **New debugAssert premise holds.** `GCCycle` is bumped once per GC (garbage_collector.go:84), the visitor stamps `SetLastGCCycle(gcCycle)` on every reached object (line 234), and `releaseBlock` zeroes the block (`*b = Block{...}`, machine.go:2307), so a pooled block has `LastGCCycle == 0` unless this recount touched it. The check sits after the full traversal and before byte accounting, so any pooled block carrying the current cycle was reached and the assert fires. It is reference-path agnostic, a strict improvement over the old `releaseBlock` assert that only walked the `Defer.Parent` path. It never fires on the current suite.
- **Goldens.** `TestFiles` (non-short and short), the gas/defer/recover units, vm `Gas`, and the new/modified integration txtar all pass on the PR head. `defer_block_recycle` and `fallthrough0` also pass under `-tags debugAssert`.

## Open questions
- The gas-golden shifts are real consensus-affecting deltas, but they are measured against #5813 (this PR's unmerged base), not master. Net effect on master depends on the whole stack merging; the PR states the goldens "revert exactly to pre-#5813," which the diffs confirm. Not posted: it is a stack-ordering observation, not an action for this PR.
- `alloc_defer_gc.gno`'s byte golden (7344) only matches in the full `TestFiles` run, not `-run alloc_defer_gc` in isolation (7952), as its own comment documents and consistent with the other `alloc_*` MemStats goldens. Not posted: documented and intentional.
