# PR [#5921](https://github.com/gnolang/gno/pull/5921): fix(gnovm): reject go1.18 generics syntax at deploy-time type check

URL: https://github.com/gnolang/gno/pull/5921
Author: ltzmaxwell | Base: master | Files: 4 | +168 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 6bcdde7e3 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5921 6bcdde7e3`

**TL;DR:** Gno is a subset of Go that stops at go1.17 and has no generics. Today a package can still declare a generic type or function and pass the deploy-time checker. This PR adds a small check that rejects that syntax up front with a clear, located error.

**Verdict: APPROVE** — correct, deterministic, deploy-gated guard with unit and end-to-end coverage; one Suggestion to broaden the interface type-set check past `|`/`~`, which the doc comment understates as undetectable.

## Summary
`typeCheckMemPackage` runs go/types, and modern go/types accepts go1.18 generics that Gno cannot run. The new `checkNoGenerics` walks each parsed file's AST after Go-parse and rejects type parameters (`type W[P any]`, `func Bar[T any]`) and interface type-set terms (unions `A | B`, approximations `~T`), reporting the earliest-positioned construct as a deterministic error. It runs at every `TypeCheckMemPackage` call, including the deploy path in [`keeper.go:647`](https://github.com/gnolang/gno/blob/6bcdde7e3/gno.land/pkg/sdk/vm/keeper.go#L647) · [↗](../../../../../.worktrees/gno-review-5921/gno.land/pkg/sdk/vm/keeper.go#L647), where a returned error becomes `ErrTypeCheck` and blocks the add-package. Before the guard these forms are not silently accepted downstream: they pass go/types and then panic in the gno preprocessor, so the guard replaces an unlocated panic with a located deploy-time rejection.

## Examples
| Written form | Before (master) | After |
|---|---|---|
| `type Foo[T any] int` | passes go/types, preprocessor nil-deref panic | `...:N:M: generic type declarations are not supported (Gno targets go1.17)` |
| `func Bar[T any]() {}` | passes go/types, preprocessor nil-deref panic | `generic functions are not supported` |
| `interface{ int \| string }` | passes go/types, `operator \| not defined on: TypeKind:` panic | `interface type unions are not supported` |
| `interface{ ~int }` | passes go/types, `checker for ILLEGAL does not exist` panic | `interface approximation (~) terms are not supported` |
| `interface{ int }` | passes go/types, preprocessor nil-deref panic | still passes the guard (residual gap) |

## Glossary
- type-check: go/types-based validation of gno source (`TypeCheckMemPackage`), distinct from preprocessing.
- preprocess: the static pass that resolves names, types, and blocks before execution.
- filetest: a `*_filetest.gno`-style file executed by the VM and asserted against golden directives.
- MemPackage: in-memory set of a package's source files, the unit loaded, type-checked, and run.

## Fix
Purely additive. A new `checkNoGenerics` in [`nogenerics.go:23`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/pkg/gnolang/nogenerics.go#L23) · [↗](../../../../../.worktrees/gno-review-5921/gnovm/pkg/gnolang/nogenerics.go#L23) is called from a new step in [`gotypecheck.go:435`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/pkg/gnolang/gotypecheck.go#L435) · [↗](../../../../../.worktrees/gno-review-5921/gnovm/pkg/gnolang/gotypecheck.go#L435), between the Go-parse step and `prepareGoGno0p9`, over `allgofs` (all parsed user/test/filetest files, not the injected gnobuiltins, which deliberately use a generic `revive` shim).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[stale after the guard]** [`gotypecheck.go:432`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/pkg/gnolang/gotypecheck.go#L432) · [↗](../../../../../.worktrees/gno-review-5921/gnovm/pkg/gnolang/gotypecheck.go#L432) — the new comment is the fourth `STEP 3:` in `typeCheckMemPackage` (429, 432, 439, 447). The step labels were already non-unique before this PR; the addition continues the existing pattern. Not posted; renumbering all four is out of this PR's scope and the new line matches the surrounding style.

## Missing Tests
None blocking. The two shipped tests cover the guard well: `TestCheckNoGenerics` in [`nogenerics_test.go:11`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/pkg/gnolang/nogenerics_test.go#L11) · [↗](../../../../../.worktrees/gno-review-5921/gnovm/pkg/gnolang/nogenerics_test.go#L11) asserts each rejected form and one accepted embed, and [`type_param0.gno`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/tests/files/type_param0.gno) · [↗](../../../../../.worktrees/gno-review-5921/gnovm/tests/files/type_param0.gno) proves both the type-check rejection (`// TypeCheckError:`) and the preprocess rejection (`// Error:`) end to end.

