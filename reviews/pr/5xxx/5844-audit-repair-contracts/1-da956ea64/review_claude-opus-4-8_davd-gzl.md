# PR [#5844](https://github.com/gnolang/gno/pull/5844): test: experiment with audit repair contracts

URL: https://github.com/gnolang/gno/pull/5844
Author: moul | Base: master | Files: 62 | +2528 -73
Reviewed by: davd-gzl | Model: claude-opus-4-8 (high) | Commit: da956ea64 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5844 da956ea64`

**TL;DR:** This is a small experimental follow-up stacked on the audit-pattern-harness PR [#5835](https://github.com/gnolang/gno/pull/5835). It teaches each audit-pattern record a `repair` block naming a vulnerable-to-fixed fixture pair, adds `TestRepairContracts` to check that pair (the source flags the pattern, the target does not, the `.gno` files changed, exported top-level function names stay stable), and adds the harness to the CI matrix so its Go tests finally run on every PR.

**Verdict: NEEDS DISCUSSION** — the author states this is not meant to merge urgently, it exists to expose the guide-to-harness-to-repair loop for discussion while [#5835](https://github.com/gnolang/gno/pull/5835) is settled ([moul](https://github.com/gnolang/gno/pull/5844#issuecomment-4864445634)); the harness Go suite and all 8 repair contracts pass, and the CI line makes those tests run per-PR. The repair contract checks the repaired fixture with a text scan only, so it accepts a target that does not compile: the "verify those outputs compile" step the PR body defers. The Missing Tests section carries a 32-line fix that closes it (build the target with `gno lint`, fail loudly when no gno binary is present). The whole layer still sits on an unmerged base. Decide whether to land the CI line and record support now and defer the repair experiment, or hold the whole thing behind [#5835](https://github.com/gnolang/gno/pull/5835).

## Summary

The PR's own delta over its base [3700f767f](https://github.com/gnolang/gno/commit/3700f767f) (round 4 of the [#5835](https://github.com/gnolang/gno/pull/5835) review) is 239 lines across the harness module: a `Repair` struct on `Record` with validation, a `repair:` block in each of the 8 `expected/*.yaml`, `TestRepairContracts` plus helpers in `run_test.go`, one README section, and one line adding `audit-pattern-harness` to `ci-dir-misc.yml`. That CI line closes the [#5835](https://github.com/gnolang/gno/pull/5835) round-4 W4 for the pure-Go tests: `TestAgentPatternContract`, `TestRepairContracts`, and the record validation now run on every PR touching `misc/**`. The blast radius is a dev tool under `misc/`; no VM, stdlib, consensus, or deployed realm code changes. The remaining gap is that the repair contract's "target removes the pattern" check runs `RunRule`, a text scan, and never compiles the repaired fixture, so a repaired target that is hit-free but broken passes. The Missing Tests section supplies a fix for that deferred compile-check.

## Examples

Repair invariants `TestRepairContracts` enforces per record, and what each does not catch:

| Invariant | Check | Not covered |
|-----------|-------|-------------|
| source demonstrates the pattern | `RunRule` hit count > 0 | — |
| target removes the pattern | `RunRule` hit count == 0 | target need not compile (fix in Missing Tests) |
| files actually changed | byte compare of `.gno` set | — |
| exported API stable | `^func [A-Z]\w*(` names equal | receiver methods, interface methods, renamed unexported helpers |

## Glossary

- crossing function — a `.gno` function whose first parameter is `cur realm`; entered via `f(cross, ...)`, which shifts `PreviousRealm`. Appears only in the fixtures, unchanged by this PR.
- heuristic rule — the harness matches audit patterns by scanning text lines, not an AST, so both false positives and false negatives are expected (README-disclaimed).

## Invariant catalog

Walked, no applicable class. The PR delta touches only Go tooling under `misc/audit-pattern-harness` plus one CI line. The only `.gno` files in the diff are the vulnerable/fixed test fixtures, which are inputs to the harness, not deployed realms, and the delta does not modify them. No GnoVM, stdlib, or realm invariant is in scope.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests

- **[repair contract calls a "fixed" target good without compiling it]** [`run_test.go:356-362`](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356-L362) · [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356) — Missing test: `TestRepairContracts` checks the repair target only through `RunRule`, a text scan, so a target that produces zero pattern hits passes even when it does not compile. This is the deferred compile-check; a fix that closes it is below.
  <details><summary>details</summary>

  The `to_fixture` check is `len(toHits) != 0` on `RunRule(rec.Rule, to.Path)` ([run_test.go:356-362](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L356-L362)). `RunRule` scans lines for the rule's pattern; it never builds the target. So a repaired fixture that removes the flagged text but references an undeclared identifier still passes the contract. Confirmed on da956ea64: replacing `current-guard/fixed/admin.gno`'s body with `owner = thisDoesNotExist` (same exported names, differs from vulnerable, zero `current_guard` hits) keeps `TestRepairContracts/current-guard` green. The `TestAgentPatternContractWithGNO` compile check that would catch it self-skips without a gno binary, and the misc CI job installs none, so nothing in CI compiles the repair targets. The author flags this as the "next harder version" in the PR body ("verify those outputs compile"), so it is a disclosed limitation of the experiment, not a hidden defect.

  Fix (closes the deferred compile-check): build the repair target inside `TestRepairContracts`. Two subtleties. First, `gno test .` is not enough: repair targets have no `_test.gno` file, so `gno test` prints `[no test files]` and exits 0 without type-checking the source. `gno lint .` compiles and type-checks it, so the check uses lint. Second, "when a gno binary is available" is the wrong condition for a contract that must always hold: the misc CI job installs no gno toolchain, so a skip-when-absent check passes silently there. The fix fails loudly instead: absent `GNO_BIN`/`gno`, `TestRepairContracts` fails with an instruction rather than skipping, while `TestAgentPatternContractWithGNO` keeps its opt-in skip. Diff is 32 lines: a `runGNOLint`/`runGNO` split in [`run.go`](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run.go), a shared `lookGNOBin(t, skipIfAbsent)` resolver, and one `runGNOLint` assertion on the target at the end of the per-record block in [`run_test.go`](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go). Verified on da956ea64 (with `GNOROOT` pointed at the PR worktree): the assertion passes on all 8 correct fixtures; breaking `current-guard/fixed/admin.gno` to the hit-free `owner = thisDoesNotExist` body turns `TestRepairContracts/current-guard` red with `admin.gno:6:10: undefined: thisDoesNotExist`, restoring it makes it green again; with no gno binary the test fails rather than skips. Repro test that documents the pre-fix gap: [`tests/repair_compile_gap_test.go`](../../../../../reviews/pr/5xxx/5844-audit-repair-contracts/1-da956ea64/tests/repair_compile_gap_test.go).
  </details>

## Suggestions

- [`ci-dir-misc.yml:25`](https://github.com/gnolang/gno/blob/da956ea64/.github/workflows/ci-dir-misc.yml#L25) · [↗](../../../../../.worktrees/gno-review-5844/.github/workflows/ci-dir-misc.yml#L25) — the CI line runs the harness's pure-Go tests but not the gno-compile contract: `TestAgentPatternContractWithGNO` self-skips unless `GNO_BIN` is set or `gno` is on PATH ([run_test.go:319-327](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L319-L327)), and the reusable `_ci-go.yml` job installs no gno toolchain (`grep -n 'GNO_BIN\|make install\|setup-gno' .github/workflows/_ci-go.yml .github/workflows/ci-dir-misc.yml` is empty). So the fixed fixtures never compile in CI and the repair targets never run through `gno test`. This partially closes [#5835](https://github.com/gnolang/gno/pull/5835)'s W4. To fully close it, the job also needs a gno binary and a `GNO_BIN` export. Review-file only; the CI line is still a net improvement.

- [`run_test.go:373-378`](https://github.com/gnolang/gno/blob/da956ea64/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L373-L378) · [↗](../../../../../.worktrees/gno-review-5844/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L373) — the exported-API-stable check uses `exportedFuncRE` = `^func\s+([A-Z]\w*)\s*\(`, which matches only top-level exported functions, so a repair that changes an exported receiver method or an interface method signature is not caught. The `interface-realm-param` repair itself changes `Notify(cur realm, ...)` to `Notify(caller address, ...)` on an interface method, which the check does not see. The test message says "exported top-level function names," so this matches its stated scope; noting it because the harness's audit families include exported-method and interface surfaces. Review-file only; disclaimed as a top-level-function check.

## Open questions

- The whole layer is stacked on [#5835](https://github.com/gnolang/gno/pull/5835), which is itself NEEDS DISCUSSION and unmerged. The `da956ea64` head has master-merge commits, so the diff against master carries the full 2528-line harness stack; only the top two commits (the `repair` support and the CI line, 239 lines) are this PR's own contribution. If [#5835](https://github.com/gnolang/gno/pull/5835) lands first, rebasing collapses this to that 239-line delta. Design/sequencing; not posted.
- The mechanically-checkable subset of these audit families overlaps the `gno lint` framework that the open [#5068](https://github.com/gnolang/gno/pull/5068) extends. The repair-contract idea (assert a fixture pair, source flags and target clean) could live there as lint fixtures. Direction; not posted.
