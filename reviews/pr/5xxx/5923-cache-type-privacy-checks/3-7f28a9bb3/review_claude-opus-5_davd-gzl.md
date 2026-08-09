# PR [#5923](https://github.com/gnolang/gno/pull/5923): chore(perfs): Cache type-privacy checks across commits

URL: https://github.com/gnolang/gno/pull/5923
Author: Villaquiranm | Base: master | Files: 7 | +1189 -36
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: 7f28a9bb3 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5923 7f28a9bb3`

Round 3. Head advanced 37466ba3c → 7f28a9bb3: `fix: address review`, then a merge of master. The answer to round 2's Critical is the commit gate: a verdict is buffered in [`defaultStore.pendingTypePrivacy`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/store.go#L219) and reaches the shared cache only from [`transactionStore.Write`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/store.go#L387), which [`app.go`'s endTxHook calls on a successful DeliverTx alone](https://github.com/gnolang/gno/blob/7f28a9bb3/gno.land/pkg/gnoland/app.go#L200-L204). That closes the simulation route: [`runTx` returns before the hook for every mode but Deliver](https://github.com/gnolang/gno/blob/7f28a9bb3/tm2/pkg/sdk/baseapp.go#L944-L949). Round 2's Warning, that the stated premise is not the premise the code has, survives and is now the whole finding: the gate scopes *which transactions* may write a verdict and nothing scopes *what a verdict is about*.

**TL;DR:** Before saving a realm object, the VM walks the object's type to check that nothing in it comes from a package marked private. This PR remembers the answer in a table keyed by the type's shape. Two different packages can have the same shape, so one package's answer is handed out for the other's type.

**Verdict: REQUEST CHANGES** — the memo key does not name the packages the verdict was computed over, so a public realm's committed verdict answers for a structurally identical private type, and the same transaction is accepted on a node that has been up and rejected on one that restarted (1 Critical, 1 Warning, 1 Missing test, 2 Suggestions).

## Verify first

- [`gnovm/pkg/gnolang/types.go:778-790`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/types.go#L778-L790) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/types.go#L778-L790) — the memo key. Read the NOTE, then decide whether a key that omits `PkgPath` can carry a verdict about package privacy. Same question at [`types.go:930-933`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/types.go#L930-L933) for interfaces.
- [`gnovm/pkg/gnolang/realm.go:1432`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1432) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1432) — every guarantee of the privacy feature rests on this one boolean. Drop [`tests/typeid_collision_test.go`](tests/typeid_collision_test.go) into `gnovm/pkg/gnolang/` and run it: two of its three cases fail at this sha.

## Summary

[`assertTypeIsPublic`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1424) re-walks a saved object's full type graph on every commit. This PR memoizes the realm-independent half of the question, "does anything in this type reach a private package", in [`typePrivacyCache`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/store.go#L210) on the root store, keyed by `TypeID`. Round 2 showed a verdict computed inside a discarded transaction outliving it; round 3 answers that by promoting verdicts only from a committing transaction, which holds.

The key itself is the remaining hole. [`StructType.TypeID()` omits `PkgPath` when every field name is exported](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/types.go#L778-L790), and [`InterfaceType.TypeID()` does the same for exported method names](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/types.go#L913-L942). Privacy is read from the [`PackageValue` during the walk](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L2363-L2369), never from the key. So `struct{ N int }` declared in a public realm and the same shape declared in a private one are one entry, both transactions commit, and the gate never fires.

## Diagram

```
tx A (public realm, commits)          tx B (private realm, commits)
 warm.Warm: [3]*struct{N int}          priv.Leak: [3]*struct{N int}
 PkgPath = gno.land/r/zz/warm          PkgPath = gno.land/r/zz/priv
            │                                     │
            ▼                                     ▼
  typeHasPrivateDep = false             key = [3]*struct{N int}
            │                                     │
            │ buffered, promoted on Write         │
            └────── shared typePrivacyCache ────► cache HIT: false
                                                  │
                                                  ▼
                                        assertTypeIsPublic returns
                                        PkgPath never compared
