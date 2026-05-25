# PR #5202: fix(gnovm/debugger): add bounds checks to prevent index panics

URL: https://github.com/gnolang/gno/pull/5202
Author: davd-gzl | Base: master | Files: 2 | +107 -2
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — three small defensive bounds checks in `debugger.go` that turn three plausible negative-index panics into safe early returns; behavior outside the panic case is unchanged. Note: reviewer is also the author, so this is a self-review for the workflow's sake; two external approvals already exist ([mvallenet](https://github.com/gnolang/gno/pull/5202#pullrequestreview-3860974634), [notJoon](https://github.com/gnolang/gno/pull/5202#pullrequestreview-4046068478)).

## Summary

The interactive Gno debugger keeps its own `m.Debugger.call` stack tracking `OpCall`/`OpReturn` pairs to render `stack`/`up`/`down`/`list` frames. Three call sites assumed invariants the debugger cannot enforce when it attaches mid-run or when frame counts diverge from the call stack: `m.Ops[len-1]` on an empty op stack, popping `m.Debugger.call` on an empty call stack, and `debugFrameLoc(m, n)` indexing `call[len-n]` when `n > len(call)`. Each of the three is a plain slice underflow that panics inside the VM step callback — non-deterministic from user code (debugger is opt-in via `EnableDebug()`), but a crash for anyone using `gno run -debug` against a contrived program. The fix is three guards; semantics are unchanged outside the panic case.

## Glossary

- `m.Ops` — the VM operation stack; `Debug()` runs before each `PopOp()` so the top of the stack is the *next* op to execute.
- `m.Debugger.call` — debugger-side call frame stack, pushed on `OpCall`, popped on `OpReturn`/`OpReturnFromBlock`. Independent of `m.Frames`.
- `debugFrameLoc(m, n)` — returns the source location of the n-th outer debugger frame; `n == 0` means current loc.
- `debugFrameFunc(m, n)` — returns the n-th outer `FuncValue` by walking `m.Frames` backwards (different counter than `m.Debugger.call`).

## Fix

Three guards in [`gnovm/pkg/gnolang/debugger.go`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go):

1. [`debugger.go:197-199`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go#L197-L199) — early return when `m.Ops` is empty before reading `m.Ops[len-1]`. In normal execution `m.Ops` is non-empty at `Debug()` entry because `runOnce` always has at least an unpopped op until `OpHalt` short-circuits ([`machine.go:1510-1522`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/machine.go#L1510-L1522)); the guard covers the edge cases where a test or future caller drives `Debug()` directly (the new `TestDebugEmptyOps` does exactly this).
2. [`debugger.go:204-207`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go#L204-L207) — guard the `m.Debugger.call[:len-1]` pop on `OpReturn`/`OpReturnFromBlock` when the call stack is empty. As the author notes in a [PR comment](https://github.com/gnolang/gno/pull/5202#discussion_r2859050212), this fires when the debugger attaches mid-execution: the returns of pre-attach calls were never paired with pushes.
3. [`debugger.go:885-890`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go#L885-L890) — replace `len(m.Debugger.call) == 0` with `n > len(m.Debugger.call)` so any `n` beyond the call-stack depth falls back to `m.Debugger.loc` instead of computing a negative index. As the author notes in [another comment](https://github.com/gnolang/gno/pull/5202#discussion_r2859135919), the caller `debugStack` iterates `i = 0, 1, …` driven by `debugFrameFunc(m, i)`, which counts non-nil `f.Func` frames in `m.Frames` — a different counter than `m.Debugger.call`, so `n > len(call)` is reachable in practice when frame counts diverge.

The added internal test file [`debugger_internal_test.go`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger_internal_test.go) exercises each guard with a minimal `Machine` shell.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`debugger.go:885-890`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go#L885-L890) — `debugStack` will now print the current `m.Debugger.loc` for every frame past the bottom of the call stack instead of stopping or marking them unknown. Since the loop terminates on `debugFrameFunc(m, i) == nil` (not on `loc`), the user only sees the duplicate locations when there are more `m.Frames`-counted frames than `m.Debugger.call` entries — same fallback shape as before, just no longer crashing. Worth a follow-up if the maintainer wants a more accurate stack display, but not blocking.

## Missing Tests

- [`debugger_internal_test.go:36`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger_internal_test.go#L36) — the three `Debug()` tests reach the guarded code by setting `m.Debugger.enabled = false` so the inner loop short-circuits via `break loop` at [`debugger.go:151-152`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go#L151-L152) and falls into the post-loop block. Verified by reverting each guard locally — every test fails with `index out of range [-1]`, so the coverage is real. A one-line comment in each test explaining the path (`enabled = false` is how you reach the post-loop guard) would help the next reader, since it reads as counter-intuitive.

- No end-to-end test exercising the "debugger attached mid-execution" scenario that motivates guard #2. The existing `debugger_test.go` runs scripted REPL sessions against a sample program from the start of execution, so the call stack is always balanced. An adversarial scenario (e.g. attaching after some calls have already returned) would catch the case where the underlying invariant assumption changes. Not blocking — the unit test covers the immediate panic — but the integration-level gap is worth flagging.

## Suggestions

- [`debugger.go:200`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go#L200) — the variable `op := m.Ops[len(m.Ops)-1]` reads the *next* op (not the just-executed one), because `Debug()` runs before `PopOp()` in [`machine.go:1510-1514`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/machine.go#L1510-L1514). The inline comment above (`// Keep track of exact locations when performing calls.`) is correct but easy to misread. A one-line clarification — e.g. `// op is the next op to execute (Debug runs before PopOp)` — would save the next reader a confused minute. Not part of this PR's scope; just flag it.

- [`debugger.go:57`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debugger.go#L57) — the existing comment on `call` says "ideally should be provided by machine frame". This PR doesn't change that, but the off-by-one between `m.Frames` and `m.Debugger.call` (the motivation for guard #3) is exactly what that TODO is pointing at. Worth filing an issue to track unifying the two stacks; the new guard is the right short-term fix.

## Questions for Author

- Is there a known reproducer for guard #2 (debugger attaches mid-execution) in any of the gnodev / gnobro tooling, or was this found by inspection? An adversarial txtar test under `gno.land/pkg/integration/testdata/` would prevent regression if that path is exercised by tooling.

## Compatibility & Risk

- Pure additions: three guards plus a new internal test file. No public API change; no production code path semantics change outside the panic case.
- `debugger.go` is build-only relevant when `EnableDebug()` is called (see [`debug.go:172`](../../../../../.worktrees/gno-review-5202/gnovm/pkg/gnolang/debug.go#L172)); production VM execution is not affected.
- Two external approvals on record. CI green ([gh pr checks](https://github.com/gnolang/gno/pull/5202)).
