# PR [#6032](https://github.com/gnolang/gno/pull/6032): test(gno.land): let package-local .txtar integration tests live next to their realm

URL: https://github.com/gnolang/gno/pull/6032
Author: moul | Base: master | Files: 23 | +412 -40
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 26c788ff2 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6032 26c788ff2`

**TL;DR:** Integration test scripts for a realm used to sit in one big shared folder, far from the realm they test. This moves 11 of them next to the code they exercise and teaches the test runner to find scripts in both places.

**Verdict: REQUEST CHANGES** — the new second root is resolved through `GNOROOT`, so a developer whose `GNOROOT` points elsewhere loses all 11 moved tests with a green run (1 Warning, 1 Missing test, 2 Suggestions, 1 Nit).

## Verify first

- [`gno.land/pkg/integration/testdata_test.go:23-24`](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/testdata_test.go#L23-L24) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/testdata_test.go#L23-L24) — the examples root tracks `GNOROOT`, not the tree the test binary was built from. Confirm that is intended: with `GNOROOT` pointed at a second gno checkout that predates the move, `go test ./gno.land/pkg/integration/ -run 'TestTestdata/wugnot$'` exits 0 and prints `[no tests to run]`.
- [`gnovm/pkg/integration/testscript.go:82-100`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L82-L100) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L82-L100) — one base-name namespace now spans both roots, so any future `foo.txtar` under `examples/` collides with `testdata/foo.txtar` and fails the whole suite at discovery. Confirm that is the wanted trade: `find gno.land/pkg/integration/testdata examples -name '*.txtar' -printf '%f\n' | sort | uniq -d` is empty today.

## Summary

`TestTestdata` read a single directory, [`testdata`](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/testdata_test.go#L24) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/testdata_test.go#L24), and testscript named each subtest after the script's base name. Scripts about one realm therefore lived nowhere near it. The PR adds [`FindTestScripts`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L74) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L74), which walks several roots and hands testscript an explicit file list, and moves 11 scripts under `examples/`: 175 stay in `testdata/`, 11 move, 186 total. Because base names stay the subtest names and the two roots share one namespace, discovery [errors on a duplicate](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L95) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L95) instead of letting testscript disambiguate to `name#1`.

## Diagram

```
                     testdata/                      $GNOROOT/examples/
                     175 *.txtar                    11 *.txtar
                         |                                |
                         +--------- FindTestScripts ------+
                                          |
                            one base-name namespace, 186 files
                                          |
                                  testscript.Params.Files

  the right-hand root is the one the diff adds, and the only one
  whose location is read out of the environment
```

## Fix

Discovery moves from testscript's directory mode to an explicit file list, so `Params.Dir` stays empty and `Params.Files` carries the walk result, [`testscript.go:53-59`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L53-L59) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L53-L59). The load-bearing constraint is that testscript names subtests after base names alone, so uniqueness has to hold across every root rather than within each one. [`update_gas_wanted.sh`](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/update_gas_wanted.sh#L57) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/update_gas_wanted.sh#L57) gains the second root and now maps captured gas back to a file by base name, which resolves to one file only while that same uniqueness rule holds.

## Critical (must fix)

None.

## Warnings (should fix)

