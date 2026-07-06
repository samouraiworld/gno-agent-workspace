# PR [#5844](https://github.com/gnolang/gno/pull/5844): test: experiment with audit repair contracts

URL: https://github.com/gnolang/gno/pull/5844
Author: moul | Base: master | Files: 62 | +2528 -73
Reviewed by: davd-gzl | Model: claude-opus-4-8 (high, deep) | Commit: da956ea64 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5844 da956ea64`

> Round 1 upgraded in place with a deep multi-angle pass (red-team / blue-team / correctness lenses + critic + claim verification) on the same commit da956ea64. Verdict unchanged (NEEDS DISCUSSION); two Warnings added over the first pass, both verified with real runs in the worktree.

**TL;DR:** This is a small experimental follow-up stacked on the audit-pattern-harness PR [#5835](https://github.com/gnolang/gno/pull/5835). It teaches each audit-pattern record a `repair` block naming a vulnerable-to-fixed fixture pair, adds `TestRepairContracts` to check that pair (the source flags the pattern, the target does not, the `.gno` files changed, exported top-level function names stay stable), and adds the harness to the CI matrix so its Go tests finally run on every PR.

**Verdict: NEEDS DISCUSSION** — the author states this is not meant to merge urgently, it exists to expose the guide-to-harness-to-repair loop for discussion while [#5835](https://github.com/gnolang/gno/pull/5835) is settled ([moul](https://github.com/gnolang/gno/pull/5844#issuecomment-4864445634)); the harness Go suite and all 8 repair contracts pass, and the CI line makes those tests run per-PR. Two things to weigh in that discussion. First, the check named a repair "contract" proves only that the flagged text pattern is gone and the top-level function names are unchanged, never that the vulnerability is fixed: a target that deletes the guarded feature, or one still exploitable but dodging the heuristic matcher, passes fully green. Second, the exported-API check scans raw file bytes, so a `func` inside a comment or string literal counts as an exported function, which both masks a genuine API removal and rejects otherwise-valid repairs. The prior pass's compile-check finding (target checked by text scan, never compiled) still stands but does not close the first point. The whole layer still sits on the unmerged [#5835](https://github.com/gnolang/gno/pull/5835).

## Summary

The PR's own delta over its base [3700f767f](https://github.com/gnolang/gno/commit/3700f767f) (round 4 of the [#5835](https://github.com/gnolang/gno/pull/5835) review) is 239 lines across the harness module: a `Repair` struct on `Record` with validation, a `repair:` block in each of the 8 `expected/*.yaml`, `TestRepairContracts` plus helpers in `run_test.go`, one README section, and one line adding `audit-pattern-harness` to `ci-dir-misc.yml`. That CI line closes the [#5835](https://github.com/gnolang/gno/pull/5835) round-4 W4 for the pure-Go tests: `TestAgentPatternContract`, `TestRepairContracts`, and the record validation now run on every PR touching `misc/**`. The blast radius is a dev tool under `misc/`; no VM, stdlib, consensus, or deployed realm code changes. The remaining gaps are all in what the contract actually proves: its "target removed the pattern" gate is `RunRule`, a line-scan heuristic that never compiles the target and can be satisfied by an unfixed or gutted target, and its "exported API stable" gate is a raw-text regex over top-level `func` names only.

## Examples

What each `TestRepairContracts` gate proves, and what satisfies it that should not:

| Gate | Check | Passes when it should not |
|------|-------|---------------------------|
| source demonstrates the pattern | `RunRule` hit count > 0 | — |
| target removed the pattern | `RunRule` hit count == 0 | target deletes the feature (gutted, not guarded); target still exploitable but dodges the heuristic (decorative `cur.IsCurrent()`, `escaped := path` returned raw); target does not compile |
| files actually changed | byte compare of `.gno` set | whitespace-only edit |
| exported API stable | `^func [A-Z]\w*(` name set equal, minus `allow_removed_exports` | exported var/const/type removed (`PublicVault`); same-name signature change (`GetVault() *Vault` → `GetVault() Vault`); method / interface-method change; generic `func F[T any](`; a `func` inside a comment or string literal (both false pass and false fail) |

## Glossary

- crossing function — a `.gno` function whose first parameter is `cur realm`; entered via `f(cross, ...)`, which shifts `PreviousRealm`. Appears only in the fixtures, unchanged by this PR.
- heuristic rule — the harness matches audit patterns by scanning text lines, not an AST, so both false positives and false negatives are expected (README-disclaimed).

## Invariant catalog

Walked, no applicable class. The PR delta touches only Go tooling under `misc/audit-pattern-harness` plus one CI line. The only `.gno` files in the diff are the vulnerable/fixed test fixtures, which are inputs to the harness, not deployed realms, and the delta does not modify them. No GnoVM, stdlib, or realm invariant is in scope.

## Critical (must fix)
None.

## Warnings (should fix)

- **[a "repair contract" that a gutted or still-vulnerable fix passes]** [`run_test.go:339-378`](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L339-L378) · [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L339) — the contract proves the flagged text pattern is gone and top-level function names are unchanged, not that the repair fixed anything. A target that removes the guarded code path, or one still exploitable but dodging the heuristic matcher, passes.
  <details><summary>details</summary>

  `TestRepairContracts` gates a record on: source has `RunRule` hits, target has zero, same `.gno` file set, at least one file changed, exported top-level function names equal. None of these ties the fixed fixture to its `goal`, which is only checked non-empty at [run_test.go:344](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L344). `RunRule` is a line-scan heuristic ([#5835](https://github.com/gnolang/gno/pull/5835)'s disclaimed matcher), so "zero hits" means "no longer trips the scan," not "fixed." Verified on da956ea64: replacing `current-guard/fixed/admin.gno`'s `TransferOwnership` body with `// authorization removed entirely` (feature gutted, exported names intact, zero `current_guard` hits) keeps `TestRepairContracts/current-guard` green; a body that keeps a decorative `_ = cur.IsCurrent()` with no panic and no real guard is still owner-spoofable yet also passes, because `current_guard` only checks that `IsCurrent` textually precedes `.Previous()`. This is not the prior pass's compile gap: both targets compile and `gno lint` clean, so building the target does not close it. The `goal` string is the only place the intended remediation is stated, and it is unenforced. Fix: scope the README claim to what the contract proves (pattern gone, API stable), or stop calling it a contract; do not describe a target that passes as a validated "good" fixture an agent can learn from.
  </details>

- **[exported-API check counts `func` inside comments and strings]** [`run_test.go:487-500`](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L487-L500) · [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L487) — `exportedFuncNames` runs `^func [A-Z]\w*(` over raw file bytes, so a `func Name(` at column 0 inside a block comment or a raw string literal is counted as an exported function. This masks a real API removal and also rejects valid repairs.
  <details><summary>details</summary>

  Every `RunRule` matcher scans comment/string-stripped source; `exportedFuncNames` at [run_test.go:487](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L487) alone reads the untouched bytes from `gnoFileContents` and applies `exportedFuncRE` ([run_test.go:15](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L15)). Because the gate is a symmetric set-equality, a phantom name cuts both ways. Verified on da956ea64: deleting the real `func Owner()` from `current-guard/fixed/admin.gno` and appending `/* func Owner() address { return owner } */` keeps `TestRepairContracts/current-guard` green even though the target's public API shrank; adding a `var usage = \`\nfunc Example() string {...}\n\`` doc string to an otherwise-correct `render-markdown/fixed/echo.gno` fails it with `from=[Render] to=[Example Render]`. Fix: extract exported names from parsed source (go/parser, or reuse the harness's existing comment/string stripping), not raw bytes.
  </details>

## Nits
None.

## Missing Tests

- **[repair contract calls a "fixed" target good without compiling it]** [`run_test.go:356-362`](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356-L362) · [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356) — Missing test: `TestRepairContracts` checks the repair target only through `RunRule`, a text scan, so a target that produces zero pattern hits passes even when it does not compile. This is the deferred compile-check; a fix that closes it is below.
  <details><summary>details</summary>

  The `to_fixture` check is `len(toHits) != 0` on `RunRule(rec.Rule, to.Path)` ([run_test.go:356-362](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356-L362)). `RunRule` scans lines for the rule's pattern; it never builds the target. So a repaired fixture that removes the flagged text but references an undeclared identifier still passes the contract. Confirmed on da956ea64: replacing `current-guard/fixed/admin.gno`'s body with `owner = thisDoesNotExist` (same exported names, differs from vulnerable, zero `current_guard` hits) keeps `TestRepairContracts/current-guard` green. The `TestAgentPatternContractWithGNO` compile check that would catch it self-skips without a gno binary, and the misc CI job installs none, so nothing in CI compiles the repair targets. The author flags this as the "next harder version" in the PR body ("verify those outputs compile"), so it is a disclosed limitation of the experiment, not a hidden defect. This is narrower than the first Warning: it only rules out non-compiling targets. A gutted or evasive target that compiles still passes even after this fix.

  Fix (closes the deferred compile-check): build the repair target inside `TestRepairContracts`. Two subtleties. First, `gno test .` is not enough: repair targets have no `_test.gno` file, so `gno test` prints `[no test files]` and exits 0 without type-checking the source. `gno lint .` compiles and type-checks it, so the check uses lint. Second, "when a gno binary is available" is the wrong condition for a contract that must always hold: the misc CI job installs no gno toolchain, so a skip-when-absent check passes silently there. The fix fails loudly instead: absent `GNO_BIN`/`gno`, `TestRepairContracts` fails with an instruction rather than skipping, while `TestAgentPatternContractWithGNO` keeps its opt-in skip. Adopting it also means the misc CI job must install a gno toolchain and export `GNOROOT` so `gno lint` resolves the fixtures' `examples/` imports, else that job goes red. Diff is 32 lines: a `runGNOLint`/`runGNO` split in [`run.go`](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run.go), a shared `lookGNOBin(t, skipIfAbsent)` resolver, and one `runGNOLint` assertion on the target at the end of the per-record block in [`run_test.go`](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go). Verified on da956ea64 (with `GNOROOT` pointed at the PR worktree): the assertion passes on all 8 correct fixtures; breaking `current-guard/fixed/admin.gno` to the hit-free `owner = thisDoesNotExist` body turns `TestRepairContracts/current-guard` red with `admin.gno:6:10: undefined: thisDoesNotExist`, restoring it makes it green again; with no gno binary the test fails rather than skips. Repro test that documents the pre-fix gap: [`tests/repair_compile_gap_test.go`](../../../../../reviews/pr/5xxx/5844-audit-repair-contracts/1-da956ea64/tests/repair_compile_gap_test.go).
  </details>

- **[new validation branches and helpers have no direct unit coverage]** [`record.go:90-131`](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/record.go#L90-L131) · [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/record.go#L90) — Missing test: the new `validate()` repair branches (missing from/to/goal, from==to, unknown fixture, duplicate name) and the new helpers (`sameKeys`, `anyChanged`, `withoutStrings`, `exportedFuncNames`) are exercised only by the 8 real fixtures on the happy path; no test drives a malformed record or asserts a helper output.
  <details><summary>details</summary>

  Every new invariant in [record.go:90-131](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/record.go#L90-L131) is real (a 9th contributor could trip each), yet a refactor that drops or reorders any branch would not turn a test red. One table test locks them; it compiles against the package (`strings` is already imported in `run_test.go`):

  ```go
  func TestValidateRepairRules(t *testing.T) {
  	base := func() Record {
  		return Record{ID: "x", Title: "t", Rule: "r",
  			Repair:   Repair{FromFixture: "a", ToFixture: "b", Goal: "g"},
  			Fixtures: []Fixture{{Name: "a", Path: "p", WantGNOTest: "fail", WantPatternHits: 1}, {Name: "b", Path: "p", WantGNOTest: "pass"}}}
  	}
  	cases := []struct {
  		name string
  		mut  func(*Record)
  		want string
  	}{
  		{"missing from", func(r *Record) { r.Repair.FromFixture = "" }, "missing from_fixture"},
  		{"missing to", func(r *Record) { r.Repair.ToFixture = "" }, "missing to_fixture"},
  		{"missing goal", func(r *Record) { r.Repair.Goal = "" }, "missing goal"},
  		{"from not a fixture", func(r *Record) { r.Repair.FromFixture = "nope" }, "does not match a fixture"},
  		{"to not a fixture", func(r *Record) { r.Repair.ToFixture = "nope" }, "does not match a fixture"},
  		{"from equals to", func(r *Record) { r.Repair.ToFixture = "a" }, "must differ"},
  		{"duplicate fixture name", func(r *Record) { r.Fixtures[1].Name = "a" }, "duplicate name"},
  		{"ok", func(r *Record) {}, ""},
  	}
  	for _, tc := range cases {
  		t.Run(tc.name, func(t *testing.T) {
  			r := base()
  			tc.mut(&r)
  			err := r.validate()
  			if tc.want == "" {
  				if err != nil {
  					t.Fatalf("unexpected error: %v", err)
  				}
  				return
  			}
  			if err == nil || !strings.Contains(err.Error(), tc.want) {
  				t.Fatalf("got %v, want substring %q", err, tc.want)
  			}
  		})
  	}
  }
  ```

  Review-file only; low value for an experiment, folded here for whoever hardens the harness. Not posted.
  </details>

## Suggestions

- [`run_test.go:373-378`](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L373-L378) · [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L373) — the "exported API stable" gate compares only top-level `func` names, so it is blind to exported var/const/type removals, same-name signature changes, methods, interface methods, and generics; and `allow_removed_exports` entries are never validated against the source. Concrete: `exported-pointer-leak` has no `allow_removed_exports`, yet its repair removes the exported `var PublicVault *Vault` (the exact pointer-to-mutable-state the pattern is about, [vault.gno:8](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/fixtures/exported-pointer-leak/vulnerable/vault.gno#L8)) and changes `GetVault`'s return type from `*Vault` to `Vault` ([fixed vault.gno:10](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/fixtures/exported-pointer-leak/fixed/vault.gno#L10)), and the gate sees neither. A misspelled `allow_removed_exports` entry (`SetHok` for `SetHook`) silently disables the guard rather than erroring, since `withoutStrings` ([run_test.go:544](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L544)) filters the source set unconditionally. The prior pass named the method/interface blind spot; this extends it to exported non-func symbols, signatures, and the unvalidated allow-list. Decide whether the gate should cover exported vars/types/signatures or the README should scope the claim to top-level function names.

- [`ci-dir-misc.yml:25`](https://github.com/gnolang/gno/blob/da956ea64/.github/workflows/ci-dir-misc.yml#L25) · [↗](../../../../../.worktrees/gno-review-5844/.github/workflows/ci-dir-misc.yml#L25) — the CI line runs the harness's pure-Go tests but not the gno-compile contract: `TestAgentPatternContractWithGNO` self-skips unless `GNO_BIN` is set or `gno` is on PATH ([run_test.go:319-327](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L319-L327)), and the reusable `_ci-go.yml` job installs no gno toolchain (`grep -n 'GNO_BIN\|make install\|setup-gno' .github/workflows/_ci-go.yml .github/workflows/ci-dir-misc.yml` is empty; the job runs `go test ./...` under `setup-go` only). So the fixed fixtures never compile in CI and the repair targets never run through `gno test`. This partially closes [#5835](https://github.com/gnolang/gno/pull/5835)'s W4. To fully close it, the job also needs a gno binary and a `GNO_BIN` export. Review-file only; the CI line is still a net improvement. `paths: misc/**` and `fail-fast: false` mean a future flaky repair contract would red the `ci / misc` check for unrelated `misc/**` PRs, but no flakiness exists today (deterministic across `-count`/`-shuffle`, all outputs sorted).

## Open questions

- The whole layer is stacked on [#5835](https://github.com/gnolang/gno/pull/5835), which is itself NEEDS DISCUSSION and unmerged. The `da956ea64` head has master-merge commits, so the diff against master carries the full 2528-line harness stack; only the top two commits (the `repair` support and the CI line, 239 lines) are this PR's own contribution. If [#5835](https://github.com/gnolang/gno/pull/5835) lands first, rebasing collapses this to that 239-line delta. Design/sequencing; not posted.
- The exported-API gate rejects a repair that adds a new exported top-level function (verified: appending `func NewlyAdded()` to a fixed fixture fails with `to=[NewlyAdded Owner TransferOwnership]`). That sits in tension with `exported-pointer-leak`'s own `goal` ("expose value-returning read APIs"): a repair that genuinely adds a read API would be rejected. The current fixture dodges this by keeping the same function name and only changing its return type. Design; not posted.
- The mechanically-checkable subset of these audit families overlaps the `gno lint` framework that the open [#5068](https://github.com/gnolang/gno/pull/5068) extends. The repair-contract idea (assert a fixture pair, source flags and target clean) could live there as lint fixtures. Direction; not posted.
