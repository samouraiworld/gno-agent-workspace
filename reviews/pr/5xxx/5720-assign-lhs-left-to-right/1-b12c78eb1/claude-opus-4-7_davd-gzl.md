# PR #5720: fix(gnovm): assign LHS in left-to-right order for AssignStmt

URL: https://github.com/gnolang/gno/pull/5720
Author: ltzmaxwell | Base: master | Files: 4 | +195 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `b12c78eb1` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5720 b12c78eb1`

**Verdict: APPROVE** — correct, minimal fix for a real Go-spec violation; tests are extensive and the single-LHS fast path keeps the dominant case allocation-free. CI failures (`gno-checks/lint` on `sealviolation`, integration `params_valset_rotation_throttle`) are unrelated to this diff and reproduce on master.

## Summary

`doOpAssign` popped LHS pointers in reverse AND wrote them in reverse, so `a, a, a = 1, 2, 3` left `a == 1` (Go spec §Assignments requires left-to-right, i.e. `a == 3`). Fix splits the loop into two: pop pointers in reverse (the LIFO discipline required to recover `lvs[i]` from sub-evals stacked by `op_exec.go:518-521 / PushForPointer`), then assign in forward order. Single-LHS gets a fast path that skips the buffer allocation.

```
Old:  for i = N-1..0:  pop lhs[i] -> assign rvs[i]   # write order: N-1..0  (WRONG)
New:  for i = N-1..0:  pop lhs[i] -> lvs[i]
      for i = 0..N-1:  assign lvs[i] <- rvs[i]       # write order: 0..N-1  (RIGHT)
