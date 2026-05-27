# PR #5048: feat(gnovm/lint): enforce last elem of pkg path to match pkg name

URL: https://github.com/gnolang/gno/pull/5048
Author: mvallenet | Base: master | Files: 99 | +590 -233
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]
Local worktree: `git -C gno worktree add .worktrees/gno-review-5048 103795b46` (then `gh -R gnolang/gno pr checkout 5048` inside it)

**Verdict: REQUEST CHANGES** — sound design, three prior approvals, but PR has drifted out of date with master: two unmerged-rebase `loader/load_govdao.gno` txtar files ([`addpkg_cla.txtar`](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/integration/testdata/addpkg_cla.txtar#L165-L166) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/integration/testdata/addpkg_cla.txtar#L165-L166), [`governance_param_validation.txtar`](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/integration/testdata/governance_param_validation.txtar#L51-L52) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/integration/testdata/governance_param_validation.txtar#L51-L52)) declare `package load_govdao` at path `gno.land/r/gov/dao/v3/loader` and now break CI. Two-line fix, then merge.

## Summary

Enforces Go's "package name == last path element" convention on every chain-deployed mempackage. New helpers in [`gnovm/pkg/gnolang/mempackage.go`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage.go#L159-L207) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage.go#L159-L207): `LastPathElement(pkgPath)` strips a `vN` suffix (`v0`+, regex `^v([0-9]+)$`) and returns the parent segment; `ValidatePkgNameMatchesPath(name, path)` compares it to the declared `package` clause. The check fires inside [`ValidateMemPackageAny`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage.go#L1090-L1098) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage.go#L1090-L1098) only when `mptype == MPUserAll` (so genesis loading and `MsgAddPackage` are gated, filetests/test runs are not), plus a `gno lint` pre-deploy warning ([`lint.go:182-194`](https://github.com/gnolang/gno/blob/103795b46/gnovm/cmd/gno/lint.go#L182-L194) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/cmd/gno/lint.go#L182-L194)) with a dedicated `gnoPackageNameMismatch` code. `BREAKING CHANGE` — new deployments only; chain state untouched. PR also renames the offending `examples/` packages (`gnoblog → blog`, `foo20 → grc20factory`, `eval → math_eval`, `nir1218_evaluation_proposal/* → .../evaluation/*`, etc.) and updates ~20 integration txtars and a handful of filetests.

## Glossary

- `MPUserAll` — mempackage type for "user-land package, all files including filetests"; the type Keeper.AddPackage assigns ([keeper.go:515](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/sdk/vm/keeper.go#L515) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/sdk/vm/keeper.go#L515)) and the only gate where this PR fires the new validation.
- `LastPathElement` — new helper that returns the parent segment when the last segment matches `v[0-9]+`, otherwise the last segment.
- `loader/load_govdao.gno` — bootstrap helper at `gno.land/r/gov/dao/v3/loader` reused across several govdao integration txtars; the PR renamed `package load_govdao → package loader` in most copies but missed two.

## Fix

Before: a user could deploy `package foo` to `gno.land/r/demo/bar` and the imported identifier (`foo`) would silently diverge from the import path's last segment (`bar`). After: VMKeeper.AddPackage and genesis loading reject the mismatch with `package name "foo" does not match path element "bar"`. The load-bearing gate is `mptype == MPUserAll` in `ValidateMemPackageAny` — confining the check to deployment paths and keeping filetests, in-memory `gno test`, and `xxx_test` integration packages unaffected. `gno lint` runs the same check upstream with a typed error code so contributors get a clean error before they pay gas. The version-suffix carve-out (`v0`..`vN`) is the only deliberate deviation from Go's spec (Go forbids `/v1`); justified in [`mempackage.go:163-165`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage.go#L163-L165) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage.go#L163-L165) by existing realms like `gno.land/r/sys/users/v1`.

## Critical (must fix)

- **[CI red — stale rebase, two missed renames]** [`gno.land/pkg/integration/testdata/addpkg_cla.txtar:165-166`](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/integration/testdata/addpkg_cla.txtar#L165-L166) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/integration/testdata/addpkg_cla.txtar#L165-L166), [`gno.land/pkg/integration/testdata/governance_param_validation.txtar:51-52`](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/integration/testdata/governance_param_validation.txtar#L51-L52) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/integration/testdata/governance_param_validation.txtar#L51-L52) — `package load_govdao` at path `gno.land/r/gov/dao/v3/loader` trips the new validator; `Run gno.land suite / Go Test / test` fails with `package name "load_govdao" does not match path element "loader"`.
  <details><summary>details</summary>

  The PR renamed the `loader/load_govdao.gno` body from `package load_govdao` to `package loader` across the govdao txtars (see [`update_storage_params.txtar` diff](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/integration/testdata/update_storage_params.txtar) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/integration/testdata/update_storage_params.txtar) — also the renamed filename `loader/loader.gno`). Two siblings that landed on master *after* the PR's last `git merge master` (2026-03-16) — `addpkg_cla.txtar` from PR #5138 (`r/sys/cla`) and `governance_param_validation.txtar` from PR #5200 (param validation) — kept the old `package load_govdao` and were never updated. Reproduced locally:

  ```
  --- FAIL: TestTestdata/addpkg_cla (0.07s)
      FAIL: testdata/addpkg_cla.txtar:16: unable to load pkg "load_govdao": invalid package: package name "load_govdao" does not match path element "loader"
  --- FAIL: TestTestdata/governance_param_validation (0.03s)
      FAIL: testdata/governance_param_validation.txtar:12: invalid package: package name "load_govdao" does not match path element "loader"
  ```

  Fix: in both files, change `package load_govdao` → `package loader` and rename the heredoc filename `loader/load_govdao.gno` → `loader/loader.gno` to mirror the convention used in `update_storage_params.txtar`. Then rebase and re-run `TestTestdata`. Two-line change; no design impact.
  </details>

## Warnings (should fix)

- **[PR is 4+ months in flight; needs a fresh rebase and a green CI before merge]** PR HEAD = `103795b46`, last master merge 2026-03-16. The two failing txtars above came in *after* that merge. Re-rebasing now is also the only way to be sure no further `package X` / pkgpath divergences slipped in during the gap (`grep -RIn '^package ' examples/ gno.land/pkg/integration/testdata/ | awk` against the path is a 30-second check before pushing the fix).

## Nits

- [`gnovm/pkg/gnolang/mempackage.go:163`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage.go#L163) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage.go#L163) — regex/comment say `v0, v1, v2, …` but doc strings elsewhere repeat the same list inconsistently; [`mempackage.go:174-175`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage.go#L174-L175) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage.go#L174-L175) writes "v0 v1 v2", [`mempackage.go:196`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage.go#L196) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage.go#L196) writes "v0, v1, v2", PR body says "v1, v2, v3". One canonical phrasing across all four call-sites is easier to scan.
- [`gno.land/pkg/integration/testdata/addpkg_namespace.txtar:64-66`](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/integration/testdata/addpkg_namespace.txtar#L64-L66) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/integration/testdata/addpkg_namespace.txtar#L64-L66) — top-level `gnomod.toml` (`module = "gno.land/r/mypkg"`) is no longer referenced by any test step now that every `addpkg` points at `$WORK/one` or `$WORK/two`. Dead heredoc; safe to delete in a follow-up.

## Missing Tests

- **[v0 path uncovered by filetests]** [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/103795b46/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/tests/files/) — only `z_addpkg_version_suffix_v1.gno` and `z_addpkg_version_suffix_v2.gno` exist; `v0` is documented as supported ([`docs/builders/deploy-packages.md:117`](https://github.com/gnolang/gno/blob/103795b46/docs/builders/deploy-packages.md#L117) · [↗](../../../../../.worktrees/gno-review-5048/docs/builders/deploy-packages.md#L117), [`mempackage_test.go:TestLastPathElement`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage_test.go) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage_test.go) does cover it as a unit test, but no filetest pins the on-chain behaviour). Add `z_addpkg_version_suffix_v0.gno` mirroring `v1`/`v2`.
- **[no negative filetest for `vN` parent mismatch]** Single mismatch case at [`z_addpkg_name_mismatch.gno`](https://github.com/gnolang/gno/blob/103795b46/gnovm/tests/files/z_addpkg_name_mismatch.gno) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/tests/files/z_addpkg_name_mismatch.gno) targets a non-versioned path. Add a `gno.land/r/demo/foo/v2` + `package bar` filetest so the regression net catches a future bug that mishandles version skipping (e.g. an off-by-one in `LastPathElement` returning `v2` instead of `foo`).

## Questions for Author

- The `gno lint` check at [`lint.go:182-194`](https://github.com/gnolang/gno/blob/103795b46/gnovm/cmd/gno/lint.go#L182-L194) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/cmd/gno/lint.go#L182-L194) and the keeper check at [`mempackage.go:1093-1098`](https://github.com/gnolang/gno/blob/103795b46/gnovm/pkg/gnolang/mempackage.go#L1093-L1098) · [↗](../../../../../.worktrees/gno-review-5048/gnovm/pkg/gnolang/mempackage.go#L1093-L1098) are duplicates by design (the comment notes lint runs first to surface a typed error). Is there appetite to centralise the check behind a single conditional inside `ValidateMemPackageAny` keyed on `MemPackageType` rather than maintaining two callers? `@thehowl` already raised this thread for the broader keeper question; same shape applies for lint.
- [`displayPackageName`](https://github.com/gnolang/gno/blob/103795b46/gno.land/pkg/gnoweb/handler_http.go#L411-L423) · [↗](../../../../../.worktrees/gno-review-5048/gno.land/pkg/gnoweb/handler_http.go#L411-L423) renders `gno.land/r/demo/v2` (where `v2` is the *actual* last-element realm name, not a version suffix) as "demo/v2" because the heuristic can't distinguish "single-segment realm named v2" from "versioned realm". Acceptable corner case, or worth scoping the version-suffix rule to paths with ≥4 segments? Same observation applies to `LastPathElement` itself.
