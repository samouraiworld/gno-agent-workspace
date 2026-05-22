# PR #5500: fix(gnovm): allow xtest back-imports at genesis AddPkg

URL: https://github.com/gnolang/gno/pull/5500
Author: notJoon | Base: master | Files: 6 | +226 -4
Reviewed by: davd-gzl | Model: claude-opus-4.7

**Verdict: APPROVE** — Fix is correct and consensus-safe. Two open concerns: the `readpkglist.go` change has no regression test (the txtar passes with it reverted), and the sibling wide-merge in `pkgloader.go` still carries the same bug for `loadpkg <single-arg>`.

## Summary

Issue #4530 fails in two independent places when an xtest does a legal back-import (`importee_test` → `importer` → `importee`):

1. `gnoland start --lazy` aborts with `sorting packages: cycle detected` because [`ReadPkgListFromDir`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/packages/readpkglist.go#L57-L68) merged imports across **all** file kinds before feeding `PkgList.Sort()`. xtest/filetest back-edges fool the DFS into reporting a cycle that does not exist (Go compiles xtests as a separate unit).
2. Genesis `AddPkg` aborts with `ImportNotFoundError` because [`TypeCheckMemPackage`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/gnolang/gotypecheck.go#L185) unconditionally runs STEP 4 xtest and filetest passes, and at genesis the deploy order is derived from prod imports only — an xtest import may reference a package not yet in the store.

The fix is two surgical changes plus a genesis-only gate:

- [`readpkglist.go:63-68`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/packages/readpkglist.go#L63-L68) — narrow the merge to `FileKindPackageSource + FileKindTest`. Internal `_test.gno` files share the prod compilation unit so they cannot legally form back-edges.
- [`gotypecheck.go:169-211`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/gnolang/gotypecheck.go#L169-L211) — new `TypeCheckOptions.SkipTestFileTypeCheck` field; when set, passes `wtests=&true` to `typeCheckMemPackage`, which runs prod + internal-test passes and returns before the xtest and filetest Check passes.
- [`keeper.go:556-559`](../../../../../.worktrees/gno-review-5500/.worktrees/gno-review-5500/gno.land/pkg/sdk/vm/keeper.go#L556-L559) — enable `SkipTestFileTypeCheck` only when `ctx.BlockHeight() == 0`. Post-genesis `AddPkg` is byte-identical; live-chain apphash unaffected.

## Glossary

- `xtest` — black-box test file (`package <name>_test`), compiled as a separate unit.
- `filetest` — `_filetest.gno`, each one its own package for VM file tests.
- Internal `_test.gno` — same package name, shares prod compilation unit.
- `wtests` (`*bool`) — pre-existing tri-state in `typeCheckMemPackage`: `nil` = all passes including filetests, `&false` = prod only, `&true` = prod + internal-test.

## Fix

Before, deploy ordering and genesis type-check both treated the dependency graph as a single flat set across all file kinds, so legal `xtest→sibling→pkg` topologies were rejected. After, the topo-sort sees only prod/internal-test edges (eliminating the false cycle) and the genesis type-check stops after prod + internal-test (eliminating the spurious `ImportNotFoundError`). The load-bearing gate is `BlockHeight() == 0` in the keeper: it is the only call site that sets `SkipTestFileTypeCheck`, so post-genesis behavior — and therefore consensus state and replay apphash — is unchanged.

## Critical (must fix)

None.

## Warnings (should fix)

- **[regression test missing for readpkglist fix]** [`gno.land/pkg/integration/testdata/issue_4530.txtar`](../../../../../.worktrees/gno-review-5500/gno.land/pkg/integration/testdata/issue_4530.txtar) — the txtar passes the suite even when [`readpkglist.go`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/packages/readpkglist.go) is reverted to master.
  <details><summary>details</summary>

  Verified locally: with the keeper fix kept and `readpkglist.go` reverted to `origin/master`, `go test -run 'TestTestdata/issue_4530' ./gno.land/pkg/integration/` still passes. Conversely, reverting only the keeper change while keeping the readpkglist fix makes it fail. So the txtar only exercises the keeper-side fix.

  The reason is [`pkgloader.go:130-140`](../../../../../.worktrees/gno-review-5500/gno.land/pkg/integration/pkgloader.go#L130-L140): when `loadpkg <name> <path>` is called with both arguments (the form used in the txtar), the `currentPkg.Name == ""` branch is skipped and `currentPkg.Imports` is never populated. `Sort()` then iterates an empty `Imports` for both packages and finds no cycle — regardless of what `ReadPkgListFromDir` would have computed.

  [`TestNoCycles`](../../../../../.worktrees/gno-review-5500/gno.land/pkg/gnoland/no_cycles_test.go#L33) calls `ReadPkgListFromDir` but only uses the returned `Dir`/`Ignore`; it builds its own graph via `listPkgs`, so it does not validate the merge-narrowing either.

  Fix: add either a unit test on `ReadPkgListFromDir` constructing a temp dir with xtest back-imports and asserting `PkgList.Sort()` returns no error, or a txtar that hits the `gnoland start --lazy` / `LoadGenesisPackagesFromDir` path with a populated `Imports` list.
  </details>

- **[same bug still present in sibling code path]** [`gno.land/pkg/integration/pkgloader.go:164-169`](../../../../../.worktrees/gno-review-5500/gno.land/pkg/integration/pkgloader.go#L164-L169) — wide merge (`PackageSource + Test + XTest + Filetest`) survives in `LoadPackage`.
  <details><summary>details</summary>

  This is the exact merge pattern that was just narrowed in `ReadPkgListFromDir`. When `loadpkg <path>` is used as a single argument (resolved via gnomod, hitting the `currentPkg.Name == ""` branch at [pkgloader.go:141](../../../../../.worktrees/gno-review-5500/gno.land/pkg/integration/pkgloader.go#L141)), the wide merge is applied and `Sort()` runs over an `Imports` list that includes xtest/filetest edges — the original false-cycle bug is still reachable.

  Either narrow the merge here for consistency with the `ReadPkgListFromDir` fix, or document explicitly in the ADR that integration `loadpkg <path>` keeps the legacy behavior and explain why. Fix: pull the merge set into a named helper used by both functions so the two paths cannot drift again.
  </details>

- **[misleading field name]** [`gnovm/pkg/gnolang/gotypecheck.go:169-173`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/gnolang/gotypecheck.go#L169-L173) — `SkipTestFileTypeCheck` does not skip all test-file type-checking.
  <details><summary>details</summary>

  The semantics, traced through [gotypecheck.go:207-212](../../../../../.worktrees/gno-review-5500/gnovm/pkg/gnolang/gotypecheck.go#L207-L212) → `wtests=&true` → [typeCheckMemPackage:520-535](../../../../../.worktrees/gno-review-5500/gnovm/pkg/gnolang/gotypecheck.go#L520-L535), are: run prod, run prod+internal-test, **skip** xtest, **skip** filetest. Internal `_test.gno` files are still type-checked. This is the intended trade-off (and the ADR explains it well), but the field name suggests "skip type-checking for test files" wholesale.

  A reader at the call site has to follow the `*bool` indirection through `typeCheckMemPackage` to discover that internal tests still run. Fix: rename to `SkipXTestAndFiletestTypeCheck`, or keep the short name and add the explicit mapping in the doc comment ("skips xtest and filetest passes; the prod+internal-test pass still runs").
  </details>

## Nits

- [`gno.land/adr/pr5500_xtest_genesis_handling.md:114-115`](../../../../../.worktrees/gno-review-5500/gno.land/adr/pr5500_xtest_genesis_handling.md#L114-L115) — stale instruction "Rename this ADR file to `pr<number>_xtest_genesis_handling.md` once the PR number is assigned." The filename already contains 5500; remove the trailing line.

## Missing Tests

- **[behavior trade-off unpinned]** [`gno.land/pkg/sdk/vm/keeper_test.go:1655`](../../../../../.worktrees/gno-review-5500/gno.land/pkg/sdk/vm/keeper_test.go#L1655) — no test pins the documented "internal `_test.gno` at genesis with a missing import still fails" behavior.
  <details><summary>details</summary>

  The ADR ([line 95-101](../../../../../.worktrees/gno-review-5500/gno.land/adr/pr5500_xtest_genesis_handling.md#L95-L101)) explicitly calls out this trade-off — only xtest/filetest passes are skipped, and an internal `_test.gno` file at genesis that imports a not-yet-deployed package still fails type-check. A symmetric counterpart to `TestVMKeeperAddPackage_XTestBackImport_Genesis` with an internal `*_test.gno` (same-package name) importing a missing package would lock this behavior in and prevent future regressions from silently widening the skip to internal tests as well.
  </details>

- **[`SkipTestFileTypeCheck` unit-level coverage]** [`gnovm/pkg/gnolang/gotypecheck_test.go`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/gnolang/gotypecheck_test.go) — the Codecov report flags lines 207-211 (the new branch in `TypeCheckMemPackage`) at 40% patch coverage. A direct unit test setting `SkipTestFileTypeCheck=true` on a package whose xtest imports something missing, asserting no error, would exercise the path without going through the keeper.

## Suggestions

- [`gnovm/pkg/gnolang/gotypecheck.go:207-211`](../../../../../.worktrees/gno-review-5500/gnovm/pkg/gnolang/gotypecheck.go#L207-L211) — could be a one-liner.
  <details><summary>details</summary>

  The five-line `if opts.SkipTestFileTypeCheck { t := true; wtests = &t }` could be `wtests := &opts.SkipTestFileTypeCheck` (or wrap in `if opts.SkipTestFileTypeCheck { wtests = ptr.To(true) }` if a `nil`-default is required). The current form mixes intent (skip xtest/filetest) with mechanism (the pre-existing tri-state pointer); a helper makes the call site easier to follow.
  </details>

## Questions for Author

- The ADR notes `no_cycles_test.go` already handles xtests correctly by treating them as separate graph nodes, while `ReadPkgListFromDir` did not. Was widening `ReadPkgListFromDir`'s output to also expose per-kind imports (rather than a flat list) considered, so callers like `pkgloader.go` and `Sort()` could pick the right view? The current narrow-merge is the right minimum fix for this PR, but it leaves the sibling-bug surface in `pkgloader.go` unaddressed.
- Is there an intent to deprecate `ReadPkgListFromDir` (the doc comment at [readpkglist.go:17-20](../../../../../.worktrees/gno-review-5500/gnovm/pkg/packages/readpkglist.go#L17-L20) already calls it deprecated) in favor of `Load` with a recursive pattern? If yes, that would be a natural place to fold in the per-kind import view.
