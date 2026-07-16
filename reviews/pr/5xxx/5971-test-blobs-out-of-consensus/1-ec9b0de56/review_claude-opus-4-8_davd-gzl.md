# PR [#5971](https://github.com/gnolang/gno/pull/5971): fix(gnovm): exclude package test/filetest blobs from consensus state

URL: https://github.com/gnolang/gno/pull/5971
Author: moul | Base: master | Files: 6 | +249 -30
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: ec9b0de56 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5971 ec9b0de56`

**TL;DR:** When a package is deployed on-chain, its `_test.gno` files are stored alongside its real code, and because they sit in the store the chain hashes, editing a single test file changes the number every node must agree on. This PR moves the test files into the store that is persisted but not hashed, so a test-only edit stops being a chain-breaking change while the files stay readable for tooling.

**Verdict: APPROVE** — the fix does what it claims and the revert proof isolates it to the blob move; drop the stray `config/addrbook.json` before merge. Deploying a package with test files also gets 223,600 gas cheaper, which is a correction rather than a regression and rides the same declared break, but the PR states gas is unchanged and that sentence should not land in the ADR.

## Summary
PR [#5891](https://github.com/gnolang/gno/pull/5891) split each stored package into a prod blob at `pkg:<path>` and a test/filetest blob at `pkg:<path>#allbutprod`, but wrote both to the merkleized `iavlStore`. So a one-line edit to `gnovm/stdlibs/chain/address_test.gno` moved the pinned genesis app hash even though execution was identical, making every test-file edit a hard fork. This PR routes the `#allbutprod` blob to `baseStore` instead, whose `dbadapter` backend returns a nil commit hash, so its contents are persisted and queryable but never reach the app hash. `FindPathsByPrefix` becomes a sorted two-store merge so a package still lists exactly once. The genesis app hash moves once, deliberately.

```
deploy pkg {gnomod.toml, foo.gno, foo_test.gno}
  ├── iavlStore  pkg:<path>              -> MP*Prod  {gnomod.toml, foo.gno}   --> app hash
  └── baseStore  pkg:<path>#allbutprod   -> MP*All   {foo_test.gno}           --> NOT in app hash
```

## Glossary
- app hash — per-block commitment to application state; two honest nodes disagreeing halts the chain.
- `baseStore` / `iavlStore` — the VM store's two backends; only `iavlStore` is merkleized.
- depth-metered store — a backend whose cache wrap charges depth-scaled I/O gas; flat backends charge a fixed cost.
- addpkg — the transaction that uploads a package or realm to the chain.
- gas — metered, consensus-relevant execution cost; any change is a behavior change.
- MemPackage — in-memory set of a package's source files.
- hard fork — a change to persisted format or a consensus gas constant; ships only on fresh genesis or a coordinated upgrade.

## Fix
Before this PR both blobs went to `iavlStore`, so the test blob's bytes fed the committed root. Now [`setMemPackageBlob`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1049) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1049) takes a destination store, and [the split in `AddMemPackage`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1015-L1030) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1015-L1030) sends the prod blob to `iavlStore` and [the sibling to `baseStore`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1026) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1026), with [`DeleteMemPackage`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1041-L1042) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1041-L1042), [`getMemPackageAllButProd`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1149) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1149) and [`isMemPackage`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/debugger.go#L620-L621) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/debugger.go#L620-L621) following. The load-bearing constraint is that the two blobs no longer share a store, so any listing that must see prod-less packages has to read both: hence the merge in [`FindPathsByPrefix`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1258-L1288) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1258-L1288).

## Benchmarks / Numbers
Measured at ec9b0de56 with the only variable being the sibling blob's destination store. See [repro](comment_claude-opus-4-8.md).

| addpkg shape | sibling → `baseStore` (PR) | sibling → `iavlStore` (pre-PR) | delta |
|---|---|---|---|
| prod + `_test.gno` | 21,680,911 | 21,904,511 | **−223,600** (−1.0%) |
| prod only, no test file | 3,771,428 | 3,771,428 | 0 |

