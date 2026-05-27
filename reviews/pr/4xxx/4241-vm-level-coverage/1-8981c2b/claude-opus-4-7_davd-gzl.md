# PR #4241: feat(gnovm): VM-Level Code Coverage Implementation

URL: https://github.com/gnolang/gno/pull/4241
Author: notJoon | Base: master | Files: 19 | +1903 -12
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `8981c2b` (stale — +42 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4241 8981c2b`

**Verdict: REQUEST CHANGES** — coverage report counts test-file lines as production code; coverage mode silently drops the gas meter and force-loads realm dependencies, changing test behavior vs. plain `gno test`; recursive `RLock` in `ShowFileCoverage`; the interface adds dead methods and pays interface-dispatch cost on every `doOpEval`/`doOpExec` even when disabled; `-coverprofile` writes an unparseable text dump.

## Summary
Adds VM-level line coverage to `gno test -cover`. A `CoverageTracker` interface is plugged into `Machine`; `doOpExec` and `doOpEval` call `m.trackCoverageForNode(node)` on every evaluated/executed AST node, recording the source line via the active block's source location. A separate `coverage.Analyzer` walks the package `PackageNode` to register "executable" lines so a coverage % can be computed. `gno test -cover` builds a separate test store, instruments the tested package, prints a textual report, optionally dumps it to a file (`-coverprofile`) and optionally colours the source (`-show`). Three txtar scripts smoke-test the CLI. The PR also re-introduces eager loading of realm imports under `-cover` to work around `unexpected node with location` panics.

## Glossary
- `CoverageTracker` — interface on `Machine` driving the hot-path tracking call.
- `Tracker` — concrete impl in `gnovm/pkg/test/coverage`; holds two `map[pkg]map[file]map[line]…` tables behind a single `sync.RWMutex`.
- `Analyzer` — AST visitor that registers "executable" lines (only statement lines, no expression-level resolution).
- `MPFTest` — mempackage filter that strips `_filetest.gno` and integration-test (`xxx_test`) files but KEEPS regular `_test.gno`.
- `loadFromDir` — closure in `imports.go` that lazily loads a package from disk when the store is queried.

## Fix
Before: `Machine` had no notion of coverage; `gno test` had no `-cover` flag. After: every `Machine` carries a `CoverageTracker` (defaulting to a no-op), and `op_exec`/`op_eval` call into it. A new `coverage` sub-package owns the data tables, the AST walker that identifies executable lines, the textual report, and an ANSI source visualiser. `gno test -cover` builds a coverage-aware store, instruments only the tested package, and emits a report after the test loop. The load-bearing constraint is that `Analyzer.AnalyzePackage` only registers production lines as "executable", and the reporter intersects executed lines with that set ([`report.go:58`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/report.go#L58) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/report.go#L58)) — but that invariant is violated today (see first Critical).

## Critical (must fix)

- **[`_test.gno` counted as production coverage]** [`gnovm/pkg/test/test.go:412-414`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L412-L414) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L412-L414) — analyzer runs on a package node that already contains the test files, so test source is reported as covered/uncovered.
  <details><summary>details</summary>

  `MPFTest.FilterMemPackage` only strips `_filetest.gno` and integration-test files; it KEEPS regular `_test.gno` (see [`mempackage.go:242-246`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/gnolang/mempackage.go#L242-L246) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/mempackage.go#L242-L246)). `m2.RunMemPackageWithOverrides(tmpkg, true)` then loads both production and `_test.gno` files into the store, after which `tgs.GetPackageNode(mpkg.Path)` returns a node whose `Files` contains both. `Analyzer.analyzeFile` walks every file unconditionally ([`analyzer.go:24-26`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/analyzer.go#L24-L26) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/analyzer.go#L24-L26)) and registers their lines as executable. The txtar run proves it:

  ```
  gno.land/r/test/main.gno:      66.7% (8/12 lines)
  gno.land/r/test/main_test.gno: 63.6% (7/11 lines)
  Total Coverage:                65.2% (15/23 lines)
  ```

  The report contains a `_test.gno` row, which is meaningless: test code is what runs the tests, it cannot reasonably have its own "coverage". The headline percentage is also wrong because the denominator includes test-file lines. Fix: skip files whose name ends in `_test.gno`/`_filetest.gno` inside `Analyzer.analyzeFile`, or filter `pn.Files` before walking.
  </details>

- **[`-cover` silently drops the gas meter]** [`gnovm/pkg/test/test.go:574-580`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L574-L580) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L574-L580) — coverage mode replaces `store.NewInfiniteGasMeter()` with `nil`, so `-cover` runs tests under a different machine config than plain `gno test`.
  <details><summary>details</summary>

  The non-coverage branch passes `store.NewInfiniteGasMeter()` to the per-test machine; the coverage branch goes through `MachineWithCoverage` ([`test.go:88-95`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L88-L95) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L88-L95)) which does not pass a `GasMeter` at all, leaving `m.GasMeter == nil`. The fact that the PR has to add `if … && m.GasMeter != nil` at [`test.go:662`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L662) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L662) to avoid a nil-deref shows the divergence is intentional but undocumented. Consequences: `--print-runtime-metrics` produces nothing under `-cover`; any in-VM code path that branches on a present gas meter (`machine.go:1256-1258`) behaves differently; gas-bounded panics that would fire in plain `gno test` cannot fire under `-cover`. `-cover` should preserve the same gas meter as the non-coverage path. Fix: have `MachineWithCoverage` accept a `GasMeter` (or drop it entirely and just pass `CoverageTracker` through `MachineOptions` at the existing call sites).
  </details>

- **[`-cover` changes realm test semantics]** [`gnovm/pkg/test/test.go:403-407`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L403-L407) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L403-L407) and [`gnovm/pkg/test/imports.go:405-419`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/imports.go#L405-L419) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/imports.go#L405-L419) — coverage mode opts realms into eager dependency loading that the existing code intentionally avoids.
  <details><summary>details</summary>

  Master deliberately skips eager realm loading: "Realms persist state and can change the state of other realms in initialization." The PR sets `loadRealmDeps := opts.Coverage && gno.IsRealmPath(mpkg.Path)` and threads it through `LoadImportsWithOptions`, so under `-cover` every imported realm gets initialised before the test runs. That can mutate shared realm state, flip the result of an `init`, or make a test pass under `-cover` that fails without it (or vice-versa). The in-line comment in `imports.go` admits the workaround is for `unexpected node with location` panics — meaning the real bug is elsewhere and is being papered over by changing semantics for one flag. Fix: root-cause the panic instead of switching realm-load policy on `-cover`; until then, `-cover` on realms should error rather than silently change behaviour.
  </details>

- **[recursive `RLock` can deadlock]** [`gnovm/pkg/test/coverage/visualize.go:22-28`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/visualize.go#L22-L28) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L22-L28) — `ShowFileCoverage` acquires `t.mu.RLock()` and then calls `t.GenerateReport()`, which acquires `t.mu.RLock()` again.
  <details><summary>details</summary>

  `sync.RWMutex` makes no guarantee that a goroutine already holding the read lock can acquire it again — if another goroutine calls `Lock` between the two `RLock`s, the second `RLock` blocks waiting for the writer, and the writer blocks waiting for the first reader to release: a classic three-party deadlock. Today `ShowFileCoverage` is only called from a single goroutine in `cmd/gno/test.go:409`, so the bug is latent, but the lock is part of a "thread-safe" API (`README.md:8`) and the constructor is exported. Fix: drop the outer `RLock` (the inner one in `GenerateReport` is sufficient), or factor out a `generateReportLocked` helper that runs under the already-held read lock.
  </details>

## Warnings (should fix)

- **[interface dispatch on the VM hot path even with `-cover` off]** [`gnovm/pkg/gnolang/op_exec.go:459-462`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/gnolang/op_exec.go#L459-L462) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/op_exec.go#L459-L462) and [`op_eval.go:27-30`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/gnolang/op_eval.go#L27-L30) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/op_eval.go#L27-L30) — every executed statement and evaluated expression now pays an `iface.IsEnabled()` call.
  <details><summary>details</summary>

  `m.CoverageTracker` is an interface; `IsEnabled()` is a virtual call. The default is `*NopCoverageTracker` whose `IsEnabled()` returns `false`, but the dispatch itself is on the hottest path in the interpreter, executed millions of times during chain replay/test runs. Even the no-op cost shows up under load. The `BenchmarkSimpleLoopWith[out]Coverage` pair in `benchmark_test.go` doesn't surface this — it only contrasts the two enabled states, not "instrumented but disabled" vs. "uninstrumented master". Fix: store a plain `bool m.CoverageEnabled` set once in `NewMachineWithOptions` and gate the call on that; or guard with a nil check on a non-defaulted pointer field. Either avoids the v-call for 100% of production traffic.
  </details>

- **[`-coverprofile` writes an unparseable text dump]** [`gnovm/pkg/test/test.go:509-512`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L509-L512) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L509-L512) — the file is just `Report.String()`, not Go's `mode: set/count/atomic` format.
  <details><summary>details</summary>

  Users will reach for `go tool cover -html=coverage.out` (or `gocov`, `gocovmerge`, codecov uploader, etc.) and get a parse error because the file is a human-readable table, not the `mode:`/`file:start.col,end.col stmts count` triples those tools expect. The flag name `-coverprofile` mirrors Go's, so the format expectation is the same. The in-code `TODO: Implement proper file output formats (JSON, HTML, etc.)` confirms it's a placeholder. Fix: emit Go's textual cover profile format (`mode: count` + `file:line.col,line.col N S` lines) so the existing tooling works. If that's not feasible, rename the flag to `-coverreport` so the format expectation doesn't carry over.
  </details>

- **[`-show` can't find source files for normal user packages]** [`gnovm/pkg/test/coverage/visualize.go:128-148`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/visualize.go#L128-L148) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L128-L148) — the candidate list only checks `pkgPath`, `<root>/examples/<pkgPath>`, `<root>/gnovm/stdlibs/<pkgPath>`, `<root>/<pkgPath>`.
  <details><summary>details</summary>

  The txtar run shows this failing for a `gno.land/r/test` package compiled from the script's tmpdir:

  ```
  Error showing coverage: cannot find source file: gno.land/r/test/main.gno
  ```

  Outside the `examples/` tree, no candidate matches. Fix: pass the on-disk directory (the `fsDir` already available in `Test()` at [`test.go:347`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L347) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L347), or the `pkg.Dir` from `cmd/gno/test.go`) into the tracker and use it as the first candidate.
  </details>

- **[dead interface surface enlarges the contract for no benefit]** [`gnovm/pkg/gnolang/coverage_interface.go:10-13`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/gnolang/coverage_interface.go#L10-L13) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/coverage_interface.go#L10-L13) — `TrackStatement(Stmt)` and `TrackExpression(Expr)` are defined, exported, mocked, and never called.
  <details><summary>details</summary>

  Greps for `TrackStatement(` / `TrackExpression(` outside the interface, mock, and no-op return zero call sites. The actual code path is `Machine.trackCoverageForNode → CoverageTracker.TrackExecution`. Every reimplementor of the interface now has to implement two methods that do nothing; readers spend time figuring out which method the VM actually calls. Same critique applies to `Tracker.SetCoverageData` and `Tracker.SetExecutableLines` ([`tracker.go:150-166`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/tracker.go#L150-L166) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/tracker.go#L150-L166)) — only callers are tests that never use the result. Fix: delete the unused methods (and the mock copies) until a caller materialises.
  </details>

- **[counts are conflated across `tset` and `itset` runs]** [`gnovm/pkg/test/test.go:424-450`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L424-L450) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L424-L450) — the tracker is never `Clear()`-ed between in-package tests and `xxx_test` integration tests, so coverage from the first run is added to the second.
  <details><summary>details</summary>

  `Tracker.TrackExecution` increments `coverage[pkg][file][line]++`. Both `runTestFiles(mpkg, tset, …)` and `runTestFiles(itmpkg, itset, …)` get the same `covTracker`, and no reset happens between them. For pure line coverage (set semantics) this is fine, but the report prints execution counts and the `coverage.out` file claims to be a profile; conflating runs makes the counts misleading. Fix: either clear between runs or document explicitly that counts are package-aggregate, not per-test.
  </details>

- **[duplicate skip-tested-package check is dead code]** [`gnovm/pkg/test/imports.go:218-222`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/imports.go#L218-L222) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/imports.go#L218-L222) and [`imports.go:256-259`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/imports.go#L256-L259) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/imports.go#L256-L259) — both check `opts.Coverage && pkgPath == opts.TestedPackagePath`, but `StoreOptions.TestedPackagePath` is never assigned by `cmd/gno/test.go`.
  <details><summary>details</summary>

  `cmd/gno/test.go:262-269` constructs `StoreOptions{WithExamples: true, Testing: true, Coverage: true, Packages: pkgs}` — `TestedPackagePath` is the zero string. The lone `opts.TestPackagePath = mpkg.Path` later in the loop sets the field on `TestOptions`, a different struct. So the skip never fires and the tested package is loaded from disk by `loadFromDir`, then loaded a second time by `m2.RunMemPackageWithOverrides`. This is also why `m.RunMemPackage(mpkg, false)` is gated on `tgs.GetMemPackage(mpkg.Path) == nil` ([`test.go:557`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L557) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L557)) — it's masking the duplicate load. Fix: either remove the skip-checks (since they don't fire and the second-load guard already handles it), or actually plumb `mpkg.Path` into `StoreOptions.TestedPackagePath`.
  </details>

- **[`opts.TestPackagePath = mpkg.Path` is set twice and read nowhere]** [`gnovm/pkg/test/test.go:361`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L361) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L361) and [`cmd/gno/test.go:352`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/cmd/gno/test.go#L352) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/cmd/gno/test.go#L352) — the field is mutated but no code path reads it.
  <details><summary>details</summary>

  `grep -n TestPackagePath` in the worktree shows only the two writes and the struct field declaration; no reader. Probably a left-over from an earlier design that used the field to gate the `loadFromDir` skip. Fix: delete the field and both writes.
  </details>

- **[ADR missing for a non-trivial VM change]** repo-level — `AGENTS.md` mandates an ADR for non-trivial AI-assisted GnoVM PRs; none was added under `gnovm/adr/`.
  <details><summary>details</summary>

  The PR description explicitly narrates a design pivot ("shifted from an initial AST-based instrumentation strategy to a machine-level solution … `MemPackageType` system, which enforces strict separation between production and test code"), which is exactly the kind of context an ADR captures so reviewers and future contributors can verify the assumption. The change adds an interface to `Machine`, hooks both top-level opcodes, and bends realm-load semantics under a flag — well past "trivial". If the work was AI-assisted, add `gnovm/adr/pr4241_vm_level_coverage.md` per the rule; if not, at minimum justify the realm-load workaround in code comments.
  </details>

## Nits

- [`gnovm/pkg/test/coverage/tracker.go:11`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/tracker.go#L11) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/tracker.go#L11) — `map[string]map[string]map[int]int64` triple-nested map duplicates `pkgPath` strings as map keys in two places (`coverage` and `executableLines`). For large workloads consider a struct key `{pkg, file}` or a flat key.
- [`gnovm/pkg/test/coverage/visualize.go:80-85`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/visualize.go#L80-L85) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L80-L85) — `bufio.Scanner` default token size will truncate lines >64KiB; for generated `.gno` this can hit. Use `scanner.Buffer(...)`.
- [`gnovm/pkg/test/coverage/visualize.go:13-19`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/visualize.go#L13-L19) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L13-L19) — emits raw ANSI even when stderr is not a TTY (e.g. CI logs, redirected output). Gate on `term.IsTerminal(int(os.Stderr.Fd()))` or accept a `NO_COLOR` env.
- [`gnovm/pkg/test/coverage/analyzer.go:166-176`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/analyzer.go#L166-L176) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/analyzer.go#L166-L176) — `AnalyzeMemPackage` is a public stub. Delete or implement.
- [`gnovm/pkg/test/test.go:509`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L509) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L509) — `os.WriteFile(..., 0644)` should be `0o644` to match the file's style.
- [`gnovm/pkg/gnolang/machine.go:2841`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/gnolang/machine.go#L2841) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/machine.go#L2841) — `trackCoverageForNode` repeatedly defends against nil/zero — most checks are unreachable on the real call path. Keeping the `IsEnabled()` gate at the caller and a single nil-node check is enough.
- [`gnovm/pkg/test/coverage/README.md:1`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/README.md#L1) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/README.md#L1) — README says "Comprehensive coverage" but the impl is line-only (no branch/condition). Tighten wording.

## Missing Tests

- **[realm under `-cover`]** no test exercises the very code path the PR introduces (`loadRealmDeps`).
  <details><summary>details</summary>

  The three txtar scripts (`basic.txtar`, `cover.txtar`, `coverprofile.txtar`) all use the same `gno.land/r/test` realm but with NO realm imports, so `loadRealmDeps` never triggers. The original motivation ("`unexpected node with location` panics when the tested realm calls functions from other realms") has no regression test. Add a txtar that imports another `r/...` realm and asserts both no-panic and a stable coverage report.
  </details>

- **[same package, `-cover` vs no `-cover`]** the report and the gas-meter divergence (Critical #2/#3) have no test asserting parity of pass/fail outcomes between the two invocations.
  <details><summary>details</summary>

  A txtar that runs `gno test .` and `gno test -cover .` on the same package and asserts identical PASS/FAIL would catch any future regression where coverage changes test semantics. Today this would already fail for realms that depend on other realms because of `loadRealmDeps`.
  </details>

- **[`ShowFileCoverage` concurrent access]** [`gnovm/pkg/test/coverage/visualize.go:22-28`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/visualize.go#L22-L28) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L22-L28) — no test exercises the lock interleaving.
  <details><summary>details</summary>

  A test that spawns one writer goroutine repeatedly calling `TrackExecution` and one reader calling `ShowFileCoverage`, under `-race`, would surface the recursive-RLock deadlock today. Worth adding before the API gets external consumers.
  </details>

- **[analyzer coverage of compound statements]** [`analyzer.go:67-163`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/analyzer.go#L67-L163) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/analyzer.go#L67-L163) — the switch doesn't enumerate every stmt type and there's no test verifying which lines are registered.
  <details><summary>details</summary>

  `analyzeStmt` has no case for `SendStmt`, no recursion into `GoStmt`/`DeferStmt` call expressions, and treats every `_test.gno`-only construct the same. A table-driven test that feeds parsed Gno snippets through the analyzer and asserts the resulting executable-line set would lock down the contract.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/coverage_interface.go:5-20`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/gnolang/coverage_interface.go#L5-L20) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/coverage_interface.go#L5-L20) — shrink the interface to `TrackExecution(pkgPath, file string, line int)` + `IsEnabled() bool`. The mutator `SetEnabled` belongs on the concrete `Tracker`, not on the VM's view of it.
- [`gnovm/pkg/test/test.go:88-95`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/test.go#L88-L95) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L88-L95) — `MachineWithCoverage` duplicates `Machine` for no good reason once it carries a `GasMeter`. Add an optional `CoverageTracker` to the existing `Machine(...)` helper instead.
- [`gnovm/pkg/test/coverage/report.go:94-111`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/pkg/test/coverage/report.go#L94-L111) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/report.go#L94-L111) — `Report.String()` always emits to stderr via `fmt.Fprint(opts.Error, …)` in `test.go:506`; consider separating "summary line" (always) from "per-file table" (verbose only) to keep CI logs scannable.
- [`gnovm/cmd/gno/test.go:404-419`](https://github.com/gnolang/gno/blob/8981c2b/gnovm/cmd/gno/test.go#L404-L419) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/cmd/gno/test.go#L404-L419) — surfacing the tracker via `opts.CoverageTracker` from `test.Test()` is a layering inversion (`cmd/gno` reaches back into a package-private impl detail by type-asserting `*vmcoverage.Tracker`). Expose a `ShowFileCoverage` method on `TestOptions` (or have `test.Test` accept a `show` argument) so `cmd/gno` doesn't depend on the concrete tracker type.

## Questions for Author

- Why is the realm-init panic a `-cover`-only workaround instead of a fix? The `unexpected node with location` panic the comment refers to is a parser/preprocess bug that exists irrespective of coverage; flipping load policy under a flag hides it. What's blocking the root-cause fix?
- Is `_test.gno` being counted in coverage intentional? If so, what's the user model — "test coverage of test code" makes the headline percentage meaningless.
- `-coverprofile` — is the long-term plan to match Go's `mode: count` format, or to define a Gno-specific JSON/HTML format? If the latter, renaming the flag now (before this lands) prevents the ecosystem from forming the wrong expectation.
- The PR also removes a `// XXX: add per-test metrics` comment in `test.go`. Is that intentional cleanup, or accidental?
