# PR [#6078](https://github.com/gnolang/gno/pull/6078): feat(gnovm): reject compiler and tooling directives in submitted packages

URL: https://github.com/gnolang/gno/pull/6078
Author: omarsy | Base: master | Files: 12 | +1109 -1
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: f30a4000b (latest)
Open the code: [github.dev](https://github.dev/omarsy/gno/tree/f30a4000b) · [vscode.dev](https://vscode.dev/github/omarsy/gno/tree/f30a4000b)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6078-head f30a4000b`

Round 2. The head advanced six commits from b0913fc73, the sha round 1 targeted: lint now takes its file list from the package instead of globbing the package root, `//go:generate` hidden from the token scanner is caught by a raw line scan, and the commit that moved preprocess gas ahead of validation was reverted after its premise turned out wrong. This round reads f30a4000b whole and carries no finding forward.

## Overview

A `.gno` file can carry a Go directive, `//go:build ignore` or `//line ../../etc/passwd:9999:1` or `//go:generate rm -rf /`, and the VM compiles the file exactly the same either way. Gno has one build target, no conditional compilation, and `gnomod.toml` carries the language version, so none of these lines change what the code does on chain. They change what happens around it: [`go/parser` honours `//line`](https://github.com/golang/go/blob/go1.25.9/src/go/scanner/scanner.go#L230) and rewrites the file and line a failed transaction reports, `gno tool transpile` emits its own `//go:build` header beside a stored one and the result stops compiling, and `go generate` over transpiled chain source runs whatever a `//go:generate` line names. Anyone auditing the source reads a file marked `ignore` as excluded, when the chain runs it unconditionally.

This change refuses the whole class at the door. [`FindDirectiveComment`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L1375) scans a submitted file for build constraints, both spellings, line directives, both the `//` and `/* */` forms, and the `//tool:name` pragmas, and [`ValidateMemPackageAny`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L1196) rejects the package when it finds one. That validator is what every chain entry point already calls, so `MsgAddPackage`, `MsgRun` and genesis are covered by one check. `gno lint` runs the same predicate over the same file list, so a developer sees the refusal before they spend a transaction on it. `//nolint` is the one exception. Stdlibs, which ship inside the node binary and carry Go's own directives, and the VM filetests, which deliberately pin that constraints are inert, are out of scope by package type.

**Verdict: APPROVE** — the rule is enforced where consensus reads it and the two escapes worth worrying about, the block form of a line directive and a `//go:generate` hiding from the token scanner inside a raw string, are both closed and pinned by tests (4 nits, 1 suggestion).

## Verify first

- [`gnovm/pkg/gnolang/mempackage.go:1280`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L1280) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/pkg/gnolang/mempackage.go#L1280) — the check runs only when `mptype.IsUserlib()`, so a submitter who could choose the type would skip it. Confirm every entry point sets the type itself: [`keeper.go:584`](https://github.com/gnolang/gno/blob/f30a4000b/gno.land/pkg/sdk/vm/keeper.go#L584) for `AddPackage`, [`keeper.go:1065`](https://github.com/gnolang/gno/blob/f30a4000b/gno.land/pkg/sdk/vm/keeper.go#L1065) for `Run`, both before the validate call.
- [`gnovm/adr/pr6078_forbid_directives.md:196`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/adr/pr6078_forbid_directives.md?plain=1#L196) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/adr/pr6078_forbid_directives.md#L196) — a package already stored on a live network with a directive line stops replaying under a node running this change. `examples/` is clean and CI proves it, but nothing here covers what third parties deployed on the portal loop or a test network. Enumerate the deployed package paths on each target chain, fetch each with `vm/qfile`, and run `FindDirectiveComment` over the result before the upgrade is scheduled.

## Summary

The predicate scans tokens through `go/scanner` rather than raw lines, which is the same machinery `go/parser` runs, so the two agree on a leading BOM and on a `package` line hidden inside a block comment, and a string literal that merely spells a directive is never a comment token. On top of that, one raw line scan catches `//go:generate` at column 1 inside a raw string or a block comment, because [`go generate` never parses](https://github.com/golang/go/blob/go1.25.9/src/cmd/go/internal/generate/generate.go#L360) and would run it. The rule mirroring [`go/ast.isDirective`](https://github.com/golang/go/blob/go1.25.9/src/go/ast/ast.go#L166) is copied rather than called, since a package's acceptance on chain must not move when the toolchain's own copy does, and a test compares the copy against Go's real behaviour over 200k inputs.

The cost argument holds up. Validation now reads every `.gno` file end to end where it used to stop at the package clause, and it does that before the preprocess charge. `CheckTx` never reaches it: [`baseapp.go:712`](https://github.com/gnolang/gno/blob/f30a4000b/tm2/pkg/sdk/baseapp.go#L712) skips `handler.Process` outside `DeliverTx`, so the scan only ever runs on a transaction the ante handler has already charged `TxSizeCostPerByte` for. The reverted commit's premise, that repeated `CheckTx` buys unpaid validator CPU, was wrong for exactly this reason, and the ADR records both the measurement and the prefilter that would buy margin back on a slower validator.

Four nits and one refactor, all local, none touching the rule itself: a false clause in the rejection message, a stdlib exemption that a symlink defeats, a doc comment that moved to the wrong function, a variable still named for the build-constraint rule, and a fifteen-line loop that `strings.SplitSeq` writes in nine.

## Nits

- **[wrong error text]** [`gnovm/pkg/gnolang/mempackage.go:1290`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L1290) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/pkg/gnolang/mempackage.go#L1284)— a rejected file also reports its package name as missing, which is false.
  <details><summary>details</summary>

  The directive branch appends its error and then `continue`s, skipping `PackageNameFromFileBody` for that file. `pkgNameFound` therefore stays false, and the tail of the validator appends a second error. A one-file package named `tagged` is refused with `invalid file "tagged.gno": directives are not supported: "//go:build"; package name "tagged" not found in files`, and the name is in the file. Fix: reject on the first directive before the per-file loop starts, so the directive error is the only one a submitter reads.
  </details>

- **[stdlib exemption]** [`gnovm/cmd/gno/lint.go:82`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/cmd/gno/lint.go#L82) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/cmd/gno/lint.go#L82)— a stdlib directory reached through a symlink does not match the root, so `gno lint` reports Go's own directives as errors.
  <details><summary>details</summary>

  `filepath.Abs` resolves a relative path against the working directory and stops there, leaving any symlink in it intact, so `abs` and `root` are two spellings of one directory and compare unequal. `gnoenv.RootDir` guesses the root from `go list` or the caller stack, both of which give the resolved path, while the directory argument comes from the shell's own view of the working directory. Linting `gnovm/stdlibs/math/bits` through a symlink with `GNOROOT` set to the resolved path prints `directives are not supported: "//go:generate"` and exits 1. Fix: compare the paths with the symlinks resolved, per the patch in [`tests/isstdlibdir_symlink_fix.go.txt`](tests/isstdlibdir_symlink_fix.go.txt), which clears the repro and leaves `TestLintApp` green.
  </details>

- **[doc comment]** [`gnovm/pkg/gnolang/mempackage.go:780`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L780) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/pkg/gnolang/mempackage.go#L780)— `MemPackageFilePaths` inherited the fourteen lines of doc that describe `ReadMemPackage`.
  <details><summary>details</summary>

  The extraction left the original comment block in place and appended the new function's own paragraphs under it, so godoc for `MemPackageFilePaths` opens with "ReadMemPackage initializes a new MemPackage by reading the OS directory at dir", and [`ReadMemPackage`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L852) keeps a one-line stub in its place. Fix: move the inherited block back above `ReadMemPackage`.
  </details>

- **[stale name]** [`gnovm/cmd/gno/lint.go:228`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/cmd/gno/lint.go#L228) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/cmd/gno/lint.go#L228)— `tagged` names the build-constraint rule the branch started as, not the directive rule it ships.
  <details><summary>details</summary>

  The variable records that some file in the package carried a directive of any kind, and reads as a build tag. `hasDirective` says what the three uses at [lint.go:228](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/cmd/gno/lint.go#L228), [252](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/cmd/gno/lint.go#L253) and [254](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/cmd/gno/lint.go#L255) test. Naming is not enforced by any linter in [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/f30a4000b/.github/golangci.yml?plain=1#L13), so this stays here and is not posted.
  </details>

## Suggestions

- **[Refactor]** [`gnovm/pkg/gnolang/mempackage.go:1347`](https://github.com/gnolang/gno/blob/f30a4000b/gnovm/pkg/gnolang/mempackage.go#L1347) · [↗](../../../../../.worktrees/gno-review-6078-head/gnovm/pkg/gnolang/mempackage.go#L1341)— the hand-rolled offset walk over lines is `strings.SplitSeq`, 15 lines to 9.
  <details><summary>details</summary>

  The loop slices the body, tests two prefixes, finds the next newline and advances past it, which is what `strings.SplitSeq(body, "\n")` yields, already used in 12 places in the tree. The iterator form allocates nothing, so the reason the loop was written by hand does not survive:

  ```go
  func hasRawGoGenerate(body string) bool {
  	for line := range strings.SplitSeq(body, "\n") {
  		if strings.HasPrefix(line, "//go:generate ") ||
  			strings.HasPrefix(line, "//go:generate\t") {
  			return true
  		}
  	}
  	return false
  }
  ```

  The two forms agree on 200k generated bodies built from the fragments that separate them, both separators, the near-miss prefixes, indentation, CRLF endings and a final line with no newline: [`tests/hasrawgogenerate_equiv_test.go`](tests/hasrawgogenerate_equiv_test.go).
  </details>

## Verified

- `gno run` and `gno test` on a local package whose file opens with `//go:build ignore` both still work, printing `hi` and `[no test files]`; the refusal is confined to `gno lint` and the chain entry points, so the rule does not reach a developer's scratch file.
- Applying the `isStdlibDir` patch above clears the symlink repro and leaves `TestLintApp` green; reverting it brings the `gnoDirectiveError` on `bits.gno` back.
- Green at f30a4000b: `TestFindDirectiveComment`, `TestIsDirectiveTextMirrorsGo`, `TestFindDirectiveComment_Terminates`, `TestValidateMemPackage_Directives`, `TestLintApp`, `TestMemPackageFilePathsMatchesReadMemPackage`, and `TestTestdata/addpkg_directives`.

## Open questions

- The `//nolint` exception is the one entry whose justification the ADR itself calls weaker than the rest, and it points against the ordering argument the rest of the rule rests on: dropping it now is restorable, removing it later is a consensus break. Not posted, because the ADR already argues both sides and the call belongs to whoever schedules the upgrade.
- `hasRawGoGenerate` runs before the token scan, so a file carrying both a header constraint and a `//go:generate` inside a raw string is refused naming the `//go:generate`. The refusal is right either way and the submitter has one directive to remove per attempt. Not posted, no change needed.
