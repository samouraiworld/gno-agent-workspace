# Findings in posting order, from round assemble: 5 to post, 0 SKIP, 0 refuted kept out

## Body

- Warning: gnovm/pkg/gnolang/machine.go:646 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/pkg/gnolang/machine.go#L646), outside the diff, so no anchor
  State: CONFIRMED, band: Warning, angle: removed
  TL;DR: with no -pkgpath the package path defaults to "main", IsStdlib("main") is true, so runFileDecls runs with overrides on and checkDuplicates never runs: b.gno's helper silently replaces a.gno's
  Check: gno run on a directory whose a.gno and b.gno both declare helper, no -pkgpath: prints 2, exit 0; with -pkgpath example.com/main: one redeclarations line, exit 1
  Details: machine.go:646 passes IsStdlib(pv.PkgPath) as withOverrides; execRun defaults pkgPath to the package name when neither -pkgpath, a PKGPATH directive nor gnomod.toml names one. The PR's own duplicate test reaches the new RedeclarationErrors branch only because it passes -pkgpath example.com/main. Predates the diff, in scope as the missed member of the duplicate-declaration class the diff sweeps.
  Evidence: sandboxed judge probe: head `== p7dir rc=0` stdout `2`, `== p7pkgpath rc=1` `running package "example.com/main": redeclarations for identifiers: "helper" (code=gnoPreprocessError)`; base `== p7dir rc=0` stdout `2`
  Artifact: tests/judge-1-3-1001-probes.sh

## gnovm/cmd/gno/run.go:404 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/run.go#L404) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: an initialization cycle through a function, a method, a closure or a two-function chain reaches `default: panic(r)` and aborts with a Go panic, exit 2
Check: build gno at ce6b30394, run GNO=<bin> sh tests/solo-finder-1-cycle-shapes.sh: each case exits 2 with a circular dependency panic
Details: resolveEffectiveDeps raises the cycle as a plain string carrying its own location, outside doRecover, so the recovered value is neither *gno.PreprocessError nor error and the default branch re-panics it. Other recursive shapes (const a = b; const b = a, var x = x, type A = B; type B = A) raise inside Preprocess and print one line. The base behaves the same, so the diff does not cause it; it leaves the class it sets out to close open.
Evidence: sandboxed finder script at ce6b30394: `ok   cycle_method: exit 2` / `panic: main/cycle_method/main.gno:7:5: circular dependency: a -> T.M -> a [recovered, repanicked]`, closure `a -> a`, twohop `a -> f -> g -> a`, func `a -> f -> a`, all exit 2; at 398e5960d the same four exit 2 without `[recovered, repanicked]`
Artifact: tests/solo-finder-1-cycle-shapes.sh

## gnovm/cmd/gno/common.go:236 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/common.go#L236) · Missing test
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: guessIssueFromError also serves gno lint and gno test, whose location changes from `gno.land/p/f1/l2:gno.land/p/f1/l2/l2.gno:3:1-21` to `l2.gno:3:1-21`, and no txtar asserts either
Check: grep -rl 'gnoPreprocessError\|gnoUnknownError' --include='*.txtar' returns 0 files; gno lint on a /p/ module with func F(cur realm) {} at head and base
Details: Only TestRunApp reaches the new FILE/SPAN code, through the run path with dir "."; lint and test go through printError with the package dir and path. Reverting common.go keeps every txtar green.
Evidence: grep over gnovm, gno.land, contribs *.txtar: 0 files. Sandboxed gno lint: head `l2.gno:3:1-21: crossing function (realm first argument) declared in non-realm package ... (code=gnoPreprocessError)`; base `gno.land/p/f1/l2:gno.land/p/f1/l2/l2.gno:3:1-21: crossing function ...`
Artifact: tests/judge-1-3-1001-probes.sh
State: CONFIRMED, band: Missing test, angle: removed
TL;DR: same gap as row 2 from the gnoUnknownError side: one lint txtar asserting a bare `<file>:<line>:<col>` location closes both
Check: grep -rh 'gnoPreprocessError\|gnoUnknownError' --include='*.txtar' gnovm returns nothing
Details: One edit, a lint txtar under gnovm/cmd/gno/testdata/lint, closes rows 2 and 1003; carried by row 2's section.
Evidence: grep over every *.txtar: 0 files; finder's base/head pair `main.gno:main.gno:6:2` against `main.gno:6:2` matches the pattern the judge measured for row 2

## gnovm/cmd/gno/run.go:394 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/run.go#L394) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: guessIssueFromError(".", ".", ...) falls back to `filepath.Clean(pkgPath)+":0"`, so a duplicate inside an imported package prints `.:0:` where the comment promises no location
Check: gno run -root-dir <tree with examples/gno.land/p/zz/dup duplicating Helper> on a main importing it
Details: doRecover wraps an error-typed panic with tm2 errors.Wrap, whose Error() prints the cause alone, so ReErrorLine misses and common.go's no-match branch writes `.:0`. The message still names the package, so the user is not lost.
Evidence: sandboxed judge-1002 script: head `i1 rc=1` `.:0: running package "gno.land/p/zz/dup": redeclarations for identifiers: "Helper" (code=gnoPreprocessError)`; base `i1 rc=2` with `stack 1: file main/i1/main.gno:1:1-7:2`
Artifact: tests/judge-1002-import-dup-location.sh

## gnovm/cmd/gno/run.go:387 [gh](https://github.com/gnolang/gno/blob/ce6b303940399f5fc07e1d2eea72ae03eb83ed4d/gnovm/cmd/gno/run.go#L387) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: doRecover wraps any panic inside Preprocess, so `should not happen` prints as a user source error with no trace
Check: gno run on `func (int) M() {}` at head and base
Details: doRecover (preprocess.go:692-717) wraps every non-PreprocessError value. gno lint and gno test already print it this way through printError, and DEBUG_PANIC=1 restores the stack, so the change aligns run with them.
Evidence: sandboxed: head `== p5 rc=1` `p5_builtinrecv/main.gno:3:1-18: should not happen (code=gnoPreprocessError)`; base `== p5 rc=2` `panic: should not happen [recovered]` with the preprocess stack
Artifact: tests/judge-1-3-1001-probes.sh
