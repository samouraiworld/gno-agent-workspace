# PR #5764: fix(gnovm): handle blank range key/value per-operand, validate assignment targets

URL: https://github.com/gnolang/gno/pull/5764
Author: ltzmaxwell | Base: master | Files: 28 | +481 -81
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 0509a199b (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5764 0509a199b`

Round 2. The round-1 review (`1-fa5c11d3f/`) covered a 2-file, +15 -3 change: a single nil-key guard in `assertIndexTypeIsInt`. Since then the PR grew into a full generalization of assignment-target/RHS validation across `RangeStmt`, `AssignStmt`, and `IncDecStmt`, absorbing #5804. Re-reviewed from scratch against the new head, then re-run through a deep multi-lens pass (red-team / blue-team / correctness): all three converged on one scope-completion gap (pointer-to-array range, below) and confirmed every other claim by running. No regressions or false positives found; verdict unchanged.

**TL;DR:** Makes the GnoVM preprocessor type-check every assignment target the same way: each range/assign/compound-assign/inc-dec operand is checked on its own, blank operands are skipped by syntax instead of by passing a nil type around, and the leftover checks now use real assignability (matching Go's `go/types`) instead of ad-hoc kind comparisons.

**Verdict: APPROVE** — principled, well-tested generalization of the round-1 fix; resolves both round-1 nits; no blocking issues found. One scope-completion gap: pointer-to-array range is the single container shape the rewrite leaves unchecked (Suggestion), pre-existing and backstopped by go/types, so non-blocking.

## Summary

The blank-key range crash (issue #5664) is fixed at the root: range operands are now evaluated through `evalAssignLhsType`, which returns `nil` exactly when the operand is blank, and every type check is guarded on a non-nil operand type. The PR then extends the same per-operand pattern to compound assignment and inc/dec, adds RHS validity (`i += int`, `i += f()`, `_ = nil`) and unassignable-target rejection (`c += 1`, `c++`, `for _, c = range a` with const `c`), and switches the slice/array/string operand checks from kind equality to `mustAssignableTo` over `baseOf(container)`, so declared map/string types and named int/rune operands are judged correctly. Error precedence is aligned with `go/types` (operator/type errors before target-assignability), pinned by the `assign_op_c`–`g` goldens.

## Glossary
- `AssertCompatible` — preprocess-time type-check hook on a statement/expr node.
- `evalAssignLhsType` — new helper: asserts the LHS is a valid assignment target and returns its static type, or `nil` iff the target is blank.
- `mustAssignableTo(n, src, dst)` — panics unless `src` is assignable to `dst`; `dst` must now be non-nil.
- `baseOf` — underlying type of a possibly-declared (named) type.

## Fix

Round 1 added `if kt == nil { return }` inside `assertIndexTypeIsInt`. This round deletes that helper entirely and restructures `(*RangeStmt).AssertCompatible` per-operand ([type_check.go:833-873](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/type_check.go#L833-L873) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L833-L873)): `kt`/`vt` are `nil` iff blank, the both-blank case returns early, and surviving operands are checked by assignability against `baseOf(x.X)`. The blank-target convention is centralized in `evalAssignLhsType` ([type_check.go:1044-1053](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/type_check.go#L1044-L1053) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L1044-L1053)), so `checkAssignableTo` no longer tolerates a nil `dt` and instead panics on it ([type_check.go:388-396](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/type_check.go#L388-L396) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L388-L396)); the three `checkOrConvertType` sites that legitimately passed a nil target are guarded with `if t != nil` ([preprocess.go:4716-4744](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/preprocess.go#L4716-L4744) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/preprocess.go#L4716-L4744)). The load-bearing constraint: the only previously-nil-`dt` call paths are blank targets (now skipped at source) and `checkOrConvertType` (now guarded); all other callers pass structurally non-nil destination types.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`type_check.go:1011-1012`](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/type_check.go#L1011-L1012) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L1011-L1012) — the "like go/types" comment on the compound-assign precedence overstates fidelity to `gc`. For `c += f()` (const target, no-value RHS), `gc` lists the target error first (`cannot assign to c`), then the RHS error; the `assign_op_g` golden pins only the RHS error. The golden is self-consistent with what gno's checker actually emits (test passes), so this is a comment-accuracy nit, not a behavior bug. Confirmed behaviorally: built the equivalent Go, `gc` reports both errors target-first.

Both round-1 nits are resolved: the wrong-variable `kt`-for-`vt` panic message is gone (that whole string branch was replaced by `mustAssignableTo`), and `gno fmt` leaves `for _ = range s` untouched (verified), so `range_blank_key3`/`5` stay load-bearing regressions.

## Missing Tests

- **[generalization untested for one range source]** [`type_check.go:848-872`](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/type_check.go#L848-L872) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L848-L872) — no filetest ranges over a pointer-to-array (`for _, v = range &arr`), the one permitted range source the per-operand switch never exercises. The absence is why the unchecked-operand gap (see Suggestions) ships silently.
  <details><summary>details</summary>

  The added range tests cover slice/array/map/string only. A mismatched value target over `&arr` preprocesses, runs, and prints an `int` from a `string` var; only the go/types pass flags it. Adversarial test written: [`tests/range_ptr_array_value.gno`](tests/range_ptr_array_value.gno) asserts the post-fix rejection (red on `0509a199b`, green once the `*PointerType` case lands). Ties the missing coverage to the Suggestion fix.
  </details>

Otherwise thorough: `range_blank_key2`–`5` (blank key/value across slice/array/map/string in assign form), `assign_range_f`–`k` (const target, nil operand, declared map, named int/rune), `assign_op_a`–`g` (const/string-index/blank targets, type/no-value RHS, and explicit error-order pins), `assign_index_d/e`, `assign_nil3/4`, `incdec_a5`, plus the nil-`dt` panic unit case in `TestCheckAssignableTo`. The `// TypeCheckError:` goldens tie each case to the actual `go/types` verdict.

## Suggestions

- **[generalization skips one container shape]** [`type_check.go:848-872`](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/type_check.go#L848-L872) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L848-L872) — range over `*[N]T` (pointer-to-array) is not operand-type-checked, so the two gno type-checkers disagree on exactly the kind of mismatch this PR exists to catch.
  <details><summary>details</summary>

  After `xt := baseOf(evalStaticTypeOf(x.X))`, a pointer-to-array source is a `*PointerType`, which matches none of the `*MapType` / `*SliceType, *ArrayType` / `PrimitiveType` cases, so both the key and value checks are skipped. The range guard at [preprocess.go:901](https://github.com/gnolang/gno/blob/0509a199b/gnovm/pkg/gnolang/preprocess.go#L901) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/preprocess.go#L901) explicitly lists pointer-to-array among the permitted range sources, so this is a missing switch case, not an unsupported type. `for _, v = range &arr` with `var v string` is silently accepted by `AssertCompatible`; only the separate go/types pass rejects it (`cannot use v (value of type int) as string value in assignment`), so the preprocess assertion and go/types diverge. Pre-existing on master (same silent acceptance, verified) and not a regression, but the PR's stated scope is per-operand assignment-target validation, and this is the one container shape Go accepts in `range` that the rewrite leaves out. Fix: add `case *PointerType:` unwrapping `pt.Elem()` to the array element in both switches. No on-chain type-safety escape: the go/types backstop in `TypeCheckMemPackage` still rejects the program. Demonstrated by [`tests/range_ptr_array_value.gno`](tests/range_ptr_array_value.gno): runs and prints `3` today, rejected at preprocess once the case is added.
  </details>

---

Verified on 0509a199b (CI-invisible checks):
- The new nil-`dt` panic in `checkAssignableTo` is unreachable from binary comparisons: `nil == nil` is rejected one step earlier by the `isComparable` check (`<nil> is not comparable`), identically on master and on this PR. `shouldSwapOnSpecificity` always routes a nil operand to `xt`, so `dt` is nil only when both operands are nil, which that earlier check already rejects.
- The kind→assignability switch is a real correctness gain, not just a message change: `for i = range slice` with `var i interface{}` now type-checks and runs (Go accepts it; old gno rejected with "index type should be int"), while named int/rune targets are now correctly rejected to match `go/types`.
