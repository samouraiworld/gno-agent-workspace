# PR #5813: perf(gnovm): recycle runtime blocks through a per-machine pool

URL: https://github.com/gnolang/gno/pull/5813
Author: thehowl | Base: master | Files: 45 | +657 -67
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: becc5fa87 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5813 becc5fa87`

Round 2. Head advanced 697316b4c → becc5fa87 with two new commits that change PR content (not a base-only move): a recycle/allocate gas split (recycled blocks no longer charge allocation gas; a new `OpCPUAcquireBlock` op; `OpCPUCall` 310→40, `OpCPUReturnCallDefers` 724→215) and a GC tally that counts pooled blocks against the alloc budget. The goldens are regenerated and the full `TestFiles` suite passes at this head; the round-1 APPROVE still holds.

Correction to an earlier draft of this review: an initial run flagged 19 gas/alloc goldens as stale because a subset/isolated `go test -run` reproduced diffs. That was a repro artifact. The gas values depend on block-pool hits and the `alloc_*` values read the shared-store MemStats, both order-dependent, so they only match in a full-suite run. Verified at becc5fa87 under Go 1.25.9: full `TestFiles` is `ok` (33s), zero gas/alloc failures. CI is green legitimately.

**TL;DR:** The GnoVM makes a fresh scope object every time a Gno program enters a function, loop, `if`, or `switch`, and throws it away on exit, so heavy programs churn millions of short-lived objects for Go's garbage collector. This PR keeps a small per-run stack of discarded scope objects and reuses them. This round reworks how that reuse is priced: reusing a block now costs less gas than allocating a fresh one, and a new per-block setup charge replaces gas that used to be folded into the call op.

**Verdict: APPROVE.** The new gas split charges deterministically (the pool starts empty each run, so reverting it restores the old uniform goldens), the committed goldens match, and full `TestFiles` passes at becc5fa87 under Go 1.25.9. Two non-blocking notes: the gas/alloc goldens are now run-order dependent (they pass only as a full suite, so a subset run shows spurious diffs), and a doc comment is stale.

## Summary
Round 1 charged identical allocation gas on the pool hit and miss paths, so pooling was gas-neutral. This round makes gas reflect the work done: a hit charges only `OpCPUAcquireBlock` (setup/recover, 100) and skips allocation gas, while a miss additionally charges `AllocateBlock(max(numNames, 14))` (the real cap-14 malloc). `OpCPUCall`/`OpCPUReturnCallDefers` were re-derived to exclude block creation (310→40, 724→215). `GarbageCollect` now adds each pooled block's retained footprint to the alloc tally via `Recount`, so parking a block in the pool no longer looks like freeing it. The split is deterministic because the per-machine pool starts empty every run (`Machine.Release` drops it), so the hit/miss sequence is a pure function of execution. One side effect: gas now depends on whether a block came from the pool, so the `gas/*` goldens (and the shared-store `alloc_*` MemStats counts) are run-order dependent and match only in a full-suite run. CI runs the whole suite, so it is green and correct; a subset or isolated `go test -run` shows spurious diffs.

## Glossary
- block: GnoVM scope object; a runtime block (call or scope) dies when popped, distinct from static and file/package blocks.
- gas: metered execution cost; consensus-relevant, so the recycle/allocate split, the new op, and the golden shifts are behavior changes.
- filetest: VM-run `.gno` asserted against golden directives; the `gas/*` files assert a `// Gas:` total, the `alloc_*` files a `MemStats` byte count.
- recount: GC re-walk add to the byte tally that does not charge gas; used so surviving (and now pooled) objects are counted without double-billing.

## Fix
The gas model moved from "pooling is gas-neutral" to "a recycle is cheaper than an allocation." `acquireBlock` charges `OpCPUAcquireBlock` on both paths and allocation gas only on a miss ([`machine.go:2237-2261`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/machine.go#L2237-L2261) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L2237-L2261)); the miss path bills the real cap-14 malloc, `AllocateBlock(max(numNames, 14))` ([`alloc.go:690-691`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/alloc.go#L690-L691) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/alloc.go#L690-L691)). `GarbageCollect` re-counts each pooled block by capacity ([`garbage_collector.go:107-109`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/garbage_collector.go#L107-L109) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/garbage_collector.go#L107-L109)). The load-bearing constraint is that the hit/miss sequence be deterministic across validators, which holds because `Machine.Release` resets `blockPool` to nil ([`machine.go:270-278`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/machine.go#L270-L278) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L270-L278)). The goldens are regenerated and match at this head.

## Benchmarks / Numbers
Per-path block-creation gas at becc5fa87 (`allocBlock=568`, `allocBlockItem=40`):

| event | charge | gas |
|---|---|---|
| pool hit | `OpCPUAcquireBlock` | 100 |
| pool miss | `OpCPUAcquireBlock + AllocateBlock(14)` | 100 + 263 = 363 |
| `OpCPUCall` 0-param | re-derived (was 310) | 40 |
| `OpCPUReturnCallDefers` | re-derived (was 724) | 215 |

Run-order sensitivity (full suite vs isolated `go test -run`, at becc5fa87), representative:

| filetest | full suite (golden) | isolated run |
|---|--:|--:|
| `gas/string_eql_diff_len` | 18553 | 18779 |
| `gas/slice_alloc` | 70970911 | 70971024 |
| `alloc_7` (MemStats bytes) | 5747 | 6355 |

The committed goldens are the full-suite values, which is what CI runs and what passes. The isolated-run column is the spurious diff a subset `-run` produces, not a real mismatch.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- The `noRecycle` invariant rests on a single line, [`op_call.go:678`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/op_call.go#L678) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/op_call.go#L678), with no production backstop. `Defer.Parent` is read only by the recount GC visitor (`Frame.Visit`), never by defer execution, so a forgotten `setNoRecycle` on some future defer path would silently recycle a still-referenced block in production. The `debugAssert` guard at [`machine.go:2312-2325`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/machine.go#L2312-L2325) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/machine.go#L2312-L2325) catches it, but `debugAssert` is false in production builds. Confirmed behaviorally: simulating the bug (clearing the flag on the FALLTHROUGH `bodyStmt` reassignment) makes `fallthrough0.gno` panic the guard under `-tags debugAssert`. Acceptable for this PR; ltzmaxwell's follow-up [#5856](https://github.com/gnolang/gno/pull/5856) proposes dropping `Defer.Parent` entirely, which removes the line and the field.
- The doc comment on `newBlockWithValueCap` is now wrong for the pool path: [`values.go:2487-2488`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/values.go#L2487-L2488) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/values.go#L2487-L2488) still says "gas/alloc accounting (AllocateBlock) is by numNames, independent of capacity," but its pool caller `newPooledBlock` now charges `AllocateBlock(max(numNames, 14))` ([`alloc.go:690-691`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/alloc.go#L690-L691) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/alloc.go#L690-L691)), i.e. by capacity. The function itself never calls `AllocateBlock`, so the sentence only described the caller and is now stale.

## Missing Tests
None blocking. The `fallthrough0.gno` regression test still trips the defer-parent assert under `-tags debugAssert` (verified at this head). No test pins the recycle-vs-allocate gas asymmetry directly (that a second acquire of the same scope costs less than the first), but the `gas/*` goldens cover it indirectly.

## Suggestions
- Pooling makes the `gas/*` goldens run-order dependent (gas now varies by pool hit) and the `alloc_*` goldens depend on the shared-store MemStats baseline. They pass as a full suite but a subset or isolated `go test -run` reports spurious diffs. Not a correctness issue and the suite always runs whole, but a one-line comment near the gas testdata would stop the next person from chasing a false failure.
- The recount loop charges `allocBlock + allocBlockItem*int64(cap(b.Values))` with a plain `+=` in `Recount` ([`alloc.go:290-295`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/pkg/gnolang/alloc.go#L290-L295) · [↗](../../../../../.worktrees/gno-review-5813/gnovm/pkg/gnolang/alloc.go#L290-L295)), no overflow guard. Bounded in practice (≤ `blockPoolLimit`=32 blocks × ~1128 B), so it cannot realistically wrap, and the surrounding GC re-walk uses the same unguarded `+=` for every surviving object. Flagging only for symmetry with the `overflow.Addp` used on the charging path; no change needed here.

## Open questions
- Resolved: an earlier draft questioned why `ci / gnovm` is green while a clean clone "fails" `TestFiles`. The failure was a subset/isolated-run artifact, not a real mismatch. Full `TestFiles` passes at becc5fa87 under Go 1.25.9 (`ok`, 33s), so the green check is correct.
- The PR body / ADR "Decision" section ([`adr/pr5813_block_pool.md:50-53`](https://github.com/gnolang/gno/blob/becc5fa87/gnovm/adr/pr5813_block_pool.md?plain=1#L50)) still states acquireBlock "charges `AllocateBlock(numNames)` on both the hit and miss paths" and "the VM-GC counts `len(b.Values)`," which the same ADR's later "Gas" section now contradicts (miss-only allocation gas; GC counts pooled-block capacity). Stale prose, not a code defect; the skill scopes findings to code and bars critiquing the ADR. Worth a heads-up so the body is internally consistent before merge.
