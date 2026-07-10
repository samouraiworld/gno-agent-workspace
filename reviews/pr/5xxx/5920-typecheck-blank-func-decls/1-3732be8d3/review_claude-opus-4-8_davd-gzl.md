# PR [#5920](https://github.com/gnolang/gno/pull/5920): fix(gnovm): type-check all blank func decls — uniqueDecls must not dedupe `_`

URL: https://github.com/gnolang/gno/pull/5920
Author: ltzmaxwell | Base: master | Files: 2 | +21 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 3732be8d3 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5920 3732be8d3`

**TL;DR:** Before deploying a package on-chain the node runs a go/types check as a front gate. That gate had a bug: if a file declared more than one blank function `func _()`, every one after the first was thrown away before checking, so type errors in those bodies went unseen. This PR keeps them all.

**Verdict: APPROVE** — correct, minimal, load-bearing fix; one stale code comment as a Nit, no blocker.

## Summary
`uniqueDecls` dedupes top-level function declarations by name so a Go-native override and its `.gno` twin do not read as a redeclaration during the go/types deploy gate. It treated the blank identifier like any other name: the first `func _()` was recorded, and every later `func _()` in the package was deleted from the AST before checking. Go allows any number of `func _()`, so the deletion was wrong, and it hid the deleted bodies from the gate. The fix skips `_` in the dedup loop exactly like `init`, both of which are the only top-level func names Go lets repeat. That gate is the on-chain type check in `AddPackage` and `MsgRun`, so the pre-fix behavior was an under-rejection of the deploy path.

## Examples
Package with two blank functions, the second body ill-typed:

```go
package hello
func _() { var x int = "a"; _ = x }   // error: cannot use "a"
func _() { var y string = 1;  _ = y } // error: cannot use 1
```

| | go/types gate result |
|---|---|
| Before | one error (`cannot use "a"`); the second body is deleted and unchecked |
| After | both errors reported |

## Glossary
- type-check: go/types validation of gno source (`TypeCheckMemPackage`), the front gate distinct from preprocessing.
- preprocess: the GnoVM static pass that resolves names, types, and blocks before execution.
- MemPackage: in-memory set of a package's source files, the unit loaded, type-checked, and run.

## Fix
`uniqueDecls` skips a func decl when it is a method, an `init`, and now a blank `_`, at [`gotypecheck.go:544-550`](https://github.com/gnolang/gno/blob/3732be8d3/gnovm/pkg/gnolang/gotypecheck.go#L544-L550) · [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/gotypecheck.go#L544-L550). The dedup map is created once per package at [`gotypecheck.go:584`](https://github.com/gnolang/gno/blob/3732be8d3/gnovm/pkg/gnolang/gotypecheck.go#L584) · [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/gotypecheck.go#L584) and threaded through every prod file at [`gotypecheck.go:655`](https://github.com/gnolang/gno/blob/3732be8d3/gnovm/pkg/gnolang/gotypecheck.go#L655) · [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/gotypecheck.go#L655), so the bug spanned the whole package, not just one file. Skipping `_` never reintroduces a real redeclaration error: the blank identifier binds nothing, so go/types accepts any number of blank funcs.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[comment no longer matches the code it labels]** [`gotypecheck.go:544`](https://github.com/gnolang/gno/blob/3732be8d3/gnovm/pkg/gnolang/gotypecheck.go#L544) · [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/gotypecheck.go#L544) — the loop comment reads "ignore methods and init functions" but the loop now also ignores blank funcs. Add `_` to the comment.

## Missing Tests
None required. The added case at [`gotypecheck_test.go:106-123`](https://github.com/gnolang/gno/blob/3732be8d3/gnovm/pkg/gnolang/gotypecheck_test.go#L106-L123) · [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/gotypecheck_test.go#L106-L123) pins that both blank-func bodies are checked and asserts the errors in source order. See Open questions for a cross-file variant that exercises the same shared-map path.

## Suggestions
None.

## Verified
- Reverting the fix reproduces the bug. Removing the `fd.Name.Name == "_"` line makes the new test report `expected 2 errors, got 1`: the second blank func's `cannot use 1` error is dropped by the gate. Restoring the line reports both. Repro in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md).
- The pre-fix hole is a deploy-gate hole, and the backstop is a recovered panic. `AddPackage` runs the gate at [`keeper.go:647`](https://github.com/gnolang/gno/blob/3732be8d3/gno.land/pkg/sdk/vm/keeper.go#L647) · [↗](../../../../../.worktrees/gno-review-5920/gno.land/pkg/sdk/vm/keeper.go#L647) then preprocesses the package at [`keeper.go:754`](https://github.com/gnolang/gno/blob/3732be8d3/gno.land/pkg/sdk/vm/keeper.go#L754) · [↗](../../../../../.worktrees/gno-review-5920/gno.land/pkg/sdk/vm/keeper.go#L754) under `defer doRecover` at [`keeper.go:731`](https://github.com/gnolang/gno/blob/3732be8d3/gno.land/pkg/sdk/vm/keeper.go#L731) · [↗](../../../../../.worktrees/gno-review-5920/gno.land/pkg/sdk/vm/keeper.go#L731). A filetest probe of the example package hits a bare preprocessor panic in `mustAssignableTo` at [`type_check.go:186`](https://github.com/gnolang/gno/blob/3732be8d3/gnovm/pkg/gnolang/type_check.go#L186) · [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/type_check.go#L186), which on-chain `doRecover` converts to a rejection. So without the fix the transaction still fails deterministically for the errors the preprocessor also catches; the fix restores a precise type-check rejection instead of relying on that panic.
- No existing package hits the pattern. Grep of `examples/`, `gnovm/stdlibs/`, and `gno.land/` finds no source file with two or more top-level `func _()`, so the hole was latent, not actively deployed against.
- `go test ./gnovm/pkg/gnolang -run TestTypeCheckMemPackage` green at 3732be8d3.

## Open questions
- The dedup map is shared across all prod files of a package, so two blank funcs split across two files trigger the same deletion. The test uses one file. A two-file case would cover the shared-map path directly, but it runs the identical code, so the gain is small; left unposted.
