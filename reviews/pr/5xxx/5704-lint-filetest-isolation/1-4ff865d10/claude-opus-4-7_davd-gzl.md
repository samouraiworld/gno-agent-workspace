# PR #5704: fix(gnovm/lint,tm2/std): handle expected-error filetests; route _filetest.gno to filetests/

URL: https://github.com/gnolang/gno/pull/5704
Author: thehowl | Base: master | Files: 4 | +56 -8
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `4ff865d10` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5704 4ff865d10`

**Verdict: APPROVE** — unblocks master CI. The "absorb any panic when a directive is present" rule is the only sane choice given how `// Error:` is currently overloaded across stages; the real ceiling on lint's signal is the directive design itself, which a new stage-specific directive (e.g. `// PreprocessError:`) would lift in a follow-up.

## Summary

Master CI's `gno-checks / lint` job has been red since #5669 landed because the new `r/test/sealviolation` filetest is designed to fail at preprocess (asserts a foreign type cannot satisfy a sealed interface), and lint propagated the panic as a hard error. The PR does two coupled things: (1) in lint STEP 5, each `*_filetest.gno` is preprocessed in its own `defer/recover` IIFE so a panic on one filetest no longer aborts siblings, and panics on filetests that declare `// Error:` or `// TypeCheckError:` are silently absorbed (gno test still validates); (2) `MemPackage.WriteTo` mirrors `ReadMemPackage` and routes `_filetest.gno` into a `filetests/` subdir. Without (2), the first fix lets lint reach STEP 6 (`WriteTo`), and every `gno lint` would scatter duplicate filetests into package roots, with a second run failing on `ReadMemPackage`'s collision check ([`mempackage.go:745-746`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/pkg/gnolang/mempackage.go#L745-L746) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/gnolang/mempackage.go#L745-L746)). One on-disk file, `r/tests/vm/z4_filetest.gno`, is moved to `filetests/` to complete the layout started in #5104.

## Glossary

- **filetest** — `*_filetest.gno` file run as its own package by `gno test`. Each declares `main()` and optional directives like `// Error:`, `// PKGPATH:`.
- **STEP 5 / STEP 6** — lint pipeline stages. STEP 5 runs `PreprocessFiles` on every fileset (normal, test, filetest). STEP 6 (`mpkg.WriteTo`) rewrites the package to disk after all STAGE-1 errors are clear.
- **`MemFile.Name`** — flat basename, regex-validated, slashes forbidden ([`memfile.go:25-27`](https://github.com/gnolang/gno/blob/4ff865d10/tm2/pkg/std/memfile.go#L25-L27) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L25-L27)). Read path uses subdir layout; previously the write path didn't.
- **`// Error:` / `// TypeCheckError:`** — filetest directives that assert an expected failure. `// Error:` is currently overloaded: it covers both runtime panics and gno-preprocessor panics. `// TypeCheckError:` is the only stage-specific one.

## Fix

Before: lint STEP 5 iterated `ftests` and called `tm.PreprocessFiles` directly. A panic propagated up to the outer `catchPanic` wrapping the whole package ([`lint.go:243`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L243) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L243)), so one failing filetest skipped every later filetest in the same package and set `hasError`. After: each filetest is wrapped in `func(){ defer func(){ recover()… }(); … }()` ([`lint.go:343-363`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L343-L363) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L343-L363)); if `filetestExpectsFailure(body)` is true the recovered panic is silently dropped, otherwise it's printed and `hasError=true` but the loop continues. The companion `WriteTo` change ([`memfile.go:241-256`](https://github.com/gnolang/gno/blob/4ff865d10/tm2/pkg/std/memfile.go#L241-L256) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L241-L256)) restores symmetry with `ReadMemPackage`'s `filetests/` subdir convention ([`mempackage.go:717-751`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/pkg/gnolang/mempackage.go#L717-L751) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/gnolang/mempackage.go#L717-L751)).

## Critical (must fix)

None.

## Warnings (should fix)

- **[directive overload caps lint's signal]** [`lint.go:333-353`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L333-L353) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L333-L353) — `// Error:` is used for both preprocess and runtime panics, so lint can't tell which stage the file is talking about. The "absorb any panic" rule is the only sane choice, but it caps how much signal lint can ever give on these files. A stage-specific directive would lift the ceiling.
  <details><summary>details</summary>

  Today `// Error:` is overloaded. It marks "this file is supposed to fail" without saying *where* — preprocess, type-check, or runtime. Lint only runs the preprocessor, so when it hits a panic on a file with `// Error:`, it can't distinguish "this is the preprocess panic the directive expected" from "the directive was about runtime; this preprocess panic is unrelated." Absorbing anything is the only safe rule under that ambiguity.

  The cost: a file that declares `// Error: integer divide by zero` (a runtime claim) and breaks at preprocess for an unrelated reason will be silently absorbed by lint. End-to-end signal is fine — `gno test` still catches the directive/panic mismatch in CI — but the local-developer `gno lint` loop goes quiet.

  Splitting the directive removes the ambiguity without requiring lint to do message-matching. Add a new `// PreprocessError:` directive. Then the rules per file become:

  - `// PreprocessError: X` → lint expects a preprocess panic, absorbs it. If a different stage breaks unexpectedly, flag it.
  - `// Error: X` (runtime) → lint never sees a runtime panic. Any preprocess panic on this file is unexpected — flag it.
  - `// TypeCheckError: X` → handled symmetrically at the Go-typecheck stage (today this directive doesn't gate STAGE 1 either; see next warning).

  Existing seal-violation filetests would migrate from `// Error:` to `// PreprocessError:`. `// Error:` stays available as a runtime claim. This is a follow-up to this PR, not a blocker.

  <details><summary>reproducer</summary>

  ```bash
  gh pr checkout 5704 -R gnolang/gno
  cat > examples/gno.land/p/test/seal/filetests/z_mismatch_filetest.gno <<'EOF'
  // PKGPATH: gno.land/r/test/sealviolation_mismatch
  package sealviolation_mismatch

  import "gno.land/p/test/seal"

  type foreignImpl struct{}

  func (f *foreignImpl) Hello() string { return "evil" }
  func (f *foreignImpl) isSealed()     {}

  func main() {
      var s seal.Sealed = &foreignImpl{} // preprocess panics: missing method isSealed
      println(s.Hello())
  }

  // Error:
  // integer divide by zero
  EOF
  # (1) lint absorbs the (unrelated) preprocess panic because a directive is present.
  go run ./gnovm/cmd/gno lint -C examples ./gno.land/p/test/seal/... ; echo "lint exit: $?"
  # (2) gno test catches the directive/panic mismatch.
  go run ./gnovm/cmd/gno test ./examples/gno.land/p/test/seal/ 2>&1 | head -6
  rm examples/gno.land/p/test/seal/filetests/z_mismatch_filetest.gno
  ```

  Expected output:

  ```text
  lint exit: 0
  --- FAIL: ./gno.land/p/test/seal/z_mismatch_filetest.gno (elapsed: 0.00s, gas: 162)
  Error diff:
  --- Expected
  +++ Actual
  @@ -1 +1 @@
  -integer divide by zero
  +gno.land/r/test/sealviolation_mismatch/z_mismatch.gno:13:6-36: *gno.land/r/test/sealviolation_mismatch.foreignImpl does not implement gno.land/p/test/seal.Sealed (missing method isSealed)
  ```

  `lint exit: 0` is the bug surface; the `gno test` FAIL diff is the safety net that still works.
  </details>
  </details>

- **[`DirectiveTypeCheckError` in STEP 5 is unreachable]** [`common.go:113-120`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/common.go#L113-L120) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/common.go#L113-L120) — Go-typecheck errors abort STAGE 1 before STEP 5 runs, so the `TypeCheckError` half of the predicate cannot fire here.
  <details><summary>details</summary>

  STEP 5 only runs if `lintTypeCheck` succeeded ([`lint.go:270-274`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L270-L274) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L270-L274)): on errors it sets `hasError=true` and returns from the inner closure, skipping STEP 5/6 for that package. `TypeCheckMemPackage` already includes filetests in `STEP 4: Type-check Gno0.9 AST in Go (_filetest.gno)` ([`gotypecheck.go:515-529`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/pkg/gnolang/gotypecheck.go#L515-L529) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/gnolang/gotypecheck.go#L515-L529)), so a filetest with `// TypeCheckError:` already aborts in STAGE 1. The predicate looks symmetric across the two directives but isn't — typecheck-error filetests still kill the lint run today. Either drop `DirectiveTypeCheckError` from `filetestExpectsFailure` (simpler), or also lift per-filetest isolation up into STAGE 1 so `// TypeCheckError:` actually gets silenced where it happens (out of scope for this PR but worth filing alongside the directive split).
  </details>

## Nits

- [`common.go:113-120`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/common.go#L113-L120) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/common.go#L113-L120) — `filetestExpectsFailure` re-parses the directives that `parsePkgPathDirective` parsed two lines above. Negligible cost, but the obvious factor is to parse directives once and pass `dirs` (or both results) down. Skip if not worth the churn.
- [`lint.go:340-342`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L340-L342) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L340-L342) — comment claims "a panic on one ... doesn't skip siblings," but the `panic(r)` re-throw at [`lint.go:358`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L358) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L358) propagates non-error panics to the outer `catchPanic`, which *does* skip siblings. In practice gno preprocess always panics with errors, so the asymmetry is invisible — but the comment overstates the guarantee. Tighten the wording or note the carve-out.
- [`memfile.go:238-240`](https://github.com/gnolang/gno/blob/4ff865d10/tm2/pkg/std/memfile.go#L238-L240) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L238-L240) — doc comment mentions `gnolang.ReadMemPackage` but doesn't note that the inverse (filetest-named file under a non-filetests dir on subsequent read) will fail collision validation. Useful breadcrumb for the next reader puzzled by a duplicate-file error.

## Missing Tests

- **[zero coverage for either fix]** [`lint.go:343-363`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L343-L363) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L343-L363), [`memfile.go:241-256`](https://github.com/gnolang/gno/blob/4ff865d10/tm2/pkg/std/memfile.go#L241-L256) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L241-L256) — codecov reports 0% patch coverage; no txtar test, no unit test for `WriteTo`.
  <details><summary>details</summary>

  `gnovm/cmd/gno/testdata/lint/` has six txtar files, none of which exercise filetests; `tm2/pkg/std/memfile_test.go` covers `Validate` and `SplitFilepath` only ([`memfile_test.go:10,159`](https://github.com/gnolang/gno/blob/4ff865d10/tm2/pkg/std/memfile_test.go) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile_test.go)). A regression here is invisible until someone runs `gno lint -C examples ./...` locally and notices. Two cheap tests would lock the behavior in:

  - A txtar that lints a tiny package containing one filetest with `// Error:` (expect exit 0, no output) and one without (expect exit 1, error printed, sibling still processed).
  - A `TestMemPackage_WriteTo` round-trip: `Read → WriteTo → Read` on a package with files in both root and `filetests/`, asserting idempotency and that filetests don't migrate to the root.
  </details>

## Suggestions

- [`lint.go:343-363`](https://github.com/gnolang/gno/blob/4ff865d10/gnovm/cmd/gno/lint.go#L343-L363) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L343-L363) — extracting the IIFE into a named helper (`preprocessFiletest(ctx, mfile, fset, expectsErr, …) bool`) would make the closure's captured-variable surface explicit and let you unit-test the absorb/report decision in isolation. Optional.
- [`memfile.go:244`](https://github.com/gnolang/gno/blob/4ff865d10/tm2/pkg/std/memfile.go#L244) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L244) — `os.MkdirAll` runs once per filetest file. Lift the directory creation out of the loop (first filetest seen → mkdir, then reuse).

## Questions for Author

- Was a stage-specific directive (e.g. `// PreprocessError:`) considered, rather than overloading `// Error:` to also drive lint behavior? It would let lint flag preprocess panics that don't match the declared stage, without re-implementing `gno test`'s message-matching.
- Any plan to extend per-filetest isolation up into STAGE 1 so `// TypeCheckError:` actually silences typecheck errors symmetrically with how `// Error:` now silences preprocess panics?
- The PR body mentions the `params_valset_rotation_throttle` txtar flake as out of scope. Is there an open issue to track it, or should it be filed alongside this PR?