The whole tx-level delta is one `Set`, per the `-tags gastrace` trace of the sibling key:

| `Set` on `pkg:gno.land/r/hellotest#allbutprod` | gas | trace `info` |
|---|---|---|
| → `baseStore`, flat `WillSet` | 26,926 | `depth=false` |
| → `iavlStore`, `DepthSet` | 250,526 | `depth=true` |
| difference | **223,600** | |

| genesis app hash | value |
|---|---|
| PR head, unmodified | `4ffebf22…` (pinned, passes) |
| PR head, `+1` test func in `chain/address_test.gno` | `4ffebf22…` (unchanged) |
| sibling → `iavlStore`, unmodified | `058910b2…` (= master's pinned value) |
| sibling → `iavlStore`, `+1` test func | `328fb036…` (moved) |

## Critical (must fix)
None.

## Warnings (should fix)
- **[stray artifact in the diff]** `config/addrbook.json:1` — a node runtime file committed with the fix; it is the only tracked file under `config/` and nothing ignores it.
  <details><summary>details</summary>

  `git log --diff-filter=A -- config/addrbook.json` names db302a1e0, this PR's first commit, and `git check-ignore config/addrbook.json` exits 1, so a `gnoland start` from the repo root will keep re-dirtying the tree for everyone. Content is the empty peer book `{"peers": []}`. Unrelated to the store change. Fix: drop the file from the PR.
  </details>

## Nits
- **[stale claim outlives the PR]** `gnovm/pkg/gnolang/store.go:1026` — routing the test blob to `baseStore` drops a flat 223,600 gas off deploying any package that ships a test file; the PR states per-tx gas accounting is unchanged.
  <details><summary>details</summary>

  Not a defect, and not a merge risk: the discount rides the same declared consensus break as the app-hash move, so no node ever prices this differently from its peers, and the new number is arguably the right one because a `dbadapter` write really does no tree traversal. It matters only because the claim is wrong and lands in a permanent ADR that later contributors will read as the gas model. The two backends meter writes differently. [`cacheStore.Set`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/cache/store.go#L186-L196) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/store/cache/store.go#L186-L196) charges depth-scaled `DepthSet` gas when its parent implements [`DepthEstimator`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/cache/store.go#L79-L80) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/store/cache/store.go#L79-L80), and a flat `WillSet` otherwise. [`bptree`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/bptree/store.go#L46) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/store/bptree/store.go#L46) implements it and backs `iavlStore`; `dbadapter` does not and backs `baseStore`. So the blob's write moves from depth-metered to flat. The amino-encode gas charged inside [`setMemPackageBlob`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1049-L1056) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1049-L1056) is length-driven and genuinely unchanged, but it is not the only charge on that path.

  The delta is closed-form, not an estimate. With [`ReadCostFlat` 59,000 and `WriteCostFlat` 24,000](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/types/gas.go#L404-L407) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/store/types/gas.go#L404-L407) and the default VM depths [`minSetReadDepth100Default` 200, `minWriteDepth100Default` 540](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/sdk/vm/params.go#L40-L42) · [↗](../../../../../.worktrees/gno-review-5971/gno.land/pkg/sdk/vm/params.go#L40-L42):

  ```
  depth (bptree)    = 2.0×59,000 + 5.4×24,000 + 14×len   = 247,600 + 14×len
  flat  (dbadapter) =              1.0×24,000 + 14×len   =  24,000 + 14×len
  delta                                                  = 223,600, for any len
  ```

  Two consequences. The `WriteCostPerByte × len` term is identical on both sides and cancels, so the delta does not scale with test-file size: a 200-byte `_test.gno` and a 200KB one each save exactly 223,600. And it does not track tree size either, because [`DefaultParams`](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/sdk/vm/params.go#L93-L98) · [↗](../../../../../.worktrees/gno-review-5971/gno.land/pkg/sdk/vm/params.go#L93-L98) sets `FixedSetReadDepth100`/`FixedWriteDepth100` to the same 200/540 and [`effectiveSetReadDepth100`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/cache/store.go#L105-L114) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/store/cache/store.go#L105-L114) returns the Fixed override when non-zero, so `expectedDepth100(tree.Size())` is never consulted. 223,600 is a hard per-package constant.

  Confirmed against the gas trace (`-tags gastrace`): the sole `Set` on `pkg:gno.land/r/hellotest#allbutprod` charges 26,926 with `info=depth=false` at head and 250,526 with `info=depth=true` when routed back, and 250,526 − 26,926 = 223,600 accounts for the whole 21,904,511 → 21,680,911 tx-level move. The same package without a `_test.gno` is byte-identical either way. Every gas golden in the tree deploys packages without test files, which is why nothing went red. Fix: drop the gas-unchanged claim, or say the store write repriced while the encode gas did not.
  </details>

## Missing Tests
- **[gas can drift back invisibly]** `gno.land/pkg/sdk/vm/gas_test.go:333` — no golden deploys a package that ships a test file, the one shape whose gas this PR moves.
  <details><summary>details</summary>

  [`setupAddPkg`](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/sdk/vm/gas_test.go#L333-L374) · [↗](../../../../../.worktrees/gno-review-5971/gno.land/pkg/sdk/vm/gas_test.go#L333-L374) builds `gnomod.toml` plus one `.gno` file, so no `#allbutprod` blob is written and the whole suite is blind to the blob's store. The txtar goldens are the same: [`addpkg_import_testdep_gas.txtar`](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/integration/testdata/addpkg_import_testdep_gas.txtar) · [↗](../../../../../.worktrees/gno-review-5971/gno.land/pkg/integration/testdata/addpkg_import_testdep_gas.txtar) does deploy a dep carrying a padded `_test.gno`, but it asserts two importers charge equal gas rather than pinning the deploy's own cost, so it stays green either way. That leaves a consensus-relevant number with nothing holding it. The gap predates this PR, which neither creates nor widens it, so this is a cheap add rather than something the PR owes: [`addpkg_test_blob_gas_test.go`](tests/addpkg_test_blob_gas_test.go), green at ec9b0de56, pins 21,680,911 and fails at 21,904,511 if the blob is routed back.
  </details>

- **[claimed-correct branch nothing runs]** `gnovm/pkg/gnolang/store.go:1271` — the merge's same-store branch is asserted correct in the comment above it but nothing exercises it.
  <details><summary>details</summary>

  [The comment](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1217-L1223) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1217-L1223) names `NewStore(_, base, base)` and argues the two iterators stay in lockstep, and the [`default`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1271-L1276) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1271-L1276) branch exists only for it. [`TestTransactionStore`](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L16-L22) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store_test.go#L16-L22) does build that shape but never lists paths; the three `FindPathsByPrefix` call sites in the file ([244](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L244) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store_test.go#L244), [273](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L273) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store_test.go#L273), [288](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L288) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store_test.go#L288)) all run against stores built with two distinct `memdb` backends ([196-199](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L196-L199) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store_test.go#L196-L199), [250-256](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store_test.go#L250-L256) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store_test.go#L250-L256)). The only production caller is the keeper's query handlers, which also pass distinct stores. So the branch is defensive today and a refactor could break it silently. I wrote [`findpaths_samestore_test.go`](tests/findpaths_samestore_test.go) and it is green at ec9b0de56, so this is coverage, not a bug.
  </details>

## Suggestions
None.

## Verified
- The stated goal holds: with a `TestReviewProbe` function appended to `gnovm/stdlibs/chain/address_test.gno`, `TestAppHashCrossrealm38` still passes at `4ffebf22…`. Test-file edits are consensus-neutral.
- The revert reproduces the bug and nothing else: changing [store.go:1026](https://github.com/gnolang/gno/blob/ec9b0de56/gnovm/pkg/gnolang/store.go#L1026) · [↗](../../../../../.worktrees/gno-review-5971/gnovm/pkg/gnolang/store.go#L1026) back to `ds.iavlStore` yields `058910b2…`, master's own pinned value, and the same test-file edit then moves it to `328fb036…`. So the new pinned hash is exactly the blob move, with no other state change folded in, and the old hash was exactly the bug.
- The premise that `baseStore` escapes the app hash is real, not assumed: [`baseKey` mounts `dbadapter`](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/gnoland/app.go#L106-L107) · [↗](../../../../../.worktrees/gno-review-5971/gno.land/pkg/gnoland/app.go#L106-L107) while `mainKey` mounts the bptree, [`dbadapter.Commit`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/dbadapter/store.go#L91-L97) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/store/dbadapter/store.go#L91-L97) returns a nil hash, and [`commitStores`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/store/rootmulti/store.go#L495-L524) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/store/rootmulti/store.go#L495-L524) folds only each store's `CommitID`, so a nil-hash store contributes a constant.
- Nothing executable reads the moved blob: grepping `gnovm/stdlibs/`, `gno.land/pkg/stdlibs/`, `op_*.go`, `uverse.go` and `contribs/` for `GetMemPackageAll`/`GetMemFile`/`FindPathsByPrefix` returns no hit, so the only readers are the keeper's `qfile`/`qdoc`/`qpaths` handlers and the debugger. "Test files never affect execution" is structural, not a convention.
- Both blobs still commit or discard together: [`newGnoTransactionStore`](https://github.com/gnolang/gno/blob/ec9b0de56/gno.land/pkg/sdk/vm/keeper.go#L382-L384) · [↗](../../../../../.worktrees/gno-review-5971/gno.land/pkg/sdk/vm/keeper.go#L382-L384) takes both stores from the same `ctx` cache multistore, so a failed tx cannot leave a sibling behind.
- The de-dup's adjacency assumption survives adversarial input: a package path may only contain `[a-z0-9./_-]` ([`rePkgPathURL`/`rePkgPathStd`](https://github.com/gnolang/gno/blob/ec9b0de56/tm2/pkg/std/memfile.go#L27-L28) · [↗](../../../../../.worktrees/gno-review-5971/tm2/pkg/std/memfile.go#L27-L28)), whose lowest byte `-` (0x2D) still sorts above `#` (0x23), so no key can land between `pkg:<path>` and its sibling and split the pair.
- The baseStore scan sees siblings only, as the comment claims: `oid:`/`node:` sort below `pkg:` and `tid:`/`pkgidx:` above `pkg:\xFF`, so the index counter and object keys stay outside `[startKey,endKey]`. Confirmed against the live listing tests.
- Green at ec9b0de56: full `./gno.land/pkg/sdk/vm/` (48s), `./gno.land/pkg/integration/ -run TestTestdata` (68s), and the `gnovm/pkg/gnolang` store suite, plus both tests I wrote under `tests/`.

## Open questions
- Storage deposits are genuinely untouched: `realmStorageDiffs` is only written from the object save path in `realm.go`, never from `AddMemPackage`, so no mempackage blob has ever fed a deposit. Not posted: it confirms the PR rather than questioning it.
- An in-place binary upgrade that keeps existing state would strand each pre-upgrade package's sibling in `iavlStore`, where `getMemPackageAllButProd` no longer looks, silently dropping test files from `qfile` while `DeleteMemPackage` stops reaching them. Not posted: the PR scopes activation to a fresh testnet, where this cannot arise, and gno.land relaunches genesis for storage-format changes.
- A `qfile` at a historical height now mixes prod files at that height with test files as of now, since `dbadapter` ignores `LoadVersion`. Not posted: realm objects already live in `baseStore`, so historical queries were already current-state for everything but package source, and this narrows rather than creates the gap.