- **[tests can vanish without turning CI red]** `gno.land/pkg/integration/testdata_test.go:24` — the package-local root is read from `GNOROOT`, so a `GNOROOT` pointing at a tree without those scripts drops all 11 and the run still passes.
  <details><summary>details</summary>

  [`testdata_test.go:24`](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/testdata_test.go#L24) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/testdata_test.go#L24) resolves the second root through [`gnoenv.RootDir`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/gnoenv/gnoroot.go#L21) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/gnoenv/gnoroot.go#L21), which returns the `GNOROOT` environment variable ahead of anything derived from the source tree. The only guard is [`len(files) == 0`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L53) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L53), which counts the total across roots, so the 175 scripts in `testdata/` keep it satisfied on their own. Before this PR a mismatched `GNOROOT` still ran all 186 scripts and failed loudly on whatever it got wrong; now 11 of them are never discovered and the package reports `ok`. Pointing `GNOROOT` at a `master` worktree makes `-run 'TestTestdata/wugnot$'` exit 0 with `[no tests to run]`, [repro](comment_claude-opus-5.md). The same asymmetry shows up in the tooling: [`update_gas_wanted.sh:57`](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/update_gas_wanted.sh#L57) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/update_gas_wanted.sh#L57) reaches `examples` relative to the repository it runs in, never through `GNOROOT`.

  Fix: make a root that contributes no script an error, the same way a duplicate base name already is.
  </details>

## Missing Tests

- **[a later default would silence the whole file list]** `gnovm/pkg/integration/testscript.go:53-59` — nothing pins the two properties `NewTestingParamsFromRoots` rests on: that `Params.Dir` stays empty, and that a root contributing zero scripts is reported.
  <details><summary>details</summary>

  [`testscript_test.go`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript_test.go#L12) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript_test.go#L12) covers `FindTestScripts` across six cases and stops there, so `NewTestingParamsFromRoots` itself has no test. Its comment at [`testscript.go:57`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L57) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L57) records that testscript honors `Params.Files` only while `Params.Dir` is empty. Nothing enforces it, and a later `Dir` default would send `RunT` down its directory branch and leave every discovered root unrun.

  [`tests/params_from_roots_test.go`](tests/params_from_roots_test.go) covers both, plus the empty-root case the Warning is about. All three pass at 26c788ff2; the empty-root one passes by asserting the current behavior, so it flips to the guard's assertion once the Warning is fixed.

  Fix: pin the `Dir` and `Files` coupling in `testscript_test.go`.
  </details>

## Suggestions

- **[a script in the platform folder can be skipped in silence]** `gnovm/pkg/integration/testscript.go:66-67` — `.txt` scripts stop being discovered under `testdata/` too, though the reason given for dropping them only holds next to real code.
  <details><summary>details</summary>

  testscript's own directory mode globs `*.txt` alongside `*.txtar`, and [`FindTestScripts`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L91) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L91) matches `.txtar` alone for every root. The rationale at [`testscript.go:66-67`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L66-L67) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L66-L67) is that a bare `.txt` next to real code reads as a fixture, which is an argument about `examples/` and not about `testdata/`. No `.txt` script exists in either root today, so nothing is lost now, and a future one lands as a file the suite walks past.
  </details>

- **[an unrelated realm edit now pays two extra CI jobs]** `.github/workflows/ci-dir-gnovm.yml:11` — a package-local script edit now triggers the gnovm and examples workflows on top of gnoland, because all three watch `examples/**`.
  <details><summary>details</summary>

  [`ci-dir-gnoland.yml:12`](https://github.com/gnolang/gno/blob/26c788ff2/.github/workflows/ci-dir-gnoland.yml#L12) · [↗](../../../../../.worktrees/gno-review-6032/.github/workflows/ci-dir-gnoland.yml#L12) already listed `examples/**`, so the moved scripts still gate their own edits. The two that did not fire for a `testdata/` edit now do: [`ci-dir-gnovm.yml:11`](https://github.com/gnolang/gno/blob/26c788ff2/.github/workflows/ci-dir-gnovm.yml#L11) · [↗](../../../../../.worktrees/gno-review-6032/.github/workflows/ci-dir-gnovm.yml#L11) and [`ci-dir-examples.yml:10`](https://github.com/gnolang/gno/blob/26c788ff2/.github/workflows/ci-dir-examples.yml#L10) · [↗](../../../../../.worktrees/gno-review-6032/.github/workflows/ci-dir-examples.yml#L10). On this PR's own run that is 13m15s for `main / test` and 3m55s for `gno-checks / test`, paid by every future one-line gas-number update. Unanchored, so it goes in the comment Body.
  </details>

## Nits

- **[the two lists drift without anything noticing]** `gno.land/pkg/integration/update_gas_wanted.sh:57` — the comment above it says to keep the roots in sync with `FindTestScripts`, but the two walkers disagree on hidden directories.
  <details><summary>details</summary>

  [`update_gas_wanted.sh:57`](https://github.com/gnolang/gno/blob/26c788ff2/gno.land/pkg/integration/update_gas_wanted.sh#L57) · [↗](../../../../../.worktrees/gno-review-6032/gno.land/pkg/integration/update_gas_wanted.sh#L57) runs a bare `find`, which descends into dot-directories. [`FindTestScripts`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/integration/testscript.go#L85) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/integration/testscript.go#L85) skips them. A script under a hidden directory would get its gas numbers rewritten and never be run. Confirmed by reading both walkers; no such file exists today.
  </details>

## Verified

- The 11 moved scripts are byte-identical to their originals: `git diff --find-renames --diff-filter=R --stat origin/master...HEAD -- '*.txtar'` reports `11 files changed, 0 insertions(+), 0 deletions(-)`. No `loadpkg` line in any of them is path-relative, so none depended on sitting under `testdata/`.
- All 11 pass at their new locations, run one `-run` selector at a time, over four repeats.
- Base names are unique across both roots today, so the new global collision check does not fire: `find gno.land/pkg/integration/testdata examples -name '*.txtar' -printf '%f\n' | sort | uniq -d` prints nothing.
- `.txtar` was already excluded from what a mempackage uploads on-chain: [`mempackage.go`](https://github.com/gnolang/gno/blob/26c788ff2/gnovm/pkg/gnolang/mempackage.go#L291-L297) · [↗](../../../../../.worktrees/gno-review-6032/gnovm/pkg/gnolang/mempackage.go#L291-L297) replaces a commented-out entry with a NOTE and a test, so the diff pins existing behavior rather than changing it.
- Tests run green at 26c788ff2: `TestFindTestScripts`, `TestMemPackage_TxtarIsNotPackageSource`, and the 11 moved scripts under `TestTestdata`.

## Open questions

- One early run of a six-script subset failed at the `TestTestdata` parent with no subtest failure captured. Five later runs of the same subset and one run of all 11 passed. Not reproduced and not attributable, so not posted.
- The quarantined `commondao` scripts now live under [`examples/quarantined/`](https://github.com/gnolang/gno/blob/26c788ff2/examples/quarantined/gno.land/r/nt/commondao/v0/commondao_council.txtar#L1) · [↗](../../../../../.worktrees/gno-review-6032/examples/quarantined/gno.land/r/nt/commondao/v0/commondao_council.txtar#L1), a tree the examples lint jobs treat differently from the rest. They still run under `TestTestdata`, which is the pre-PR behavior, so nothing changed. Worth knowing if quarantine ever grows a skip rule.
