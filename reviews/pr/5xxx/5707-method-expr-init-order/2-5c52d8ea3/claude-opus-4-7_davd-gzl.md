# PR #5707: fix(gnovm): track method expression deps in init order analysis

URL: https://github.com/gnolang/gno/pull/5707
Author: ltzmaxwell | Base: master | Files: 3 | +117 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `5c52d8ea3` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5707 5c52d8ea3`

**Verdict: APPROVE** — both pointer- and value-receiver method expressions now register correctly; two defensive `panic("should not happen")` arms are the only remaining smell.

## Summary
Method expressions (unbound method values like `T.M` or `(*T).M`) take a `SelectorExpr.Path.Type` of `VPField` rather than `VPValMethod`/`VPPtrMethod`, so the init-order dependency walker added in #5649 ignored them. Vars whose initializers transitively depended on package state through a method-expression call body got initialized before that state and read zero values. This revision adds a `case VPField` arm gated on `ATTR_TYPEOF_VALUE == *TypeType`, with an `extractTypeValue` helper that unwraps `*StarExpr` to also cover `(*T).M`, plus an `*PointerType` branch for the resulting `TypeValue`. New file `var_initorder28.gno` pins the pointer-receiver case (the gap flagged in the previous review).

## Glossary
- `codaInitOrderDeps` — post-preprocess pass that scans every package-level decl and records its dep set on `ATTR_DECL_DEPS`.
- `VPField` — selector path kind shared by struct-field access AND unbound method expressions; the distinguisher is whether `n.X` carries a `*TypeType` vs a value type.
- Method expression — `T.M` or `(*T).M`, callable as `T.M(receiver, args...)`; distinct from a method value `t.M`.

## Fix
Before: the `TRANS_LEAVE *SelectorExpr` switch in [`preprocess.go:6005-6033`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L6005-L6033) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6005-L6033) handled only `VPValMethod`/`VPPtrMethod`/`VPDerefValMethod`/`VPDerefPtrMethod`. Method expressions, which compile to `VPField` via [`DeclaredType.GetUnboundPathForName`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/types.go#L2146-L2153) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/types.go#L2146-L2153), were silently skipped — `addDep` never fired for `T.Method2` in `var dummy2 = T.Method2(0)`.

After: a new `case VPField` arm at [`preprocess.go:6034-6068`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L6034-L6068) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6034-L6068) gates on `n.X.GetAttribute(ATTR_TYPEOF_VALUE).(*TypeType)` to distinguish method-expression selectors from regular struct-field selectors, then calls the new helper [`extractTypeValue`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L5907-L5919) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L5907-L5919) which strips a single `*StarExpr` layer before asserting `*ConstExpr` and reading `TypeValue`. A type switch on `tv.Type` accepts both `*DeclaredType` (value receiver, `T.M`) and `*PointerType` whose `Elt` is `*DeclaredType` (pointer receiver, `(*T).M`); same-package gate (`dt.PkgPath != pn.PkgPath`) mirrors the existing arm. Both bundled tests (`var_initorder27.gno`, `var_initorder28.gno`) pass; the runtime dispatch at [`values.go:1859-1884`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/values.go#L1859-L1884) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/values.go#L1859-L1884) accepts the same two type shapes, so the preprocess arm matches the runtime invariant.

## Critical (must fix)

None.

## Warnings (should fix)

- **[panic on theoretically-unreachable arms]** [`gnovm/pkg/gnolang/preprocess.go:6059-6063`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L6059-L6063) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6059-L6063) — two `panic("should not happen, …")` calls in init-order analysis preempt unrelated preprocess errors with a misleading stack.
  <details><summary>details</summary>

  The `*PointerType` case panics if `t.Elt` is not `*DeclaredType`; the `default` arm panics on any other `tv.Type`. Both protect an invariant — `ATTR_TYPEOF_VALUE == *TypeType && n.X is a *ConstExpr/TypeValue` is supposed to imply `tv.Type ∈ {*DeclaredType, *PointerType[*DeclaredType]}`. I tried reaching them with interface method expressions (`I.M(t)`) and embedded-promoted method expressions (`Outer.MethodFromInner(o)`); both fail earlier at [`preprocess.go:2635`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L2635) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L2635) with `unknown *DeclaredType method named ...` (pre-existing limitation on master), so the panics are unreachable today.

  Risk if a future preprocessor change loosens the upstream gate: an unrelated user var decl panics during init-order analysis with a message that points at a debug-only invariant. The matching runtime path at [`values.go:1879`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/values.go#L1879) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/values.go#L1879) panics too, but only when the method is actually invoked; the preprocess panic fires for any decl that mentions the expression, fan-out is worse. Fix: replace both `panic` calls with `break` to silently skip — missing one dep is cheaper than a preprocess-time crash on a structural change elsewhere.
  </details>

- **[closure-wrapped method expression uncovered]** [`gnovm/tests/files/var_initorder27.gno:18-20`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/tests/files/var_initorder27.gno#L18-L20) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/var_initorder27.gno#L18-L20) — the new tests only exercise top-level direct method expressions; nested-in-closure path also relies on the new arm but has no in-tree guard.
  <details><summary>details</summary>

  `var dummy = func() int { return T.M(0) }()` exercises the same VPField arm via the func-lit body walk in TranscribeB, and works correctly on this PR — I verified (see [tests/var_initorder_methodexpr_closure.gno](tests/var_initorder_methodexpr_closure.gno)). Without an in-tree regression test, a future change to the func-lit skip guard at [`preprocess.go:5963-5967`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L5963-L5967) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L5963-L5967) could silently regress this. Fix: extend `var_initorder27.gno` with a closure-wrapped call, or add a sibling `var_initorder29.gno`.
  </details>

## Nits

- [`gnovm/tests/files/var_initorder27.gno:7-9`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/tests/files/var_initorder27.gno#L7-L9) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/var_initorder27.gno#L7-L9) — header comment "Method calls are ignored when deciding initialization order" is verbatim from Go issue 3824 but reads as a current-state claim in a test that asserts the opposite. Trim to a one-line reference (`// Issue 3824: method calls and method expressions must be tracked in initialization order.`).
- [`gnovm/pkg/gnolang/preprocess.go:6056-6061`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L6056-L6061) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6056-L6061) — the inner `var ok bool` shadow inside `case *PointerType` is only needed because `dt` is set in the outer scope. Collapsing to a single `dt, ok := t.Elt.(*DeclaredType)` declaration plus an outer `var dt *DeclaredType` would scan cleaner; current form reads as a Go quiz.

