# PR #5320: fix(gnovm): support gno fmt for filetest

URL: https://github.com/gnolang/gno/pull/5320
Author: ltzmaxwell | Base: master | Files: 98 | +431 -245
Reviewed by: davd-gzl | Model: claude-opus-4-7-1m
Local worktree: `git -C gno worktree add .worktrees/gno-review-5320 d2e3201c` (then `gh -R gnolang/gno pr checkout 5320` inside it)

**Verdict: REQUEST CHANGES** — fast-path + auto-update across all filetests silently strips test-only unused imports from `recover1b.gno`, `assign37b.gno`, and `const49.gno`, losing the very `TypeCheckError` assertions those files were written to verify; resolve thehowl's open thread on `OnPackageConflict` before merge.

## Summary

`gno fmt` previously refused to format any directory whose `.gno` files declared different package names — which is the entire shape of `gnovm/tests/files/` (hundreds of independent filetests, most `package main` but also `package foo`, `package other`, …). The PR adds an `ErrPackageConflict` sentinel in `ParsePackage`, has `FormatFile` swallow it and fall back to per-file formatting, and short-circuits the package probe entirely for paths under `gnovm/tests/{files,challenges}` via a new `formatOneFile` router in the CLI. The `Makefile` `fmt` target now runs `gno fmt -w ./tests/files/...`, and 91 filetests are reformatted accordingly. Mechanism is sound; the load-bearing concern is that the imports-aware formatter prunes "unused" imports — and three filetests in this very diff intentionally carry an unused import to assert the typechecker emits `"X" imported and not used`.

## Glossary

- **filetest** — single `.gno` file under `gnovm/tests/files/` with embedded `// Output:` / `// Error:` / `// TypeCheckError:` directives; the test runner executes the file and string-matches each directive.
- **`TypeCheckError:` directive** — exact-match assertion against the typechecker output for that file; if removed but the typechecker still emits errors, the test fails (`filetest.go:200`).
- **`ParsePackage`** — `gnofmt` helper that reads a directory, parses the package clause of every non-test `.gno`, and errors if names disagree (`package.go:61`).
- **filetestsRoot / challengesRoot** — `$GNOROOT/gnovm/tests/files` and `$GNOROOT/gnovm/tests/challenges`; the two paths the CLI hard-codes as "always per-file format" (`fmt.go:242-243`).

## Fix