```

## Fix

The verdict is a fact about a set of packages, and the key names none of them. Nothing about the commit gate is wrong; it scopes when a verdict may be written, while the collision is about what the verdict is written under. A key that determines the set of packages the walk read, or a memo restricted to the type kinds whose `TypeID` already carries their `PkgPath`, would make the cached answer belong to the type it is returned for. [`TestTypeHasPrivateDep_TypeIDCollidesNotOnDeclared`](tests/typeid_collision_test.go) pins the second option's boundary: a `DeclaredType` keys on `PkgPath` and does not collide.

## Benchmarks / Numbers

The four shipped benchmarks, at 7f28a9bb3 on this machine, `-benchtime 300x`:

| benchmark | ns/op | allocs/op |
|---|--:|--:|
| `ColdAcyclic` | 2755 | 5 |
| `WarmAcyclic` | 265 | 0 |
| `ColdSelfReferential` | 18469 | 18 |
| `WarmSelfReferential` | 1752 | 6 |

The cold-to-warm ratio holds in the direction the [ADR's own table](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/adr/pr5923_type_privacy_dependency_cache.md?plain=1#L198-L201) reports. The PR description reports three other benchmarks, none of which the diff contains.

## Critical (must fix)

- **[one cache entry serves two packages, so a private type passes the check]** [`gnovm/pkg/gnolang/realm.go:1299`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1299) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1299) — `TypeID` drops `PkgPath` for a struct whose field names are all exported, so a public realm's committed verdict answers for a structurally identical private type, and `assertTypeIsPublic` returns before it compares any package path.
  <details><summary>details</summary>

  The source states the collision where the key is built: ["Struct types expressed or declared in different packages may have the same TypeID if and only if neither have unexported fields"](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/types.go#L780-L783), and [`InterfaceType.TypeID()` carries the same note for method names](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/types.go#L930-L933). Both transactions in the repro commit, so the buffer-and-promote gate never applies: the public verdict is promoted by [`transactionStore.Write`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/store.go#L395) and the private type then [hits it](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1299) and [skips the walk](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1432). A `[3]*Data` of nil pointers carries the private type with no private object, so the object-level check has nothing to see.

  The consequence is not only the leak. [`tests/typeprivacy_collide_restart.txtar`](tests/typeprivacy_collide_restart.txtar) runs the identical call twice around a `gnoland restart`: accepted warm, rejected cold, both at this sha. [`tests/typeprivacy_collide_poison.txtar`](tests/typeprivacy_collide_poison.txtar) is the leak on its own. Both pass at the merge base ddb752cac. [`tests/typeid_collision_test.go`](tests/typeid_collision_test.go) is the same defect in three lines with no node, using the store helper the PR's own test file already defines. Fix: make the memo key determine the packages the verdict was computed over.
  </details>

## Warnings (should fix)

- **[the description's benchmark table measures benchmarks the diff does not contain]** [`gnovm/pkg/gnolang/realm_assertpublic_bench_test.go:100-160`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L100-L160) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L100-L160) — the PR description reports `RepeatedCommits`, `AlwaysNewType` and `RepeatedCommits_SelfReferential`, including a 25% regression, and the file ships `ColdAcyclic`, `WarmAcyclic`, `ColdSelfReferential` and `WarmSelfReferential`.
  <details><summary>details</summary>

  A reader cannot reproduce the reported numbers or judge the reported regression, and the regression is the one line in the table that argues against the change. The four benchmarks that do ship measure cold against warm for two shapes, which is a different question from the description's per-call-count framing. The numbers above are what the shipped file produces at this sha. Fix: report the shipped benchmarks, and if the self-referential regression is still real under them, say so with the number they give.
  </details>

## Missing Tests

- **[nothing in the suite can see a key collision, because every case gets its own store]** [`gnovm/pkg/gnolang/realm_privatedep_test.go:28`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm_privatedep_test.go#L28) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_privatedep_test.go#L28) — `TestTypeHasPrivateDep_PublicStruct` and [`_OwnPackagePrivate`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm_privatedep_test.go#L39) already build the colliding pair, and pass only because [`newPrivateDepTestStore`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm_privatedep_test.go#L15-L22) hands each of them a fresh store.
  <details><summary>details</summary>

  Every test that could observe cross-package reuse of one key isolates itself from it by construction, so the suite would stay green through any fix that only moves the gate around. The two structs differ solely in `PkgPath`, which is the field the key omits. [`tests/typeid_collision_test.go`](tests/typeid_collision_test.go) is the missing case in the file's own style: it drives both types through one store, and covers the interface shape and the declared-type shape that does not collide.
  </details>

## Suggestions

- **[the realm's own package is now looked up where the base branch short-circuited]** [`gnovm/pkg/gnolang/realm.go:1337`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L1337) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1337) — `computeTypeHasPrivateDep` calls `isPkgPrivateFromPkgPath` for every package path it meets, the realm's own included, and that function [panics when the store has no `PackageValue`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm.go#L2365-L2367).
  <details><summary>details</summary>

  The base branch reached the lookup only after `pkgPath != rlm.Path`, so a realm saving its own type never needed its own package in the store. A sweep of every `Type` kind through both walks agrees on every other case; this one turns `ok` into `cannot find package value from store for path`. No path from gno source reaches it, since the realm's `PackageValue` is loaded before its objects are saved, so this is a new panic in a state the VM does not appear to produce rather than a defect. Fix: return false for `pkgPath == rlm.Path` before the lookup, or say in the comment that the store is required to hold it.
  </details>

- **[the benchmark helper writes the mutex-guarded map directly]** [`gnovm/pkg/gnolang/realm_assertpublic_bench_test.go:80`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L80) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L80) — `clearTypePrivacyMemo` replaces `ds.typePrivacyCache.m` without taking `mu`, which is the one lock the new type exists to hold.
  <details><summary>details</summary>

  It is a benchmark helper and nothing runs concurrently beside it today, so no run detects it. The cost is that the only in-tree example of touching this map shows it being touched the way the type forbids. Fix: add a `reset` method beside `get` and `set`.
  </details>

## Verified

- The commit gate closes the simulation route round 2 found. [`runTx`](https://github.com/gnolang/gno/blob/7f28a9bb3/tm2/pkg/sdk/baseapp.go#L944-L949) returns before `endTxHook` for every mode but Deliver, and `CheckTx` returns before `beginTxHook` runs at all, so no query, simulation or mempool check reaches `Write`.
- The refactor preserves the walk. A sweep of 42 cases covering every `Type` kind through `assertTypeIsPublic`, run at both shas from [`tests/equiv_typewalk_test.go`](tests/equiv_typewalk_test.go), agrees everywhere except the own-package lookup in the Suggestion, two renamed panic strings, and four cases built on an unnamed `FieldType`, which the preprocessor cannot produce: it [renames every unnamed parameter and result](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/preprocess.go#L486-L503) before a `FieldType` is built.
- Declared types do not collide. `TestTypeHasPrivateDep_TypeIDCollidesNotOnDeclared` passes at this sha, which bounds the Critical to structural keys.
- The private-realm path is not slower. A chain of private structs at depths 8, 32 and 128, cold cache per iteration, three runs of 3000 iterations at each sha: the ranges overlap at every depth. An earlier single run suggested a regression at depth 8 and did not survive repetition.
- Red CI is not this branch. `main / test` fails on `TestNodeBootWithInitialHeight`, which fails the same way on master.
- Green at 7f28a9bb3: `TestTypeHasPrivateDep*`, `TestAssertTypeIsPublic*`, `TestTestdata/typecache_poison`, `TestTestdata/typecache_restart_gas`.

## Open questions

- `isPkgPrivateFromPkgPath` goes through [`GetPackage`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/store.go#L459), which falls through to [`loadObjectSafe`](https://github.com/gnolang/gno/blob/7f28a9bb3/gnovm/pkg/gnolang/store.go#L634) and bills I/O gas for a non-stdlib package. Skipping the walk therefore skips a metered read whenever the walk would have been the first thing in the transaction to load that package. Round 2 measured warm against cold on the two shapes most likely to reach it and got identical gas, and no shape found since reaches it either, so there is nothing to report beyond the mechanism being present in the code.
- The memo has no eviction and is keyed by `TypeID`, which users control through the packages they deploy. Growth against a realistic deployment rate is still unmeasured.
