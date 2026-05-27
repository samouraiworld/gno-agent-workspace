# PR #5733: fix(gnovm): avoid having empty pointer if range nil array is discarded

URL: https://github.com/gnolang/gno/pull/5733
Author: Villaquiranm | Base: master | Files: 2 | +25 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `8867633` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5733 8867633`

**Verdict: APPROVE** — one-line fix matches Go semantics, txtar covers the headline case, only nits (test filename names the wrong issue, helper comment typo, `Name == blankIdentifier` would be more idiomatic than `Name == "_"`).

## Summary

Issue [#5665](https://github.com/gnolang/gno/issues/5665): `for i, _ := range nilPtr` panics in Gno but is valid in Go — `_` discards the value, so the nil dereference never has to happen. The fix gates the value-read path in `OpRangeIter`'s `-1` (assign-element) phase on a new `isDiscardedValue` predicate: when the AST value is a `*NameExpr` named `_`, skip both the existing nil-ptr-array panic check and the `GetPointerAtIndex().Deref()` lookup. Key reads are unaffected (`i` is just the index). Behavior parity with Go was verified across the matrix: `for i, _ := range nilPtr` succeeds (sum=45), `for _, _ = range nilPtr` succeeds, `for _, v := range nilPtr` still panics, non-nil iteration unchanged.

## Fix

Before: [`op_exec.go:212`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/op_exec.go#L212) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/op_exec.go#L212) entered the value-assign branch whenever `bs.Value != nil`, and the very first check inside ([`op_exec.go:215`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/op_exec.go#L215) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/op_exec.go#L215)) raised `nil pointer dereference` for nil array pointers. After: the guard becomes `bs.Value != nil && !isDiscardedValue(bs.Value)` — when the user wrote `_`, the deref-and-assign path is skipped wholesale. The new helper [`op_exec.go:848-852`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/op_exec.go#L848-L852) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/op_exec.go#L848-L852) returns true iff `e` is a `*NameExpr` whose `Name == "_"`. Safe across `Op`s: for `DEFINE`, nothing is on the stack to pop; for `ASSIGN`, [`PushForPointer(*NameExpr)`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/machine.go#L2586-L2589) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/machine.go#L2586-L2589) is a no-op (`// no Lhs eval needed.`), so skipping the corresponding `PopAsPointer` leaks nothing.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/cmd/gno/testdata/test/issue_5667.txtar`](https://github.com/gnolang/gno/blob/8867633/gnovm/cmd/gno/testdata/test/issue_5667.txtar) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/cmd/gno/testdata/test/issue_5667.txtar) — filename references issue #5667, but the PR body says "fixes #5665". Issue #5667 exists and is a different bug (interface-method nil-ptr panic for `fixedbugs/issue19040.go`). Rename to `issue_5665.txtar` to match the actual fixed issue.
- [`op_exec.go:848`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/op_exec.go#L848) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/op_exec.go#L848) — comment typo `discarted` → `discarded`; also slightly tighter as `// isDiscardedValue reports whether e is the blank identifier.`
- [`op_exec.go:851`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/op_exec.go#L851) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/op_exec.go#L851) — every other check in the package uses the `blankIdentifier` const (defined at [`preprocess.go:18`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/preprocess.go#L18) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/preprocess.go#L18) and used in `machine.go`, `nodes.go`, `values.go`, `preprocess.go`, `op_eval.go`). Use `namexp.Name == blankIdentifier` for grep-consistency.
- [`testdata/test/issue_5667.txtar:18`](https://github.com/gnolang/gno/blob/8867633/gnovm/cmd/gno/testdata/test/issue_5667.txtar#L18) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/cmd/gno/testdata/test/issue_5667.txtar#L18) — file ends without trailing newline (`\ No newline at end of file` in diff). Most existing txtars in the same directory terminate with `\n`.

## Missing Tests

- **[blank-blank variant uncovered]** [`gnovm/cmd/gno/testdata/test/issue_5667.txtar`](https://github.com/gnolang/gno/blob/8867633/gnovm/cmd/gno/testdata/test/issue_5667.txtar) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/cmd/gno/testdata/test/issue_5667.txtar) — the new txtar only exercises `for i, _ := range arr` (DEFINE op). The fix also covers `for _, _ = range arr` (ASSIGN op, both blanks) and `for k, _ = range arr` (ASSIGN op, single blank value with non-`_` named key). Both work today, but a second case in the same file would lock in the behavior cheaply.
  <details><summary>details</summary>

  Repro for the ASSIGN-with-blank case:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5733 -R gnolang/gno
  cat > /tmp/blank.gno <<'EOF'
  package main

  func main() {
  	var arr *[5]int
  	var i int
  	s := 0
  	for i, _ = range arr {
  		s += i
  	}
  	println("sum", s)
  }
  EOF
  go run ./gnovm/cmd/gno run /tmp/blank.gno
  rm /tmp/blank.gno
  ```

  ```
  sum 10
  ```

  Add as a second `-- main2.gno --` block (or a sibling `gno run main2.gno` invocation) inside the same txtar.
  </details>

## Suggestions

- [`op_exec.go:212-218`](https://github.com/gnolang/gno/blob/8867633/gnovm/pkg/gnolang/op_exec.go#L212-L218) · [↗](../../../../../.worktrees/gno-review-5733/gnovm/pkg/gnolang/op_exec.go#L212-L218) — minor structural alternative: keep the outer `if bs.Value != nil` and short-circuit the discarded case right where the comment lives, so the nil-deref panic stays paired with its rationale rather than gated by a boolean further out. Cosmetic; current form is fine.
  <details><summary>details</summary>

  ```go
  if bs.Value != nil {
      if isDiscardedValue(bs.Value) {
          // `_` discards the value; skip the deref so `for i, _ := range nilPtr`
          // does not panic (matches Go).
      } else {
          if op == OpRangeIterArrayPtr && xv.V == nil {
              m.pushPanic(typedString("runtime error: nil pointer dereference"))
              return
          }
          // ... existing assignment ...
      }
  }
  ```

  Same control flow, comment lives next to the panic it explains. Take or leave.
  </details>

## Questions for Author

- Was the test file's `issue_5667.txtar` name intentional, or a typo for `issue_5665.txtar`? (PR body says "fixes #5665"; issue #5667 is a different open bug.)
