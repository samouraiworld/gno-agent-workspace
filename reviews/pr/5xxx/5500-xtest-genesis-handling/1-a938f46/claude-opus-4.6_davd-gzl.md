# PR #5500: fix(gnovm): allow xtest back-imports at genesis AddPkg

**URL:** https://github.com/gnolang/gno/pull/5500
**Author:** notJoon | **Base:** master | **Files:** 6 | **+226 -4**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes issue #4530, where a perfectly legal Go xtest topology — package `importee_test` (black-box test) importing `importer`, while `importer` imports `importee` — was incorrectly rejected by Gno during genesis in two independent places:

1. **`readpkglist.go` topo-sort false cycle:** `ReadPkgListFromDir` merged imports from *all* file kinds (including `XTest` and `Filetest`) into the dependency graph fed to `PkgList.Sort()`. Since xtest/filetest are compiled as separate units in Go, their back-imports created artificial back-edges that the DFS cycle detector mistook for real cycles. The fix excludes `FileKindXTest` and `FileKindFiletest` from the merge, keeping only `FileKindPackageSource` and `FileKindTest` (internal tests share the prod compilation unit and cannot legally form back-edges).

2. **`keeper.go` genesis type-check failure:** `TypeCheckMemPackage` unconditionally ran xtest and filetest type-check passes (STEP 4 in `gotypecheck.go`). At genesis, deploy order is derived from production imports only, so an xtest import may reference a package not yet in the store, causing `ImportNotFoundError`. The fix introduces a new `TypeCheckOptions.SkipTestFileTypeCheck` boolean that, when set, passes `wtests = &true` to `typeCheckMemPackage`, which executes the prod and prod+internal-test passes but returns before the xtest and filetest Check passes. This flag is set **only** when `ctx.BlockHeight() == 0` in the keeper, preserving post-genesis behavior byte-identically.

The approach is well-scoped: `SkipTestFileTypeCheck` has exactly one setter (keeper.go at genesis), and `ReadPkgListFromDir` is only used for genesis ordering / offline tooling, never at runtime after `genesis.json` is finalized. The PR includes an ADR (`gno.land/adr/pr5500_xtest_genesis_handling.md`), a keeper unit test pinning both branches (reject post-genesis, accept at genesis), and an end-to-end txtar integration test reproducing the original issue.

## Test Results

- **Existing tests:** PASS
  - `go test ./gno.land/pkg/sdk/vm/` — PASS (17.9s)
  - `go test ./gnovm/pkg/gnolang/ -run TestTypeCheckMemPackage` — PASS
  - `go test ./gno.land/pkg/gnoland/ -run TestNoCycles` — PASS
  - `go test ./gnovm/pkg/packages/` — PASS
- **New tests:** PASS
  - `TestVMKeeperAddPackage_XTestBackImport_Genesis` — PASS (3.5s)
  - `TestTestdata/issue_4530` (txtar integration) — PASS (3.2s)
- **CI status:** All green (except the bot merge-requirements check, which is pending human approval — unrelated)
- **Edge-case tests:** skipped (changes are narrow and well-tested)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/gotypecheck.go:207-211` — The `wtests` mechanism uses a `*bool` tri-state (`nil` = check all including filetests, `&false` = prod-only, `&true` = prod+internal-tests). While the PR correctly uses this existing convention, the indirection through `*bool` is the source of the only non-obvious semantics in this PR. `SkipTestFileTypeCheck = true` sets `wtests = &true`, which causes `typeCheckMemPackage` to run prod + internal-test passes but skip xtest and filetest passes. The name `SkipTestFileTypeCheck` could be misread as skipping *all* test type-checking (including internal tests). Consider renaming to `SkipXTestAndFiletestTypeCheck` or adding a code comment at the call site in `TypeCheckMemPackage` (lines 207-212) explicitly stating: "wtests=&true means: run prod + internal-test passes, skip xtest + filetest passes." This would prevent future confusion. Currently the doc comment on the field (lines 169-173) is clear enough, but the mapping from field to behavior goes through a non-obvious `*bool` indirection.

- [ ] `gno.land/pkg/sdk/vm/keeper.go:557-558` — Minor alignment inconsistency: line 557 uses three spaces before the `//` comment (`gno.TCGenesisStrict   //`) while line 558 uses one space (`opts.SkipTestFileTypeCheck = true //`). The original code at line 557 was aligned with one space; this PR introduced the extra spaces. Suggest aligning comments consistently.

## Nits

- [ ] `gno.land/adr/pr5500_xtest_genesis_handling.md:114-115` — The ADR ends with "Rename this ADR file to `pr<number>_xtest_genesis_handling.md` once the PR number is assigned." The PR number *is* 5500 and the filename already contains it (`pr5500_xtest_genesis_handling.md`). This trailing note should be removed.

## Missing Tests

- [ ] No dedicated unit test for `ReadPkgListFromDir` with the xtest-exclusion behavior. The txtar integration test `issue_4530.txtar` covers it end-to-end (via `loadpkg` which exercises `ReadPkgListFromDir`'s output through `PkgList.Sort()`), and `TestNoCycles` validates the examples directory. A unit test specifically constructing a directory with xtest back-imports and verifying `ReadPkgListFromDir` produces a sortable `PkgList` (no cycle error) would make the fix more explicit. Low priority since the integration test covers the scenario.

## Suggestions

- The Codecov report shows `gotypecheck.go` at 40% patch coverage (2 missing + 1 partial). Lines 208-210 (the `SkipTestFileTypeCheck` branch) are not exercised by the existing `gotypecheck_test.go` tests. Adding a test case to `TestTypeCheckMemPackage` that sets `SkipTestFileTypeCheck = true` with a package containing an xtest that imports a non-existent dependency would directly cover this code path at the unit level (`gnovm/pkg/gnolang/gotypecheck_test.go`).

## Questions for Author

- The ADR notes that `no_cycles_test.go` already handles xtests correctly (treating them as separate graph nodes), while `ReadPkgListFromDir` did not. Is there a plan to deprecate or align `ReadPkgListFromDir` with the graph-building approach used in `no_cycles_test.go`? The function is already marked as deprecated in its doc comment (line 17).

## Verdict

APPROVE — Clean, minimal, consensus-safe fix for a well-understood bug. The two changes are independent and surgically scoped: `readpkglist.go` excludes xtest/filetest from topo-sort inputs, and `gotypecheck.go`+`keeper.go` skip xtest/filetest type-check passes at genesis only. The `BlockHeight == 0` guard ensures zero impact on live chains. Tests cover both directions (reject post-genesis, accept at genesis) and the original reproduction scenario. ADR is thorough and well-written.
