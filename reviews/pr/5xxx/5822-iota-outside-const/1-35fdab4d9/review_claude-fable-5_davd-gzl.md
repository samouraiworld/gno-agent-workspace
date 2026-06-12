# PR #5822: fix(preprocess): using iota outside constant declaration

URL: https://github.com/gnolang/gno/pull/5822
Author: Villaquiranm | Base: master | Files: 3 | +40 -0
Reviewed by: davd-gzl | Model: claude-fable-5 | Commit: 35fdab4d9 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5822 35fdab4d9`

**TL;DR:** In Go, the special identifier `iota` only has meaning inside a `const` declaration. GnoVM crashed with an internal error when a program used `iota` anywhere else; this PR makes it report the same clear error Go does, with the right source position, plus two tests pinning that.

**Verdict: APPROVE** — guard is correct and nil-safe, both failure branches are pinned by the new tests, the message matches go/types exactly; only nits (define-site `iota` gets the use-site message, leftover Go-corpus `// ERROR` markers).

## Summary
Issue [#5668](https://github.com/gnolang/gno/issues/5668) (Go corpus `fixedbugs/bug186.go`): any use of `iota` outside a const declaration hit the unguarded attribute read at [`preprocess.go:1305`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/preprocess.go#L1305) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/preprocess.go#L1305), and since only const ValueDecls carry `ATTR_IOTA` (set at [`go2gno.go:912`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/go2gno.go#L912) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/go2gno.go#L912)), the `.(int)` assertion blew up as `interface conversion: interface {} is nil, not int` — an internal panic with no user-facing source position. The fix checks that the nearest enclosing declaration is a const ValueDecl before reading the attribute, and panics with `cannot use iota outside constant declaration` otherwise — byte-identical to the go/types message, so the `// Error:` and `// TypeCheckError:` blocks in the new filetests agree.

## Glossary
- **preprocess** — GnoVM's static-analysis pass over the AST before execution; panics there become positioned compile errors.
- **ATTR_IOTA** — AST attribute holding the spec index inside a const declaration; only const ValueDecls get it.
- **filetest** — `gnovm/tests/files/*.gno` whose behavior must match its `// Error:` / `// TypeCheckError:` block.

## Fix
Before: [`lastDecl`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/preprocess.go#L4675-L4682) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/preprocess.go#L4675-L4682) returned whatever declaration encloses the `iota` (a FuncDecl for `f(iota)`, a non-const ValueDecl for `var a = iota`, nil at worst) and the attribute read crashed. After: the guard at [`preprocess.go:1300-1303`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/preprocess.go#L1300-L1303) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/preprocess.go#L1300-L1303) requires a `*ValueDecl` with `Const` set. The comma-ok assertion is nil-safe, so the `lastDecl == nil` case also lands on the proper message. The residual unguarded `.(int)` is fine: every const ValueDecl gets `ATTR_IOTA` from the parser, the decl-split path re-propagates it ([`preprocess.go:152-153`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/preprocess.go#L152-L153) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/preprocess.go#L152-L153)), and the attribute-dropping `ValueDecl.Copy` ([`nodes_copy.go:326-333`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/nodes_copy.go#L326-L333) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/nodes_copy.go#L326-L333)) has no production caller reaching preprocess (`copyDecls` is used only by `FileNode.Copy`, itself unused outside tests).

Verified on 35fdab4d9: `go test -run 'TestFiles/iota' ./gnovm/pkg/gnolang/` passes (all four iota filetests, both VM-error and typecheck modes). Exercised live via `gno run`: function-level `const a = iota` and multi-name `const a, b = iota, iota*2` (split path) still evaluate correctly; package-level `var x = iota`, `type T [iota]int`, and `iota` inside a func literal in a var decl all report the new positioned error where master crashed with the interface-conversion panic ([repro](comment_claude-fable-5.md)).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`preprocess.go:1300-1303`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/preprocess.go#L1300-L1303) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/preprocess.go#L1300-L1303) — declaring a variable named `iota` (e.g. `iota := 5`, legal in Go) now reports "cannot use iota outside constant declaration", but the code declares `iota`, it doesn't use it; Gno's own message for that rule is `builtin identifiers cannot be shadowed: iota`. Fix: when `n.Type == NameExprTypeDefine`, panic with the shadowing message instead.
  <details><summary>details</summary>

  Gno deliberately forbids shadowing builtins: package-level `const iota = 5` already dies with `builtin identifiers cannot be shadowed: iota` via [`preprocess.go:5791-5794`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/pkg/gnolang/preprocess.go#L5791-L5794) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/pkg/gnolang/preprocess.go#L5791-L5794). But a local `iota := 5` reaches `case "iota"` first, with the define-site NameExpr itself, so the user is told they "use" iota outside a const declaration at the spot where they declare it. Confirmed behaviorally: error position is the left-hand side of the `:=`. Not a regression — master crashed with the internal interface-conversion panic on the same input — so this PR strictly improves it; the suggested branch just finishes the job. Fix: when `n.Type == NameExprTypeDefine`, panic with the shadowing message instead.
  </details>
- [`iota_outside_const.gno:9`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/tests/files/iota_outside_const.gno#L9) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/tests/files/iota_outside_const.gno#L9), [`iota_outside_const_2.gno:9`](https://github.com/gnolang/gno/blob/35fdab4d9/gnovm/tests/files/iota_outside_const_2.gno#L9) · [↗](../../../../../.worktrees/gno-review-5822/gnovm/tests/files/iota_outside_const_2.gno#L9) — the inline `// ERROR "iota"` comments are Go-corpus errorcheck directives the gno filetest harness ignores (only the `// Error:` / `// TypeCheckError:` blocks are asserted); no other file in `gnovm/tests/files` carries them. Drop them.

## Missing Tests
None — the two filetests pin both guard branches: `f(iota)` lands on a FuncDecl (non-ValueDecl arm) and `var a = iota` on a non-const ValueDecl (`!Const` arm). Package-level `var x = iota` and `type T [iota]int` exercise the same two arms (verified live, [repro](comment_claude-fable-5.md)).

## Suggestions
None.

## Open questions
- Unused parameter named `iota` is still accepted (`func f(iota int) int { return 0 }` runs; the shadowing rule fires only on predefine and path-fill paths, and `case "iota"` only on use). Pre-existing, outside this diff, so not posted.
- `iota` inside a function literal inside a const declaration (go/types resolves it to the enclosing spec index) is unreachable in Gno — const expressions cannot contain function literals without `unsafe.Sizeof`-style constant builtins — so the theoretical divergence has no input that triggers it; not posted.
