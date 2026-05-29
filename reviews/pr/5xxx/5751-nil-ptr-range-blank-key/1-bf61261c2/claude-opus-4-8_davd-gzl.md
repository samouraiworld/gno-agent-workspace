# PR #5751: fix: nil ptr when discarded key and value not present on range

URL: https://github.com/gnolang/gno/pull/5751
Author: Villaquiranm | Base: master | Files: 2 | +32 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: bf61261c2 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5751 bf61261c2`

**Verdict: REQUEST CHANGES** — the one-line fix is correct for the reported reproducer, but it patches the symptom in `AssertCompatible`'s early-return instead of the root cause, so the sibling form `for _, v = range slice` (blank key, value present) still panics with the identical nil deref.

## Summary

`(*RangeStmt).AssertCompatible` crashed on `for _ = range a.e` because a blank-identifier key with an absent value (`x.Value == nil`) skipped the both-blank early return, then `evalStaticTypeOf(_)` returned nil and `assertIndexTypeIsInt(nil)` dereferenced it ([type_check.go:823](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L823) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L823)). The PR widens the early-return guard to also fire when `x.Value == nil`. That fixes the exact case in #5664, but the real defect is that the slice/array branch index-checks the key type without a nil guard — the same branch the string branch already guards with `if kt != nil`. Any range where the key is `_` produces a nil key type, and a present non-blank value keeps it out of the early return, so `for _, v = range slice` still crashes.

## Glossary
- `AssertCompatible` — preprocess-time type-check hook on `RangeStmt`.
- `evalStaticTypeOf` — resolves the static type of an expression; returns nil for the blank identifier `_`.
- `assertIndexTypeIsInt` — asserts a range key type is `int`; calls `kt.Kind()` with no nil check.

## Fix

Before: `AssertCompatible` returned early only when both key and value were blank identifiers; `for _ = range x` (Op `ASSIGN`, blank key, nil value) fell through to the per-X-type checks and crashed. After: the guard returns when the key is blank and the value is absent-or-blank ([type_check.go:832-835](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L832-L835) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L832-L835)). The load-bearing constraint is unchanged: a blank key means there is nothing to type-check, so skipping is safe. The gap is that "blank key" can also occur with a present value, which this guard does not cover.

## Critical (must fix)

- **[same crash, different range form]** [`type_check.go:853`](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L853) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L853) — `for _, v = range slice` (blank key, value present) still nil-derefs in `assertIndexTypeIsInt`.
  <details><summary>details</summary>

  **Shape:** blank key + present non-blank value + slice/array operand.
  **Mechanism:** the value is present and non-blank, so the widened early return at [type_check.go:832](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L832) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L832) does not fire. Execution reaches the `SliceType`/`ArrayType` branch, which calls `assertIndexTypeIsInt(kt)` with `kt = evalStaticTypeOf(_) = nil` ([type_check.go:853](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L853) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L853), [type_check.go:858](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L858) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L858)). `assertIndexTypeIsInt` calls `kt.Kind()` with no nil check ([type_check.go:822-826](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L822-L826) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L822-L826)) → uncaught nil pointer dereference. The `MapType` branch survives because `mustAssignableTo` tolerates a nil type, and the `StringKind` branch survives because it guards `if kt != nil` ([type_check.go:864](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L864) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L864)) — only slice/array is unguarded.
  **Result:** same stack as #5664 — `assertIndexTypeIsInt({0x0,0x0})` at type_check.go:823, called from `AssertCompatible` at type_check.go:853.
  **Fix:** address the root cause rather than the early-return. Either skip the key index-check when the key is blank (`if !isBlankIdentifier(x.Key) { assertIndexTypeIsInt(kt) }` in both the slice and array branches), or guard `assertIndexTypeIsInt` against a nil type like the string branch does. Then `for _ = range slice` is also covered without the special-cased early return.
  </details>

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5751 -R gnolang/gno
  cat > gnovm/tests/files/range_blank_key_5751.gno <<'EOF'
  package main

  func main() {
  	a := []int{1, 2, 3}
  	var v int
  	for _, v = range a {
  	}
  	_ = v
  	println("ok")
  }

  // Output:
  // ok
  EOF
  go test -run 'TestFiles/range_blank_key_5751.gno$' ./gnovm/pkg/gnolang/
  rm gnovm/tests/files/range_blank_key_5751.gno
  ```
  ```
  --- FAIL: TestFiles/range_blank_key_5751.gno (0.00s)
      files_test.go:111: unexpected panic: runtime error: invalid memory address or nil pointer dereference
      ...
      gnolang.assertIndexTypeIsInt({0x0, 0x0})  type_check.go:823
      gnolang.(*RangeStmt).AssertCompatible(...) type_check.go:853
  ```

## Warnings (should fix)

None.

## Nits

- [`issue_5664.txtar:1`](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/cmd/gno/testdata/test/issue_5664.txtar#L1) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/cmd/gno/testdata/test/issue_5664.txtar#L1) — header comment "Issue 2821" is copied verbatim from Go's `fixedbugs/bug406.go`; this is gno issue #5664. Update or drop the stale reference.
- [`issue_5664.txtar:30`](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/cmd/gno/testdata/test/issue_5664.txtar#L30) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/cmd/gno/testdata/test/issue_5664.txtar#L30) — no trailing newline at end of file.
- [`issue_5664.txtar:1`](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/cmd/gno/testdata/test/issue_5664.txtar#L1) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/cmd/gno/testdata/test/issue_5664.txtar#L1) — the header says "several combinations of key and value (presence)" but only two forms are tested (`for _ = range` and `for range`). Either widen the test (see Missing Tests) or tighten the comment.

## Missing Tests

- **[regression hole]** [`issue_5664.txtar`](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/cmd/gno/testdata/test/issue_5664.txtar) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/cmd/gno/testdata/test/issue_5664.txtar) — add the `for _, v = range slice` form, which currently panics.
  <details><summary>details</summary>

  The test covers the two forms that now early-return but omits the blank-key-present-value form that still crashes (Critical above). Adding it to the same txtar would have surfaced the incomplete fix. Adversarial filetest: [`tests/range_blank_key_5751.gno`](tests/range_blank_key_5751.gno) — asserts `// Output: ok`; flips green once the slice/array branch handles a nil key type.
  </details>

## Suggestions

- [`type_check.go:828-874`](https://github.com/gnolang/gno/blob/bf61261c2/gnovm/pkg/gnolang/type_check.go#L828-L874) · [↗](../../../../../.worktrees/gno-review-5751/gnovm/pkg/gnolang/type_check.go#L828-L874) — once the root cause is fixed (skip key check when key is blank / nil), the widened early-return at line 832 becomes redundant and can revert to the original both-blank check, keeping a single code path for "blank key" handling instead of two.

## Questions for Author

- Was the blank-key-with-present-value form (`for _, v = range slice`) considered? It hits the identical `assertIndexTypeIsInt(nil)` deref and isn't covered by either the fix or the test.
