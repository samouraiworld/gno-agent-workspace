# PR [#5978](https://github.com/gnolang/gno/pull/5978): fix(gnovm): pin the per-file Go version in the consensus type-check

URL: https://github.com/gnolang/gno/pull/5978
Author: davd-gzl | Base: master | Files: 4 | +139 -11
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: da74644bf (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5978 da74644bf`

**TL;DR:** A `.gno` file can carry a `//go:build go1.N` comment, and Go's type checker obeys that line instead of the Go version the chain pins. This PR erases the line's effect right after parsing, so the uploader can no longer pick the rules their own package is judged by and the verdict stops depending on which Go release a node was built with.

**Verdict: APPROVE** — the fix is correct, minimal, and proven at both ends; one coverage gap and two nits (1 Missing test, 2 Nits, 2 Suggestions).

## Summary

`TypeCheckMemPackage` sets [`GoVersion: "go1.18"`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L190) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L190) on its `types.Config` so a submitted package gets the same verdict on every node. That pin governs only files with no version of their own: `go/parser` fills `ast.File.GoVersion` from any `//go:build go1.N` line sitting before the package clause, as [gno's vendored copy of the same parser shows](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/parser/parser.go#L153-L157) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/parser/parser.go#L153-L157), and `go/types` then uses `max(fileVersion, go1.21)` for that file and errors `file requires newer Go version goX (application built with goY)` when the file version exceeds the compiling toolchain's (`go/types/check.go:394-423` in Go 1.26). Package bodies arrive from `addpkg` and `maketx run`, so both halves were reachable from a transaction. The fix blanks the field on every parsed `.gno` file in [`GoParseMemPackage`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L645) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L645), the single parse entry point feeding the type-check.

```
                 master                                 this PR
  //go:build go1.N                        //go:build go1.N
        |                                       |
   go/parser  -> ast.File.GoVersion = go1.N   go/parser -> GoVersion = go1.N
        |                                       |
        |                                  gof.GoVersion = ""     <- the fix
        |                                       |
   go/types: v = max(go1.N, go1.21)        go/types: v = Config.GoVersion
             and reject if go1.N > builder                = go1.18
```

## Examples

Measured with `TypeCheckMemPackage(MPUserProd, TCLatestRelaxed)` on a binary built with Go 1.26.5.

| file source | master | this PR |
|---|---|---|
| `for range 10` | reject | reject |
| `//go:build go1.22` + `for range 10` | accept | reject |
| `//go:build go1.21` + `min(1, 2)` | accept | reject |
| `//go:build go1.16` + `min(1, 2)` | accept | reject |
| `//go:build go1.23` + range-over-func | accept | reject |
| `//go:build go1.26` + trivial body | accept | accept |
| `//go:build go1.27` + trivial body | reject | accept |
| imports a dependency tagged `//go:build go1.22` | accept | reject |
| `// +build go1.22` (old style) + `for range 10` | reject | reject |
| `//go:build ignore` + `for range 10` | reject | reject |

An older tag raises the file too: `go/types` takes `max(fileVersion, go1.21)`, so `//go:build go1.16` unlocked the go1.21 builtins.

## Glossary

- type-check: go/types-based validation of gno source (`TypeCheckMemPackage`), distinct from preprocessing.
- preprocess: the static pass that resolves names, types, and blocks before execution.
- MemPackage: in-memory set of a package's source files, the unit loaded, type-checked, and run.
- addpkg: the transaction (`maketx addpkg`) that uploads a package or realm to the chain.
- uverse: the GnoVM's universe block, the outermost scope holding built-in types, values, and functions.
- Store: the backing store for packages and objects (`gno.Store`/`defaultStore`).
- transactionStore: the per-transaction Store wrapper (struct `{*defaultStore}`), whose methods promote to the embedded `*defaultStore`.
- filetest: a `*_filetest.gno` file executed by the VM and asserted against golden directives.

## Fix

Before, an `*ast.File` returned by [`GoParseMemPackage`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L587) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L587) carried whatever version its build line declared, and `types.Config.GoVersion` applied only to the files that declared none. After, the field is blanked on every file before it reaches the checker, so `Config.GoVersion` is the only version input for the submitted package and for every package it imports. The load-bearing constraint is that this is the sole parse path into the checker: [`typeCheckMemPackage` at line 440](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L440) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L440) is its only non-test caller, and it recurses through itself for imports, so one assignment covers the whole graph.

The second commit deletes the `mpkg == nil` skip in [`PreprocessAllFilesAndSaveBlockNodes`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/machine.go#L328-L333) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/machine.go#L328-L333). The skip is unreachable: the producer already drops prod-less packages [before its only channel send](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/store.go#L1275-L1282) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/store.go#L1275-L1282), and the [`Store` interface documents that drop](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/store.go#L88-L92) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/store.go#L88-L92).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[exported function mutates what it returns, silently]** `gnovm/pkg/gnolang/gotypecheck.go:645` — [`GoParseMemPackage`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L587) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L587) is exported, and its [doc comment](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L580-L586) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L580-L586) lists what comes back without saying every returned file has had its Go version cleared.
  <details><summary>details</summary>

  A caller reading godoc gets ASTs that no longer match the source they were parsed from. The [inline note at the assignment](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L640-L645) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L640-L645) reaches only someone reading the body. Fix: say in the doc comment that the per-file Go version is cleared, so `Config.GoVersion` stays the sole authority.
  </details>

- **[invariant note sits where the guard used to be]** `gnovm/pkg/gnolang/machine.go:331-332` — the "never yields nil" comment describes the channel but sits inside the loop body, one line before an unrelated statement. Reads better above the `for`. Not posted, cosmetic.

## Missing Tests

