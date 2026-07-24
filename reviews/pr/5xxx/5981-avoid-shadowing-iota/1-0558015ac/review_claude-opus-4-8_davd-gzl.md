# PR [#5981](https://github.com/gnolang/gno/pull/5981): fix(preprocess): avoid shadowing of iota

URL: https://github.com/gnolang/gno/pull/5981
Author: Villaquiranm | Base: master | Files: 8 | +84 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 0558015ac (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5981 0558015ac`

**TL;DR:** `iota` is the counter you use inside `const` blocks. Writing it as an ordinary variable name used to blow up with a confusing internal panic in some places and quietly work in others; this PR makes the compiler reject it up front with a clear message.

**Verdict: REQUEST CHANGES** — the rule the PR introduces still does not fire for a three-clause `for` init, so `for iota := 0; iota < 2; iota++` keeps compiling while `for iota := range s` is now an error (2 Warnings, 2 Nits, 1 Missing test, 1 Suggestion).

## Summary

Since [PR 5822](https://github.com/gnolang/gno/pull/5822), naming something `iota` outside a const declaration reaches [`case iotaIdentifier:`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L1298) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L1298) in the preprocessor and panics with `cannot use iota outside constant declaration`, an internal message that names no fix. The PR moves rejection earlier: [`StaticBlock.Reserve`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/nodes.go#L2317-L2324) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2317-L2324) now rejects any name binding called `iota`, so the error arrives at the declaration site with the same wording gno already uses for `var iota` and `type iota`. This is a deliberate divergence from Go, which allows `iota` as an ordinary identifier; [issue 5876](https://github.com/gnolang/gno/issues/5876) states that choice outright.

`Reserve` is the wrong side of one earlier pass. [`initStaticBlocks`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L181-L183) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L181-L183) runs `initStaticBlocks1` before `initStaticBlocks2`, and the first pass [renames every name declared in a three-clause `for` init to `<name>.loopvar`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L284-L300) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L284-L300). `Reserve` runs in the second pass, so for that one binding form it sees `iota.loopvar` and the guard never fires.

```
source            initStaticBlocks1            initStaticBlocks2
                  (.loopvar rename)            (Reserve → guard)

iota := 5         iota                    →    iota          → rejected
for iota := range iota                    →    iota          → rejected
for iota := 0;    iota  →  iota.loopvar   →    iota.loopvar  → accepted
```

## Examples

| Written form | Go | gno on master | gno at 0558015ac |
|---|---|---|---|
| `iota := 5` | runs | internal panic | rejected, clear message |
| `for iota := range s` | runs | internal panic | rejected, clear message |
| `switch iota := x.(type)` | runs | internal panic | rejected, clear message |
| `func f(iota int) int { return iota }` | runs | internal panic | rejected, clear message |
| `func f(iota int) { println("hi") }` | runs | runs | rejected |
| `func f() (iota int)` | runs | runs | rejected |
| `func (iota T) M()` | runs | runs | rejected |
| `for iota := 0; iota < 2; iota++` | runs | runs | runs |
| `func f(len int) int { return len }` | runs | runs | runs |
| `type T struct{ iota int }` | runs | runs | runs |

## Glossary

- preprocess: the static pass (`PredefineFileSet`/`initStaticBlocks`) that resolves names, types, and blocks before execution.
- uverse: the GnoVM's universe block, the outermost scope holding built-in types, values, and functions.
- filetest: a `*_filetest.gno` file executed by the VM and asserted against golden directives (`// Output:`, `// Error:`, ...).
- type-check: go/types-based validation of gno source (`TypeCheckMemPackage`), distinct from preprocessing.
- loopvar rename: `initStaticBlocks1`'s pass renaming names declared in a three-clause `for` init to `<name>.loopvar`, so later preprocessing sees the suffixed name.

## Fix

The guard sits at the top of [`Reserve`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/nodes.go#L2322-L2324) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2322-L2324), before the existing-name lookup, so it fires for every `Reserve` call site: import alias, `var`/`const`/`type`/`func` declaration, receiver, parameter, named result, type-switch guard, short variable declaration, and `range` key/value. `iota` itself is registered into uverse through [`def("iota", undefined)`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/uverse.go#L761) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/uverse.go#L761), which reaches `Define2` without going through `Reserve`, so the builtin registration is not self-rejecting. The message matches the two pre-existing sites at [preprocess.go:5363](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L5361-L5364) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L5361-L5364) and [preprocess.go:5869](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L5867-L5869) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L5867-L5869), which both gate on [`isUverseName`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/misc.go#L175-L178) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/misc.go#L175-L178) rather than on `iota` alone.

## Critical (must fix)

None.

## Warnings (should fix)

- **[rule the compiler states but does not enforce]** `gnovm/pkg/gnolang/nodes.go:2318-2324` — a three-clause `for` init still binds `iota`, so `for iota := 0; iota < 2; iota++` compiles and runs while `for iota := range s` is rejected.
  <details><summary>details</summary>

  [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L284-L300) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L284-L300) rewrites names declared in a `ForStmt` init to `<name>.loopvar` and [rewrites every body reference to match](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L350-L366) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L350-L366), and it runs before [`Reserve`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/nodes.go#L2322-L2324) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2322-L2324) is ever called. `Reserve` sees `iota.loopvar`, the equality test fails, and the binding goes through. The body references were rewritten too, so [`case iotaIdentifier:`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L1298) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L1298) does not catch it either. `for iota := 0; iota < 2; iota++ { println(iota) }` prints `0` then `1` at this head, same as on master and same as Go, while every neighbouring form errors. The comment on the guard enumerates the binding sites it covers and this one is absent from both the list and the filetests. [repro](comment_claude-opus-4-8.md), [ready-to-add filetest](tests/iota_identifier_forinit.gno). Fix: reject the name before the loopvar rename consumes it, or compare against the base name in `Reserve`.
  </details>

- **[upgrade can brick a node that boots today]** `gnovm/pkg/gnolang/nodes.go:2322-2324` — three binding forms that run on master are rejected here, and node startup re-preprocesses every stored package, so a package already on chain that uses one of them fails at boot rather than at its next call.
  <details><summary>details</summary>

  `func f(iota int) { println("hi") }`, `func f() (iota int)` and `func (iota T) M()` all run on master: the name is bound but never referenced, so nothing ever reaches the `iota` branch in the preprocessor. At this head `Reserve` rejects all three at the declaration. Reverting the guard restores each one, see [repro](comment_claude-opus-4-8.md). The forms that already panicked on master could never have been deployed, so the exposure is exactly these three. [`VMKeeper.Initialize`](https://github.com/gnolang/gno/blob/0558015ac/gno.land/pkg/sdk/vm/keeper.go#L168) · [↗](../../../../../.worktrees/gno-review-5981/gno.land/pkg/sdk/vm/keeper.go#L168) calls [`PreprocessAllFilesAndSaveBlockNodes`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/machine.go#L328-L364) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/machine.go#L328-L364), which iterates every stored mem package and calls `Preprocess` on each file with no per-package recover. Fix: confirm with maintainers that narrowing what the VM accepts is acceptable for chains that are not reset, or note the requirement in the PR description.
  </details>

## Nits

- **[message promises a rule that is not the rule]** `gnovm/pkg/gnolang/nodes.go:2323` — the text says builtin identifiers cannot be shadowed, but every other uverse name is still accepted as a parameter name.
  <details><summary>details</summary>

  All 39 user-spellable names in the universe block were run as a parameter name at this head: 38 compile and only `iota` is rejected. The set is every `def` and `defNative` entry in [`makeUverseNode`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/uverse.go#L746) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/uverse.go#L746), which is what [`isUverseName`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/misc.go#L175-L178) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/misc.go#L175-L178) reads, minus the dot-prefixed internals user source cannot spell. gno rejects every uverse name for a `type` declaration, a short variable declaration and a `var` declaration, which is what [`TestBuiltinIdentifiersShadowing`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/gno_test.go#L43-L125) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/gno_test.go#L43-L125) covers. An author who reads the new message on a parameter named `iota` will conclude the same holds for `len`. Fix: name `iota` in the message.
  </details>

- **[green CI on a formatting-only failure]** `gnovm/tests/files/iota_identifier_param.gno:13` — four of the new filetests end with a trailing blank line, which is what turns `main / build` red.
  <details><summary>details</summary>

  The [failing job](https://github.com/gnolang/gno/actions/runs/30069639090/job/89407628624) is the formatting check, and its diff removes one blank line from each of `iota_identifier_param.gno`, `iota_identifier_recv.gno`, `iota_identifier_result.gno` and `iota_identifier_typeswitch.gno`. Fix: run `make fmt`.
  </details>

## Missing Tests

- **[the uncovered form is the common one]** `gnovm/tests/files/iota_identifier_range.gno:5` — no filetest covers a three-clause `for` init, which is the one binding form the guard misses.
  <details><summary>details</summary>

  The six new filetests cover short variable declaration, parameter, named result, receiver, `range` key and type-switch guard. `for iota := 0; iota < 2; iota++` is absent, and it is the form that still compiles. A filetest here is what stops the gap from reappearing after any future change to the loopvar rename. Ready to add: [`tests/iota_identifier_forinit.gno`](tests/iota_identifier_forinit.gno), which fails at 0558015ac by printing `0` and `1` instead of erroring.
  </details>

## Suggestions

- **[two spellings of one rule drift apart]** `gnovm/pkg/gnolang/nodes.go:2322` — the guard hardcodes `iota` while the two sibling checks for the same message gate on `isUverseName`.
  <details><summary>details</summary>

  [preprocess.go:5361-5364](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L5361-L5364) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L5361-L5364) and [preprocess.go:5867-5869](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L5867-L5869) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L5867-L5869) raise the identical string through [`isUverseName`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/misc.go#L175-L178) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/misc.go#L175-L178). Widening the new guard to `isUverseName` would be a much larger change than this PR intends: it would reject `func f(len int)`, which compiles in Go and in gno today. Keeping the narrow test is the right call for this PR; the note is that a reader comparing the three sites has no way to tell the narrow one is deliberate. Not posted, no change needed.
  </details>

## Verified

- The guard is what rejects the three previously-running forms: restoring `gnovm/pkg/gnolang/nodes.go` from the merge base makes `func f(iota int) { println("hi") }` print `hi` again, and reinstating it makes the same program error at the declaration.
- `for iota := 0; iota < 2; iota++ { println(iota) }` runs to completion at 0558015ac, printing `0` and `1`; a filetest asserting the shadowing error fails with `unexpected output`.
- gno matches Go for every form the guard leaves alone: nested `const` groups inside a function print `0 1 0 1` in both, a struct field named `iota` and a method named `iota` both work, and 38 of the 39 user-spellable uverse names compile as parameter names in Go, on master and at this head, `iota` being the only rejection. Table asserted by [`tests/iota_go_parity_test.go`](tests/iota_go_parity_test.go), which type-checks each snippet with `go/types`.
- `gno lint` reports the same rejection as the VM (`code=gnoPreprocessError`) for a package whose exported function takes an `iota` parameter, so the tool and the VM agree even though `go/types` accepts the source.
- Error text and location are unchanged from master for every form master already rejected: package-level and local `var iota`, local `const iota`, `type iota` at both scopes, `func iota()`, and `import iota "strings"`.
- Run at 0558015ac: the six new filetests and `TestBuiltinIdentifiersShadowing` pass, and `go test ./gnovm/pkg/gnolang/...` is green apart from five filetests (`redeclaration3`, `redeclaration4`, `type41`, `types/and_f0`, `types/or_f0`) that fail identically at the merge base d14a03770; those are `go/types` message drift from a newer local Go, not this diff.

## Existing threads

None.

## Open questions

- The user-facing error for a `const` initialised from a `for`-init variable leaks the internal name: `iota.loopvar<VPBlock(1,0)> (variable of type int) is not constant`. Identical on master, so it belongs to the loopvar rename rather than to this PR; not posted.
- The new filetests carry inline `// ERROR "..."` comments, which the filetest harness does not read; the assertion is the trailing `// Error:` block. Two pre-existing `iota_outside_const*.gno` files use the same inert form, so it is house style rather than a defect; not posted.
