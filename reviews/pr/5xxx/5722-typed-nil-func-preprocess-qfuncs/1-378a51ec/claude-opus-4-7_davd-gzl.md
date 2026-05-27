# PR #5722: fix(gnovm): handle typed-nil func value in preprocess and vm/qfuncs

URL: https://github.com/gnolang/gno/pull/5722
Author: ltzmaxwell | Base: master | Files: 5 | +48 -8
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]
Local worktree: `git -C gno worktree add .worktrees/gno-review-5722 378a51ec` (then `gh -R gnolang/gno pr checkout 5722` inside it)

Verdict: APPROVE — minimal, surgical fix for two related typed-nil-func crashes; both preprocess call sites and the QueryFuncs site are correctly guarded, the new test (`func31.gno`) is a verbatim port of Go's `issue8047.go`, the regression test for QueryFuncs reproduces the live bug, and the CI failures are unrelated to this diff.

## Summary

A package-level typed-nil func variable (e.g. `var Hook func()`) crashed the VM in two places. (1) In preprocess, calling `((func())(nil))()` reached the `*FuncType` LEAVE branch at [`preprocess.go:1759-1762`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/preprocess.go#L1759-L1762) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/preprocess.go#L1759-L1762) and the generic-specify block at [`preprocess.go:2206-2212`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/preprocess.go#L2206-L2212) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/preprocess.go#L2206-L2212), both of which assumed `(*ConstExpr).V` is a non-nil `*FuncValue` — a typed-nil `ConstExpr` (V==nil but T==FuncType) blew up the type assertion. (2) In `QueryFuncs`, iterating a realm's package block at [`keeper.go:1234-1245`](https://github.com/gnolang/gno/blob/378a51ec/gno.land/pkg/sdk/vm/keeper.go#L1234-L1245) · [↗](../../../../../.worktrees/gno-review-5722/gno.land/pkg/sdk/vm/keeper.go#L1234-L1245) gated on `tv.T.Kind() == FuncKind` but not on `tv.V != nil`, so any deployed realm declaring a public typed-nil func crashed `vm/qfuncs` (RPC-reachable). The fix demotes `(*TypedValue).GetFunc()` from "panic on nil V" to "return nil on nil V" at [`values.go:2104-2110`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/values.go#L2104-L2110) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/values.go#L2104-L2110) and adds matching nil guards at the three call sites.

`var Hook func()` is not theoretical — `examples/gno.land/r/tests/vm/crossrealm/crossrealm.gno:118` already ships `var Closure func()` (and two more at L128/L141), so any chain RPC issuing `vm/qfuncs gno.land/r/tests/vm/crossrealm` panicked pre-fix.

## Glossary

- `ConstExpr` — preprocess-resolved literal/constant expression; for func references holds the bound `*FuncValue` in `.V`.
- `FuncValue` — concrete callable; `nil` of type `func(...)` is `(T=*FuncType, V=nil)`, not `(T=nil, V=nil)`.
- `QueryFuncs` / `vm/qfuncs` — ABCI query returning exported function signatures of a realm.
- `uversePkgPath` — the predeclared/universe pseudo-package (`append`, `copy`, `make`, etc.).

## Fix

`(*TypedValue).GetFunc()` previously did `tv.V.(*FuncValue)` (panic on nil V). It now returns nil instead. The two preprocess call sites were rewritten to the same nil-tolerant shape already used at [`nodes.go:439`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/nodes.go#L439) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/nodes.go#L439) and [`preprocess.go:2007-2013`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/preprocess.go#L2007-L2013) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/preprocess.go#L2007-L2013) — extract `fv` with `if cx, ok := ... ; ok` plus a `fv != nil` guard before reading `PkgPath`/`Name`. `QueryFuncs` adds `if fv == nil { continue }` between the existing `Kind() != FuncKind` and `IsMethod` checks. The new regression test plants `var Hook func()` in `TestVmHandlerQuery_Funcs`'s realm fixture so the JSON expectation (which excludes `Hook`) locks the skip semantics in. `gnovm/tests/files/func31.gno` exercises the defer path.

## Critical (must fix)

None.

## Warnings (should fix)

- [public API contract change] [`values.go:2104-2110`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/values.go#L2104-L2110) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/values.go#L2104-L2110) — `GetFunc()` silently changes from panic-on-nil to return-nil; out-of-tree callers that deref the result now SIGSEGV instead of getting a clear `interface conversion` panic.
  <details><summary>details</summary>

  In-tree callers are fine — I audited all seven: the two `values.go` method-table reads ([L1908](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/values.go#L1908) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/values.go#L1908), [L1942](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/values.go#L1942) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/values.go#L1942)), [`nodes.go:1535`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/nodes.go#L1535) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/nodes.go#L1535), and [`op_types.go:385`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/op_types.go#L385) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/op_types.go#L385) all read from `DeclaredType.Methods[i]` which stores concrete `*FuncValue`s, never typed-nil. [`test.go:520`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/test/test.go#L520) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/test/test.go#L520) evaluates a discovered test func name. The pre-existing guarded sites at [`nodes.go:439`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/nodes.go#L439) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/nodes.go#L439) and [`preprocess.go:2009`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/preprocess.go#L2009) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/preprocess.go#L2009) already check `!= nil`. So the in-tree blast radius is zero. The concern is the implicit contract change for any third-party Go embedder of `gnovm/pkg/gnolang` — `*TypedValue` is exported and `GetFunc` is exported. Fix: either keep the new behavior and add a one-line godoc to `GetFunc` explicitly stating "returns nil if `tv.V` is not a `*FuncValue` (e.g. typed-nil func variable)", or rename and add a parallel `MustGetFunc` to preserve the old contract — godoc is the lighter touch and matches the in-tree usage pattern.
  </details>

- [latent crash on same shape] [`preprocess.go:2068`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/preprocess.go#L2068) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/preprocess.go#L2068) — `GetUnboundFunc()` still panics on typed-nil V; not reachable from a user-defined typed-nil through this PR's repro shape, but the symmetry is incomplete and one refactor away from biting.
  <details><summary>details</summary>

  The cur-call check at [`preprocess.go:2062-2072`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/preprocess.go#L2062-L2072) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/preprocess.go#L2062-L2072) runs `ftv, err := tryEvalStatic(...)` on `n.Func` for a `f(cur, ...)` call, then dereferences `ftv.GetUnboundFunc().PkgPath`. `GetUnboundFunc` at [`values.go:2112-2121`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/values.go#L2112-L2121) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/values.go#L2112-L2121) panics on nil V via the default arm. A typed-nil `var F func(cur realm)` followed by `F(cur)` is currently unreachable through ordinary source because `cur` only resolves inside crossing functions, but the asymmetry is fragile — if `cur`-resolution ever broadens, this becomes a second crash class. Fix: either mirror the GetFunc change (return nil for nil V, leave the bound-method case as-is) and add a nil guard at the call site, or leave `GetUnboundFunc` as-is and add a comment at the call site documenting why ftv.V is guaranteed non-nil here.
  </details>

## Nits

- [`gnovm/tests/files/func31.gno:25`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/tests/files/func31.gno#L25) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/tests/files/func31.gno#L25) — missing trailing newline; companion file `func30.gno` ends with one.
- [`gnovm/tests/files/func31.gno:6-7`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/tests/files/func31.gno#L6-L7) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/tests/files/func31.gno#L6-L7) — no blank line between the file comment and `package main`, and none between `package main` and the first `func`; sibling `func30.gno` uses both.
- [`gnovm/pkg/gnolang/values.go:2104-2110`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/pkg/gnolang/values.go#L2104-L2110) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/pkg/gnolang/values.go#L2104-L2110) — the body
  ```go
  fv, ok := tv.V.(*FuncValue)
  if !ok {
      return nil
  }
  return fv
  ```
  collapses to `fv, _ := tv.V.(*FuncValue); return fv` (failed type assertion on a single return returns the zero value), but the explicit form is fine and arguably more readable.

## Missing Tests

- [defer/preprocess separation] [`gnovm/tests/files/func31.gno`](https://github.com/gnolang/gno/blob/378a51ec/gnovm/tests/files/func31.gno) · [↗](../../../../../.worktrees/gno-review-5722/gnovm/tests/files/func31.gno) — exercises only the defer path; a direct call `((func())(nil))()` (no defer) hits the same preprocess fix but has no filetest.
  <details><summary>details</summary>

  The PR body claims both crashes are fixed by the preprocess guard, and the defer wrapper happens to test the same preprocess path as a bare call would. A 5-line companion filetest with `((func())(nil))()` at the top level (no defer) would lock in the non-defer surface explicitly and matches what the PR title advertises ("typed-nil func value in preprocess"). Low value, but cheap.
  </details>

## Suggestions

- [`gno.land/pkg/sdk/vm/handler_test.go:146`](https://github.com/gnolang/gno/blob/378a51ec/gno.land/pkg/sdk/vm/handler_test.go#L146) · [↗](../../../../../.worktrees/gno-review-5722/gno.land/pkg/sdk/vm/handler_test.go#L146) — the comment `// typed-nil func: QueryFuncs must skip rather than crash` is helpful; consider a one-line equivalent in `keeper.go:1240` so a future reader of just the keeper doesn't have to chase the regression test to learn what the guard is for.
  <details><summary>details</summary>

  The current keeper comment "typed-nil func variable, no signature to expose" describes *what* the code does. Adding "(regression: PR #5722 — crashed `vm/qfuncs` for any realm with `var X func(...)`)" makes the *why* discoverable without git-blame.
  </details>

## Questions for Author

None — the diff matches the description, the regression test reproduces the live bug, and the call-site audit checks out.
