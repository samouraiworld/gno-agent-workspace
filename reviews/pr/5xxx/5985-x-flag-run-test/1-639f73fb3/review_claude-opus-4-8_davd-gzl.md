# PR [#5985](https://github.com/gnolang/gno/pull/5985): gnovm: add -X flag to gno run and gno test

URL: https://github.com/gnolang/gno/pull/5985
Author: ygd58 | Base: master | Files: 6 | +458 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 639f73fb3 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5985 639f73fb3`

**TL;DR:** Adds a `-X name=value` option to `gno run` and `gno test` that replaces the text of a package-level string variable before the code runs, so you can override a default like a version string without editing the file.

**Verdict: REQUEST CHANGES** — the rewrite works and resists literal injection, but it moves reported source positions, one flag silently rewrites every package under test, and every mismatch is silent (3 Warnings, 1 Missing test, 2 Nits).

## Summary

`-X` collects `name=value` pairs into a map ([`xFlag.Set`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/xflag.go#L119-L128) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L119-L128)). Before the GnoVM parses a file, [`patchXVars`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/xflag.go#L32-L89) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L32-L89) parses it with `go/parser`, walks only the top-level declarations, replaces the string literal of any matching `var`, and re-serializes the whole file with `go/printer`. `gno run` swaps `MustReadFile` for [`mustReadAndPatchFile`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/run.go#L385-L394) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/run.go#L385-L394); `gno test` rewrites every file body of the [MemPackage](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/test.go#L449-L453) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/test.go#L449-L453) after reading it. The AST walk is the right call: it is what makes a raw string that merely contains `var myVar = "..."` immune, which a regex pass could not do. The cost is that the VM no longer sees the file on disk, it sees a `go/printer` render of it, and the override name carries no package qualifier.

## Examples

| Command | Declaration in the package | Result |
| --- | --- | --- |
| `gno run -X myVar=ovr f.gno` | `var myVar = "default"` | patched to `ovr` |
| `gno run -X main.myVar=ovr f.gno` | `var myVar = "default"` | no change, exit 0, no message |
| `gno run -X Count=7 f.gno` | `var Count = 3` | no change, exit 0, no message; `go build` errors here |
| `gno run -X Konst=z f.gno` | `const Konst = "k"` | no change, exit 0, no message |
| `gno test -X Version=1.2.3 ./...` | `var Version` in two packages | both packages patched |
| `gno test -X Greeting=bye .` | `var Greeting` in a `_filetest.gno` | filetest patched, its golden is not, test fails |

## Glossary

- MemPackage: in-memory set of a package's source files, the unit loaded, type-checked, and run.
- filetest: a `*_filetest.gno` file executed by the VM and asserted against golden directives (`// Output:`, `// Error:`, `// Realm:`).

## Warnings (should fix)

