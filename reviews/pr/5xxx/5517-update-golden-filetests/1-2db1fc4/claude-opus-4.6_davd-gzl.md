# PR #5517: chore(gno): fix `gno test` to update golden filetests in place

**URL:** https://github.com/gnolang/gno/pull/5517
**Author:** jeronimoalbi | **Base:** master | **Files:** 4 | **+7 -7**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a bug where `gno test -update-golden-tests=true` would write updated filetest files to the **package root directory** instead of the `filetests/` subdirectory where they actually live. The result was duplicate files: the original in `filetests/` was left untouched while a new copy was created in the package root.

The root cause is a single line in `gnovm/pkg/test/test.go:308`. When constructing the filesystem path for writing back updated golden filetests, the code used `filepath.Join(fsDir, testFileName)`, which resolves to the package root. However, since the `filetests/` subdirectory convention was introduced (commits `f0e3a022` and `718e4594`, Jan-Feb 2026), filetest files physically reside in `<pkgDir>/filetests/`. The `MemFile.Name` field stores only the base filename (e.g., `x_filetest.gno`) because `ReadMemPackageFromList` uses `filepath.Base()` when loading files — the `filetests/` prefix is stripped during loading.

The fix changes the path construction to `filepath.Join(fsDir, "filetests", testFileName)`, which correctly writes back to the source location. Three txtar test files (`error_sync.txtar`, `output_sync.txtar`, `realm_sync.txtar`) are updated to match: their filetest files are moved from the txtar root (`-- x_filetest.gno --`) into a `filetests/` subdirectory (`-- filetests/x_filetest.gno --`), and the `cmp` assertion is updated accordingly (`cmp filetests/x_filetest.gno x_filetest.gno.golden`).

## Test Results

- **Existing tests:** PASS
  - `go test -v -run 'Test_Scripts/test' ./gnovm/cmd/gno/` — PASS (12.3s, all 38 subtests pass)
  - `Test_Scripts/test/error_sync` — PASS
  - `Test_Scripts/test/output_sync` — PASS
  - `Test_Scripts/test/realm_sync` — PASS
- **CI status:** All green
- **Edge-case tests:** skipped (trivial one-line path fix)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gnovm/pkg/test/test.go:308` — The hardcoded `"filetests"` string duplicates the same constant used in `ReadMemPackage` (`gnovm/pkg/gnolang/mempackage.go:688`), where the `filetests` directory name is also a hardcoded string literal. If this directory name ever changes, both locations must be updated in sync. Consider extracting a shared constant (e.g., `const FiletestsDir = "filetests"`) in a common location. Low severity since the convention is well-established and unlikely to change, but it would improve maintainability.

## Nits

None

## Missing Tests

None — the three existing txtar tests (`error_sync`, `output_sync`, `realm_sync`) directly exercise the golden-update codepath and verify the file is written to the correct location via `cmp filetests/x_filetest.gno x_filetest.gno.golden`.

## Suggestions

None

## Questions for Author

None

## Verdict

APPROVE — Clean, minimal one-line fix for a clear bug. The filetest write-back path was not updated when the `filetests/` subdirectory convention was introduced. The fix correctly reconstructs the filesystem path, and all three affected txtar tests are properly updated to match.