Before: `gno fmt ./gnovm/tests/files/...` failed on the first directory with mixed package names; the directory had been ungovernable by `gno fmt` since inception. After: `ParsePackage` returns a sentinel ([`package.go:89`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/pkg/gnofmt/package.go#L89) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/pkg/gnofmt/package.go#L89)) instead of an opaque `fmt.Errorf`; `FormatFile` recognises it, caches `nil` against the directory, and routes the file through `FormatImportFromSource` ([`processor.go:117-141`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/pkg/gnofmt/processor.go#L117-L141) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/pkg/gnofmt/processor.go#L117-L141)). For known filetest roots, `formatOneFile` skips the package probe entirely ([`fmt.go:277-282`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/cmd/gno/fmt.go#L277-L282) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/cmd/gno/fmt.go#L277-L282)). The 91 reformatted filetests are mostly whitespace (tabs vs spaces, operator spacing, trailing blanks); three carry semantic regressions because the formatter pruned imports that were part of the test surface.

## Critical (must fix)

- **[silently drops `TypeCheckError` coverage for unused imports]** [`gnovm/tests/files/recover1b.gno`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/tests/files/recover1b.gno) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/tests/files/recover1b.gno), [`gnovm/tests/files/assign37b.gno`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/tests/files/assign37b.gno) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/tests/files/assign37b.gno), [`gnovm/tests/files/const49.gno`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/tests/files/const49.gno) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/tests/files/const49.gno) — `gno fmt -w` stripped `import "fmt"` / `import "io"` that each file deliberately carried to exercise the typechecker's `"X" imported and not used` diagnostic; the corresponding lines were then deleted from `// TypeCheckError:` to keep the file passing.
  <details><summary>details</summary>

  Pre-PR, `recover1b.gno` asserted (in addition to the panic) `// TypeCheckError: main/recover1b.gno:3:8: "fmt" imported and not used`. The fmt run removed both the `import "fmt"` line and the assertion — the file now exercises the recover/panic path only, and the codepath that emits the unused-import error has lost a witness here. Same shape for `const49.gno` (`"io" imported and not used` — the `// TypeCheckError:` directive is now fully gone, deleted via [`const49.gno:14-16`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/tests/files/const49.gno#L14) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/tests/files/const49.gno#L14)) and `assign37b.gno` (the trailing `; main/assign37b.gno:3:8: "fmt" imported and not used` clause dropped from a longer assertion).

  This is not a one-shot regression — the new `make fmt` target ([`gnovm/Makefile:61`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/Makefile#L61) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/Makefile#L61)) extends `gno fmt -w` to `./tests/files/...`, so the next contributor who writes a filetest containing an intentionally unused import will have it stripped on the very next `make fmt`, and the new assertion silently dropped from the regenerated `// TypeCheckError:` block. The mechanism is repeatable for every typechecker diagnostic that the unused-import pruner can trigger (unused vars are arguably the next target, since `imports.Process` doesn't touch them, but any "_ imported but not used" test will decay).

  Fix: restore the three imports as named blank imports so the pruner leaves them alone — `import _ "fmt"` for `recover1b.gno` / `assign37b.gno` and `import _ "io"` for `const49.gno` — and verify the `// TypeCheckError:` directive still receives the unused-import line under the typechecker (a blank import is technically used, so this may need a `//gnofmt:keep` style escape hatch or a separate filetest convention). Alternative: gate the unused-import pruning behind a flag and disable it for `./tests/files/...` in the Makefile invocation. Whichever path is chosen, the three reverts in this PR need to be undone and the underlying mechanism documented so future filetests don't decay the same way.
  </details>

## Warnings (should fix)

- **[unresolved review thread]** [@thehowl](https://github.com/gnolang/gno/pull/5320#discussion_r3102173136) [`gnovm/pkg/gnofmt/processor.go:131`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/pkg/gnofmt/processor.go#L131) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/pkg/gnofmt/processor.go#L131) — thehowl asks to drop `OnPackageConflict` and have `FormatFile` return the error; the CLI then explicitly enumerates known filetest dirs. The PR keeps both the silent fallback and the explicit enumeration; the resulting matrix is the source of the user-hostile note below.
  <details><summary>details</summary>

  The current shape — `FormatFile` swallows the conflict, calls back to `OnPackageConflict`, the CLI prints a note under `-v` telling the user to edit `gnovm/cmd/gno/fmt.go` — leaks an internal-contributor concern into end-user output. thehowl's proposal removes the silent path entirely: external users get an explicit error ("directory has mixed package names; pass individual files"), gno contributors maintain the allowlist, and `OnPackageConflict` plus its docstring disappear. Fewer moving parts, no decision tree at runtime.

  Fix: either close the thread by adopting thehowl's suggestion (delete `OnPackageConflict`, return `ErrPackageConflict` from `FormatFile`, have the CLI catch it only for the known roots), or push back in-thread with the case for the current design before merging.
  </details>

- **[contributor-only note shown to end users]** [`gnovm/cmd/gno/fmt.go:250-258`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/cmd/gno/fmt.go#L250-L258) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/cmd/gno/fmt.go#L250-L258) — the verbose-mode diagnostic tells users to "register this path in `gnovm/cmd/gno/fmt.go`'s filetest roots", which only makes sense for someone hacking on gno itself.
  <details><summary>details</summary>

  An external user running `gno fmt -v ./some/mixed-pkg/dir/` is told to edit a file inside the gno source tree they may not even have a checkout of. The note doesn't tell them the operationally useful thing: "pass individual files instead, or split the directory by package." If the design from the thread above stands, the note should be rewritten for the audience that actually sees it; if thehowl's design wins, the note can disappear entirely.
  </details>

- **[fast-path covers real packages too]** [`gnovm/cmd/gno/fmt.go:262`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/cmd/gno/fmt.go#L262) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/cmd/gno/fmt.go#L262) — `formatOneFile` short-circuits the package-aware path for everything under `gnovm/tests/files/`, including legitimate multi-file packages under `extern/*` (e.g. `extern/redeclaration1/`, `extern/ct/`, `extern/foo/`).
  <details><summary>details</summary>

  `gnovm/tests/files/extern/` holds real testdata packages — `redeclaration1/` has two `package redeclaration` files, `extern/ct/` has three `package ct` files, etc. (`find gnovm/tests/files/extern -mindepth 2 -maxdepth 2 -type d` finds 8 dirs). For these, package-level formatting (where `processor.go` pools top-level decls across files) is the more correct mode; per-file formatting risks the very import-pruning failure mode the Known-Limitation doc warns about ([`processor.go:100-116`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/pkg/gnofmt/processor.go#L100-L116) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/pkg/gnofmt/processor.go#L100-L116)).

  Spot-checking the current extern packages, no file references a symbol defined-only-in-a-sibling file with an imported package, so today's tree is safe. But future testdata under `extern/` won't be — adding a sibling-decl test there will silently lose imports. Fix: scope the fast-path to `tests/files` excluding `extern/`, or move the conflict detection into `formatOneFile` and only fast-path directories that actually conflict.
  </details>

- **[per-file mode loses the consistency check]** [`gnovm/pkg/gnofmt/processor.go:138-141`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/pkg/gnofmt/processor.go#L138-L141) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/pkg/gnofmt/processor.go#L138-L141) — the doc block above acknowledges that a directory of independent `package main` filetests cannot be detected and that imports will be pooled across unrelated files, causing silent import-list corruption.
  <details><summary>details</summary>

  The Known-Limitation block lays out the failure shape plainly: file A imports `pkg.Sym`, file B in the same dir top-level-declares its own `Sym`, the processor treats A's `Sym` as resolved by B, prunes A's import. This is mitigated for `tests/files/` by the fast-path (per-file mode treats each file in isolation), and the doc says "add new dirs of this shape to the enumeration" — but there is no mechanism to detect when a maintainer has missed one. A regression test or lint that walks `tests/files/`-like dirs and flags any with package-name uniformity but cross-file unresolved symbols would close the loop.
  </details>

## Nits

- [`gnovm/cmd/gno/fmt.go:287-304`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/cmd/gno/fmt.go#L287-L304) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/cmd/gno/fmt.go#L287-L304) — `isUnderAnyRoot` runs `filepath.Abs` on every input file's directory; for `tests/files/...` (~1000 files) that's ~1000 stat-free syscalls. Trivial in absolute terms, but the function could cache `filepath.Abs(dir)` per dir or take pre-cleaned absolute roots and a pre-cleaned absolute dir. Skip if benchmarks show no win.
- [`gnovm/pkg/gnofmt/processor.go:127`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/pkg/gnofmt/processor.go#L127) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/pkg/gnofmt/processor.go#L127) — comment "cache nil to avoid re-parsing" is accurate but reads as if storing nil is a hack; clarify that `pkgdirCache` is `map[string]Package` and a stored `nil` value is the conflict sentinel.

## Missing Tests

- **[no regression test for fast-path scoping]** [`gnovm/cmd/gno/fmt.go:242-243`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/cmd/gno/fmt.go#L242-L243) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/cmd/gno/fmt.go#L242-L243) — `filetestsRoot` and `challengesRoot` are hard-coded; if either path moves or a typo creeps in, the fallback path still works (because `ErrPackageConflict` is caught) so no test fails. A txtar test that asserts a file under `gnovm/tests/files/` does NOT trigger the verbose conflict note would catch a regression.
  <details><summary>details</summary>

  Today, the only way to know the fast-path is wired correctly is via behaviour difference: per-file mode is silent under `-v`, fallback mode emits the conflict note. The existing `conflict_fallback.txtar` covers the fallback path; an inverse test covering the fast-path would lock the wiring.
  </details>

- **[no test for the documented limitation]** [`gnovm/pkg/gnofmt/processor.go:100-116`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/pkg/gnofmt/processor.go#L100-L116) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/pkg/gnofmt/processor.go#L100-L116) — the doc describes a concrete failure shape (same-pkg-name dir, cross-file decl, pruned import) but there is no test asserting it actually behaves that way today, nor a TODO regression flag for when it's fixed.
  <details><summary>details</summary>

  Without a witness, the next refactor of `processPackageFiles` could accidentally fix or worsen the behaviour silently. Add a `processes_test.go` case that constructs the shape and asserts the (currently wrong) output, so any future change is forced to address it explicitly.
  </details>

## Suggestions

- [`gnovm/Makefile:61`](https://github.com/gnolang/gno/blob/d2e3201c/gnovm/Makefile#L61) · [↗](../../../../../.worktrees/gno-review-5320/gnovm/Makefile#L61) — adding `./tests/files/...` to `make fmt` makes every contributor's `make fmt` invocation a load-bearing test-coverage decision (see Critical above). Either gate this behind a separate target (`make fmt-tests`) or pair it with a CI check that fails if `// TypeCheckError:` directives lose lines across the same PR.

## Questions for Author

- Did you intend to drop the `"fmt" imported and not used` / `"io" imported and not used` clauses from `recover1b.gno`, `assign37b.gno`, `const49.gno`? If yes, what's the new test that exercises that typechecker codepath? If not, see Critical for the suggested fix.
- Have you closed [thehowl's discussion on `OnPackageConflict`](https://github.com/gnolang/gno/pull/5320#discussion_r3102173136) elsewhere, or is the current design (silent fallback + verbose note) the considered response?
- The fast-path enumerates `tests/files` and `tests/challenges`. Are there other dirs in the repo with the same shape (mixed package names, intentional)? `gnovm/tests/backup/` is one — should it be in the list?
