# PR #5704: fix(gnovm/lint,tm2/std): defer expected-failure filetests to gno test; route _filetest.gno into filetests/

URL: https://github.com/gnolang/gno/pull/5704
Author: thehowl | Base: master | Files: 6 | +134 -44
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `050597de7` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5704 050597de7`

**Verdict: APPROVE** — second-pass review of follow-ups since `4ff865d10`. The duplicate-parsing nit, the DEBUG_PANIC carve-out, and the lint-side missing-tests gap are addressed. One cleanup ask: `cb1abaedb` (`excludeExpectedTypeCheckErrors`) is dead code in practice — `// TypeCheckError:` is gnovm-internal-only ([`filetest.go:51-53`](https://github.com/gnolang/gno/blob/050597de7/gnovm/pkg/test/filetest.go#L51-L53) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/test/filetest.go#L51-L53), [`test.go:90-91`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/test.go#L90-L91) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/test.go#L90-L91)) and no user file uses it. The previous review's "unreachable" finding meant "drop the dead code," not "make it reachable." Worth reverting, but not a blocker.

## Summary

Three changes layered into one PR: (1) `excludeExpectedTypeCheckErrors` filters `// TypeCheckError:` filetests out of `mpkg` before `lintTypeCheck`, so the package-level Go-typecheck doesn't propagate their intentional failure; (2) STEP 5's per-filetest preprocess panic is isolated, gated on `DEBUG_PANIC=1` like `catchPanic`, so a panic on one filetest no longer kills siblings and expected-failure filetests don't fail the whole lint run; (3) `MemPackage.WriteTo` routes `*_filetest.gno` into a `filetests/` subdir to mirror `ReadMemPackage`'s read convention, completing the migration started in #5104. Two new txtar tests cover the lint behavior (positive and negative). The `parsePkgPathDirective` / `filetestExpectsFailure` helpers were inlined per the previous review's nit.

## Glossary

