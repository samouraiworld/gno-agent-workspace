# PR #4241: feat(gnovm): VM-Level Code Coverage Implementation

URL: https://github.com/gnolang/gno/pull/4241
Author: notJoon | Base: master | Files: 19 | +1907 -13
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `ae7f297b4` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4241 ae7f297b4`

Round 2 of 2. Round 1 reviewed `8981c2b` ([`1-8981c2b/claude-opus-4-7_davd-gzl.md`](../1-8981c2b/claude-opus-4-7_davd-gzl.md)).

**Verdict: REQUEST CHANGES** — nothing from round 1 was addressed. The only change since `8981c2b` is a single master merge (`ae7f297b4`), and its conflict resolution broke gofmt at `machine.go:98` (space-indented comment), which now fails `main / build` and `main / lint` on CI. All four round-1 Criticals stand verbatim: `_test.gno` counted as production coverage (reproduced), `-cover` drops the gas meter, `-cover` flips realm dependency-load policy, recursive `RLock` in `ShowFileCoverage`. No ADR added.

## What changed since round 1

`8981c2b` is a direct ancestor of head; `ae7f297b4` is `Merge branch 'master' into instrumental-coverage` (parents `8981c2b` + master `d8a351e7`). No coverage logic was touched. The merge only:

- absorbed master's interrealm-v2 / allocator changes into `test.go` (forced `MaxAllocBytes`, `NewOriginRealmTV` seeding of `cur`) and `op_exec.go` (`Assign2`/`isEql`/`GetPointerAtIndex` signature churn, nil-pointer-range fix), and dropped `FixFrom` from `StoreOptions` in `imports.go` — all master-driven, none coverage-specific;
- introduced a **new** gofmt defect at [`machine.go:98`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/gnolang/machine.go#L98) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/machine.go#L98): the `CoverageTracker` field was inserted directly above master's new `BoundedPanicRender` doc block, and the first comment line got two-space indentation instead of a tab.

Verified every round-1 finding is still live at head (line numbers below are head). No new coverage functionality, no fixes.

## CI status

`main / build` and `main / lint` both fail, same root cause:

```
machine.go:98:1: File is not properly formatted (gofmt)
machine.go:98:1: File is not properly formatted (goimports)
make generate creates files that differ from the git tree
```

`go build ./gnovm/...` and the `coverage` / `gnolang` unit tests pass locally — the failure is purely the formatting/`make generate` gate, not compilation.

## Critical (must fix)

- **[CI red — gofmt break introduced by the merge]** [`gnovm/pkg/gnolang/machine.go:98`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/gnolang/machine.go#L98) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/machine.go#L98) — comment line indented with two spaces instead of a tab; fails `gofmt`, `goimports`, and `make generate`.
  <details><summary>details</summary>

  New in `ae7f297b4`. The merge slotted the PR's `CoverageTracker` field into `MachineOptions` immediately above master's `BoundedPanicRender` doc block and left the first comment line space-indented. `gofmt -d` flags exactly this line, and the `main / build` job's `make generate` diff check rejects the tree. Build and lint are hard-required. Fix: `gofmt -w gnovm/pkg/gnolang/machine.go` (replace the two leading spaces on line 98 with a tab) and re-run `make generate`.

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 4241 -R gnolang/gno
  gofmt -l gnovm/pkg/gnolang/machine.go
  ```

  ```
  gnovm/pkg/gnolang/machine.go
  ```
  </details>

