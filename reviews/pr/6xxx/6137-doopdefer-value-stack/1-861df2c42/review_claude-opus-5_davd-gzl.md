# PR [#6137](https://github.com/gnolang/gno/pull/6137): fix(gnovm): correct doOpDefer value-stack handling for defer f(g())

URL: https://github.com/gnolang/gno/pull/6137
Author: omarsy | Base: master | Files: 4 | +183 -1
Reviewed by: davd-gzl | Model: claude-opus-5, effort xhigh | Commit: 861df2c42 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6137 861df2c42`
Overview: [overview](../overview.md)

## Overview

Deferring a call whose one written argument is a multi-value call, the
`defer f(g())` spread form, read the wrong slot of the value stack.
[`doOpDefer`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L712)
counted the arguments with `len(ds.Call.Args)`, which is 1 for that form while
the stack holds one value per expanded result. It therefore picked up an
argument value where the function value sat. When that argument was an `int` the
VM raised `defer called a nil function`; when it was itself a `func` value the VM
ran it and never ran the deferred function. The fix reads
[`ds.Call.NumArgs`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L720)
instead, the expanded count the preprocessor already computes, and adds a
`PopValues` to the nil-func branch so it consumes its arguments like the two
branches above it. Two filetests come with it.

**Verdict: APPROVE** — the fix is right, matches Go on every shape tried, and
leaves no Warning; the added coverage is thinner than the bug it closes and the
`default:` branch keeps the imbalance the change removes elsewhere (1 missing
test, 1 suggestion, 1 nit).

## Verify first

- [`gnovm/pkg/gnolang/op_call.go:722`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L722) · [↗](../../../../../.worktrees/gno-review-6137/gnovm/pkg/gnolang/op_call.go#L722). `PeekValue(numArgs + 1)` reaches the function only if `NumArgs` is set on every deferrable call expression. Run `go test ./gnovm/pkg/gnolang/ -run 'TestFiles/defer'` and confirm 0 failures across the 35 defer filetests.
- [`gnovm/pkg/gnolang/op_call.go:750`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L750) · [↗](../../../../../.worktrees/gno-review-6137/gnovm/pkg/gnolang/op_call.go#L750). The new `PopValues(numArgs)` runs before the `m.PopValue()` at the bottom of the function, so the nil branch now pops `numArgs + 1` values in total. Confirm the func value is the last one off by reading the two lines together.

## Summary

`CallExpr` carries two counts and they differ only for the spread form:
[`nodes.go:420`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/nodes.go#L420)
defines `NumArgs` as `len(Args) or len(Args[0].Results)`, and
[`countNumArgs`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/preprocess.go#L6074-L6090)
returns the result count only when a single argument is a call that is not a
type conversion. The ordinary call path already reads the expanded count at
[`op_call.go:11`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L11),
so the defer path was the one place still using the syntactic count. The change
makes the two agree.

## Fix

`numArgs` moves from `len(ds.Call.Args)` to `ds.Call.NumArgs`, which feeds both
the `PeekValue` offset that locates the function value and the `popCopyArgs`
count that drains its arguments. The nil-func branch at
[`op_call.go:745-751`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L745-L751)
gains `m.PopValues(numArgs)`, since it pushed the deferred entry and left the
evaluated arguments behind. The load-bearing constraint is that `NumArgs` equals
`len(Args)` for every non-spread call, so no ordinary defer changes behaviour.

## Benchmarks / Numbers

Each shape run as a filetest at the merge base `bc35e978a` with the fix
reverted, then at `861df2c42`. The Go column is `go run` on the same source
under go1.25.9.

| Deferred call | Go | base | head |
|---|---|---|---|
| `h(twovals())`, plain func | `h: 15` | panic, `defer called a nil function` | `h: 15` |
| `f(g())`, `g` returning `(func(), int)` | `f: n = 99` | `WRONG: marker ran` | `f: n = 99` |
| `variadic(two())` | `variadic: 7 1 8` | panic, `defer called a nil function` | `variadic: 7 1 8` |
| `counter{n: 1}.add(two())` | `method: 16` | panic, `defer called a nil function` | `method: 16` |
| `println(two())` | `7 8` | panic, `defer called a nil function` | `7 8` |
| `nf(1, 2)`, `nf` nil | call-of-nil | call-of-nil | call-of-nil |
| `nf(two())`, `nf` nil | call-of-nil | call-of-nil | call-of-nil |

## Missing Tests

- **[test coverage]** `gnovm/tests/files/defer_multivalue_arg.gno:18` — three spread shapes broke identically at the base and none of them is covered.
  <details><summary>details</summary>

  The added filetest exercises a plain function callee twice. A variadic callee,
  a bound method and a builtin reach the same `PeekValue(numArgs + 1)` line
  through different `switch` branches, and each panicked with `defer called a nil
  function` at the merge base. The next change to touch `doOpDefer` gets no
  signal from any of them. Fix: add
  [`tests/defer_multivalue_arg_shapes.gno`](tests/defer_multivalue_arg_shapes.gno),
  which covers all three in one file and fails at `bc35e978a`.
  </details>

## Nits

- **[test claim]** `gnovm/tests/files/defer_nil_func_args.gno:3-8` — the header says the test asserts the value stack stays balanced, and the test passes with the fix reverted.
  <details><summary>details</summary>

  Reverting
  [`op_call.go`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/op_call.go#L745-L751)
  to the base and re-running leaves this filetest green, so it pins claim (a),
  call-of-nil at the deferred call, and nothing of claim (b). Nothing else can
  pin (b) through this harness:
  [`PopFrameAndReturn`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/gnolang/machine.go#L2694-L2702)
  truncates the stack to the frame's recorded height on return, and
  [`CheckEmpty`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/pkg/test/filetest.go#L159-L161)
  runs only for a filetest that produces neither output nor error, which this
  one cannot be. Fix: drop clause (b) from the header, or say there that the
  balance is unobservable from a filetest.
  </details>

## Suggestions

- **[state safety]** `gnovm/pkg/gnolang/op_call.go:752-753` — `default:` returns before the trailing `m.PopValue()`, so it leaves the function value and its arguments on the stack.
  <details><summary>details</summary>

  The change gives `case nil` the same stack discipline as `*FuncValue` and
  `*BoundMethodValue`. `default:` returns before the trailing `m.PopValue()`, so
  it leaves `numArgs + 1` values behind. The only shape found to reach it is
  `defer int64(x)` with `x` an `int`, which the go typechecker rejects with
  `defer requires function call, not conversion`, so the branch runs under the
  filetest harness and not on source a chain would accept. That is why this is a
  suggestion and not a defect. `m.PopValues(numArgs + 1)` before the `return`
  keeps the filetest's `invalid defer function call: typeval{int64}` and its
  typecheck error unchanged, and the 35 defer filetests stay green.
  </details>

## Verified

- Go parity on both cases the ADR claims: `go run` on the same source under
  go1.25.9 prints `body`, `h: 15`, `f: n = 99`, matching the filetest's
  `// Output:` block exactly.
- Every added filetest was re-run with `op_call.go` reverted to `bc35e978a`.
  [`defer_multivalue_arg.gno`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/tests/files/defer_multivalue_arg.gno)
  fails there;
  [`defer_nil_func_args.gno`](https://github.com/gnolang/gno/blob/861df2c42/gnovm/tests/files/defer_nil_func_args.gno)
  passes, which is the nit above.
- The head is a merge of master into the branch. `git show 861df2c42 --cc`
  prints no hunk, so it carries no conflict resolution.
- The 35 filetests matching `TestFiles/defer` are green at the reviewed sha.
