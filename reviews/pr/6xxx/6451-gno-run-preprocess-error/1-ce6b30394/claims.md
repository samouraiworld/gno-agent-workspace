# Claims: gnolang/gno#6451 round 1, ce6b30394, claude-opus-5-5, standard review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`run.go:404`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/run.go#L404) | 1 | Warning | CONFIRMED | same | posts |
| [`machine.go:646`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/pkg/gnolang/machine.go#L646) | 1001 | Warning | CONFIRMED | same | posts |
| [`common.go:236`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/common.go#L236) | 2 | Missing test | CONFIRMED | differs | posts |
| [`common.go:236`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/common.go#L236) | 1003 | Missing test | CONFIRMED | differs | with row 2 |
| [`run.go:394`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/run.go#L394) | 1002 | Nit | CONFIRMED | differs | posts |
| [`run.go:387`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/run.go#L387) | 3 | Suggestion | CONFIRMED | differs | posts |

### 1 · Warning · CONFIRMED · `gnovm/cmd/gno/run.go:404`

An initialization cycle through a function, a method, a closure or a two-function chain reaches `default: panic(r)` and aborts with a Go panic, exit 2

- Check: build gno at ce6b30394, run GNO=<bin> sh tests/solo-finder-1-cycle-shapes.sh: each case exits 2 with a circular dependency panic
- Observed: sandboxed finder script at ce6b30394: `ok cycle_method: exit 2` / `panic: main/cycle_method/main.gno:7:5: circular dependency: a -> T.M -> a [recovered, repanicked]`, closure `a -> a`, twohop `a -> f -> g -> a`, func `a -> f -> a`, all exit 2; at 398e5960d the same four exit 2 without `[recovered, repanicked]`
- Test: [`tests/solo-finder-1-cycle-shapes.sh`](tests/solo-finder-1-cycle-shapes.sh)

### 1001 · Warning · CONFIRMED · `gnovm/pkg/gnolang/machine.go:646`

With no -pkgpath the package path defaults to "main", IsStdlib("main") is true, so runFileDecls runs with overrides on and checkDuplicates never runs: b.gno's helper silently replaces a.gno's

- Check: gno run on a directory whose a.gno and b.gno both declare helper, no -pkgpath: prints 2, exit 0; with -pkgpath example.com/main: one redeclarations line, exit 1
- Observed: sandboxed judge probe: head `== p7dir rc=0` stdout `2`, `== p7pkgpath rc=1` `running package "example.com/main": redeclarations for identifiers: "helper" (code=gnoPreprocessError)`; base `== p7dir rc=0` stdout `2`
- Test: [`tests/judge-1-3-1001-probes.sh`](tests/judge-1-3-1001-probes.sh)

### 2 · Missing test · CONFIRMED · `gnovm/cmd/gno/common.go:236`

guessIssueFromError also serves gno lint and gno test, whose location changes from `gno.land/p/f1/l2:gno.land/p/f1/l2/l2.gno:3:1-21` to `l2.gno:3:1-21`, and no txtar asserts either

- Check: grep -rl 'gnoPreprocessError\\|gnoUnknownError' --include='*.txtar' returns 0 files; gno lint on a /p/ module with func F(cur realm) {} at head and base
- Observed: grep over gnovm, gno.land, contribs *.txtar: 0 files. Sandboxed gno lint: head `l2.gno:3:1-21: crossing function (realm first argument) declared in non-realm package ... (code=gnoPreprocessError)`; base `gno.land/p/f1/l2:gno.land/p/f1/l2/l2.gno:3:1-21: crossing function ...`
- Test: [`tests/judge-1-3-1001-probes.sh`](tests/judge-1-3-1001-probes.sh)

### 1003 · Missing test · CONFIRMED · `gnovm/cmd/gno/common.go:236`

Same gap as row 2 from the gnoUnknownError side: one lint txtar asserting a bare `<file>:<line>:<col>` location closes both

- Check: grep -rh 'gnoPreprocessError\\|gnoUnknownError' --include='*.txtar' gnovm returns nothing
- Observed: grep over every *.txtar: 0 files; finder's base/head pair `main.gno:main.gno:6:2` against `main.gno:6:2` matches the pattern the judge measured for row 2

### 1002 · Nit · CONFIRMED · `gnovm/cmd/gno/run.go:394`

guessIssueFromError(".", ".", ...) falls back to `filepath.Clean(pkgPath)+":0"`, so a duplicate inside an imported package prints `.:0:` where the comment promises no location

- Check: gno run -root-dir <tree with examples/gno.land/p/zz/dup duplicating Helper> on a main importing it
- Observed: sandboxed judge-1002 script: head `i1 rc=1` `.:0: running package "gno.land/p/zz/dup": redeclarations for identifiers: "Helper" (code=gnoPreprocessError)`; base `i1 rc=2` with `stack 1: file main/i1/main.gno:1:1-7:2`
- Test: [`tests/judge-1002-import-dup-location.sh`](tests/judge-1002-import-dup-location.sh)

### 3 · Suggestion · CONFIRMED · `gnovm/cmd/gno/run.go:387`

doRecover wraps any panic inside Preprocess, so `should not happen` prints as a user source error with no trace

- Check: gno run on `func (int) M() {}` at head and base
- Observed: sandboxed: head `== p5 rc=1` `p5_builtinrecv/main.gno:3:1-18: should not happen (code=gnoPreprocessError)`; base `== p5 rc=2` `panic: should not happen [recovered]` with the preprocess stack
- Test: [`tests/judge-1-3-1001-probes.sh`](tests/judge-1-3-1001-probes.sh)

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`run.go:393`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/run.go#L393) | reach | TrimPrefix(pkgPath+"/") plus guessIssueFromError(".", ".") might misplace the file for an absolute path, a -pkgpath realm path with a ../ file, a directory argument, or a run from inside the directory | ran gno-head in the sandbox: abs path -> `c7_pkgvar/main.gno:3:5-18: bigint overflows target kind (code=gnoPreprocessError)` (relativized to cwd, same file); `-pkgpath gno.land/r/a/b ../c7_pkgvar/main.gno` -> `../c7_pkgvar/main.gno:3:5-18`; `run .` inside the dir -> `main.gno:3:5-18`; every location names the right file |
| [`common.go:230`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/common.go#L230) | lines | a multi-line preprocess message would fail ReErrorLine (no (?s)) and fall back to location `.:0` | ran three messages most likely to span lines (crossing func with a multi-statement body dumped in the message, struct conversion, func-literal conversion with a multi-line span): each printed one line with its file location, e.g. `e3_funclitconv/main.gno:4:6-9:3: cannot use func(string) string as func(int) int (code=gnoPreprocessError)` |
| [`common.go:273`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/common.go#L273) | reach | guessFilePathLoc's GNOROOT branch slices the suffix `dir[len(dir)-len(pkgPath):]` where the root is the prefix `dir[:len(dir)-len(pkgPath)]`, so a location in another package under the same root never resolves to a filesystem path | out of the diff's scope: pre-existing, guessFilePathLoc is unchanged by the PR and the own-package case is caught by the earlier `s in pkgPath` branch; sed -n 273p gnovm/cmd/gno/common.go -> `gnoRoot := dir[len(dir)-len(pkgPath):]`. Candidate for an issue draft, not this review. |
| [`run.go:397`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/run.go#L397) | lines | the RedeclarationErrors branch might swallow a non-duplicate error, or a runtime init panic, as a preprocess error | errors.As only matches a chain holding RedeclarationErrors, raised solely by checkDuplicates (grep: machine.go:524,670,762); the PR's own run_init_panic case and a run of `panic("boom")` in init re-panic as before; gno-level panics surface as UnhandledPanicError, not RedeclarationErrors |
| [`run.go:409`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/run.go#L409) | lines | Self-reference through a func literal or a method (`var a = func() int { return a }()`, `var a = T{}.M()`) still exits 2 | same resolveEffectiveDeps string panic as notJoon's open report, not a new case: `[gno-head rc=2] panic: main/zz/cases3/v3/main.gno:3:5: circular dependency: a -> a [recovered, repanicked]` and `... v4/main.gno:7:5: circular dependency: a -> T.M -> a`; direct var cycles (`var a = b; var b = a`, `var a = a`) are PreprocessErrors and print one line, rc 1 |
| [`machine.go:500`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/pkg/gnolang/machine.go#L500) | rollout | Exporting redeclarationErrors could change node output read by an older binary | gno.land/pkg/sdk/vm/keeper.go formats VM panics with fmt.Errorf("%v", r), so the type name never reaches a tx result; the message is unchanged and pinned by gnovm/tests/files/redeclaration6.gno, redeclaration8.gno, redeclaration10.gno |
| [`run.go:387`](https://github.com/gnolang/gno/blob/ce6b30394/gnovm/cmd/gno/run.go#L387) | lines | Ordinary preprocess-error shapes might escape the PreprocessError branch or lose their location | 14 shapes run at head (unknown import, undefined name, recursive type, method on non-local type, return count, missing return, duplicate method, const overflow, assignment mismatch, ...): each prints `<path as given>:<span>: msg (code=gnoPreprocessError)` and exits 1; base panics with exit 2 on each |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gnovm/cmd/gno/run.go:404 | build gno at ce6b30394, run GNO=<bin> sh tests/solo-finder-1-cycle-shapes.sh: each case exits 2 with a circular dependency panic | warm | unwrapped-string-panic |
| 2 | CONFIRMED | Missing test | gnovm/cmd/gno/common.go:236 | grep -rl 'gnoPreprocessError\\|gnoUnknownError' --include='*.txtar' returns 0 files; gno lint on a /p/ module with func F(cur realm) {} at head and base | warm |  |
| 3 | CONFIRMED | Suggestion | gnovm/cmd/gno/run.go:387 | gno run on `func (int) M() {}` at head and base | warm |  |
| 1001 | CONFIRMED | Warning | gnovm/pkg/gnolang/machine.go:646 | gno run on a directory whose a.gno and b.gno both declare helper, no -pkgpath: prints 2, exit 0; with -pkgpath example.com/main: one redeclarations line, exit 1 | warm | stdlib-override-skips-duplicate-check |
| 1002 | CONFIRMED | Nit | gnovm/cmd/gno/run.go:394 | gno run -root-dir <tree with examples/gno.land/p/zz/dup duplicating Helper> on a main importing it | warm |  |
| 1003 | CONFIRMED | Missing test | gnovm/cmd/gno/common.go:236 | grep -rh 'gnoPreprocessError\\|gnoUnknownError' --include='*.txtar' gnovm returns nothing | warm |  |

</details>

Hit rate per tier, from the record: hot 0/0 confirmed over 0 files, warm 6/6 confirmed over 3 files, cold 0/0 confirmed over 5 files.

## Completeness

- Reflector, completeness: no candidate added and none dropped. Catalog classes walked against the diff: error and panic handling, VM-fault recoverability and type-check and preprocess are the ones it touches (rows 1, 3 and 1001); twin input paths is row 1001, the duplicate check keyed on `IsStdlib`; determinism, gas, realm state, caller identity, coin, storage deposit, global state, older data and Go semantics are untouched, since the diff changes CLI error reporting and renames one error type.
- Sibling sweep by shape: `ReErrorLine` has one other caller, `gnovm/pkg/repl/repl.go:142` and `:150`, which prints `MSG` alone and never joins a location, so it carries no doubled path. `m.RunFiles` has one caller under `gnovm/cmd`, `run.go:409`, the one the diff wraps; `runExpr` keeps its own recover (`run.go:436`).
- Judge probes beyond the candidates, sandboxed at head and base (`tests/judge-1-3-1001-probes.sh`, `tests/judge-1002-import-dup-location.sh`): a missing import prints `p1_missing_import/main.gno:3:8-27: unknown import path gno.land/p/nope/x (code=gnoPreprocessError)`, rc 1 (base rc 2); `-pkgpath gno.land/r/x/y ../p8_rel/main.gno` prints `../p8_rel/main.gno:4:6-18`, rc 1; an absolute file path run from another directory prints `../abs/main.gno:4:6-18`, a correct path relative to the working directory, so no finding.
- Draft repros: the blocks were run through the judge's binary built at ce6b30394 in the sandbox, not extracted verbatim, since the round may not run `gh pr checkout`; the outputs quoted are from those runs.
- The lint txtar in the common.go section is a direction, not run.

## Retro

- What failed: the writer's Verdict line reads `NEEDS DISCUSSION`, not one of the three verdict words, while its Event is `COMMENT`; left as written for the user to read. 17 min against 21; $8.48 against $7.
- What worked: the second finder's brief on the gnolang side found the default-package-path duplicate case no forge thread named.
- Hit rate per tier: warm 3 files carry every confirmed row; the 5 cold test files none.
- Upgrade: none beyond the batch-wide `--notes` the 6533 retro proposes.