- **[`_test.gno` counted as production coverage]** (round 1, unaddressed) [`gnovm/pkg/test/test.go:418-423`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L418-L423) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L418-L423) — `AnalyzePackage` walks a package node that includes the test files, so `_test.gno` shows its own coverage row and inflates the denominator.
  <details><summary>details</summary>

  `Analyzer.AnalyzePackage` iterates `pn.Files` ([`analyzer.go:24-26`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/analyzer.go#L24-L26) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/analyzer.go#L24-L26)) with no `_test.gno` filter; `MPFTest` keeps regular `_test.gno`, so the loaded node carries both production and test files. Reproduced at head:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 4241 -R gnolang/gno
  make -C gnovm install.gno >/dev/null 2>&1 || go build -o /tmp/gno ./gnovm/cmd/gno
  d=$(mktemp -d); cat > "$d/main.gno" <<'EOF'
  package testcoverage
  func Add(a, b int) int { return a + b }
  func Subtract(a, b int) int { return a - b }
  func Multiply(a, b int) int { r := 0; for i := 0; i < b; i++ { r += a }; return r }
  EOF
  cat > "$d/main_test.gno" <<'EOF'
  package testcoverage
  import "testing"
  func TestAdd(t *testing.T) { if Add(2,3)!=5 { t.Error("bad") } }
  EOF
  printf 'module = "gno.land/r/test"\ngno = "0.9"\n' > "$d/gnomod.toml"
  (cd "$d" && GNOROOT=$OLDPWD /tmp/gno test -cover .)
  rm -rf "$d"
  ```

  ```
  Coverage Report
  ===============

  gno.land/r/test/main.gno: 20.0% (1/5 lines)
  gno.land/r/test/main_test.gno: 100.0% (1/1 lines)

  Total Coverage: 33.3% (2/6 lines)
  ok      . 	0.80s
  ```

  The `main_test.gno` row is meaningless and the headline 33.3% is wrong: the denominator counts a test-file line. Fix: skip files ending in `_test.gno`/`_filetest.gno` inside `analyzeFile`, or filter `pn.Files` before walking. Adversarial txtar: [`tests/cover_excludes_testfiles.txtar`](tests/cover_excludes_testfiles.txtar).
  </details>

- **[`-cover` silently drops the gas meter]** (round 1, unaddressed) [`gnovm/pkg/test/test.go:587-591`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L587-L591) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L587-L591) — coverage branch uses `MachineWithCoverage` (no gas meter); non-coverage branch passes `store.NewInfiniteGasMeter()`.
  <details><summary>details</summary>

  Inside the per-test loop the two branches diverge: `MachineWithCoverage(...)` at [`test.go:588`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L588) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L588) takes no `GasMeter`, while `Machine(..., store.NewInfiniteGasMeter())` at [`test.go:590`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L590) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L590) does. `MachineWithCoverage` ([`test.go:88-95`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L88-L95) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L88-L95)) leaves `GasMeter` nil. The guard `m.GasMeter != nil` at [`test.go:690`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L690) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L690) confirms the divergence is known. Consequence: `-cover` runs tests under a different machine config than plain `gno test` (no runtime metrics, no gas-bounded panics). Fix: give `MachineWithCoverage` a `GasMeter`, or fold `CoverageTracker` into the existing `Machine(...)` so both paths share one config.
  </details>

- **[`-cover` changes realm test semantics]** (round 1, unaddressed) [`gnovm/pkg/test/test.go:412`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L412) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L412) and [`gnovm/pkg/test/imports.go:409-417`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/imports.go#L409-L417) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/imports.go#L409-L417) — `loadRealmDeps := opts.Coverage && gno.IsRealmPath(mpkg.Path)` opts realms into eager dependency loading the non-coverage path deliberately avoids.
  <details><summary>details</summary>

  The in-line comment still states the workaround papers over `unexpected node with location` panics, and the non-coverage path still skips eager realm loading because "Realms persist state and can change the state of other realms in initialization." Threading `loadRealmDeps` only under `-cover` means a realm test can pass/fail differently with vs without the flag. The author noted the panic is only "partially fixed" (issue comment, 2025-06-25). Fix: root-cause the panic rather than switching load policy on a flag; until then, error on `-cover` for realms instead of silently changing behavior.
  </details>

- **[recursive `RLock` can deadlock]** (round 1, unaddressed) [`gnovm/pkg/test/coverage/visualize.go:23-28`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/visualize.go#L23-L28) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L23-L28) — `ShowFileCoverage` holds `t.mu.RLock()` then calls `GenerateReport()`, which re-acquires `t.mu.RLock()` at [`report.go:30`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/report.go#L30) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/report.go#L30).
  <details><summary>details</summary>

  `sync.RWMutex` gives no re-entrancy guarantee: a writer arriving between the two `RLock`s deadlocks both. Latent today (single caller), but the README advertises a thread-safe API and the type is exported. Fix: drop the outer `RLock` (inner one suffices) or factor a `generateReportLocked` helper run under the held read lock.
  </details>

## Warnings (should fix)

All carried from round 1, still present at head:

- **[interface dispatch on the VM hot path with `-cover` off]** [`gnovm/pkg/gnolang/op_exec.go:476-477`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/gnolang/op_exec.go#L476-L477) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/op_exec.go#L476-L477) and [`op_eval.go:28-29`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/gnolang/op_eval.go#L28-L29) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/op_eval.go#L28-L29) — every executed statement / evaluated expression pays a virtual `CoverageTracker.IsEnabled()` call even when the default `NopCoverageTracker` is installed.
  <details><summary>details</summary>

  The gate is an interface method call on the interpreter's hottest path, executed millions of times during replay/test runs. The default returns `false`, but the dispatch cost is unconditional. Fix: cache a plain `bool m.CoverageEnabled` set once in `NewMachineWithOptions` and branch on that; or guard with a nil check on a non-defaulted pointer field.
  </details>

- **[`-coverprofile` writes an unparseable text dump]** [`gnovm/pkg/test/test.go:516-518`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L516-L518) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L516-L518) — the file is `Report.String()`, not Go's `mode:`/`file:start.col,end.col stmts count` profile.
  <details><summary>details</summary>

  The flag name `-coverprofile` mirrors Go's, so users will run `go tool cover -html=...` and hit a parse error. Fix: emit Go's textual cover-profile format, or rename to `-coverreport` so the expectation doesn't carry over.
  </details>

- **[`-show` can't find source for non-`examples/` packages]** [`gnovm/pkg/test/coverage/visualize.go`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/visualize.go#L55-L62) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L55-L62) — candidate list misses packages compiled outside the examples tree; `cannot find source file: <pkg>/<file>`.
  <details><summary>details</summary>

  Same gap as round 1: the on-disk dir (`fsDir` in `Test()` / `pkg.Dir` in `cmd/gno`) is not threaded into the tracker, so a tmpdir realm can't be visualized. Fix: pass the source dir as the first candidate.
  </details>

- **[dead interface surface: `TrackStatement` / `TrackExpression`]** [`gnovm/pkg/gnolang/coverage_interface.go:9-13`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/gnolang/coverage_interface.go#L9-L13) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/coverage_interface.go#L9-L13) — defined, mocked, implemented, never called.
  <details><summary>details</summary>

  The real path is `trackCoverageForNode → TrackExecution`; greps for `TrackStatement(`/`TrackExpression(` find only the interface, the `Nop` impl, and the `Tracker` impl ([`tracker.go:51,62`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/tracker.go#L51-L62) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/tracker.go#L51-L62)), no callers. Same for `SetCoverageData` / `SetExecutableLines` ([`tracker.go:152,161`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/tracker.go#L152-L161) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/tracker.go#L152-L161)) — used only by tests. Fix: delete until a caller materializes.
  </details>

- **[counts conflated across `tset` and `itset` runs]** [`gnovm/pkg/test/test.go:435,456`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L435) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L435-L456) — the tracker is never `Clear()`-ed between in-package and `xxx_test` runs.
  <details><summary>details</summary>

  Both `runTestFiles` calls share one `covTracker`; `Clear()` exists ([`tracker.go:142`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/tracker.go#L142) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/tracker.go#L142)) but is only called from a unit test. Fine for set-semantics line coverage, misleading for the execution counts the report prints. Fix: clear between runs, or document that counts are package-aggregate.
  </details>

- **[dead skip-check on unassigned `TestedPackagePath`]** [`gnovm/pkg/test/imports.go:217,254`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/imports.go#L217) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/imports.go#L217-L254) — both branches test `opts.Coverage && pkgPath == opts.TestedPackagePath`, but `StoreOptions.TestedPackagePath` is never assigned.
  <details><summary>details</summary>

  `grep -rn TestedPackagePath gnovm/` shows only the field declaration and the two reads, no write. `cmd/gno/test.go` constructs `StoreOptions{...Coverage:true...}` without it, and `opts.TestPackagePath = mpkg.Path` ([`test.go:361`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L361) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L361)) sets a different struct's field. The skip never fires; the second-load guard at [`test.go:568`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L568) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L568) masks the duplicate load. Fix: remove the dead checks, or actually plumb `mpkg.Path` into `TestedPackagePath`.
  </details>

- **[`opts.TestPackagePath` written, read nowhere]** [`gnovm/pkg/test/test.go:361`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L361) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L361) and [`gnovm/cmd/gno/test.go:352`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/cmd/gno/test.go#L352) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/cmd/gno/test.go#L352) — field mutated twice, no reader.
  <details><summary>details</summary>

  `grep -n TestPackagePath` finds only the declaration and the two writes. Fix: delete the field and both writes.
  </details>

- **[no ADR for a non-trivial VM change]** repo-level — `AGENTS.md` mandates an ADR under `gnovm/adr/` for non-trivial AI-assisted GnoVM PRs; none present.
  <details><summary>details</summary>

  The PR adds an interface to `Machine`, hooks both top-level opcodes, and bends realm-load policy under a flag — past "trivial". The description narrates a design pivot (AST instrumentation → machine-level) that an ADR exists to capture. Add `gnovm/adr/pr4241_vm_level_coverage.md`.
  </details>

## Nits

All carried from round 1, still present:

- [`gnovm/pkg/test/test.go:518`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L518) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L518) — `os.WriteFile(..., 0644)` should be `0o644`; the sibling write at [`test.go:487`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/test.go#L487) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/test.go#L487) already uses `0o644`.
- [`gnovm/pkg/test/coverage/analyzer.go:166-176`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/analyzer.go#L166-L176) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/analyzer.go#L166-L176) — `AnalyzeMemPackage` is a public stub with no callers. Delete or implement.
- [`gnovm/pkg/test/coverage/visualize.go`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/visualize.go#L10-L18) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/visualize.go#L10-L18) — emits raw ANSI unconditionally; gate on `term.IsTerminal` or honor `NO_COLOR`.
- [`gnovm/pkg/test/coverage/tracker.go:11`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/test/coverage/tracker.go#L11) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/test/coverage/tracker.go#L11) — triple-nested `map[string]map[string]map[int]…` duplicates pkg/file keys across `coverage` and `executableLines`; consider a struct key.
- [`gnovm/pkg/gnolang/machine.go:3024-3057`](https://github.com/gnolang/gno/blob/ae7f297b4/gnovm/pkg/gnolang/machine.go#L3024-L3057) · [↗](../../../../../.worktrees/gno-review-4241/gnovm/pkg/gnolang/machine.go#L3024-L3057) — `trackCoverageForNode` has six sequential nil/zero guards; most are unreachable on the real call path. The `IsEnabled()` caller gate plus one nil-node check suffices.

## Missing Tests

Carried from round 1, still absent:

- **[realm under `-cover`]** no test exercises `loadRealmDeps`; the three coverage txtars use `gno.land/r/test` with no realm imports, so the panic-workaround path is never hit.
- **[`-cover` vs no `-cover` parity]** no test asserts identical PASS/FAIL between `gno test .` and `gno test -cover .` on the same package (would catch the gas-meter and realm-load divergences).
- **[`ShowFileCoverage` under `-race`]** no concurrent-access test to surface the recursive-RLock deadlock.
- **[`_test.gno` exclusion]** no test asserts test files are excluded from the report; the bug ships untested. Adversarial txtar provided: [`tests/cover_excludes_testfiles.txtar`](tests/cover_excludes_testfiles.txtar) (asserts the current buggy output, with the post-fix assertion one comment-flip away).

## Questions for Author

- Why merge master without re-running `gofmt`/`make generate`? Build and lint are red on exactly the line the merge touched. Please re-run the format gate before the next push.
- Realm-init panic is still only "partially fixed" per your 2025-06-25 comment. Is the root-cause fix in scope for this PR, or will `-cover` ship with the `loadRealmDeps` workaround?
- `_test.gno` rows in the report and the inflated denominator — intended, or a bug? If intended, what's the user model for "coverage of test code"?
- `-coverprofile` — match Go's `mode: count` format, or define a Gno-specific format? If the latter, rename the flag before this lands.
