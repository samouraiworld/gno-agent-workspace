# PR [#5981](https://github.com/gnolang/gno/pull/5981): fix(preprocess): avoid shadowing of iota

URL: https://github.com/gnolang/gno/pull/5981
Author: Villaquiranm | Base: master | Files: 9 | +96 -1
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 055d85cbc (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5981 055d85cbc`

Round 3. The head advanced af3accbce → 055d85cbc and the patch-ids differ, so this is a full round. Since round 2 the branch gained a second guard in [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L299-L304) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L299-L304) that rejects `iota` in a three-clause `for` init before the `.loopvar` rename hides it, a filetest for that form, an `iotaIdentifier` constant replacing two string literals, and the `make fmt` pass that cleared `main / build`. Round 2's `for`-init Warning, formatting Nit and Missing test are resolved. The compatibility Warning and the error-message Nit carry, and the compatibility Warning now covers a fourth binding form.

## Overview

`iota` is the counter that steps through a `const` block. Go also lets the same word be an ordinary variable, parameter or field name anywhere else, and gno half-allowed that: some spellings ran, some died with an internal message naming no fix, and which was which depended on whether the name was ever read. This branch picks one answer and applies it at every site that binds a name, rejecting `iota` at the declaration with the wording gno already uses for `var iota`. Two checks do the rejecting rather than one, because a three-clause `for` init is renamed by an earlier pass before the main check can see it.

**Verdict: NEEDS DISCUSSION** — every binding form now rejects consistently and round 2's `for`-init hole is closed, so what is left is a maintainer call: nine forms that compile and run on master stop compiling here, and node startup re-preprocesses every stored package with no per-package recover (1 Warning, 2 Nits, 1 Suggestion).

## Verify first

- [`gnovm/pkg/gnolang/preprocess.go:302`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L302) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L302) is the one rejection path that never reaches [`Reserve`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/nodes.go#L2325) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2325). Delete it and `go test -run 'TestFiles/iota_identifier_forinit.gno$' ./gnovm/pkg/gnolang/` prints `0` and `1` instead of erroring.
- [`gnovm/pkg/gnolang/nodes.go:2325`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/nodes.go#L2325) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2325) decides what the VM stops accepting. Run [`tests/iota_binding_sweep.sh`](tests/iota_binding_sweep.sh) and read the nine rows that go `RUNS` at the merge base and `REJECT` at the head.

## Summary

Since [PR 5822](https://github.com/gnolang/gno/pull/5822), naming something `iota` outside a const declaration reaches [`case iotaIdentifier:`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L1304) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L1304) and panics with `cannot use iota outside constant declaration`, an internal message that names no fix, but only when the name is read; bind it and never read it and the program runs. [Issue 5876](https://github.com/gnolang/gno/issues/5876) asks for one consistent compile-time rejection instead, and states the divergence from Go outright. The branch delivers it in two places: [`StaticBlock.Reserve`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/nodes.go#L2320-L2327) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2320-L2327) covers every name-binding site the preprocessor reserves, and a second guard inside [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L299-L304) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L299-L304) covers the three-clause `for` init, whose names [`initStaticBlocks1` renames to `<name>.loopvar`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L305) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L305) before [`Reserve` runs in the second pass](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L182-L183) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L182-L183).

## Examples

Measured by [`tests/iota_binding_sweep.sh`](tests/iota_binding_sweep.sh) over 30 forms; the Go column by [`tests/iota_go_parity.go`](tests/iota_go_parity.go).

| Written form | Go | gno on master | gno at 055d85cbc |
|---|---|---|---|
| `iota := 5` | runs | internal panic | rejected, clear message |
| `for iota := range s` | runs | internal panic | rejected, clear message |
| `switch iota := x.(type)` | runs | internal panic | rejected, clear message |
| `if iota := 0; iota == 0` | runs | internal panic | rejected, clear message |
| `func f(iota int) int { return iota }` | runs | internal panic | rejected, clear message |
| `func f(iota int) { println("hi") }` | runs | runs | rejected |
| `func f() (iota int)` | runs | runs | rejected |
| `func (iota T) M()` | runs | runs | rejected |
| `func(iota int) {}` as a closure | runs | runs | rejected |
| `for iota := 0; iota < 2; iota++` | runs | runs | rejected |
| `func f(len int) int { return len }` | runs | runs | runs |
| `type T struct{ iota int }` | runs | runs | runs |
| `func (t T) iota() int` | runs | runs | runs |
| `iota:` as a loop label | runs | runs | runs |
| `const ( a = iota )` | runs | runs | runs |

## Fix

The guard in `Reserve` sits before the existing-name lookup, so it fires for every site that reserves a name: import alias, `var`, `const`, `type` and `func` declaration, receiver, parameter, named result, type-switch guard, short variable declaration, `if` and `switch` init, and `range` key and value. `iota` itself is registered into uverse through [`def("iota", undefined)`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/uverse.go#L761) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/uverse.go#L761), which reaches `Define2` without passing through `Reserve`, so the builtin registration is not self-rejecting. The three-clause `for` init is the one binding form that never reaches `Reserve` under its own name, which is why the branch carries a second guard at the rename site. The message matches the two pre-existing sites at [preprocess.go:5367-5370](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L5367-L5370) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L5367-L5370) and [preprocess.go:5873-5875](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L5873-L5875) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L5873-L5875), which both gate on [`isUverseName`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/misc.go#L175-L178) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/misc.go#L175-L178) rather than on `iota` alone.

## Warnings (should fix)

- **[upgrade can brick a node that boots today]** `gnovm/pkg/gnolang/nodes.go:2325` — nine forms that compile and run on master are rejected here, and node startup re-preprocesses every stored package, so a package already on chain that uses one fails at boot rather than at its next call.
  <details><summary>details</summary>

  [`tests/iota_binding_sweep.sh`](tests/iota_binding_sweep.sh) runs 30 binding forms against the merge base and the head; nine go from `RUNS` to `REJECT`. They fall into four classes: an unreferenced parameter, whether on a top-level function or a closure and whether first or later in the list; a named result; a receiver; and a three-clause `for` init, referenced or not. The first three run on master because the name is bound and never reaches [`case iotaIdentifier:`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L1304) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L1304); the fourth runs because the `.loopvar` rename [rewrites the body references](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L372) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L372) to match. [`VMKeeper.Initialize`](https://github.com/gnolang/gno/blob/055d85cbc/gno.land/pkg/sdk/vm/keeper.go#L162) · [↗](../../../../../.worktrees/gno-review-5981/gno.land/pkg/sdk/vm/keeper.go#L162) calls [`PreprocessAllFilesAndSaveBlockNodes`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/machine.go#L356-L361) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/machine.go#L356-L361), which iterates every stored mem package and calls `Preprocess` on each file with no per-package recover. Nothing under `examples/` or `gnovm/stdlibs/` binds `iota`, so the exposure is third-party packages already deployed. Raised in round 2 as [thread r3660033767](https://github.com/gnolang/gno/pull/5981#discussion_r3660033767), whose posted text still says three forms and needs the fourth added. Fix: confirm with maintainers that narrowing what the VM accepts is acceptable for chains that are not reset, or say so in the PR description.
  </details>

## Nits

- **[message promises a rule that is not the rule]** `gnovm/pkg/gnolang/nodes.go:2326` — the text says builtin identifiers cannot be shadowed, and every other uverse name is still accepted as a parameter name.
  <details><summary>details</summary>

  `func f(len int) int { return len }` compiles and prints `3` at this head, and round 2 measured the same for 38 of the 39 user-spellable names in [`makeUverseNode`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/uverse.go#L746) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/uverse.go#L746), which is the set [`isUverseName`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/misc.go#L175-L178) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/misc.go#L175-L178) reads. An author who meets the new message on a parameter named `iota` will conclude the same holds for `len`. Raised in round 2 as [thread r3660033770](https://github.com/gnolang/gno/pull/5981#discussion_r3660033770) and unchanged at this head, so nothing new is posted. Fix: name `iota` in the message.
  </details>

- **[the constant leaves its own definition site spelling the string]** `gnovm/pkg/gnolang/preprocess.go:20` — `def("iota", undefined)` is the literal this constant should have replaced.
  <details><summary>details</summary>

  After this diff the only remaining `"iota"` literal in the package is [`def("iota", undefined)`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/uverse.go#L761) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/uverse.go#L761), which is the line the constant is about. `blankIdentifier` beside it has the same shape and the same gap. Fix: spell it `def(iotaIdentifier, undefined)`.
  </details>

## Suggestions

- **[one guard where the branch has two]** `gnovm/pkg/gnolang/preprocess.go:299-304` — `Reserve` sees this name as `iota.loopvar`, so trimming that suffix there covers the `for` init and this block comes out.
  <details><summary>details</summary>

  `Reserve` already sees the `for`-init name, as `iota.loopvar`. Trimming that suffix before the equality test makes the existing guard fire on it, which removes the six-line block at the rename site and leaves one place to read and one to keep correct. [`tests/single_guard.patch`](tests/single_guard.patch) is the change, net seven lines shorter. Applied at 055d85cbc it leaves all 30 rows of [`tests/iota_binding_sweep.sh`](tests/iota_binding_sweep.sh) identical and the whole `TestFiles` suite green except one golden line: the error on a `for` init moves from the whole statement, `4:2-6:3`, to its init clause, `4:6-15`, which is where the same error already points for `iota := 5`. Fix: apply the patch and update that line in [`iota_identifier_forinit.gno:10`](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/tests/files/iota_identifier_forinit.gno#L10) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/tests/files/iota_identifier_forinit.gno#L10).
  </details>

## Verified

- The `for`-init guard is what closes round 2's hole: deleting the six-line block at [preprocess.go:299-304](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L299-L304) · [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/preprocess.go#L299-L304) makes `iota_identifier_forinit.gno` print `0` and `1` again, and restoring it makes the same file error at the declaration.
- `gno lint` reports both rejection paths as `gnoPreprocessError` carrying the file and the range, not a raw panic: `gno.land/p/demo/forinit/a.gno:4:2-6:3: builtin identifiers cannot be shadowed: iota` for the `for` init and `gno.land/p/demo/param/a.gno:3:1-5:2:` for the parameter. The second pass panics from outside `Reserve`, so the two paths reaching the same diagnostic is not automatic.
- Go compiles and runs all nine newly-rejected forms, asserted by [`tests/iota_go_parity.go`](tests/iota_go_parity.go).
- Both packages in the `gno lint` check above are accepted at the merge base with no diagnostic, which is what makes them deployable today.
- Run at 055d85cbc: the seven new filetests pass, and `go test -run 'TestFiles$' ./gnovm/pkg/gnolang/` is green over the whole suite in 188s, with none of the `go/types` message drift round 2 saw.
- No `.gno` source under `examples/`, `gnovm/stdlibs/` or `gno.land/` binds `iota` as an identifier.

## Existing threads

- [r3660033759](https://github.com/gnolang/gno/pull/5981#discussion_r3660033759), the round 2 `for`-init Warning, is fixed by this head and left for the author to resolve. Its text stays true of the `Reserve` line it anchors, which is why it reads as live.
- [r3660033767](https://github.com/gnolang/gno/pull/5981#discussion_r3660033767), the compatibility Warning, is open and its posted count of three forms is now four.
- [r3660033770](https://github.com/gnolang/gno/pull/5981#discussion_r3660033770), the message Nit, is open and unchanged.

## Open questions

- A label named `iota` still compiles, since a label is not a block name and never reaches `Reserve`. It is the one spelling left that a reader of the error message would expect to be rejected; too marginal to post.
- The new filetests carry inline `// ERROR "..."` comments the harness does not read; the assertion is the trailing `// Error:` block. Two pre-existing `iota_outside_const*.gno` files use the same inert form, so it is house style rather than a defect; not posted.
