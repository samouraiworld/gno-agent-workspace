# PR [#5891](https://github.com/gnolang/gno/pull/5891): feat(gnovm): split mempackage storage into prod and test blobs

URL: https://github.com/gnolang/gno/pull/5891
Author: jaekwon | Base: master | Files: 10 | +509 -24
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: 82e5cb868 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5891 82e5cb868`

Round 2 (deep). Head advanced 057894796 → 82e5cb868: two follow-up commits from [@ltzmaxwell](https://github.com/gnolang/gno/pull/5891#issuecomment-4966682416) plus a merge of master (`5b989cad5`). The patch-id gate is not decisive here — the merge-base moved d2737d84e → 5b989cad5 and patch-ids differ (`26893098…` → `093ccdb4…`) — but the move is not base-only in any case: `git show 82e5cb868 --cc` prints conflict-resolution content in four files, all of it PR content. [`3a30d0928`](https://github.com/gnolang/gno/commit/3a30d0928) moved the prod-less-package skip out of `Machine.PreprocessAllFilesAndSaveBlockNodes` and into `IterMemPackage`; [`209b0b90a`](https://github.com/gnolang/gno/commit/209b0b90a) added a doc line on the implementation. The merge re-derived the pinned app hash to `d5764060…` and re-pinned three gas goldens onto master's bptree pricing. Round 1's Missing Test (no keeper-level proof of the redeploy clear) is carried, now with the test written. Round 1's lone Suggestion is **retracted**: its premise was wrong, and the reason it was wrong is this round's Warning at `store.go:88-91`. Round 1's APPROVE is **overturned**. The PR is MERGED (squash [`af23ea2ae`](https://github.com/gnolang/gno/commit/af23ea2ae) on master), so every finding below is framed as a follow-up against master rather than a change to this branch.

**TL;DR:** When a package is deployed on-chain, all its `.gno` files are stored together in one blob, including the `_test.gno` files no importer ever uses. This PR stores each package as two blobs, production files under the usual key and test files under a sibling key, so importing a package no longer reads and type-checks its test files.

**Verdict: REQUEST CHANGES** — the split itself is lossless and deterministic and every consensus number is re-pinned, but the sibling key is addressable as a package path, and reaching it through `vm/qfile` panics the query handler on unauthenticated input. Open PR [#5971](https://github.com/gnolang/gno/pull/5971) happens to fix that, untested and as a side effect; two doc contracts also state the opposite of what the code does.

## Summary
The chain stored each package's full file set in one `pkg:<path>` blob, so type-checking an importer decoded the dependency's `_test.gno` bytes too. This PR writes production files under `pkg:<path>` typed `MP*Prod` and the test/filetest complement under a `pkg:<path>#allbutprod` sibling. `GetMemPackage` returns prod-only, so the import hot path is charged on prod bytes; a new `GetMemPackageAll` merges both blobs for the query handlers that must still see test files. Three correctness fixes ride along: reject packages with no production `.gno` file, clear both keys before a private redeploy, and de-dup the sibling key in `FindPathsByPrefix`. The sibling key is what the round-2 Warning turns on: `pkg:<path>#allbutprod` is exactly the key `GetMemPackage("<path>#allbutprod")` computes, so the alias resolves, and `GetMemPackageAll` then hands that unvalidated path to `MPAnyAll.Decide`, which panics.

```
deploy pkg {gnomod.toml, foo.gno, foo_test.gno}
  ├── pkg:<path>              -> MP*Prod  {gnomod.toml, foo.gno}   (import/type-check reads this)
  └── pkg:<path>#allbutprod   -> MP*All   {foo_test.gno}           (query paths read this)
                    ▲
                    └── also reachable as GetMemPackage("<path>#allbutprod") --> Decide() panics
```

## Glossary
- MemPackage — in-memory set of a package's source files, the unit loaded, type-checked, and run.
- Amino — gno's deterministic serialization codec for on-chain state.
- app hash — per-block commitment to application state; two honest nodes disagreeing halts the chain.
- gas — metered, consensus-relevant execution cost; any change is a behavior change.
- `baseStore` / `iavlStore` — the VM store's two backends; only `iavlStore` is merkleized.
- addpkg — the transaction that uploads a package or realm to the chain.
- filetest — a `*_filetest.gno` file executed by the VM and asserted against golden directives.

