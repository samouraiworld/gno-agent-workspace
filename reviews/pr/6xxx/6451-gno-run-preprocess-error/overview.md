# `runFiles` in `gno run`: preprocess errors as one line
Written by claude-opus-5-5.
PR: [gnolang/gno#6451](https://github.com/gnolang/gno/pull/6451)

## TLDR

When `gno run` met a mistake in the code it was asked to run, such as a constant that does not fit its type or the same function declared in two files, it crashed with a Go stack trace. This change catches those mistakes and prints one line naming the file, the line and the problem, then exits with code 1, the way `gno lint` and `gno test` already do.

## Preprocessing and `PreprocessError`

Before any gno code runs, the VM walks every file once to resolve names, check types and fold constants: that walk is `Preprocess`. When it finds a problem it panics, and `doRecover` in `gnovm/pkg/gnolang/preprocess.go` catches the panic on its way out of each node, adds the node's location (`<pkgpath>/<file>:<line>:<col>`) and re-panics it as a `*PreprocessError`. A panic already of that type passes through untouched, so the innermost location wins. `gno lint` and `gno test` catch these in `catchPanic` and print them; `gno run` called `m.RunFiles` bare, so the panic reached the Go runtime.

## The new `runFiles` wrapper

[`runFiles`](https://github.com/gnolang/gno/pull/6451/files#diff-27f332114c6bfa7b15dd3385042edd0efd3700c020ec01e293d88f0644c5e1c0R379-R411) wraps `m.RunFiles` in a `recover` (skipped when `DEBUG_PANIC=1`, which keeps the raw panic for debugging). It sorts what it catches three ways:

- a `*gno.PreprocessError`: it strips the leading `<pkgPath>/` from the message, since `gno run` names files with the paths typed on the command line, and prints the result through `guessIssueFromError` with a `(code=gnoPreprocessError)` suffix;
- an `error` that is a `gno.RedeclarationErrors`: printed as is, since it carries no location;
- anything else, a runtime panic in a package initializer for one: re-thrown unchanged.

The first two return `commands.ExitCodeError(1)`, which `execRun` returns to the CLI.

## Duplicate declarations and `RedeclarationErrors`

`checkDuplicates` in `gnovm/pkg/gnolang/machine.go` lists every top-level name declared twice across a package's files. Its error type was the unexported `redeclarationErrors`; the change [exports it](https://github.com/gnolang/gno/pull/6451/files#diff-0830c8c26a4042165bbd447eb4955f11a0789f85ddfe85b794f6db2f26115aa5R498-R524) as `RedeclarationErrors` so `runFiles` can match it with `errors.As`. Whether `checkDuplicates` runs at all depends on the package path: `Machine.RunFiles` passes [`IsStdlib(pv.PkgPath)`](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/pkg/gnolang/machine.go#L646) as `withOverrides`, and a stdlib-looking path, the bare `main` that `gno run` uses by default included, lets a later declaration replace an earlier one instead of reporting it.

## Reading an error location: `ReErrorLine`

`gno.ReErrorLine` splits an error string into named parts: `PATH` (the directory part), an optional `FILE`, `SPAN` (`<line>:<col>` or `<line>:<col>-<endline>:<endcol>`) and `MSG`. Its `LOC` part is the whole location, path and file included. `guessIssueFromError` used to join `PATH` with `LOC`, which repeated the path: `gno.land/p/f1/l2:gno.land/p/f1/l2/l2.gno:3:1-21`. It [now joins](https://github.com/gnolang/gno/pull/6451/files#diff-d7622258d9c25719013624e7bdf138c9c9476033e8a7588ee4aec6cce4949bb3R235-R245) `PATH`, `FILE` and `SPAN`, then shortens the path against the package directory, giving `l2.gno:3:1-21`. When the regex does not match, the location falls back to `<pkgPath>:0`. The helper serves `gno lint` and `gno test` as well, so their locations change the same way.

## Before and after

**Before:** `gno run` on a file with `var u uint8 = -1` printed `panic: bigint underflows target kind [recovered]`, the preprocess stack and a Go goroutine dump, and exited 2, through `m.RunFiles` with no recover.

**After:** the same run prints `main.gno:4:6-18: bigint underflows target kind (code=gnoPreprocessError)` and exits 1, through `runFiles`.

## Words used here

| Name | What it is |
| --- | --- |
| `Preprocess` | The VM pass over every file before execution that resolves names and checks types; a problem panics. |
| `doRecover` | The deferred function in `preprocess.go` that adds a node's location to a panic and re-panics it as `*PreprocessError`. |
| `*PreprocessError` | The panic value carrying the located error and the preprocess node stack; `Unwrap()` returns the error. |
| `RedeclarationErrors` | The error `checkDuplicates` returns, a list of names declared more than once in one package. |
| `withOverrides` | The `runFileDecls` flag that lets a later declaration replace an earlier one; true when `IsStdlib(pkgPath)` holds. |
| `ReErrorLine` | The regex in `nodes_location.go` that splits `<path>/<file>:<span>: <msg>` into named groups. |
| `DEBUG_PANIC` | Environment variable; `DEBUG_PANIC=1` turns off the recover in `runFiles` and `catchPanic`. |
| `gnoPreprocessError` | The issue code printed in parentheses after a preprocess error, shared by `gno run`, `gno lint` and `gno test`. |
