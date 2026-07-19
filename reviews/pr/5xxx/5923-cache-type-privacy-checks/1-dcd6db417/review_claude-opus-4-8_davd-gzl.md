# PR [#5923](https://github.com/gnolang/gno/pull/5923): chore(perfs): Cache type-privacy checks across commits

URL: https://github.com/gnolang/gno/pull/5923
Author: Villaquiranm | Base: master | Files: 6 | +783 -32
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: dcd6db417 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5923 dcd6db417`

**TL;DR:** Before saving a realm object, the VM walks the object's whole type graph to check that nothing in it comes from a package marked private. This PR remembers the answer on the type itself so the walk can be skipped next time.

**Verdict: REQUEST CHANGES** — the memo is correct but almost never reaches the case it targets, and the regression guard shipped with it still passes with the memo switched off (3 Warnings, 1 Nit, 1 Suggestion).

## Summary

`assertTypeIsPublic` re-walks a saved object's full type graph on every commit. This PR adds `typeHasPrivateDep`, a realm-independent "does anything here come from a private package" check whose answer is memoized in a new `privateDep uint8` tristate on `StructType`, `InterfaceType`, and `DeclaredType`. A walk that crosses a cycle discards everything it computed rather than risk freezing a premature answer. The two traversals were folded into one shared `typePkgPathAndChildren` helper.

The memo lives on the Go object, so it only pays off when the identical `Type` object reaches `assertTypeIsPublic` twice. Two things stop that from happening in a node. Types reloaded from the store come back as a fresh object every transaction, because `cacheTypes` is transaction-scoped. And a declared type carrying any method is walked as a cycle, because its method values carry the receiver, so it is never committed to the memo at all. Measured on the PR's own example realm across five consecutive transactions: zero memo hits, and the same seven `isPkgPrivateFromPkgPath` lookups per call as before.

```
 microbenchmark                      node
 ──────────────                      ────
 root ──► same *StructType           tx1: GetType ──► *DeclaredType @0x…d860 ─┐
          every b.Loop()             tx2: GetType ──► *DeclaredType @0x…2000 ─┤ different objects,
          ⇒ memo hits                tx3: GetType ──► *DeclaredType @0x…      ─┘ privateDep = 0 each time
