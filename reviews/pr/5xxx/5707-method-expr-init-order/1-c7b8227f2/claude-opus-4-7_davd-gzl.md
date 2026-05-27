# PR #5707: fix(gnovm): track method expression deps in init order analysis

URL: https://github.com/gnolang/gno/pull/5707
Author: ltzmaxwell | Base: master | Files: 2 | +66 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]
Local worktree: `git -C gno worktree add .worktrees/gno-review-5707 c7b8227f2` (then `gh -R gnolang/gno pr checkout 5707` inside it)

**Verdict: REQUEST CHANGES** — fix is correct for the canonical `T.M` form but misses the pointer-receiver form `(*T).M`, which still produces incorrect init order.

## Summary
Method expressions (unbound method values like `T.Method` or `(*T).Method`) take a `SelectorExpr.Path.Type` of `VPField` rather than `VPValMethod`/`VPPtrMethod`, so the init-order dependency walker added in PR #5649 ignored them. Vars whose initializers transitively depend on package-level state through a method-expression call body got initialized before that state, yielding zero values. The fix adds a `VPField` arm that recognises a type-valued `n.X` and records `T.Method` as a dep, mirroring the existing `VPValMethod` arm.

## Glossary
- `codaInitOrderDeps` — post-preprocess pass that scans every package-level decl and records its dep set on `ATTR_DECL_DEPS`.
- `VPField` — selector path kind shared by struct-field access AND unbound method expressions; the distinguisher is whether `n.X` is a type vs a value.
- Method expression — `T.M` or `(*T).M`, callable as `T.M(receiver, args...)`; distinct from a method value `t.M`.

