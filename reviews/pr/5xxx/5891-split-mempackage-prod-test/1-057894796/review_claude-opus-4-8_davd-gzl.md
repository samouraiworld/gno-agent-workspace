# PR [#5891](https://github.com/gnolang/gno/pull/5891): feat(gnovm): split mempackage storage into prod and test blobs

URL: https://github.com/gnolang/gno/pull/5891
Author: jaekwon | Base: master | Files: 11 | +507 -22
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 057894796 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5891 057894796`

**TL;DR:** When a package is deployed on-chain, all its `.gno` files are stored together in one blob, including `_test.gno`/`_filetest.gno` files that no importer ever uses. This PR stores each package as two blobs, production files under the usual key and test files under a sibling key, so importing a package no longer reads and type-checks its test files. It also rejects deploying a package that has only test files, clears stale blobs when a private package is redeployed, and updates the app hash and gas goldens for the new storage layout.

**Verdict: APPROVE** — the split is lossless and deterministic, the prod-less-package reject closes a real restart-history divergence, and every consensus-relevant number is re-pinned. Only optional cleanups remain; the app-hash change is intended and requires the usual genesis relaunch.

## Summary
The chain stored each package's full file set in one `pkg:<path>` IAVL blob, so type-checking an importer decoded the dependency's `_test.gno` bytes too. Measured on master that is +40620 gas for importing a dep carrying a padded test file. This PR writes production files (non-.gno plus non-test `.gno`) under `pkg:<path>` typed `MP*Prod`, and the test/filetest complement under a `pkg:<path>#allbutprod` sibling typed with the original `MP*All` as an inert sentinel. `GetMemPackage` returns prod-only, so the import/preprocess hot path is charged on prod bytes; a new `GetMemPackageAll` merges both blobs for query paths that must still see test files. Three correctness fixes ride along: reject packages with no production `.gno` file, clear both keys before a private redeploy, and de-dup the sibling key in `FindPathsByPrefix`. Stored bytes change, so the genesis app hash moves (intended, verified by `TestAppHashCrossrealm38`).

```
deploy pkg {gnomod.toml, foo.gno, foo_test.gno}
  ├── pkg:<path>              -> MP*Prod  {gnomod.toml, foo.gno}      (import/type-check reads this)
  └── pkg:<path>#allbutprod   -> MP*All   {foo_test.gno}             (query paths read this)

deploy pkg {gnomod.toml, only_test.gno}   ->  REJECTED at AddPackage (no prod .gno)
```

## Glossary
- MemPackage — in-memory set of a package's source files, the unit loaded, type-checked, and run.
- Amino — gno's deterministic serialization codec for on-chain state.
- app hash — per-block commitment to application state; two honest nodes disagreeing halts the chain.
- gas — metered, consensus-relevant execution cost; any change is a behavior change.