- **[same vector one hop away, untested]** `gnovm/pkg/gnolang/gotypecheck_buildtag_test.go:44-61` — a `//go:build go1.N` line inside an imported package raises that import's own version too, and [no case](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L44-L61) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L44-L61) covers it.
  <details><summary>details</summary>

  `GoParseMemPackage` runs for imports as well, through the [`typeCheckMemPackage` recursion](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L440) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L440), so a dependency carrying `//go:build go1.22` raised its own version the same way. Measured: a package importing a dependency whose file starts with `//go:build go1.22` and contains `for range 10` type-checks clean without the blanking and is rejected with it. Fix: add the import case, shipped as [`tests/buildtag_import_test.go`](tests/buildtag_import_test.go).
  </details>

## Suggestions

- **[two tests for one invariant, in two files]** `gnovm/pkg/gnolang/gotypecheck_buildtag_test.go:41-42` — the [precondition](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L41-L42) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L41-L42) restates what [`TestTypeCheckMemPackage_GoVersionPinned`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck_test.go#L469-L481) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck_test.go#L469-L481) already asserts. Not posted, low gain.
  <details><summary>details</summary>

  Both assert that range-over-int is rejected and that the message names `go1.22`. The two tests guard the same pin from opposite sides, so keeping them adjacent in `gotypecheck_test.go` would make a future edit to the pin land in one place.
  </details>

- **[an older tag raises the file too]** `gnovm/pkg/gnolang/gotypecheck_buildtag_test.go:44-61` — every case uses a tag above the go1.18 pin, but `go/types` applies `max(fileVersion, go1.21)`, so `//go:build go1.0` also unlocked the go1.21 builtins. Not posted; the fix already covers it and the case only documents the rule.

## Verified

- Removing the fix reproduces both halves. Replacing [`gof.GoVersion = ""`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L645) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L645) with a comment makes all four assertions of [the new test](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L36-L78) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L36-L78) fail, three with "An error is expected but got nil" and one with `file requires newer Go version go1.99 (application built with go1.26)`. Restoring the line turns them green.
- The rejection boundary sits exactly at the compiling toolchain. Without the fix, on a binary built with Go 1.26.5, `//go:build go1.26` is accepted and `//go:build go1.27` is rejected as `application built with go1.26`. `go/types` derives that bound from the compiler itself (`go_current = asGoVersion(fmt.Sprintf("go1.%d", goversion.Version))`, `go/types/version.go:49` in Go 1.26), and [`go.mod` declares `go 1.25.9`](https://github.com/gnolang/gno/blob/da74644bf/go.mod#L3) · [↗](../../../../../.worktrees/gno-review-5978/go.mod#L3), so a node built with Go 1.25 and a node built with Go 1.26 disagreed on `//go:build go1.26`.
- The tag reaches imports too. A package importing a dependency whose file starts with `//go:build go1.22` and uses range-over-int is accepted without the fix and rejected with it, so the blanking covers the whole import graph, not just the submitted package.
- No effect on the existing corpus. The three `.gno` files in the tree carrying a build line ([build0.gno](https://github.com/gnolang/gno/blob/da74644bf/gnovm/tests/files/build0.gno#L3) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/tests/files/build0.gno#L3), [ct2.gno](https://github.com/gnolang/gno/blob/da74644bf/gnovm/tests/files/extern/ct/ct2.gno#L1) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/tests/files/extern/ct/ct2.gno#L1), [ct3.gno](https://github.com/gnolang/gno/blob/da74644bf/gnovm/tests/files/extern/ct/ct3.gno#L1) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/tests/files/extern/ct/ct3.gno#L1)) each parse to an empty `GoVersion` with and without the fix, because none of the three expressions yields a Go version. `TestFiles/build0.gno` passes at the reviewed commit, and the two `ct` files type-check as one package with a nil error both ways.
- The deleted guard is dead. A store holding a prod package, a test-only package, and a second prod package reports three indexed packages and a nil `GetMemPackage` for the test-only one, yet [`IterMemPackage`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/store.go#L1255) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/store.go#L1255) yields exactly the two prod packages and never nil. `defaultStore` is the only implementation of the method in the tree, and [`transactionStore`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/store.go#L288-L290) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/store.go#L288-L290) inherits it by embedding.
- The unlocked features stay unrunnable independently of this gate. Range-over-int and range-over-func are rejected by the [preprocessor's range switch](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/preprocess.go#L886-L901) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/preprocess.go#L886-L901), and `min`/`max`/`clear` appear nowhere in the [uverse builtin definitions](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/uverse.go#L754) · [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/uverse.go#L754), so the tag bought reach into the type-check only.
- Green at da74644bf: `go test ./gnovm/pkg/gnolang/ -run TestTypeCheck`, `-run 'TestStore|TestMachine|TestPreprocess|TestMemPackage'`, `go test ./gno.land/pkg/sdk/vm/ -run TestVMKeeper`, and `go test ./gnovm/pkg/gnolang/ -run Files -test.short` (ten pre-existing local-toolchain wording failures, identical to master, no new ones).

## Open questions

- The commit message and PR body call both halves state forks. Only the toolchain half diverges between honest nodes: `max(fileVersion, go1.21)` is computed identically on every build, so the acceptance-floor bypass is a uniform policy hole, not a fork. Not posted; the code and its comments are accurate, only the narrative overreaches.
- The pin closes the declared-version axis, not the toolchain axis in general: `go/types` itself changes between Go releases, which is visible right now as ten filetests whose error wording differs between a Go 1.26 build and CI's. Out of this PR's scope, worth its own look.
- The two commits are unrelated, so a bisect over a consensus fix carries a dead-code deletion along with it. Not posted; both are small and each is self-explanatory.