- **filetest** — `*_filetest.gno` file run as its own package by `gno test`. Declares `main()` plus directives like `// Error:`, `// TypeCheckError:`, `// PKGPATH:`.
- **STEP 5 / STAGE 1 / STAGE 2** — lint pipeline stages. STEP 5 preprocesses every fileset (normal, test, filetest). STAGE 1 ends with `lintTypeCheck`+preprocess; STAGE 2 (`mpkg.WriteTo`) writes the package to disk.
- **`// TypeCheckError:`** — filetest directive asserting an expected Go-typecheck failure. Documented as "only available for gnovm internal test files" ([`test.go:90-91`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/test.go#L90-L91) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/test.go#L90-L91), [`filetest.go:51-53`](https://github.com/gnolang/gno/blob/050597de7/gnovm/pkg/test/filetest.go#L51-L53) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/test/filetest.go#L51-L53)).

## Fix

Before: a `_filetest.gno` with `// TypeCheckError:` failed STAGE 1's Go-typecheck on the package mpkg (lint reported `gnoTypeCheckError`, never reached STEP 5); a preprocess panic on one filetest aborted every later filetest in the same package via the outer `catchPanic`. After: [`lint.go:268`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L268) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L268) substitutes a filtered mpkg into `lintTypeCheck` via `excludeExpectedTypeCheckErrors` ([`lint.go:410-434`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L410-L434) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L410-L434)); STEP 5 ([`lint.go:332-370`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L332-L370) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L332-L370)) parses directives once, then wraps preprocess in a per-filetest IIFE — recover is registered only when `DEBUG_PANIC != "1"`, mirroring `catchPanic` at [`common.go:177`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/common.go#L177) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/common.go#L177); recovered errors on directive-tagged filetests are absorbed, untagged ones flow through `printError`. The `WriteTo` change ([`memfile.go:241-257`](https://github.com/gnolang/gno/blob/050597de7/tm2/pkg/std/memfile.go#L241-L257) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L241-L257)) restores symmetry with `ReadMemPackage`'s filetests-subdir read at [`mempackage.go:717-751`](https://github.com/gnolang/gno/blob/050597de7/gnovm/pkg/gnolang/mempackage.go#L717-L751) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/gnolang/mempackage.go#L717-L751).

## Critical (must fix)

None.

## Warnings (should fix)

- **[`excludeExpectedTypeCheckErrors` is dead code; revert `cb1abaedb`]** [`lint.go:410-434`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L410-L434) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L410-L434) — `// TypeCheckError:` is documented as "only available for gnovm internal test files" ([`test.go:90-91`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/test.go#L90-L91) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/test.go#L90-L91)) and `runFiletest` panics on it when `tcheck=false` ([`filetest.go:51-53`](https://github.com/gnolang/gno/blob/050597de7/gnovm/pkg/test/filetest.go#L51-L53) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/test/filetest.go#L51-L53), the path `gno test` always takes for user packages at [`test.go:428`](https://github.com/gnolang/gno/blob/050597de7/gnovm/pkg/test/test.go#L428) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/test/test.go#L428)). Users can't legitimately put the directive in a `_filetest.gno`. `grep -rln "// TypeCheckError:" examples/` returns nothing.
  <details><summary>details</summary>

  The filter currently fires on zero files in the repo. Its only effect is to make lint accept input that `gno test` rejects — which the docstring frames as "gno test matches the actual error against the directive," but `gno test` does the opposite: it runs the same `lintTypeCheck` over the full mpkg at [`test.go:312`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/test.go#L312) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/test.go#L312) and FAILs the package with `gnoTypeCheckError`.

  The previous review at [`1-4ff865d10/claude-opus-4-7_davd-gzl.md`](../1-4ff865d10/claude-opus-4-7_davd-gzl.md) flagged the original STEP 5 predicate's `DirectiveTypeCheckError` half as unreachable. The intended fix was "drop the dead `|| DirectiveTypeCheckError` half" — making the unreachability explicit. `cb1abaedb` instead added `excludeExpectedTypeCheckErrors` to make it reachable for user packages, which the directive's design doesn't support.

  **Fix:** revert `cb1abaedb` (the `excludeExpectedTypeCheckErrors` function, its call site at [`lint.go:268`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L268) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L268), the `|| DirectiveTypeCheckError` term in STEP 5's `expectsErr` at [`lint.go:342-343`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L342-L343) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L342-L343), and the `filetest_typecheck_error.txtar`+`filetest_unannotated_typecheck_error.txtar` pair). Keep `4ff865d10`'s STEP 5 isolation (the `// Error:` half, which is the only legitimately-user-facing directive) and `050597de7`'s DEBUG_PANIC carve-out.

  **Repro (showing the divergence today; gone after the revert):**

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5704 -R gnolang/gno
  mkdir -p /tmp/gno-tc-repro/p/test/tcdemo/filetests
  cat > /tmp/gno-tc-repro/gnowork.toml <<'EOF'
  use = "p/test/tcdemo"
  EOF
  cat > /tmp/gno-tc-repro/p/test/tcdemo/good.gno <<'EOF'
  package tcdemo

  func Hello() string { return "hi" }
  EOF
  cat > /tmp/gno-tc-repro/p/test/tcdemo/gnomod.toml <<'EOF'
  module = "gno.land/p/test/tcdemo"
  gno = "0.9"
  EOF
  cat > /tmp/gno-tc-repro/p/test/tcdemo/filetests/z_tc_filetest.gno <<'EOF'
  // PKGPATH: gno.land/r/test/tcdemoft
  package tcdemoft

  func main() {
  	var x int = "hello"
  	_ = x
  }

  // TypeCheckError:
  // cannot use "hello" (untyped string constant) as int value in variable declaration
  EOF
  GNOROOT=$PWD go run ./gnovm/cmd/gno lint -C /tmp/gno-tc-repro ./p/test/tcdemo ; echo "lint exit: $?"
  GNOROOT=$PWD go run ./gnovm/cmd/gno test -C /tmp/gno-tc-repro ./p/test/tcdemo 2>&1 | head -5 ; echo "test exit: ${PIPESTATUS[0]}"
  rm -rf /tmp/gno-tc-repro
  ```

  ```
  lint exit: 0
  p/test/tcdemo/z_tc_filetest.gno:5:14: cannot use "hello" (untyped string constant) as int value in variable declaration (code=gnoTypeCheckError)
  FAIL    ./p/test/tcdemo 	0.00s
  FAIL
  FAIL: 0 build errors, 1 test errors
  test exit: 1
  ```

  `lint exit: 0` + `test exit: 1` on the same synthetic input. Post-revert: both exit 1, the directive is rejected uniformly.
  </details>

- **[stale filetest at package root collides on next read]** [`memfile.go:241-257`](https://github.com/gnolang/gno/blob/050597de7/tm2/pkg/std/memfile.go#L241-L257) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L241-L257) — `WriteTo` always creates the file under `filetests/` but never deletes a stale copy at the package root. If a user runs `gno lint` on a package containing a leftover `z_old_filetest.gno` at root (older than #5104), the next `ReadMemPackage` returns `"cannot add %q in filetests: same filename in package dir"` from [`mempackage.go:745-746`](https://github.com/gnolang/gno/blob/050597de7/gnovm/pkg/gnolang/mempackage.go#L745-L746) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/gnolang/mempackage.go#L745-L746).
  <details><summary>details</summary>

  This PR migrates the one example that was caught in this state (`r/tests/vm/z4_filetest.gno`) by hand. For repositories outside `examples/`, the migration is left to the user: their first `gno lint` will succeed silently, but the second one (or any subsequent `ReadMemPackage`-based tool) fails on the dup-filename collision. The lint output gives no warning.

  **Fix:** before `WriteTo` rewrites filetests to `filetests/`, remove any same-named `*_filetest.gno` at the package root — or detect-and-warn so the user knows to delete the stale copy. Since `ReadMemPackage` reads from both locations and rejects duplicates, the in-memory mpkg is already de-duped at read time; the on-disk cleanup is what's missing.
  </details>

- **[directive overload caps lint's signal]** [`lint.go:332-370`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L332-L370) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L332-L370) — carried from prior review at [`5704-lint-filetest-isolation/1-4ff865d10/claude-opus-4-7_davd-gzl.md`](../1-4ff865d10/claude-opus-4-7_davd-gzl.md). `// Error:` covers both preprocess and runtime panics, so lint must absorb anything on a directive-tagged file. A new `// PreprocessError:` would lift the ceiling; author explicitly declined in the PR thread ([comment](https://github.com/gnolang/gno/pull/5704#issuecomment-4565156077)) on the grounds that `gno test` would still catch mismatches via `// Error:`'s preprocess→runtime unwrap path. Position noted; non-blocking.

## Nits

- [`lint.go:414-434`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L414-L434) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L414-L434) — `excludeExpectedTypeCheckErrors` drops `mpkg.Info` when constructing the filtered copy. The parallel helper [`FilterMemPackage`](https://github.com/gnolang/gno/blob/050597de7/gnovm/pkg/gnolang/mempackage.go#L353-L375) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/pkg/gnolang/mempackage.go#L353-L375) preserves it. The returned mpkg is only used by `lintTypeCheck` (which doesn't read `Info`), so functionally harmless — but inconsistent.
- [`lint.go:345-347`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L345-L347) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L345-L347) — carried from prior review. Comment says "a panic on one ... doesn't skip siblings," but the `panic(r)` re-throw at [`lint.go:364`](https://github.com/gnolang/gno/blob/050597de7/gnovm/cmd/gno/lint.go#L364) · [↗](../../../../../.worktrees/gno-review-5704/gnovm/cmd/gno/lint.go#L364) propagates non-error panics to the outer `catchPanic`, which *does* skip siblings. In practice gno preprocess always panics with errors, so invisible — but the comment overstates the guarantee.
- [`memfile.go:238-240`](https://github.com/gnolang/gno/blob/050597de7/tm2/pkg/std/memfile.go#L238-L240) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L238-L240) — carried from prior review. Doc-comment doesn't note that a stale filetest at the package root will collide on the next `ReadMemPackage`. Cross-link to the [stale-filetest warning above](#warnings-should-fix).
- [`memfile.go:244-249`](https://github.com/gnolang/gno/blob/050597de7/tm2/pkg/std/memfile.go#L244-L249) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L244-L249) — `os.MkdirAll` runs per filetest file. Lift it to the first filetest seen, then reuse. Trivial.

## Missing Tests

- **[no `WriteTo` round-trip coverage]** [`memfile.go:241-257`](https://github.com/gnolang/gno/blob/050597de7/tm2/pkg/std/memfile.go#L241-L257) · [↗](../../../../../.worktrees/gno-review-5704/tm2/pkg/std/memfile.go#L241-L257) — carried from prior review. Codecov still reports 0% patch coverage for the `tm2/pkg/std/memfile.go` change. The two new txtar tests don't exercise `WriteTo`'s filetest-routing in any observable way: they only `cmp stdout/stderr`, never check the on-disk layout, and the unannotated test exits early (STAGE 2 not reached). A `TestMemPackage_WriteTo` unit test asserting `Read → WriteTo → Read` idempotency on a package with both root and `filetests/` files would lock the behavior in.
- **[no test for the stale-filetest collision]** — flagged in the warnings section. A txtar that starts a package with a filetest at the root, runs `gno lint`, runs `gno lint` a second time (or `gno test`) would surface the silent migration failure.

## Suggestions

None beyond the warnings.

## Questions for Author

- For the stale-filetest collision: would you rather have `WriteTo` aggressively clean up root-level `_filetest.gno` files (risky — silent file deletion), or surface a warning at `ReadMemPackage` time pointing to the new layout? The current behavior — silent success on the first lint, dup-filename error on the second — is the worst of both worlds for unmigrated repos.
