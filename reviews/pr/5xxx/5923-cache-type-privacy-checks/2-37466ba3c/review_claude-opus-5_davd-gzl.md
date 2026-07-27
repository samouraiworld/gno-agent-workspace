# PR [#5923](https://github.com/gnolang/gno/pull/5923): chore(perfs): Cache type-privacy checks across commits

URL: https://github.com/gnolang/gno/pull/5923
Author: Villaquiranm | Base: master | Files: 6 | +894 -36
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 37466ba3c (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5923 37466ba3c`

Round 2. Head advanced dcd6db417 → 37466ba3c: `fix review comments`, a merge of master, then `update comment`. The memo was redesigned in response to round 1: it moved off the `Type` object onto a store-level, TypeID-keyed map shared by reference into every transaction. That resolves round 1's first two Warnings and its Missing Test, and the stale ADR filename Nit is fixed. The third Warning is answered by disclosure rather than a change, which holds. The redesign promotes round 1's Suggestion into the blocking path: the verdict now outlives the transaction that computed it, and nothing scopes it to state that actually landed.

**TL;DR:** Before saving a realm object, the VM walks the object's whole type graph to check that nothing in it comes from a package marked private. This PR remembers the answer in a table on the node, keyed by the type's name, so later blocks can skip the walk.

**Verdict: REQUEST CHANGES** — a verdict computed inside a transaction that is thrown away is kept forever, so simulating a public build of a path disables the private-realm check for that path permanently and makes the outcome of a later transaction depend on which node was asked to simulate (1 Critical, 1 Warning, 1 Suggestion).

## Verify first

- [`gnovm/pkg/gnolang/realm.go:1293-1296`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1293-L1296) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1293-L1296) — the verdict is written with nothing tying it to whether the surrounding transaction commits. Confirm there is no path that discards it when the transaction is discarded: add the two txtars in `tests/` and run both, the control passes and the poison one fails.
- [`gnovm/pkg/gnolang/realm.go:1420`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1420) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1420) — a `false` verdict here returns before any enforcement runs, so every guarantee of the privacy feature rests on that one boolean. Confirm the memo can only be written from state that reached the chain.

## Summary

`assertTypeIsPublic` re-walks a saved object's full type graph on every commit. This PR memoizes the realm-independent half of that question, "does anything in this type reach a private package", so the walk can be skipped. Round 1 measured the original design getting zero hits in a node, because the memo lived on the `Type` object and every transaction reloads types as fresh objects. The redesign moves it to [`defaultStore.typePrivacyCache`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/store.go#L197) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L197), keyed by `TypeID` and [shared by reference into every transaction](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/store.go#L320) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L320), which does reach the update path and does now cache method-bearing and self-referential types.

Sharing the map by reference is also what breaks it. The write is unconditional and unscoped, so a transaction whose effects are thrown away still leaves its verdict behind. A simulated deployment is one such transaction, and it travels the [`.app/simulate` ABCI query](https://github.com/gnolang/gno/blob/37466ba3c/tm2/pkg/crypto/keys/client/broadcast.go#L243) · [↗](../../../../../.worktrees/gno-review-5923/tm2/pkg/crypto/keys/client/broadcast.go#L243), so it is answered by whichever single node the client happened to ask and never enters consensus.

## Diagram

```
simulate addpkg gno.land/r/zz/priv          real addpkg, private = true
(public build, ABCI query, discarded)       (broadcast, reaches the chain)
            │                                           │
            │ runs the package, commits                 │
            ▼                                           ▼
 typeHasPrivateDep(priv.Data) = false          typeHasPrivateDep(priv.Data)
            │                                           │
            │ cache.set  ── shared root-store map ──►  cache HIT: false
            │              (no rollback, no tx scope)    │
            ▼                                           ▼
   tx state discarded                    assertTypeIsPublic returns early
                                         private-realm check never runs
```

## Fix

`typeHasPrivateDep` consults the store memo, and [`assertTypeIsPublic` returns immediately on a `false` verdict](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1420-L1427) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1420-L1427) without walking or comparing any package path. The load-bearing constraint is stated in the code as "package privacy never changes once a package is created", which is true of packages that exist but says nothing about verdicts computed against packages that never came into existence. Committing a verdict only when the transaction that produced it commits restores the assumption the rest of the design rests on.

## Benchmarks / Numbers

Warm and cold gas for the identical no-op write, measured through the node with a restart between, on the two shapes most likely to diverge:

| shape | warm (same process) | cold (after restart) |
|---|---|---|
| realm holding a type from a package it does not import | 1348492 | 1348492 |
| the transaction that first pushes that type across a realm boundary | 1885842 | 1885842 |

No gas divergence in either shape, so round 1's Suggestion about a store-keyed memo eliding metered reads does not materialise as written.

## Critical (must fix)

- **[a discarded transaction turns the privacy check off permanently]** [`gnovm/pkg/gnolang/realm.go:1293-1296`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1293-L1296) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1293-L1296) — the verdict is written from inside any transaction that runs the walk, including one whose state never lands, so simulating a public build of a path and then deploying a private package there leaves the private-realm type check disabled for those types for the life of the process.
  <details><summary>details</summary>

  The write goes through the pointer [the root store shares with every transaction](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/store.go#L320) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L320), and nothing unwinds it when a transaction is discarded. A simulated `addpkg` runs the package and commits inside a throwaway transaction, so its verdict for each `TypeID` lands in the permanent map. Deploying a private package at the same path afterwards produces the same `TypeID`s, [`typeHasPrivateDep` hits the stale `false`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1288-L1292) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1288-L1292), and [`assertTypeIsPublic` returns before it can reject anything](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1420) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1420). A value carrying the private type but no object from the private realm, such as an array of nil pointers, then persists into a public realm; the object-level check does not see it because there is no private object to see.

  Simulation is an [ABCI query](https://github.com/gnolang/gno/blob/37466ba3c/tm2/pkg/crypto/keys/client/broadcast.go#L243) · [↗](../../../../../.worktrees/gno-review-5923/tm2/pkg/crypto/keys/client/broadcast.go#L243), answered by one node and never replicated, so the second consequence is that validators disagree: a node that served the simulation accepts the later transaction, a node that did not panics on it. The two txtars in [`tests/`](tests/typeprivacy_simulate_poison.txtar) differ only by the simulated `addpkg`; the control passes at 37466ba3c and the poison one fails, and both pass at the merge-base d14a03770. [`tests/privdep_poison_test.go`](tests/privdep_poison_test.go) is the same thing without a node: it drives a discarded transaction straight through `typeHasPrivateDep`, passes at the merge-base, and fails at 37466ba3c with `assertTypeIsPublic accepted gno.land/r/x/secret.Token from another realm`. Fix: commit a verdict only when the transaction that computed it commits.
  </details>

## Warnings (should fix)

- **[the comment states a guarantee the code does not have]** [`gnovm/pkg/gnolang/store.go:192-193`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/store.go#L192-L193) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L192-L193) — "the verdict is a pure, immutable function of TypeID" holds only across packages that exist, and the memo is also written from packages that never existed.
  <details><summary>details</summary>

  The same claim appears again at [`realm.go:1421-1423`](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm.go#L1421-L1423) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1421-L1423) as the stated reason the enforcement walk may be skipped, so it is the argument the whole optimization rests on, not decoration. `TypeID` does not distinguish a type declared in a private package from a structurally identical one declared in a public package at the same path, and privacy is read from the `PackageValue` at walk time rather than from anything the key carries. Whatever shape the fix for the Critical takes, both comments need to say which transactions a verdict may be drawn from. Fix: state the condition under which a verdict is cacheable rather than asserting that privacy is immutable.
  </details>

## Nits

None.

## Missing Tests

None. Round 1's benchmark gap is closed: the four benchmarks now come in [cold](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L80-L91) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L80-L91) and [warm](https://github.com/gnolang/gno/blob/37466ba3c/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L98-L108) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L98-L108) pairs, and the cold ones reset the map each iteration rather than reusing a warmed object.

## Suggestions

- **[the regression guard cannot fail for the reason it exists]** [`gno.land/pkg/integration/testdata/typecache_restart_gas.txtar:12-14`](https://github.com/gnolang/gno/blob/37466ba3c/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L12-L14) · [↗](../../../../../.worktrees/gno-review-5923/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L12-L14) — the file now says outright that it also passes with the cache disabled, which answers round 1's Warning honestly, but the same restart it already performs would catch the Critical if it asserted behavior rather than only gas.
  <details><summary>details</summary>

  Round 1 asked for a realm that produces memo hits. The disclosure added here is the better answer to that specific point: with the path unmetered, no realm shape can make warm and cold gas differ, so the test is a canary for a future metering change and nothing more. Measured warm against cold on two further shapes and both matched exactly, which is the table above. The gap that remains is different: the guard exercises cache warmth across a restart and checks only `GAS USED`, so a warmth-dependent correctness change passes it untouched. Not a blocker, and the shipped `tests/` pair covers the Critical directly.
  </details>

## Verified

- A simulated deployment decides the outcome of a later real transaction. The two txtars in `tests/` are identical but for one `-simulate only` line; the control rejects the leak at 37466ba3c and the poison one persists it with `GAS USED: 2105566`. Both pass at the merge-base d14a03770, so the behavior arrives with this diff.
- The bypass needs a type-only carrier. An earlier draft of the repro leaked `&Data{N: 1}` and was still rejected, by the object-level check rather than the type-level one; switching to `[3]*Data` of nil pointers is what isolates `assertTypeIsPublic`.
- The bypass reproduces without a node. [`tests/privdep_poison_test.go`](tests/privdep_poison_test.go) uses no symbol the PR introduces, so it runs on both sides: green at d14a03770, red at 37466ba3c.
- The memo is genuinely poisoned rather than merely bypassed. Tracing every `typePrivacyCache` read and write through the poison run shows `gno.land/r/zz/priv.Data -> false` set during the simulated deployment and hit afterwards.
- Cross-account poisoning is not reachable through `addpkg`: [`checkNamespacePermission`](https://github.com/gnolang/gno/blob/37466ba3c/gno.land/pkg/sdk/vm/keeper.go#L743) · [↗](../../../../../.worktrees/gno-review-5923/gno.land/pkg/sdk/vm/keeper.go#L743) runs before [`RunMemPackage`](https://github.com/gnolang/gno/blob/37466ba3c/gno.land/pkg/sdk/vm/keeper.go#L808) · [↗](../../../../../.worktrees/gno-review-5923/gno.land/pkg/sdk/vm/keeper.go#L808), so a package at a namespace the signer does not control never runs and never writes a verdict.
- Warm and cold gas match on both shapes in the table above, so the store-keyed memo does not elide metered reads.
- Green at 37466ba3c: `TestTypeHasPrivateDep*`, `TestAssertTypeIsPublic*`, `TestTestdata/typecache_restart_gas`, `TestTestdata/addpkg_private`.

## Open questions

- The memo has no eviction and is keyed by `TypeID`, which users control through the packages they deploy. Did not measure growth against a realistic deployment rate, so there is nothing to raise beyond noting that the map only ever grows.
