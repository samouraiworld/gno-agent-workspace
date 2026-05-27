# PR #5689: fix(gnolang): allow indirect cur-call through a local func variable

URL: https://github.com/gnolang/gno/pull/5689
Author: omarsy | Base: master | Files: 4 | +127 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `f28f535` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5689 f28f535`

**Verdict: APPROVE** — minimal, well-targeted fix that defers an inconclusive preprocess-time check to the authoritative runtime check; new filetests cover both paths; adversarial tests confirm the fix also unblocks func-parameter indirection without weakening cross-realm rejection.

## Summary

The preprocessor's best-effort same-realm check on `f(cur, ...)` was rejecting any indirection through a local variable (e.g. `p := myHandler; p(cur)`) with a misleading "expected function or bound method but got <nil>" panic. Root cause: [`tryEvalStatic`](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/preprocess.go#L4185-L4212) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/preprocess.go#L4185-L4212) for a local `NameExpr` returns a `TypedValue` with the function type bound but `V == nil` (the throwaway machine has type info but no runtime values), and the existing else branch calls `GetUnboundFunc()` on that nil value. Fix inserts an `else if ftv.V == nil` arm that skips the static check and lets the runtime path in [`PushFrameCall`](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/machine.go#L2202-L2221) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/machine.go#L2202-L2221) enforce same-realm — same error message, same observable behavior for misuse, only moved compile→runtime.

## Glossary

- `tryEvalStatic` — best-effort evaluation of an expression on a throwaway machine; catches panics, returns `(TypedValue, error)`.
- `ftv.V == nil` — the resolved `TypedValue` has a type but no underlying value (typical for locals that the throwaway machine can't bind).
- `PushFrameCall` — runtime call-frame setup; performs the authoritative `m.Realm != pv.Realm` check for crossing functions.

## Fix

Before: any `p(cur, …)` where `p` is a local function variable panicked at preprocess time with a confusing "got <nil>" diagnostic, even when the underlying function is same-realm. The static check's `else` branch [unconditionally invoked `GetUnboundFunc()`](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/preprocess.go#L1994) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/preprocess.go#L1994) on an `ftv` that — for local-variable resolution — has `T` set but `V == nil`. After: a new arm at [preprocess.go:1984-1992](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/preprocess.go#L1984-L1992) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/preprocess.go#L1984-L1992) catches the `V == nil` case and falls through; cross-realm misuse is now rejected at runtime by [machine.go:2202-2221](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/machine.go#L2202-L2221) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/machine.go#L2202-L2221) with the same error text. The load-bearing constraint — same-realm crossing — is enforced authoritatively by the runtime path, which existing cross-realm filetests already exercise.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/preprocess.go:1989-1991`](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/preprocess.go#L1989-L1991) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/preprocess.go#L1989-L1991) — comment references "machine.go (~line where IsCrossing is checked in PushFrameCall)"; that line will drift. Replace with the stable symbol pair, e.g. `// Runtime check: see Machine.PushFrameCall — the IsCrossing branch checking m.Realm != pv.Realm.`
- [`gnovm/tests/files/zrealm_curcall_indirect_external.gno:5-7`](https://github.com/gnolang/gno/blob/f28f535/gnovm/tests/files/zrealm_curcall_indirect_external.gno#L5-L7) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_external.gno#L5-L7) — comment says "Companion to z_indirect_cur_call_local", but the actual sibling file is `zrealm_curcall_indirect_local.gno`. Update the name.
- [`gnovm/tests/files/zrealm_curcall_indirect_external.gno:12`](https://github.com/gnolang/gno/blob/f28f535/gnovm/tests/files/zrealm_curcall_indirect_external.gno#L12) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_external.gno#L12) — column-aligned trailing comment (`p(cur)             //`) is stylistically inconsistent with the rest of the diff. Drop the alignment.
- [`gnovm/adr/pr5689_preprocess_indirect_cur_call.md`](https://github.com/gnolang/gno/blob/f28f535/gnovm/adr/pr5689_preprocess_indirect_cur_call.md) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/adr/pr5689_preprocess_indirect_cur_call.md) — confirm whether the project wants per-fix ADRs for ~5-line bugfixes; this is documentation overhead for a small change. If the team has been moving toward ADRs for all gnolang semantic changes, fine; otherwise consider folding the rationale into the PR description.

## Missing Tests

- **[fix is broader than advertised]** [`gnovm/tests/files/zrealm_curcall_indirect_local.gno`](https://github.com/gnolang/gno/blob/f28f535/gnovm/tests/files/zrealm_curcall_indirect_local.gno) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_local.gno) — same fix also unblocks indirection through a function parameter, which is currently broken on master.
  <details><summary>details</summary>

  Confirmed empirically: the test below panics on master with the same "expected function or bound method but got <nil>" diagnostic and passes once this PR is applied. Worth adding as a regression test, since the fix narrative only mentions local-variable indirection but the actual blast radius covers any `NameExpr` whose static resolution yields `T != nil, V == nil` — locals, params, and likely closure captures.

  ```gno
  // PKGPATH: gno.land/r/test/indirectcurparam
  package indirectcurparam

  func myHandler(cur realm) { println("called") }

  func wrap(cur realm, f func(realm)) { f(cur) }

  func main(cur realm) {
      wrap(cur, myHandler)
      println("done")
  }
  // Output:
  // called
  // done
  ```

  Reproducer saved at [`reviews/pr/5xxx/5689-indirect-cur-call/1-f28f535/tests/zrealm_curcall_indirect_param.gno`](tests/zrealm_curcall_indirect_param.gno).
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/preprocess.go:1984-1992`](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/preprocess.go#L1984-L1992) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/preprocess.go#L1984-L1992) — 1 line of code with a 7-line comment block is top-heavy.
  <details><summary>details</summary>

  Trim to ~2 lines:

  ```go
  } else if ftv.V == nil {
      // Local/param/field: only the type was bound; defer the
      // same-realm check to Machine.PushFrameCall.
  }
  ```

  The full context already lives in `pr5689_preprocess_indirect_cur_call.md`.
  </details>

- [`gnovm/pkg/gnolang/preprocess.go:4185-4212`](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/preprocess.go#L4185-L4212) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/preprocess.go#L4185-L4212) — out of scope, but worth flagging: `tryEvalStatic`'s `recover()` defer unconditionally sets `err = fmt.Errorf("recovered panic with: %v", nil)` on the successful (non-panic) path, because `r := recover()` returns nil and the else-branch fires.
  <details><summary>details</summary>

  Effect: `err == nil` is only ever true for the early `*ConstExpr` return; the comment at [preprocess.go:1981](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/preprocess.go#L1981) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/preprocess.go#L1981) ("This is fine; e.g. somefunc()(cur,...)") is therefore misleading — `somefunc()(cur,...)` is a `CallExpr`, would hit the defer, and `err` would be set. Not a defect introduced by this PR, but the existing comments around `tryEvalStatic` consumers (here and at [transpile_gno0p9.go:156](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/transpile_gno0p9.go#L156) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/transpile_gno0p9.go#L156), [transpile_gno0p9.go:529](https://github.com/gnolang/gno/blob/f28f535/gnovm/pkg/gnolang/transpile_gno0p9.go#L529) · [↗](../../../../../.worktrees/gno-review-5689/gnovm/pkg/gnolang/transpile_gno0p9.go#L529)) all assume two-valued semantics that don't match the implementation. Consider a follow-up to either fix the always-non-nil err, or document the actual behavior at the helper.
  </details>

## Questions for Author

- Was the function-parameter case (per the missing-test note above) intentionally left out, or just not exercised? If intentional, what's the rationale for narrowing the regression coverage to locals?
- Worth pulling the `ftv.V == nil` arm out of the `Name("cur")` branch — should the same defensive arm cover other `tryEvalStatic` consumers that follow the same `IsUndefined → else → GetUnboundFunc` pattern? (Looking at the two `transpile_gno0p9.go` call sites in particular.)
