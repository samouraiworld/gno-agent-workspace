# PR #5813: perf(gnovm): recycle runtime blocks through a per-machine pool

URL: https://github.com/gnolang/gno/pull/5813
Author: thehowl | Base: master | Files: 40 | +257 -62
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 697316b4c (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5813 697316b4c`

**TL;DR:** The GnoVM creates a fresh scope object every time a Gno program enters a function, loop, `if`, or `switch`, and throws it away on exit, so heavy programs make millions of short-lived objects that Go's garbage collector then has to clean up. This PR keeps a small per-run stack of discarded scope objects and reuses them instead of allocating new ones, cutting allocations and wall-time sharply on ordinary Gno execution.

**Verdict: APPROVE** — the recycle-safety invariant holds under adversarial probing, the `fallthrough` defer-recycle hole is fixed structurally and its regression test is load-bearing, gas accounting is deterministic and consistent across hit and miss; no blocking concerns. One reviewer-only note: the PR body's "Gas: recycling is cheaper than allocating" section describes a gas redesign (`OpCPUAcquireBlock`, `OpCPUCall` 310→40) that is not in the merged code.

## Summary
Runtime scope and call blocks dominated remaining heap allocations after the byte-access fixes in part 2: 45% of objects in the bytes stdlib suite, 135M of them. The pool recycles a block the moment it is discarded from the machine's block stack, resting on a heap-items invariant: closures capture `HeapItemValue`s rather than blocks, `&local` is heap-promoted by the preprocessor, and frames store indices not block pointers, so a popped runtime block provably has no surviving reference. `acquireBlock`/`releaseBlock` route through every discard site; `releaseBlock` excludes the populations that travel the stack but must not be pooled (static, file/package, defer-site, and realm-attached blocks), zeroes a released block so it retains no references, and a `debugAssert` guard panics if a pooled block is still a pending `Defer.Parent`. Consensus-visible gas shifts only from the `_allocBlock` 528→536 bump (the new `noRecycle` field, +8 B/block); recycling itself charges identical allocation gas to a fresh allocation, so goldens move by small uniform deltas.

## Glossary
- block: GnoVM scope object; a runtime block (call or scope) dies when popped, distinct from static and file/package blocks.
- heap item: heap-promoted variable slot a closure captures instead of the defining block; the invariant the pool rests on.
- new-real: object reachable from the realm graph but pre-ObjectID, the mid-transaction window the recycle guard also excludes.
- gas: metered execution cost; consensus-relevant, so the `_allocBlock` bump and golden shifts are behavior changes.
- filetest: VM-run `.gno` asserted against golden directives; `fallthrough0.gno` is the new regression test.

## Fix
On every block discard site ([`machine.go:1755`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/machine.go#L1755) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L1755), and the `releaseBlocksFrom` calls in `GotoJump`/`PopFrameAndReset`/`PopFrameAndReturn`/`PeekFrameAndContinueFor`/`PeekFrameAndContinueRange`), the popped block is offered to `releaseBlock` instead of left for Go GC. Acquire sites ([`op_call.go:255`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/op_call.go#L255) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/op_call.go#L255), [`op_call.go:585`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/op_call.go#L585) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/op_call.go#L585), and the five statement cases in `op_exec.go`) call `acquireBlock` instead of `Alloc.NewBlock`. The load-bearing constraint is that nothing reachable after a pop points at a runtime block; the defer-site exception is handled by a dedicated `noRecycle` `Block` field set in [`doOpDefer`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/op_call.go#L675-L678) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/op_call.go#L675-L678), and the realm-attached exception by the `IsFinalized() || GetIsNewReal()` guard at [`machine.go:2300`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/machine.go#L2300) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L2300).

## Benchmarks / Numbers
| metric | master | this PR |
|---|---|---|
| Benchdata wall-time (geomean) | 134.2µs | 105.4µs (−21.5%) |
| Benchdata B/op (geomean) | — | −89% |
| Benchdata allocs/op (geomean) | — | −59% |
| bytes suite heap objects | 300M | 165M |
| full pkg/gnolang long mode, 16 cores | 184.6s | 154.0s |
| `_allocBlock` (`unsafe.Sizeof(Block{})`) | 528 | 536 |
| `gas/const` golden | 2343 | 2346 |
| `gc.txtar` GAS USED | 151380803 | 151380819 |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- The `noRecycle` invariant rests on a single line, [`op_call.go:678`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/op_call.go#L678) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/op_call.go#L678), with no production backstop. `Defer.Parent` is read only by the recount GC visitor (`Frame.Visit`), never by defer execution, so a forgotten `setNoRecycle` on some future defer path would silently recycle a still-referenced block in production. The `debugAssert` guard at [`machine.go:2303-2316`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/machine.go#L2303-L2316) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L2303-L2316) catches it, but `debugAssert` is false in production builds. Confirmed behaviorally: removing `b.noRecycle = false` reasoning aside, simulating the bug (clearing the flag on the FALLTHROUGH `bodyStmt` reassignment) makes `fallthrough0.gno` panic the guard under `-tags debugAssert`. Acceptable for this PR; ltzmaxwell's follow-up #5856 proposes dropping `Defer.Parent` entirely, which removes the line and the field.

## Missing Tests
None blocking. The new `fallthrough0.gno` covers the defer-in-fallthrough hole; the existing closure/defer/recover/heap/goto escape filetests cover the recycle invariant and pass under `-tags debugAssert`. A realm-execution (`/r/`) recycle filetest would pin the `IsFinalized() || GetIsNewReal()` guard directly rather than relying on the existing `zrealm*` corpus, but the guard is belt-and-suspenders (runtime blocks are never realm-reachable) so this is not blocking.

## Suggestions
None.

## Open questions
- The PR body's "Gas: recycling is cheaper than allocating" subsection and the `### noRecycle placement` paragraph describe `OpCPUAcquireBlock`, an `AllocateBlock`-only-on-miss split, and re-derived `OpCPUCall` 310→40 / `OpCPUReturnCallDefers` 724→215. None of that is in the merged code: `OpCPUAcquireBlock` does not exist, `OpCPUCall` is still 310 ([`machine.go:1407`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/machine.go#L1407) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L1407)), `OpCPUReturnCallDefers` still 724 ([`machine.go:1527`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/machine.go#L1527) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L1527)), and both `acquireBlock` paths charge `AllocateBlock(numNames)` identically ([`machine.go:2245`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/machine.go#L2245) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L2245), [`alloc.go:686`](https://github.com/gnolang/gno/blob/697316b4c/gnovm/pkg/gnolang/alloc.go#L686) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/alloc.go#L686)). The author evidently reverted to the simpler gas model. Not posted: it is a stale-description matter, not a code defect, and the skill scopes findings to code. Worth a heads-up to the author so the body matches before merge.
- Re-running the full `gnovm/pkg/gnolang` filetest suite under `-tags debugAssert` is not clean on this base: several `zrealm*` tests panic "non-escaped object should not have zero hash", and they fail identically with pooling disabled, so the breakage predates this PR. Consequence: the new `releaseBlock` debugAssert invariant can only be exercised via targeted tests, not a clean full-suite debugAssert run. Not posted: pre-existing, not introduced here.