## Suggestions
- **[guard narrower than its comment]** [`nogenerics.go:17`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/pkg/gnolang/nogenerics.go#L17) · [↗](../../../../../.worktrees/gno-review-5921/gnovm/pkg/gnolang/nogenerics.go#L17) — the comment says bare non-interface type-set elements are not detectable syntactically, but only the bare-ident case (`interface{ int }`) is. `interface{ *int }`, `interface{ []byte }`, and `interface{ struct{} }` also slip past the guard, and those forms cannot be legal embedded interfaces, so they are syntactically distinguishable.
  <details><summary>details</summary>

  The `InterfaceType` branch at [`nogenerics.go:44-62`](https://github.com/gnolang/gno/blob/6bcdde7e3/gnovm/pkg/gnolang/nogenerics.go#L44-L62) · [↗](../../../../../.worktrees/gno-review-5921/gnovm/pkg/gnolang/nogenerics.go#L44-L62) flags an embedded field only when its type is a `BinaryExpr` with `|` or a `UnaryExpr` with `~`. A legal interface embed is a plain or qualified identifier (`ast.Ident` or `ast.SelectorExpr`); any other embedded element type is a go1.18 type-set element. Confirmed at 6bcdde7e3 by type-checking each form through `TypeCheckMemPackage`: `interface{ int }`, `interface{ *int }`, `interface{ []byte }`, and `interface{ struct{} }` all return no error, while `interface{ int | string }` and `interface{ ~int }` are rejected. Like the acknowledged `interface{ int }` case, these then reach the preprocessor and panic there rather than getting a clean type-check error. The genuinely undetectable case is only the bare ident, which is indistinguishable from embedding an interface named `int`. Fix: reject any embedded element whose type is not a plain or qualified identifier, or narrow the comment's claim to the bare-ident case.
  </details>

## Verified
- Deploy path is gated: [`keeper.go:647-650`](https://github.com/gnolang/gno/blob/6bcdde7e3/gno.land/pkg/sdk/vm/keeper.go#L647-L650) · [↗](../../../../../.worktrees/gno-review-5921/gno.land/pkg/sdk/vm/keeper.go#L647-L650) returns `ErrTypeCheck` when `TypeCheckMemPackage` errors, so a generic declaration fails add-package.
- Revert-proof of value: on master (guard removed) `type Foo[T any] int` and `interface{ int }` pass go/types then panic with a nil pointer dereference in the gno preprocessor; `interface{ int | string }` panics with `operator | not defined on: TypeKind:` and `interface{ ~int }` with `checker for ILLEGAL does not exist`. With the guard each returns a located type-check error instead. Verified by running `RunMemPackage` on each form in the worktree package.
- Coverage of forms: type parameters on a nested-in-body local type (`func F(){ type L[T any] int }`), generic type aliases (`type A[T any] = B[T]`), inline func-constraint unions (`func F[T int|string]()`), and unions of approximations (`~int | ~string`) are all rejected; generic methods are rejected earlier by go/parser (`method must have no type parameters`). Verified by type-checking each in the worktree.
- No on-chain break: no package under `examples/` or `gnovm/stdlibs/` declares generics, so the guard rejects nothing that currently deploys. Grep of the worktree.
- Determinism: the error is `fset.Position(off)` of the earliest-position construct across files in deterministic file order; no map iteration, wall clock, or randomness on the path.
- Tests green at 6bcdde7e3: `TestCheckNoGenerics` and `TestFiles/type_param0.gno`. The full `TestFiles` suite reddens only on `type41.gno` (`nil is not a type` vs `nil (untyped nil) is not a type`), a go/types message drift from the local go1.26.4 toolchain versus CI's go1.25.9; `type41.gno` is untouched by this PR and CI's `main / test` passed.

## Open questions
- The residual type-set gap (`interface{ int }` and the pointer/slice/struct forms above) still reaches the preprocessor and panics there rather than surfacing a clean error. The PR defers this to a future preprocess check; whoever adds that check should make it a located rejection, matching this guard's error shape. Not posted: deferred scope with no action required in this PR beyond the Suggestion.
- CI `build` check is red for an unrelated reason: `misc/gendocs` installs `golang.org/x/pkgsite@latest`, whose v0.3.0 requires go >= 1.26.0 while CI runs go1.25.9. Infra/toolchain, not this PR's code. Noted in the comment Body, no code change.