## Fix
Before, `AddMemPackage` wrote one blob and `GetMemPackage` returned all files. After, [`splitProdAllButProd`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1055-L1090) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1055-L1090) partitions the package so `prod ∪ allButProd == mpkg.Files` with no overlap, and [`AddMemPackage`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L982-L1024) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L982-L1024) writes each blob conditionally. The load-bearing constraint is that conditional two-key writes are not a full replace, so a re-add must first [`DeleteMemPackage`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1033-L1036) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1033-L1036) both keys, which the keeper does on private redeploy.

## Benchmarks / Numbers
| Path | master (`5b989cad5`) | this PR (`82e5cb868`) | note |
|------|--------|---------|------|
| import dep with padded `_test.gno` (useb) vs without (usea) | +40620 | equal (2691401 each) | import decodes prod blob only |
| `gnokey_gasfee.txtar` addpkg | 2452592 | 2452609 | +17: `MPUserAll`→`MPUserProd` is one more amino byte |
| `restart_gas.txtar` addpkg ×3 | 2476212 / 2478531 / 2478558 | 2476229 / 2478548 / 2478575 | +17 each |
| genesis app hash | `b04e01f8…` | `d5764060…` | stored byte-set changed |

The +17 is closed-form: one extra amino byte at `GasEncodePerByte` 3 + `WriteCostPerByte` 14. The `addpkg_import_testdep_gas.txtar` pin fell 3113401 → 2691401 in the merge because master's bptree mount moved the flat store costs; the usea/useb *equality* the golden actually guards is untouched, since the +40620 delta is per-byte only (`ReadCostPerByte` 17 + `GasAminoDecode` 3, × 2031 bytes) and per-byte gas is not depth-scaled.

## Critical (must fix)
None.