## Fix
Before: the `TRANS_LEAVE *SelectorExpr` switch in [`preprocess.go:5991-6019`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/preprocess.go#L5991-L6019) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L5991-L6019) handled only `VPValMethod`/`VPPtrMethod`/`VPDerefValMethod`/`VPDerefPtrMethod`. Method expressions, which compile to `VPField` via [`DeclaredType.GetUnboundPathForName`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/types.go#L2146-L2153) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/types.go#L2146-L2153), were silently skipped — `addDep` never fired for `T.Method2` in `var dummy2 = T.Method2(0)`.

After: a new `case VPField` arm at [`preprocess.go:6020-6045`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/preprocess.go#L6020-L6045) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6020-L6045) gates on `n.X.GetAttribute(ATTR_TYPEOF_VALUE).(*TypeType)` to distinguish method-expression selectors from regular struct-field selectors, then extracts `*DeclaredType` from `n.X.(*ConstExpr).V.(TypeValue)` and registers `T.Method` as a dep — same shape as the existing `VPValMethod` arm.

The load-bearing constraint is the `n.X.(*ConstExpr)` cast: it only matches when the type ref `T` has been replaced by a `*ConstExpr` carrying a `TypeValue`. That happens for bare `T.M` (NameExpr `T` is replaced inline), but NOT for `(*T).M` where `n.X` remains a `*StarExpr`. See Critical finding below.

## Critical (must fix)

- **[pointer-receiver method expressions still mis-order]** [`gnovm/pkg/gnolang/preprocess.go:6026-6029`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/preprocess.go#L6026-L6029) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6026-L6029) — `(*T).Method(&t)` does not register `T.Method` as a dep; same init-order bug remains.
  <details><summary>details</summary>

  **Shape:** `var dummy = (*T).Method(&x)` where `Method` reads package-level `c`. With this PR applied, `dummy` still initializes before `c`, so `dummy == 0` instead of the expected value.

  **Mechanism:** For `(*T).Method`, the outer `SelectorExpr.X` is a `*StarExpr` wrapping the type-name node, not the `*ConstExpr(TypeValue)` form the new arm requires. The existing SelectorExr handler at [`preprocess.go:2610-2635`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/preprocess.go#L2610-L2635) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L2610-L2635) handles both forms correctly because it calls `evalStaticType(store, last, n.X)` and switches on the result — covering both `*DeclaredType` and `*PointerType` wrapping. The new arm in this PR re-implements that detection but stops at `n.X.(*ConstExpr)`, missing the StarExpr form. `ATTR_TYPEOF_VALUE` on `n.X` is set to `*TypeType` for `(*T)` too (verified — the `isType` gate at line 6023 passes), so only the `*ConstExpr` assertion is the discriminator.

  **Repro:** fails on PR head with `panic: dummy != 7`.

  ```bash
  gh pr checkout 5707 -R gnolang/gno
  cat > gnovm/tests/files/var_initorder_methodexpr_ptrrcvr.gno <<'EOF'
  // run
  package main

  type T int
  func (r *T) Method() int { return c }

  var x T = 0
  var dummy = (*T).Method(&x)
  var c = identity(7)
  func identity(a int) int { return a }

  func main() {
      if dummy != 7 { panic("dummy != 7") }
      println(dummy)
  }

  // Output:
  // 7
  EOF
  go test -v -run 'TestFiles/var_initorder_methodexpr_ptrrcvr.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/var_initorder_methodexpr_ptrrcvr.gno
  ```

  **Fix:** unwrap a `*StarExpr` before the `*ConstExpr` assertion, or replace the AST-shape check with `evalStaticType(store, last, n.X)`. The cleaner approach mirrors line 2612:

  ```go
  case VPField:
      if _, isType := n.X.GetAttribute(ATTR_TYPEOF_VALUE).(*TypeType); !isType {
          break
      }
      xt, ok := n.X.GetAttribute(ATTR_GNO_TYPE).(Type) // or wherever the resolved type lives
      // ... or drill through StarExpr explicitly:
      cx, ok := n.X.(*ConstExpr)
      if !ok {
          if sx, ok2 := n.X.(*StarExpr); ok2 {
              cx, ok = sx.X.(*ConstExpr)
          }
      }
      if !ok { break }
      // rest unchanged
  ```

  Either approach should also add a `var_initorder28.gno` covering the pointer-receiver form so this doesn't regress.
  </details>

## Warnings (should fix)

- **[missing regression test for closure / func-lit path]** [`gnovm/tests/files/var_initorder27.gno:18-20`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/tests/files/var_initorder27.gno#L18-L20) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/var_initorder27.gno#L18-L20) — the bundled test only covers top-level direct method expressions; nested-in-closure is also affected by the fix but uncovered.
  <details><summary>details</summary>

  `var dummy = func() int { return T.M(0) }()` exercises the new VPField arm transitively through the func-literal walk. I confirmed it both fails on master and passes with the fix (see [tests/var_initorder_methodexpr_closure.gno](tests/var_initorder_methodexpr_closure.gno)). Without an in-tree regression test, a future change to func-lit preprocessing skip flags could regress this silently.

  Fix: add `var_initorder_methodexpr_closure.gno` (or fold into 27) as a permanent guard.
  </details>

## Nits

- [`gnovm/pkg/gnolang/preprocess.go:6022`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/preprocess.go#L6022) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6022) — comment says "n.X is a type reference" but the load-bearing condition is "n.X is a `*ConstExpr` carrying a `TypeValue`", which is narrower. Loose comment hid the pointer-receiver gap from the author. Tighten to: "n.X is a const type expression — does NOT cover `(*T).M` where n.X is a `*StarExpr`" (or fix the gap, then the comment can stay generic).
- [`gnovm/pkg/gnolang/preprocess.go:6020`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/preprocess.go#L6020) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6020) — three early-`break`s in a row read as filler. Collapsing into a single conditional chain (or, ideally, a helper) would make the gate readable. Optional.

## Missing Tests

- **[pointer-receiver method expression]** [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/) — see Critical.
- **[closure-wrapped method expression]** [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/) — see Warning.
- **[cross-package method expression negative case]** [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/tests/files/) — added [`tests/var_initorder_methodexpr_xpkg.gno`](tests/var_initorder_methodexpr_xpkg.gno) confirming `pkg.T.GetB(...)` does NOT panic the resolver. Currently passes (the `dt.PkgPath != pn.PkgPath` gate works), but the `var_initorder_xpkgmethod.gno` sibling only covers the VPValMethod path. Mirroring the test for VPField is cheap insurance.

## Suggestions

- [`gnovm/pkg/gnolang/preprocess.go:6020-6045`](https://github.com/gnolang/gno/blob/c7b8227f2/gnovm/pkg/gnolang/preprocess.go#L6020-L6045) · [↗](../../../../../.worktrees/gno-review-5707/gnovm/pkg/gnolang/preprocess.go#L6020-L6045) — consider factoring the receiver-type extraction shared between the `VPValMethod` arm (lines 5992-6019) and this new arm into a helper `declaredTypeFromSelector(n) *DeclaredType`. Both arms duplicate the "unwrap pointer, assert DeclaredType, check PkgPath" sequence. A helper would also make it natural to add the missing StarExpr branch in one place.
  <details><summary>details</summary>

  The two arms differ only in where the type comes from (ATTR_REF_ELEM_TYPE / ATTR_TYPEOF_VALUE for the value-method case; `n.X.(*ConstExpr).V.(TypeValue)` for the type-method case). After a helper extracts `dt`, the rest — `if dt == nil || dt.PkgPath != pn.PkgPath { break }; addDep(...)` — is identical and can collapse.
  </details>

## Questions for Author

- Was `(*T).Method` considered? It's the canonical Go form for pointer-receiver method expressions and the same Go issue 3824 mention. The current test only covers value-receiver `T.Method2`. Suggest extending `var_initorder27.gno` with a `(*T).Method` line (mirrors the wider parity goal stated in the PR body).
- The CI failures (`gno-checks/lint` and `main/test`) appear unrelated to this change — lint is `r/test/sealviolation` (interface compliance, untouched files) and the test job is `params_valset_rotation_throttle` (validator-set throttle, also untouched). Worth a rerun before merge to confirm they're flakes vs. real bisect candidates.