## Fix
Before, `AddMemPackage` wrote one blob and `GetMemPackage` returned all files; importers paid decode gas on test bytes. After, [`splitProdAllButProd`](https://github.com/gnolang/gno/blob/057894796/gnovm/pkg/gnolang/store.go#L1074-L1096) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1074-L1096) partitions the package so `prod ∪ allButProd == mpkg.Files` with no overlap, and [`AddMemPackage`](https://github.com/gnolang/gno/blob/057894796/gnovm/pkg/gnolang/store.go#L1014-L1029) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1014-L1029) writes each blob conditionally. The load-bearing constraint is that conditional two-key writes are not a full replace, so a re-add must first [`DeleteMemPackage`](https://github.com/gnolang/gno/blob/057894796/gnovm/pkg/gnolang/store.go#L1038-L1041) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1038-L1041) both keys, which the keeper does on private redeploy.

## Benchmarks / Numbers
| Path | master | this PR | note |
|------|--------|---------|------|
| import dep with padded `_test.gno` (useb) vs without (usea) | +40620 | equal | import decodes prod blob only |
| addpkg gas (prod blob type byte `MPUserAll`→`MPUserProd`) | — | +17 | per author; goldens re-pinned |
| genesis app hash | `28f55f0a…` | `adef42a3…` | stored byte-set changed |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- `gno.land/pkg/sdk/vm/keeper.go:635-643` — no test proves the keeper clears the sibling on a private redeploy.
  <details><summary>details</summary>

  The stale-blob clearing is exercised only at the store layer ([`TestDeleteMemPackageClearsStaleBlobsOnReAdd`](https://github.com/gnolang/gno/blob/057894796/gnovm/pkg/gnolang/store_test.go#L115-L161) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store_test.go#L115-L161)), whose own comment says it mirrors the keeper's delete-then-add. Nothing asserts that [`VMKeeper.AddPackage`](https://github.com/gnolang/gno/blob/057894796/gno.land/pkg/sdk/vm/keeper.go#L635-L643) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L635-L643) actually calls `DeleteMemPackage` on the private-redeploy path, so a future edit that drops the call would leave `qfile`/`GetMemPackageAll` serving a deleted `_test.gno` and every store-level test would stay green. Impact is query correctness, not consensus, and private redeploy is a niche path, so this is a low-priority guard rather than a blocker. A keeper test would redeploy a private package with a file set that removes a `_test.gno`, then assert `getGnoTransactionStore(ctx).GetMemFile(pkgPath, "<removed>_test.gno")` is nil.
  </details>

## Suggestions
- `gnovm/pkg/gnolang/machine.go:341` — the prod filter is now redundant.
  <details><summary>details</summary>

  [`IterMemPackage`](https://github.com/gnolang/gno/blob/057894796/gnovm/pkg/gnolang/store.go#L1246-L1272) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1246-L1272) now yields the prod blob (typed `MP*Prod`, no test files), so [`mpkg = MPFProd.FilterMemPackage(mpkg)`](https://github.com/gnolang/gno/blob/057894796/gnovm/pkg/gnolang/machine.go#L341) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/machine.go#L341) filters nothing and only re-copies each file per package at restart. It was load-bearing before the split, when `IterMemPackage` returned the full package. Harmless and arguably defensive, so keep or drop; no change needed for correctness.
  </details>

## Verified
- Split round-trips losslessly across re-add: `TestDeleteMemPackageClearsStaleBlobsOnReAdd` green, and I confirmed by reading `splitProdAllButProd` that both branches (has-prod and prod-less) reconstruct `mpkg.Files` exactly — non-.gno files fold to prod when a prod `.gno` exists and to the sibling when none does.
- A dependency's padded `_test.gno` does not inflate import gas: `TestVMKeeperAddPackage_ImportTestDepGasEqual` and `addpkg_import_testdep_gas.txtar` both green at 057894796 (usea == useb == 3113401), so import-time type-checking never decodes the `#allbutprod` sibling.
- Prod-less package is rejected up front: `TestVMKeeperAddPackage_NoProdFiles` green; the reject and the store's prod predicate share `MPFProd.FilterMemPackage(...).IsEmpty()`, so the check cannot drift from what gets stored.
- `FindPathsByPrefix` lists a split package once and drops `#`-prefix probes: `TestFindByPrefixDeDupesSplitPackages` green; the prod key and its sibling are adjacent in IAVL order because no valid path byte sorts between end-of-path and `#` (0x23), so de-dup against the previous key suffices.
- Query compensations preserve pre-split behavior: `QueryFile`/`QueryDoc`/`GetMemFile`/`debugger` switched to `GetMemPackageAll`, which returns the same file set the single-blob `GetMemPackage` returned on master, so these are behavior-preserving, not new behavior.
- Goldens hold at the reviewed sha: `TestAppHashCrossrealm38`, `restart_gas.txtar`, `gnokey_gasfee.txtar` green.

## Open questions
- Merging this shifts the genesis app hash, so it needs a coordinated genesis relaunch like any storage-format change; fine on a pre-mainnet chain, and PR 5892 stacks on this new layout. Not posted: it is called out in the PR description and is not a code defect.