## Warnings (should fix)
- **[unauthenticated input panics a query handler]** `gnovm/pkg/gnolang/store.go:1168` — `GetMemPackageAll` hands its raw `path` to `MPAnyAll.Decide`, which panics on any path that is neither stdlib nor userlib; `vm/qfile` and `vm/qdoc` pass raw query data straight through, and the `#allbutprod` sibling guarantees a resolvable key for every package that ships a test file.
  <details><summary>details</summary>

  `backendPackagePathKey("gno.land/r/hello#allbutprod")` is byte-identical to the sibling key the split writes for `gno.land/r/hello`, so [`GetMemPackage`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1095-L1097) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1095-L1097) returns the sibling blob rather than nil. `GetMemPackageAll` therefore gets past its `prod == nil && allButProd == nil` early return and reaches [`MPAnyAll.Decide(path)`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1168) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1168), whose [`default` branch panics](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/mempackage.go#L568) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/mempackage.go#L568) on the `#`. The reachable entry points are [`QueryFile`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/keeper.go#L1415) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L1415) and [`QueryDoc`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/keeper.go#L1435) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L1435); [`queryFile`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/handler.go#L274-L283) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/handler.go#L274-L283) forwards `req.Data` unvalidated, and neither [`handleQueryCustom`](https://github.com/gnolang/gno/blob/82e5cb868/tm2/pkg/sdk/baseapp.go#L492-L531) · [↗](../../../../../.worktrees/gno-review-5891/tm2/pkg/sdk/baseapp.go#L492-L531) nor the vm handler recovers.

  The node survives: [`RecoverAndLogHandler`](https://github.com/gnolang/gno/blob/82e5cb868/tm2/pkg/bft/rpc/lib/server/http_server.go#L174) · [↗](../../../../../.worktrees/gno-review-5891/tm2/pkg/bft/rpc/lib/server/http_server.go#L174) catches it, so the caller gets a 500 and the node logs a full goroutine stack at ERROR, one per request, from any unauthenticated client. That is the whole impact: no consensus effect, no state effect, no content leak, since the panic fires before the merge returns. It is still a regression, and the query surface itself shows what the expected answer is: `QueryEval` and `QueryFuncs`, which this PR does not touch, take the same alias and return `invalid package path` as an error. Only the two handlers switched to `GetMemPackageAll` panic on it. `#` is the one byte a package path can never contain ([`rePkgPathURL`/`rePkgPathStd`](https://github.com/gnolang/gno/blob/82e5cb868/tm2/pkg/std/memfile.go#L27-L28) · [↗](../../../../../.worktrees/gno-review-5891/tm2/pkg/std/memfile.go#L27-L28)), and the PR already guards `FindPathsByPrefix` against exactly this input class while `GetMemPackageAll` does not.

  Two notes for whoever picks this up. Open PR [#5971](https://github.com/gnolang/gno/pull/5971) already fixes it, incidentally: routing the sibling to `baseStore` leaves nothing at `iavlStore`'s `pkg:<path>#allbutprod`, so the alias resolves to nil and the query returns the clean error. I ran [`qfile_sibling_alias.txtar`](tests/qfile_sibling_alias.txtar) at both heads — red at 82e5cb868, green at 5971's ec9b0de56 — so the fix is real but rests on a store-routing detail nothing pins. Fix: reject a path containing `#` in `GetMemPackageAll` before the store read, and land the txtar so the guard survives whichever way the sibling is routed.
  </details>

- **[doc invites deleting the line that makes it true]** `gnovm/pkg/gnolang/store.go:88-91` — the `Store.IterMemPackage` contract states test files are never yielded; that holds only for `MP*All` packages, and the caller-side filter it invites removing is what makes the other types safe.
  <details><summary>details</summary>

  The [interface doc](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L88-L91) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L88-L91) says each yielded package is the prod mempackage and that test files "are not included". The split only runs on the [`mpkgtype.IsAll()` branch](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1009) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1009); an `MP*Test` or `MP*Integration` add takes the [`else`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1021-L1023) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1021-L1023) and is stored whole, test files and all, which `IterMemPackage` then yields verbatim. Confirmed behaviorally against a store carrying one `MPStdlibTest` add: `IterMemPackage yielded: path="math" type=MPStdlibTest files=[math.gno math_test.gno]`. That shape is written in production code at [`gnovm/pkg/test/imports.go:313`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/test/imports.go#L313) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/test/imports.go#L313).

  Nothing breaks today, because [`MPFProd.FilterMemPackage`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/machine.go#L330) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/machine.go#L330) still strips them in the one caller. The risk is precisely that a reader trusts the doc and drops that line as redundant — which is what round 1 of this review proposed, on this doc's reasoning. It is redundant for the chain store, which only ever holds `MP*All`, and load-bearing for every other store. [`store_itermempackage_doc_test.go`](tests/store_itermempackage_doc_test.go) asserts the doc as written and is red at 82e5cb868, so it goes green with either remedy. Fix: filter inside `IterMemPackage`, or narrow the doc to say the split applies to `MP*All` only.
  </details>

- **[method changed meaning, doc did not]** `gnovm/pkg/gnolang/store.go:1093-1094` — `GetMemPackage`'s doc comment still describes the pre-split contract, so nothing at the declaration says the returned package omits test files.
  <details><summary>details</summary>

  [The comment](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1093-L1094) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1093-L1094) reads "retrieves the MemPackage at the given path", byte-identical to master's, and appears in the diff only as a context line. The method's meaning changed under it: it now returns prod-only for `MP*All`. `GetMemPackageAll` right below it does document its own role ("for query/tooling paths that must see test files"), which makes the silence on the prod-only sibling read as deliberate rather than stale. The pair matters because picking the wrong one is how a test file reaches a consensus read, the exact hazard the split exists to remove. Fix: say prod-only at the declaration.
  </details>

## Nits
- **[comment points the wrong way]** `gno.land/pkg/sdk/vm/keeper.go:1433-1434` — the `QueryDoc` comment frames `GetMemPackageAll` as widening doc generation to test files "for any future test-derived examples"; it restores master, where the single blob already carried them.
  <details><summary>details</summary>

  On master [`QueryDoc` called `GetMemPackage`](https://github.com/gnolang/gno/blob/5b989cad5/gno.land/pkg/sdk/vm/keeper.go#L1415), and master's `AddMemPackage` wrote the whole package under one key, so doc generation already saw test files. Every query-path switch in this PR is compensation for the split, not new behavior. A reader who takes [the comment](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/keeper.go#L1433-L1434) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L1433-L1434) at face value could "revert" it to `GetMemPackage` as a cleanup and silently narrow the query. Fix: say it preserves pre-split behavior.
  </details>

- **[unrelated change, half applied]** `gno.land/pkg/sdk/vm/params.go:44` — a gas-params style edit rides in a mempackage-storage PR, and the merge left the const block inconsistent.
  <details><summary>details</summary>

  Commit [`e5235a533`](https://github.com/gnolang/gno/commit/e5235a533) ("remove redundant gas conversions") is PR-branch-only — `git merge-base --is-ancestor e5235a533 origin/master` fails — and un-cast both `minWriteDepth100Default = int64(440)` → `440` and `iterNextCostFlatDefault = int64(1_000)` → `1_000`. Master then rewrote the first line with its own value, so the merge kept master's `int64(540)` and the branch's bare `1_000`, leaving [three cast constants and one bare one](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/params.go#L40-L44) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/params.go#L40-L44) in the same block. Value-inert: `iterNextCostFlatDefault` only feeds [`NewParams`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/params.go#L71) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/params.go#L71), whose parameter is `int64`, and `maxIterNextCostFlat` is only compared against an `int64` and printed with `%d`. It landed on master with the squash. Fix: finish the un-cast or drop it, in its own PR.
  </details>

- `gnovm/pkg/gnolang/store.go:83` — `DeleteMemPackage` joins the `Store` interface with no doc comment, while the delete-before-re-add requirement it exists to serve is documented only on [`defaultStore.AddMemPackage`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L974-L980) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L974-L980). A caller coding against [the interface](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L82-L83) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L82-L83) cannot discover it.

## Missing Tests
- **[bug can come back invisibly]** `gno.land/pkg/sdk/vm/keeper.go:642` — no test proves the keeper clears the sibling on a private redeploy. Carried from round 1, now written.
  <details><summary>details</summary>

  The stale-blob clearing is exercised only at the store layer ([`TestDeleteMemPackageClearsStaleBlobsOnReAdd`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store_test.go#L115-L161) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store_test.go#L115-L161)), whose own comment says it mirrors the keeper's delete-then-add by hand, and the existing [`TestVMKeeperAddPackage_UpdatePrivatePackage`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/keeper_test.go#L359) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper_test.go#L359) redeploys the same file set, so it never removes a file. Nothing asserts that [`VMKeeper.AddPackage`](https://github.com/gnolang/gno/blob/82e5cb868/gno.land/pkg/sdk/vm/keeper.go#L635-L643) · [↗](../../../../../.worktrees/gno-review-5891/gno.land/pkg/sdk/vm/keeper.go#L635-L643) calls `DeleteMemPackage`. [`keeper_private_redeploy_test.go`](tests/keeper_private_redeploy_test.go) is green at 82e5cb868; deleting the keeper's `DeleteMemPackage` call turns it red while every other `TestVMKeeperAddPackage_*` stays green.
  </details>

- **[silent data loss stays green]** `gnovm/pkg/gnolang/store.go:1085` — the prod-less branch's non-`.gno` fold is asserted only by the comment above it.
  <details><summary>details</summary>

  [The `prodSkipped && !strings.HasSuffix(mfile.Name, ".gno")` clause](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store.go#L1085) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store.go#L1085) is what keeps a prod-less package's `gnomod.toml`, `LICENSE` and `README.md` in storage instead of dropping them on the floor. Removing it leaves both of the PR's store tests green, because [`TestFindByPrefixDeDupesSplitPackages`](https://github.com/gnolang/gno/blob/82e5cb868/gnovm/pkg/gnolang/store_test.go#L250-L293) · [↗](../../../../../.worktrees/gno-review-5891/gnovm/pkg/gnolang/store_test.go#L250-L293) builds its prod-less package from a lone `_test.gno` with no non-`.gno` file, so the branch never runs. The de-dup's adjacency assumption has the same shape of gap: only a nested path (`alpha` plus `alpha/sub`) separates an adjacent sibling suffix from a non-adjacent one, and no test uses one. [`store_split_contract_test.go`](tests/store_split_contract_test.go) is green at 82e5cb868 and covers the lossless split, the `GetMemPackageAll` merge, the `IterMemPackage` prod-only contract, and nested de-dup; each case was mutation-checked against the line it pins.
  </details>

- **[fix rests on an untested side effect]** `gnovm/pkg/gnolang/store.go:1168` — nothing pins that the `#allbutprod` alias is rejected rather than panicking, in either this PR or [#5971](https://github.com/gnolang/gno/pull/5971).
  <details><summary>details</summary>

  See the Warning for the mechanism. [`qfile_sibling_alias.txtar`](tests/qfile_sibling_alias.txtar) deploys one package carrying a `_test.gno`, then hits `vm/qfile` and `vm/qdoc` with the sibling key spelled as a path; it is red at 82e5cb868 and green at 5971's ec9b0de56. Landing it with either fix keeps the alias closed whichever store the sibling ends up in.
  </details>

## Suggestions
None. Round 1's `machine.go` Suggestion is retracted; see the `store.go:88-91` Warning.

## Verified
- The sibling key is addressable as a package path: `GetMemPackage("gno.land/p/demo/dep#allbutprod")` returns the sibling blob (`Path="gno.land/p/demo/dep" Type=MPUserAll files=1`), not nil. Its content does not leak through any handler, because `GetMemPackageAll` panics before merging, and it is not importable: the type-checker rejects the import path outright (`invalid import path (invalid character U+0023 '#')`), so no consensus-executed path can name it.
- The panic is remote-triggerable and the node survives it: through real `gnoland start` + `gnokey query vm/qfile`, the alias logs `Panic in RPC HTTP handler` with a full stack and answers `invalid status code received, 500`; the next query on the real path succeeds. Red at 82e5cb868, green at 5971's ec9b0de56 with `package "gno.land/r/hello#allbutprod" is not available`.
- The panic is confined to the two handlers this PR moved onto `GetMemPackageAll`, and the rest of the query surface answers the way they should: `QueryEval` and `QueryFuncs` fed the same alias return `invalid package path` as an error, as does `QueryEval` on an arbitrary `bogus#path`.
- A prod-less package cannot land on chain, so the new `IterMemPackage` skip cannot make a restarted node diverge: `AddPackage` rejects all three shapes tried (test-only `.gno`, filetest-only, `xxx_test` package plus README).
- `IterMemPackage` violates its own interface doc for non-`MP*All` types: a store carrying one `MPStdlibTest` add yields `files=[math.gno math_test.gno]`.
- The new prod-less skip changes no consensus number: on-chain it is unreachable, since the keeper rejects prod-less packages at `AddPackage`, and a prod-less package still lists once through `FindPathsByPrefix` while `IterMemPackage` yields 0 against `NumMemPackages()==1`.
- The merge dropped no master content beyond the `params.go` const noted as a Nit: the app-hash comment block took master's two entries verbatim and added the split's as a third, and the +17 gas delta is consistent across all four re-pinned deploy goldens.
- `FindPathsByPrefix` holds against adversarial prefixes: `foo#`, `foo#allbutprod` and `foo#allbutpro` all yield nothing, the empty prefix and `gno.land` list every package exactly once, and a package listing is unaffected by whether it carries a test file.
- Green at 82e5cb868: `TestAppHashCrossrealm38`, the `gnovm/pkg/gnolang` store suite, the full `TestVMKeeperAddPackage_*` set, and `gnokey_gasfee` / `restart_gas` / `addpkg_import_testdep_gas` (usea == useb == 2691401).

## Open questions
- Re-adding at an existing path as `MP*Prod` after an `MP*All` add leaves the stale sibling, and `GetMemPackageAll` then serves the deleted test file. No caller does this: the chain store only ever sees `MP*All`, and the four non-keeper `AddMemPackage` call sites are tooling. Not posted: latent, and the delete-before-re-add doc already covers it once `DeleteMemPackage` gets its interface comment.
- The `#allbutprod` suffix is a valid sub-realm subpath under [#5890](https://github.com/gnolang/gno/pull/5890)'s `realm.Sub`: `subRealmPathError(host, "allbutprod")` returns empty. Sub-realms address objects by a hash of the path, never through `pkg:`, so the namespaces do not collide today. Not posted: no reachable defect, but the two features now mint `host#subpath` strings from different rules.
- Merging shifts the genesis app hash, so it needs a coordinated genesis relaunch. Not posted: called out in the PR description, and the fork tool replays addpkg txs rather than copying blobs, so the split applies naturally on re-execution.