```

## Glossary

- `PushForPointer` — pushes the sub-expressions of an LHS (e.g. `X`, `Index` for `IndexExpr`) onto the eval stack; consumed later by `PopAsPointer`.
- `PopAsPointer` — pops the LHS sub-eval values and constructs a `PointerValue` for assignment; for maps, creates the entry if absent.
- `Block.Blank` — single shared slot all `_` writes land in for a given block.

## Fix

Before, the loop popped each LHS pointer from the stack and called `Assign2` immediately, which produced right-to-left writes whenever LHS targets aliased (same variable, same map key, same `*p`, same field). After, [`gnovm/pkg/gnolang/op_assign.go:32-53`](https://github.com/gnolang/gno/blob/b12c78eb1/gnovm/pkg/gnolang/op_assign.go#L32-L53) · [↗](../../../../../.worktrees/gno-review-5720/gnovm/pkg/gnolang/op_assign.go#L32-L53) splits the work: a single-LHS fast path skips the buffer (the dominant case), and the multi-LHS path buffers `len(s.Lhs)` pointers in reverse before writing in forward order. The reverse-pop is load-bearing — see [`machine.go:2717-2732`](https://github.com/gnolang/gno/blob/b12c78eb1/gnovm/pkg/gnolang/machine.go#L2717-L2732) · [↗](../../../../../.worktrees/gno-review-5720/gnovm/pkg/gnolang/machine.go#L2717-L2732) (IndexExpr `PopAsPointer` consumes `iv` then `xv` from the value stack, in the inverse of `PushForPointer`'s push order at [`machine.go:2594-2600`](https://github.com/gnolang/gno/blob/b12c78eb1/gnovm/pkg/gnolang/machine.go#L2594-L2600) · [↗](../../../../../.worktrees/gno-review-5720/gnovm/pkg/gnolang/machine.go#L2594-L2600)).

The fix also tightens Go-spec compliance on panics during LHS resolution: previously a panic at `PopAsPointer(lhs[k])` could leave `lhs[k+1..N-1]` already assigned; now all pointers are resolved before any write, so a recoverable panic leaves no partial assignment. Not flagged below — strictly an improvement.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/op_assign.go:30-31`](https://github.com/gnolang/gno/blob/b12c78eb1/gnovm/pkg/gnolang/op_assign.go#L30-L31) · [↗](../../../../../.worktrees/gno-review-5720/gnovm/pkg/gnolang/op_assign.go#L30-L31) — the fast-path comment says "no duplicate-LHS ambiguity"; that's true but understates it. With one LHS, there's no ordering question at all — no comment needed beyond "single-LHS path skips the buffer". Trim if you respin.
- [`gnovm/tests/files/assign41.gno:64-72`](https://github.com/gnolang/gno/blob/b12c78eb1/gnovm/tests/files/assign41.gno#L64-L72) · [↗](../../../../../.worktrees/gno-review-5720/gnovm/tests/files/assign41.gno#L64-L72) — `multiBlankMixedType` is honest about being a smoke test, but it doesn't exercise the surviving-type effect at all (no realm, no finalization). Either trim it (the other six functions already cover the fix) or upgrade it to a filetest under a realm that exercises `assertTypeIsPublic`. Either is fine; in the current form it pads the file without adding coverage.

## Missing Tests

- [`gnovm/pkg/gnolang/op_assign_test.go`](https://github.com/gnolang/gno/blob/b12c78eb1/gnovm/pkg/gnolang/op_assign_test.go) · [↗](../../../../../.worktrees/gno-review-5720/gnovm/pkg/gnolang/op_assign_test.go) — no test for the panic-during-resolution invariant.
  <details><summary>details</summary>

  The new code resolves all LHS pointers before any `Assign2` call. That's a behavioral change: previously, a panic mid-loop (e.g. `a[0], a[bad_idx], a[1] = 1, 2, 3` where `bad_idx` is out-of-range) could leave `a[0] = 1` if the reverse-pop reached LHS 1 before LHS 0 panicked — but actually under the old code the reverse pop hit LHS 2 first (`a[1]`, fine), then LHS 1 (`a[bad_idx]`, panic), and never touched LHS 0. So the old code happened to also leave nothing assigned in this specific shape. Worth a `defer recover()` Gno test asserting "no partial assignment on out-of-bounds LHS" to lock the invariant. Low priority — the assigned-order tests already gate the headline bug.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/op_assign.go:44`](https://github.com/gnolang/gno/blob/b12c78eb1/gnovm/pkg/gnolang/op_assign.go#L44) · [↗](../../../../../.worktrees/gno-review-5720/gnovm/pkg/gnolang/op_assign.go#L44) — the `make([]PointerValue, len(s.Lhs))` allocates every multi-LHS assignment. For the typical `a, b = c, d` shape (N=2) this is a small heap alloc per execution. If hot-path microbench shows it matters, a stack-allocated `[8]PointerValue` slice header would cover all observed N in the corpus without an alloc. Not pressing — the fast path already handles N=1, which is the overwhelming case.
  <details><summary>details</summary>

  Pattern: `var stack [8]PointerValue; lvs := stack[:0]; if n := len(s.Lhs); n > cap(stack) { lvs = make([]PointerValue, n) } else { lvs = lvs[:n] }`. Run `BenchmarkOpAssign_{1,10,100,1000}` (already in `bench_ops_test.go:2667-2670`) before/after to measure. Skip if the numbers don't move.
  </details>
- ADR: the AI agents guide ([`AGENTS.md:83-101`](https://github.com/gnolang/gno/blob/b12c78eb1/AGENTS.md#L83-L101) · [↗](../../../../../.worktrees/gno-review-5720/AGENTS.md#L83-L101)) requires an ADR for non-trivial AI-assisted PRs. The diff is small but the test commentary reads AI-thorough (`bench_ops_test.go` reference annotations, "shape-only placeholder; PopAsPointer reads from the stack, not the AST", explicit production-push-order comment in `op_assign_test.go:52-53`). If AI-assisted, add `gnovm/adr/pr5720_assign_lhs_order.md` with the bug shape and the reverse-pop / forward-assign split as the load-bearing decision. If it isn't, ignore.

## Questions for Author

- Is this AI-assisted? If so, an ADR is required per `AGENTS.md`.
- Was `BenchmarkOpAssign_*` rerun before/after? The commit message `perf(gnovm): fast-path single-LHS doOpAssign` implies measurement; the PR body doesn't include numbers. A one-line `benchstat` would justify the fast-path branch concretely.