- **[every reported line number can be wrong]** `gnovm/cmd/gno/xflag.go:83-88` — reported line numbers stop matching the file on disk, because re-quoting collapses a multi-line raw string initializer to one line and the VM parses that re-render.
  <details><summary>details</summary>

  The VM and the type-checker are handed [`buf.String()`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/xflag.go#L83-L88) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L83-L88), not the bytes on disk, and `strconv.Quote` turns a multi-line raw string into a single-line quoted one. On a gofmt-clean file whose `var Banner` holds a three-line raw string, a panic below it reports `main.gno:7` under `-X` and `main.gno:10` without it; the same file under `gno test` reports a type error at `e.gno:6:9` instead of `e.gno:9:9`. Runs of two or more blank lines collapse the same way. Overriding a banner or template string is exactly what `-X` is for, so this is the common case, not the corner. Fix: splice the quoted value into the original source bytes at the literal's own offsets and leave the rest of the file untouched, instead of re-serializing the AST. [repro](comment_claude-opus-4-8.md), test: [`flag_x_lines.txtar`](tests/flag_x_lines.txtar).
  </details>

- **[one flag rewrites unrelated packages]** `gnovm/cmd/gno/test.go:449-453` — an override name carries no package qualifier, so one flag rewrites the same-named var in every package under test, `_test.gno` and `_filetest.gno` files included.
  <details><summary>details</summary>

  The loop applies the same map to every file of every [MemPackage](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/test.go#L449-L453) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/test.go#L449-L453) it tests, and `MPAnyAll` includes `_test.gno` and `_filetest.gno`. Two unrelated packages each declaring `var Version = "dev"` both came back `1.2.3` from a single `-X Version=1.2.3 ./...`. A filetest whose own `var Greeting` matched was patched while its `// Output:` golden was not, turning a passing package red. The [flag help](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/test.go#L196-L197) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/test.go#L196-L197) points at `go build -ldflags "-X ..."`, whose [documented form](https://pkg.go.dev/cmd/link) is `importpath.name=value` and which rejects the unqualified spelling outright with `-X flag requires argument of the form importpath.name=value`. Note that the unqualified spelling is the one [issue #1021](https://github.com/gnolang/gno/issues/1021) asks for, so this is a scoping decision, not a plain bug. Fix: decide whether `-X` accepts a package qualifier, and stop citing `go build -X` in the help text for the part where the two differ. [repro](comment_claude-opus-4-8.md)
  </details>

- **[a typo produces a green run with the old value]** `gnovm/cmd/gno/xflag.go:46-75` — a name that matches nothing is silently ignored, so the run stays green on the default value, and that includes the qualified `main.myVar=...` form the Go linker requires.
  <details><summary>details</summary>

  The [walk](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/xflag.go#L46-L75) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L46-L75) `continue`s past every non-match and `patchXVars` never reports which overrides went unused. `gno run -X main.myVar=ovr`, `-X Count=7` on `var Count = 3`, and `-X Konst=z` on a `const` each exited 0 printing the untouched defaults. The [Go linker](https://pkg.go.dev/cmd/link) diverges on two of these: it errors with `main.Count: cannot set with -X: not a var of type string (type:int)`, and it errors on the unqualified name. Silence for an unknown name does match it. Its documented scope is also wider: it sets a var "initialized to a constant string expression", where the [`*ast.BasicLit` test](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/xflag.go#L68-L71) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L68-L71) drops `var x = "a" + "b"`. Fix: report the override names that matched nothing. [repro](comment_claude-opus-4-8.md)
  </details>

## Nits

- **[a broken contract nobody trips over yet]** `gnovm/cmd/gno/xflag.go:103-114` — [`String()`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/xflag.go#L103-L114) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag.go#L103-L114) joins the map with commas in Go's random map order, and `Set` cannot parse the result back. No current trigger: `flag.FlagSet.Var` snapshots `DefValue` from `String()` at registration, when the map is still empty, and [the usage renderer reads `f.DefValue`](https://github.com/gnolang/gno/blob/639f73fb3/tm2/pkg/commands/command.go#L304-L315) · [↗](../../../../../.worktrees/gno-review-5985/tm2/pkg/commands/command.go#L304-L315), so `gno run -h` always prints `-X ...`. Not posted, no change needed.
- **[help text promises parity the flag does not have]** `gnovm/cmd/gno/run.go:94-95` — the help string names `go build -ldflags "-X ..."` as the model, and the same string is repeated verbatim at [`test.go:196-197`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/test.go#L196-L197) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/test.go#L196-L197). Covered by the two Warnings above; not posted separately.

## Missing Tests

- **[the gno test path ships untested]** `gnovm/cmd/gno/test.go:449-453` — `-X` coverage stops at `gno run`, so deleting the MemPackage patch loop keeps the suite green.
  <details><summary>details</summary>

  [`run_test.go`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/run_test.go#L22-L33) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/run_test.go#L22-L33) covers `gno run -X` and [`xflag_test.go`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/xflag_test.go#L62-L222) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/xflag_test.go#L62-L222) covers `patchXVars` in isolation, but the MemPackage loop that wires the two together on the `gno test` side has no coverage at all. The txtar harness at [`testdata/test/`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/testdata_test.go#L13-L41) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/testdata_test.go#L13-L41) is the natural home. Fix: add [`flag_x.txtar`](tests/flag_x.txtar), which asserts the override lands and the default applies without the flag; it passes at 639f73fb3.
  </details>

## Verified

- `go/parser` accepts every real `.gno` file in the tree, so the "unparseable body is returned unchanged" branch has no trigger today: 1401 files under `examples`, 253 under `gnovm/stdlibs`, 2503 under `gnovm/tests/files`, zero rejections. The only 9 failures sit under `gnovm/tests/integ` and are deliberately-invalid fixtures (`empty_gno3`, `invalid_gno_file`, `several-lint-errors`).
- Re-serializing preserves comments and filetest directives. Patching all 41 repo `.gno` files that declare a top-level string var with their own current values changed bytes in 28 of them, lost 0 comment lines, and changed 0 line counts. Every one of those 28 diffs is alignment padding rewritten from spaces to tabs inside `var` groups and before trailing comments, because `printer.Fprint` runs without gofmt's `UseSpaces` mode. Four hand-built files carrying `// PKGPATH:`, a var doc comment, a trailing line comment, `// Output:`, `// Realm:` and `// Error:` each came back with every directive in place.
- The override value cannot escape its literal. An override whose text is a Go string-concatenation fragment printed verbatim, and a value containing a newline printed as two lines, both because `strconv.Quote` re-escapes before the literal is written.
- [`mustReadAndPatchFile`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/run.go#L385-L394) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/run.go#L385-L394) reports errors identically to the [`MustReadFile`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/pkg/gnolang/go2gno.go#L48-L54) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/pkg/gnolang/go2gno.go#L48-L54) it replaces: both `os.ReadFile` then `ParseFile`, both panic with the same value, both still land in [`catchPanic`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/cmd/gno/run.go#L369-L371) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/cmd/gno/run.go#L369-L371).
- `go build -ldflags` comparison run locally on go1.26.5: unknown name exits 0 silently, `-X main.Count=7` on an int var exits 1 with `cannot set with -X: not a var of type string (type:int)`, `-X main.Const=z` on a const exits 0 silently, `-X Version=v` exits 1 with `-X flag requires argument of the form importpath.name=value`.
- `gno test` runs packages concurrently, and each `testPkg` gets a fresh MemPackage from [`MustReadMemPackage`](https://github.com/gnolang/gno/blob/639f73fb3/gnovm/pkg/gnolang/mempackage.go#L847) · [↗](../../../../../.worktrees/gno-review-5985/gnovm/pkg/gnolang/mempackage.go#L847), which re-reads from disk with no cache, so the patch loop mutates per-goroutine state only and the shared override map is read-only after flag parsing.
- Green at 639f73fb3: `go test ./gnovm/cmd/gno/...` (23s), the 11 `TestPatchXVars` subtests, `TestXFlag_SetAndString`, `TestXFlag_SetInvalid`, `TestXFlag_NilString`, plus both review test artifacts run through the txtar harness.

## Existing threads

None. No review comments and no human comments on the PR; the only comment is the Gno2D2 bot summary.

## Open questions

- The red `check` job is the conventional-commit title linter rejecting `gnovm: add -X flag ...`. Not a code problem and not posted.
- `gno run -X` and `gno test -X` patch only the files named on the command line or belonging to the package under test, never their dependencies. That is defensible but undocumented; worth a line in the help text if the flag stays. Not posted, no decision hangs on it.
- Issue [#1021](https://github.com/gnolang/gno/issues/1021) also asks for `gno publish -X`. The PR body explains that subcommand does not exist, which reads correct against the current CLI. Not posted.