```

## Glossary

- transactionStore: the per-transaction Store wrapper returned by `BeginTransaction`, carrying per-tx caches and the tx-scoped gas meter.
- TypeID: a gno type's canonical string identity, deciding type equality and persisted in on-chain object state.
- defined type: a type introduced by `type N U`, a `DeclaredType` in the GnoVM, carrying its own method set.
- private package: a package whose `gnomod.toml` sets `private = true`; another realm may not persist a value whose type reaches into it, which is what `assertTypeIsPublic` enforces.
- txtar: testscript-based integration tests under `gno.land/pkg/integration/testdata/`.

## Fix

`assertTypeIsPublic` used to walk the graph itself and check each node's `pkgPath` against `rlm.Path`, at [`realm.go:1446-1470`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm.go#L1446-L1470) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1446-L1470). It now consults `typeHasPrivateDep` first and returns immediately when the graph touches no private package at all. The load-bearing constraint is that the realm-own exemption (`pkgPath == rlm.Path`) cannot enter the memo, since it depends on which realm is asking; only the realm-independent half is cached, and the exact realm-aware walk still runs whenever the fast path cannot rule out a violation.

## Benchmarks / Numbers

Microbenchmarks, merge-base [42c8946c7](https://github.com/gnolang/gno/commit/42c8946c71639ca0a566dee5a00cee8bcdd31eda) vs dcd6db417, PR's own benchmark file on both sides, min ns/op over 6-12 runs:

| Benchmark | base | PR |
|---|---|---|
| `RepeatedCommits` | 9527 ns, 4048 B, 51 allocs | 62 ns, 0 B, 0 allocs |
| `AlwaysNewType` | 52726 ns, 20509 B, 245 allocs | 30332 ns, 17728 B, 204 allocs |
| `RepeatedCommits_SelfReferential` | 1902 ns, 1032 B, 17 allocs | 1310 ns, 1160 B, 20 allocs |
| private-own-realm graph (added for this review) | 3554 ns, 4048 B, 51 allocs | 349 ns, 320 B, 1 alloc |

The self-referential row measures faster here, not the ~25% slower the PR description reports; only its allocation count regresses, by 3 allocs and 128 B per call.

Memo hit rate through the real keeper path, one transaction store per call, four to five consecutive calls per realm shape:

| realm shape | hits | misses | walks discarded for a cycle |
|---|---|---|---|
| updates an existing object (the PR's own `typecache` realm) | 0 | 8 | 1 |
| creates a new object of a reused, method-free type | 2 | 7 | 1 |
| self-referential `avl.Node`-shaped type | 0 | 7 | 3 |

## Critical (must fix)

None.

## Warnings (should fix)

- **[the speedup does not reach a running node]** [`gnovm/pkg/gnolang/realm.go:1273-1276`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm.go#L1273-L1276) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1273-L1276) — the memo lives on the `Type` object, and a type reloaded in a later transaction is a different object, so the update path never gets a hit.
  <details><summary>details</summary>

  `BeginTransaction` gives every transaction a fresh `cacheTypes` map at [`store.go:254`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/store.go#L254) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L254), so [`GetTypeSafe`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/store.go#L798) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L798) re-decodes from the backend and hands back a new object whose `privateDep` is zero. Running the PR's own `typecache` realm through the keeper for five consecutive transactions gives 0 hits and 8 misses on every call, with `isPkgPrivateFromPkgPath` called 7 times each time, the same as before the change. A realm that creates new objects of a reused method-free type does hit, twice out of nine lookups, because those types come from the package node rather than the store. Fix: hang the memo off something that outlives a transaction, or scope the PR's claim to the create path.
  </details>

- **[any type with a method is silently excluded]** [`gnovm/pkg/gnolang/realm.go:1371-1379`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm.go#L1371-L1379) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1371-L1379) — a declared type's children include its own method values, which carry the receiver, so walking any method-carrying type crosses a cycle and commits nothing.
  <details><summary>details</summary>

  `typePkgPathAndChildren` appends `method.T` and `mv.Type` for each method, and a bound method's signature names the receiver type, which closes a self-loop back to the declared type. The walk sets `sawCycle` at [`realm.go:1314`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm.go#L1314) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1314) and drops the whole `pending` list. Measured on real gno source: `type Coin struct{ Amount int }` memoizes, and the same type plus `func (c Coin) Get() int` does not, even though `Get`'s signature never names `Coin`. The exclusion is documented as covering linked-list shapes; it covers every type that has a method. See [`privdep_reach_test.go`](tests/privdep_reach_test.go). Fix: exclude only the nodes actually at risk, or say in the code comment that method-carrying types are out.
  </details>

- **[the guard passes with the feature switched off]** [`gno.land/pkg/integration/testdata/typecache_restart_gas.txtar:44-58`](https://github.com/gnolang/gno/blob/dcd6db417/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L44-L58) · [↗](../../../../../.worktrees/gno-review-5923/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L44-L58) — the realm it exercises produces no memo hits, so the warm and cold calls it compares do identical work.
  <details><summary>details</summary>

  The test's stated property is that billed gas must not vary with memo warmth, and it names a future gas-metering change as the trap it guards. `SaveItem` only updates an existing object, which is the case measured at 0 hits above, so calls 2 and 3 are both cold. Making `getPrivateDepCache` return a miss unconditionally leaves the test green at the same `EXACT_GAS`, which is the direct proof that it never touches the memo. Fix: drive the guard with a realm that actually produces hits, per the create-path numbers above.
  </details>

## Nits

- **[a reader cannot find the referenced document]** [`gno.land/pkg/integration/testdata/typecache_restart_gas.txtar:2`](https://github.com/gnolang/gno/blob/dcd6db417/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L2) · [↗](../../../../../.worktrees/gno-review-5923/gno.land/pkg/integration/testdata/typecache_restart_gas.txtar#L2) — points at `gnovm/adr/prxxxx_type_privacy_dependency_cache.md`; the file added by this PR is [`pr5923_type_privacy_dependency_cache.md`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/adr/pr5923_type_privacy_dependency_cache.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/adr/pr5923_type_privacy_dependency_cache.md#L1). Same stale name at [`realm_assertpublic_bench_test.go:50`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L50) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L50) and [`:86`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L86) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L86).

## Missing Tests

- **[the benchmark cannot show the gap it is meant to expose]** [`gnovm/pkg/gnolang/realm_assertpublic_bench_test.go:53-60`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L53-L60) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm_assertpublic_bench_test.go#L53-L60) — the two repeated-commit benchmarks hand `assertTypeIsPublic` the same Go object on every iteration, which no node does, so neither measures the memo's real hit rate.
  <details><summary>details</summary>

  `BenchmarkAssertTypeIsPublic_RepeatedCommits` reuses one `*StructType` built once outside the loop, and its self-referential counterpart reuses one `*DeclaredType` the same way, so both measure the best case the memo can ever have rather than the one the realm save path presents. A benchmark that reloads the type through `BeginTransaction`/`GetType` between iterations would have surfaced the first Warning before the ~92x figure went into the PR description. [`privdep_reach_test.go`](tests/privdep_reach_test.go) is the assertion-shaped version of that: it fails at dcd6db417 and passes once the memo outlives a transaction.
  </details>

## Suggestions

- **[moving the memo re-opens a question the current guard cannot answer]** [`gnovm/pkg/gnolang/realm.go:1319`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/realm.go#L1319) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/realm.go#L1319) — a memo keyed off the store rather than the type object would start eliding `isPkgPrivateFromPkgPath` calls that are cold backend reads, which are gas-metered.
  <details><summary>details</summary>

  `isPkgPrivateFromPkgPath` calls `store.GetPackage`, and the first such call per transaction for a given path falls through to `loadObjectSafe`, which charges amino-decode gas and allocation at [`store.go:551-568`](https://github.com/gnolang/gno/blob/dcd6db417/gnovm/pkg/gnolang/store.go#L551-L568) · [↗](../../../../../.worktrees/gno-review-5923/gnovm/pkg/gnolang/store.go#L551-L568). Today nothing diverges, because the memo never elides a first-in-transaction lookup: the only types it caches belong to packages the transaction has already loaded in order to run them. A cross-transaction memo would remove that property, and the guard as written would not notice.
  </details>

## Verified

- The memo never contradicts a cache-free reachability walk. 5000 random type graphs over four packages with a random private subset, each graph queried in four shuffled rounds against a shared set of type objects so earlier answers feed later ones, method self-references included: no disagreement. [`privdep_differential_test.go`](tests/privdep_differential_test.go).
- The type object handed to `assertTypeIsPublic` changes identity across transactions: `gno.vm/t/hello.Coin` is `0x…d860` with `privateDep=1` in the first transaction and `0x…2000` with `privateDep=0` in the second.
- `typecache_restart_gas.txtar` stays green with `getPrivateDepCache` forced to always miss, at the same `EXACT_GAS`.
- Memo hit counts through `gno.land/pkg/sdk/vm`'s keeper, one transaction store per call: see the table above.
- Benchmarks re-measured on both sides by restoring `realm.go` and `types.go` from the merge-base under the PR's own unmodified benchmark file.
- Green at dcd6db417: `TestTypeHasPrivateDep*`, `TestAssertTypeIsPublic*` (also under `-race`), `./gno.land/pkg/sdk/vm/ -run Gas`, `TestTestdata/typecache_restart_gas`.

## Open questions

- `typeHasPrivateDep` calls `isPkgPrivateFromPkgPath` for own-realm types too, where `assertTypeIsPublic` short-circuits on `pkgPath != rlm.Path` first. Benchmarked the private-own-realm shape on both sides and it is 10x faster on the PR, so there is nothing to raise.
- The fast path stops populating `visited` with the subtree's TypeIDs, so a later object of a nested type is re-checked instead of skipped. Costs nothing observable and no behavior depends on the map's contents, so not posted.