## Missing Tests

- **[closure-wrapped method expression]** [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/) — see Warning. Working copy at [tests/var_initorder_methodexpr_closure.gno](tests/var_initorder_methodexpr_closure.gno).
- **[type alias to declared type]** [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/) — `type U = T; var d = U.M(0)` passes (alias resolves to the underlying declared type before the gate), but it's not in-tree. Working copy at [tests/var_initorder_methodexpr_alias.gno](tests/var_initorder_methodexpr_alias.gno). Cheap to add as a guard against alias-resolution drift.

## Suggestions

- [`gnovm/pkg/gnolang/preprocess.go:6005-6068`](https://github.com/gnolang/gno/blob/5c52d8ea3/gnovm/pkg/gnolang/preprocess.go#L6005-L6068) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6005-L6068) — receiver-type extraction is now duplicated between the `VPValMethod` arm and the new `VPField` arm: both unwrap a pointer, assert `*DeclaredType`, check `dt.PkgPath != pn.PkgPath`, then `addDep(dt.Name + "." + n.Sel)`. A helper `recordMethodDep(addDep, dt, pn, sel Name)` would collapse the four-line tail. Left as a follow-up — diff size minimal as written.

## Questions for Author

- The two CI failures (`gno-checks/lint` re `r/test/sealviolation`, `main/test` re `params_valset_rotation_throttle`) appear unrelated to this change — same files / same job names as on the previous revision, untouched here. Worth a rerun before merge to confirm flake vs. real bisect candidate.
