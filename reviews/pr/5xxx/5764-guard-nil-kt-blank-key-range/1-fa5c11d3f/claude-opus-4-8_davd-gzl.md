# PR #5764: fix(gnovm): guard nil kt in assertIndexTypeIsInt for blank-key range

URL: https://github.com/gnolang/gno/pull/5764
Author: ltzmaxwell | Base: master | Files: 2 | +15 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `fa5c11d3f` (stale — +65 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5764 fa5c11d3f`

**Verdict: APPROVE** — this is the root-cause fix the [#5751 review](https://github.com/gnolang/gno/pull/5751) asked for: it guards `assertIndexTypeIsInt` against a nil key type and folds the duplicated string-branch guard into it, so all four range branches (slice/array/string/map) now tolerate a blank key. Supersedes #5751's narrower early-return approach. Only nits: a pre-existing wrong-variable panic message on the line right below the change, and overlap with two sibling open PRs that should be reconciled.

## Summary

`for _ = range slice` panicked with a nil-pointer dereference. A blank key `_` has no static type, so `evalStaticTypeOf(_)` returns nil, and the `SliceType`/`ArrayType` branches of `(*RangeStmt).AssertCompatible` called `assertIndexTypeIsInt(kt)` which dereferenced `kt.Kind()` with no nil guard — the issue #5664 crash. This PR adds `if kt == nil { return }` at the top of `assertIndexTypeIsInt` ([type_check.go:822-829](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L822-L829) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L822-L829)) and replaces the string branch's inline `if kt != nil && kt.Kind() != IntKind` with a call to the same helper ([type_check.go:867](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L867) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L867)). One filetest pins the regression.

Guarding `kt == nil` is the correct fix, not a mask over a preprocessing gap. A discarded range key is genuinely unconstrained — Go's own type checker imposes no `int` requirement on the key of `for _ = range s` because nothing receives it. "Blank key ⇒ nil static type ⇒ nothing to assert" is the right invariant. The map branch already encoded this implicitly (`mustAssignableTo` returns nil when the destination type is nil, [type_check.go:410](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L410) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L410)); this PR brings slice/array/string in line.

## Glossary
- `AssertCompatible` — preprocess-time type-check hook on `RangeStmt`.
- `evalStaticTypeOf` — static type of an expression; returns nil for the blank identifier `_`.
- `assertIndexTypeIsInt` — asserts a range key type is `int`; now returns early on nil.
- `mustAssignableTo` — panics unless `xt` is assignable to `dt`; tolerates `dt == nil`.

## Fix

Before: `assertIndexTypeIsInt(nil)` dereferenced `nil.Kind()`, and the slice/array branches called it unconditionally on the (possibly nil) blank-key type. The string branch hand-rolled its own `kt != nil` guard. After: the helper returns immediately when `kt == nil`, so the three call sites at [type_check.go:856](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L856) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L856), [:861](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L861) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L861), [:867](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L867) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L867) are all nil-safe with one guard, and the duplicated inline check is removed. Load-bearing constraint: a non-nil key with the wrong kind still panics, so the int-key assertion is preserved for real keys (verified — see Missing Tests baseline).

## Relationship to #5733, #5751 (same area, all three open)

All three PRs target the blank-key-range crash family and are open simultaneously:

- #5733 — fixes the *execution-time* deref (`for i, _ := range nilPtr` in `op_exec.go`). Disjoint code path; no conflict.
- #5751 — fixes the *same preprocess-time* crash as this PR but by widening the both-blank early-return in `AssertCompatible` to also fire when `x.Value == nil`. My [#5751 review](https://github.com/gnolang/gno/pull/5751) flagged that approach as Critical-incomplete: it still crashed on `for _, v = range slice` (blank key, *present* value), which skips the early return. This PR (#5764) fixes the root cause instead, so it covers that form too (verified below) and does **not** need the widened early-return.

Recommendation: merge #5764 and close #5751 as superseded, or reduce #5751 to just its txtar (the issue_5664 regression file) on top of this fix. The two should not both land — #5751's widened early-return becomes dead/redundant once `assertIndexTypeIsInt` is nil-safe. Both still carry stale issue-number filenames (#5751 review nits).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`type_check.go:870`](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L870) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L870) — pre-existing wrong-variable panic message in the branch this PR edits: the check is `vt.Kind() != Int32Kind` but the message prints `kt` (`"value type should be int32, but got %v", kt`). For `for k, v = range str` with a wrong-typed value it reports the *key* type, not the offending value type. Not introduced here, but the PR touches lines 864-867 of this exact `if` block — cheap to flip `kt` → `vt` while in the neighborhood.
- [`range_blank_key.gno:7`](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/tests/files/range_blank_key.gno#L7) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/tests/files/range_blank_key.gno#L7) — `for _ = range s {}` is the precise reproducer (Op `ASSIGN`, blank key, nil value, which is what crashed) so it is correct to keep, but `gofmt`/`gno fmt` rewrites bare `for _ = range s` to `for range s` — and `for range s` has `Op != ASSIGN`, so it returns at [type_check.go:832](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/pkg/gnolang/type_check.go#L832) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/pkg/gnolang/type_check.go#L832) *before* reaching the fixed code and would silently stop exercising the regression. Confirm `gno fmt` leaves this file alone (the explicit `_ =` form is load-bearing for the test).

## Missing Tests

- **[blank-key + present-value form not pinned]** [`range_blank_key.gno`](https://github.com/gnolang/gno/blob/fa5c11d3f/gnovm/tests/files/range_blank_key.gno) · [↗](../../../../../.worktrees/gno-review-5764/gnovm/tests/files/range_blank_key.gno) — the filetest covers only `for _ = range slice` (blank key, absent value). The fix also rescues `for _, v = range {slice,array,string,map}` (blank key, *present* value), which is the exact form #5751's approach left broken. Worth a second case to lock in the differentiator and prevent a future regression back to an early-return-only fix.
  <details><summary>details</summary>

  Adversarial filetest: [`tests/range_blank_key_present_value.gno`](tests/range_blank_key_present_value.gno) — `for _, v = range` over all four operand types, asserts `// Output: ok`. Passes on this PR; crashes on master with the issue-#5664 stack (`assertIndexTypeIsInt({0x0,0x0})` ← `(*RangeStmt).AssertCompatible`). Verified both directions.
  </details>

  **Repro (the present-value form + the wrong-type-key baseline in one run):**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5764 -R gnolang/gno
  cat > gnovm/tests/files/range_blank_present_value.gno <<'EOF'
  package main

  func main() {
  	s := []int{1, 2, 3}
  	var sv int
  	for _, sv = range s {
  	}
  	_ = sv
  	str := "abc"
  	var rv rune
  	for _, rv = range str {
  	}
  	_ = rv
  	println("ok")
  }

  // Output:
  // ok
  EOF
  go test -run 'TestFiles/range_blank_present_value.gno$' ./gnovm/pkg/gnolang/
  # baseline: a non-nil wrong-kind key still panics (assertion preserved)
  go test -run 'TestFiles/types/assign_range_c.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/range_blank_present_value.gno
  ```
  ```
  ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.137s
  ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.140s
  ```

## Suggestions

None.

## Questions for Author

- #5751 and #5764 fix the same preprocess crash two different ways and are both open. Plan to close #5751 as superseded by this root-cause fix, or keep its regression txtar on top of #5764? Landing both leaves #5751's widened early-return redundant.
