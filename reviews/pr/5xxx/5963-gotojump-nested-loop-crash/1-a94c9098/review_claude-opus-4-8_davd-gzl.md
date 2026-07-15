# PR [#5963](https://github.com/gnolang/gno/pull/5963): fix(gnovm): correct GotoJump stmt-stack truncation for goto out of nested loops

URL: https://github.com/gnolang/gno/pull/5963
Author: omarsy | Base: master | Files: 3 | +153 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: a94c9098 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5963 a94c9098`

**TL;DR:** A backward `goto` that jumps out of three or more nested `for`/`range` loops crashed the interpreter with `slice bounds out of range [:-1]`. This removes a duplicate stmt-stack truncation in `GotoJump` so those gotos run correctly.

**Verdict: APPROVE** — correct, minimal crash fix; only additive missing-test and comment-hardening suggestions remain.

## Summary
`Machine.GotoJump` reset the statement stack to the outermost popped frame's `NumStmts`, then subtracted the crossed-frame count a second time. `NumStmts` is captured before that frame pushes its own `bodyStmt`, so the first reset already drops every crossed loop's `bodyStmt`; the second subtraction is redundant. When the crossed-frame count exceeds the outermost frame's `NumStmts` (three loops directly under the function body: `NumStmts == 2`, `depthFrames == 3`), the index underflows and the slice expression panics. In shallower cases it over-truncated silently, because the `GOTO` handler in `op_exec.go` re-truncates `m.Stmts` to the target block's `bodyStmt.NumStmts` immediately after and re-grows the slice from retained capacity. The fix deletes the second truncation; the final stmt-stack length is unchanged on previously-working gotos.

## Fix
Remove `m.Stmts = m.Stmts[:len(m.Stmts)-depthFrames]` from `GotoJump`, leaving `m.Stmts = m.Stmts[:fr.NumStmts]` as the sole baseline reset, consistent with the neighboring `Ops`/`Values`/`Exprs`/`Blocks` resets at [`machine.go:2450-2454`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/machine.go#L2450-L2454) · [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/machine.go#L2450). The `GOTO` handler at [`op_exec.go:708-717`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/op_exec.go#L708-L717) · [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/op_exec.go#L708) authoritatively sets the final `m.Stmts` length afterward, and it is `GotoJump`'s only caller.

The `m.Blocks` second pop is kept and is correct: `findGotoLabel` resets `blockDepth` to 0 on every frame crossing at [`preprocess.go:4685`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/preprocess.go#L4685) · [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/preprocess.go#L4685), so `BlockDepth` counts only scopes within the target frame, which the baseline `fr.NumBlocks` reset does not already remove. Statements have no equivalent second component.

## Examples
Byte-for-byte parity with `go run`, all crashing on master before the fix:

| Construct | master | PR / `go run` |
|-----------|--------|---------------|
| 3 nested `for`, backward goto to func body | panic `[:-1]` | `done 10` |
| 4 nested `for` | panic `[:-2]` | `done 17` |
| 3 nested `range` | panic `[:-1]` | `done 10` |
| 3 loops + `{}` block scopes crossed | panic `[:-1]` | `done 10` |
| 2 loops inside a `switch` clause (frame depth 3) | panic `[:-1]` | `done 10` |
| 2 nested `for` (depth below threshold) | `done 6` | `done 6` |

## Glossary
- bodyStmt: the sticky loop/clause body statement each frame pushes onto `m.Stmts`; the stack slots this fix accounts for.
- block: a GnoVM scope frame; `m.Blocks` and `depthBlocks` track these, distinct from the stmt stack.
- Exception: the VM's Go-level panic wrapper; `runOnce` re-raises anything that is not an `*Exception`, so a bare Go panic escapes gno `recover()`.
- filetest: a `*.gno` file run by the VM and asserted against an `// Output:` golden.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[typo in a panic message]** `machine.go:2436`, `machine.go:2458` — both guards read `exeeds`, should be `exceeds`. Pre-existing, in the touched function. Unreachable from user gno code (`depthFrames`/`depthBlocks` are bounded by lexical nesting from `findGotoLabel`), so they correctly stay bare panics for an internal-invariant violation, not VM Exceptions.

## Missing Tests
- **[frame-type coverage gap]** [`gnovm/tests/files/goto10.gno`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/tests/files/goto10.gno) · [↗](../../../../../.worktrees/gno-review-5963/gnovm/tests/files/goto10.gno) — the added regression covers `ForStmt` frames only; the fix also repairs the crash across `RangeStmt` and `SwitchClauseStmt` frames, which have no golden asserting them.
  <details><summary>details</summary>

  `findGotoLabel` counts three frame types at [`preprocess.go:4676`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/preprocess.go#L4676) · [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/preprocess.go#L4676): `ForStmt`, `RangeStmt`, `SwitchClauseStmt`. Existing `goto*` filetests cross a `RangeStmt` or `SwitchClauseStmt` only at frame depth ≤ 1, which never underflows. Two sibling filetests close the gap; both panic `[:-1]` on master and pass here. Fix: add [`goto_nested_range.gno`](../tests/goto_nested_range.gno) (three nested `range` loops) and [`goto_switch_loops.gno`](../tests/goto_switch_loops.gno) (two loops inside a `switch` clause) to `gnovm/tests/files/`. See [repro](comment_claude-opus-4-8.md).
  </details>

## Suggestions
- **[guard against a symmetric re-introduction]** `machine.go:2444-2449` — the added comment explains why re-subtracting `depthFrames` is wrong but not why truncating to `fr.NumStmts` alone is sufficient.
  <details><summary>details</summary>

  The final `m.Stmts` length is owned by the `GOTO` handler's re-truncation at [`op_exec.go:715`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/op_exec.go#L715) · [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/op_exec.go#L715). The removed line mirrored `PopFrameAndReset` at [`machine.go:2464-2470`](https://github.com/gnolang/gno/blob/a94c9098/gnovm/pkg/gnolang/machine.go#L2464-L2470) · [↗](../../../../../.worktrees/gno-review-5963/gnovm/pkg/gnolang/machine.go#L2464), which resets to `fr.NumStmts` then does one `PopStmt` (valid there: it pops exactly one frame). A maintainer "restoring symmetry" between the two is the path that reintroduces this bug. One sentence citing the `op_exec.go` re-truncation and contrasting the single-frame `PopFrameAndReset` would harden it. Fix: no behavioral change; comment only.
  </details>

## Verified
- Reverting the fix reproduces the crash: master is the pre-fix state and panics `slice bounds out of range [:-1]` on the 3-nested-loop `goto`; the PR prints `done 10`.
- Output matches `go run` byte-for-byte across `for`, `range`, and `switch`-clause frame crossings and additional block-scope crossings, for 3- to 6-deep nesting, forward and backward gotos; non-crashing cases (depth below threshold, label between loops, label inside a block) are unchanged on the PR.
- Two new frame-type regressions ([`goto_nested_range.gno`](../tests/goto_nested_range.gno), [`goto_switch_loops.gno`](../tests/goto_switch_loops.gno)) panic on master and pass here.
- `goto10.gno` and the `goto*` / `heap_alloc_gotoloop*` / `loopvar_goto*` filetest families pass at a94c9098. The one red case in the full `-run Files -test.short` suite (`types/or_f0.gno`) fails identically on master: a go/types 1.26 message-text drift on the `|` operator, unrelated to this diff.
