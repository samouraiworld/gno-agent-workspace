# PR #5559: fix(gnovm): allow private realm unit tests to import their own package

**URL:** https://github.com/gnolang/gno/pull/5559
**Author:** aronpark1007 | **Base:** master | **Files:** 2 | **+41 -1**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR fixes issue #5053, where a realm marked `private = true` in gnomod.toml could not be imported by its own `xxx_test` package files, making it impossible to write external unit tests for private realms.

The fix adds a single condition to the private import check in `gnoImporter.ImportFrom()`: when `gimp.testing` is true (type-checking test files) AND the import path matches the package being tested (`pkgPath == gimp.pkgPath`), the private import restriction is bypassed. This allows `xxx_test` files to import their own private realm while all other callers remain blocked.

**Files changed:**
- `gnovm/pkg/gnolang/gotypecheck.go` — Modified the private import guard from `mod.Private` to `mod.Private && !(gimp.testing && pkgPath == gimp.pkgPath)`, adding a comment explaining the exception.
- `gnovm/cmd/gno/testdata/test/realm_private_unit_test.txtar` — New end-to-end txtar test: creates a private realm with an `xxx_test` file that imports and calls it, verifying the test passes.

## Test Results

- **Existing tests:** PASS
  - `Test_Scripts/test/realm_private_unit_test` — PASS (1.77s)
  - `TestVMKeeperAddPackage_PrivatePackage` — PASS
  - `TestVMKeeperAddPackage_ImportPrivate` — PASS (confirms on-chain import of private still blocked)
  - `TestVMKeeperRunImportPrivate` — PASS
  - All CI checks: PASS (build, lint, test, gno-checks)
- **Edge-case tests:** skipped

## Critical (must fix)

None

## Warnings (should fix)

None

## Nits

- [ ] `gnovm/pkg/gnolang/gotypecheck.go:348` — The exception `!(gimp.testing && pkgPath == gimp.pkgPath)` applies transitively: if `xxx_test` of private realm R imports package A, and A imports R, the private check for R is also bypassed during that test type-checking phase (because `gimp.testing` remains `true` and `gimp.pkgPath` remains R throughout the recursive import chain). This has no on-chain security impact (on-chain AddPackage never sets `gimp.testing = true`), but it means the type-checker is slightly more permissive during `gno test` than strictly necessary. A package A that imports private R would fail its own on-chain type-checking anyway, so this scenario is very unlikely in practice. Flagging for awareness only.

## Missing Tests

- [ ] No negative txtar test verifying that an external (non-self) package is still blocked from importing a private realm during `gno test`. The on-chain case is covered by `addpkg_import_private.txtar` and `TestVMKeeperAddPackage_ImportPrivate`, but an off-chain `gno test` scenario is not explicitly tested. Consider adding a test with two packages: a private realm and a separate public package that tries to import it, running `gno test` on both and verifying the external import is still rejected.

## Suggestions

- The PR description and code comment clearly explain the safety guarantees (`gimp.testing` is off-chain only, `pkgPath == gimp.pkgPath` limits to self-import). The rationale is sound. No ADR needed for a one-line bug fix.

## Questions for Author

- The commit message body includes `Assisted-By: Claude Sonnet 4.6`. The repo's `AGENTS.md` says "Do NOT add Co-Authored-By lines or AI tool credits in commits — they are legally misleading and carry no useful information. Disclose AI usage in the PR description instead." Is `Assisted-By` an accepted alternative per team convention, or should this be removed from the commit and moved to the PR description?

## Verdict

APPROVE — Minimal, well-targeted fix that correctly solves the problem with a clear safety argument: the exception only activates off-chain during `gno test` and is strictly limited to self-imports. On-chain behavior is unchanged, confirmed by existing keeper tests.
