# PR [#6082](https://github.com/gnolang/gno/pull/6082): fix(tm2): Parallel queries on localClient by mocking mutex on ro client

URL: https://github.com/gnolang/gno/pull/6082
Author: Villaquiranm | Base: master | Files: 5 | +498 -13
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: 5c2227c96 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6082 5c2227c96`
Overview: [visual overview](../overview.html)

## Overview

A gno.land node opens three ABCI connections to its own application: consensus, mempool, and query. PR 5431 gave the query connection its own mutex so an `.app/simulate` could no longer stall block production. This PR removes that mutex outright, handing the query connection a `noopMutex` whose `Lock` and `Unlock` do nothing, so queries no longer serialise against each other either. The `localClient` field changes from `*sync.Mutex` to `sync.Locker` to make that possible.

The change trades a lock for an invariant. The pull request writes it down: everything reachable from `Application.Query` and `Application.Info` must be goroutine-safe. Whether that invariant actually holds is the whole review, because the lock was what made it not matter.

It does not hold. Two paths reachable from an ordinary RPC query race under `-race`, each with a negative control proving the diff is what exposes them.

```
                       consensus ─┐
                                  ├─ &l.mtx  (real mutex, unchanged)
                       mempool   ─┘

  before:  query ────────────────── &l.queryMtx  (real mutex)
  after:   query ────────────────── noLock       ← every caller enters together
                                       │
                     ┌─────────────────┴─────────────────┐
              .app/simulate                    handleQueryCustom
              height >= 1  snapshot, safe      vm/qeval, vm/qrender
              height <  1  shared checkState   shared gnolang.Type memos
                           ▲ C2                ▲ C1
```

**Verdict: REQUEST CHANGES** — two reproduced data races on the query path the diff opens, and the ADR's reachability argument for leaving one of them alone is contradicted by the node's own startup order (2 Critical, 6 Warnings, 2 Missing tests, 6 Nits, 2 Suggestions).

## Verify first

- [`gnovm/pkg/gnolang/types.go:1149-1156`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1149-L1156) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/types.go#L1149) — `BoundType` publishes `ft.bound` and fills its `Params`/`Results` non-atomically, and [`gnovm/pkg/gnolang/types.go:1281-1293`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1281-L1293) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/types.go#L1281) derives `TypeID` from exactly those fields. Run `go test -race -run TestParallelVMEval ./gno.land/pkg/gnoland/` with the fixture in `tests/`: the read-only connection fails, the same app behind a real mutex passes.
- [`tm2/pkg/sdk/helpers.go:56-64`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/sdk/helpers.go#L56-L64) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/sdk/helpers.go#L56) — below height 1, `Simulate` copies `app.checkState.ctx`, whose gas meter is one shared pointer. Run `go test -race -run TestPreFirstBlockSimulate ./gno.land/pkg/gnoland/`: 57 races, all on that meter.
- [`tm2/pkg/bft/proxy/client.go:16-18`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L16-L18) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L16) — the interface contract every future `ClientCreator` reads still promises "an independent mutex". Confirm the precondition a new implementer now has to satisfy is written where they will see it.

## Summary

The diff is five files: two code changes totalling +36 -13, a 114-line ADR, a six-line addendum to the 5431 ADR, and a 342-line integration test. `localClient.mtx` becomes a `sync.Locker` ([`tm2/pkg/bft/abci/client/local_client.go:22`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/abci/client/local_client.go#L22) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/abci/client/local_client.go#L22)), and `localClientCreator.queryMtx` is replaced by a shared stateless `noopMutex` ([`tm2/pkg/bft/proxy/client.go:52`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L52) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L52)). Every method body is untouched, so the change is entirely in which `Locker` the query client holds.

The mechanism works: queries genuinely run in parallel, and the new test proves overlap with a gauge placed beneath the ABCI client, which is the correct place for it. What the change does not carry is the validation its own invariant needs. The new test exercises `.app/simulate` of a bank send, which never enters the GnoVM, so the `gnoStore`, `cacheNodes` and type-graph sharing the ADR names as the load-bearing machinery is the one thing it does not touch. Both Criticals below live in that gap.

## Critical (must fix)

- **[correctness — data race]** `gnovm/pkg/gnolang/types.go:1149-1156` — two concurrent `vm/qeval` or `vm/qrender` calls race on the lazily memoised fields of type objects the whole process shares, and the raced field is what decides interface satisfaction.
  <details><summary>details</summary>

  `VMKeeper.newGnoTransactionStore` forks `cacheObjects`, `cacheTypes` and the allocator per call but keeps `cacheNodes: txlog.Wrap(ds.cacheNodes)` ([`gnovm/pkg/gnolang/store.go:255`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/store.go#L255) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/store.go#L255)), so every query's store fork reaches the same `*PackageNode` graph and the same `*FuncType` and `*StructType` objects. `BoundType` ([`gnovm/pkg/gnolang/types.go:1149-1156`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1149-L1156) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/types.go#L1149)) and `TypeID` ([`gnovm/pkg/gnolang/types.go:1281-1293`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1281-L1293) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/types.go#L1281)) both memoise with an unsynchronised check-then-set. `BoundType` is the sharper one: it assigns `ft.bound` to a composite literal, so a second goroutine can follow that pointer and read `Params`/`Results` while the first is still writing them — and `TypeID`, derived from those two fields, is what `InterfaceType.VerifyImplementedBy` ([`gnovm/pkg/gnolang/types.go:1029-1040`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/types.go#L1029-L1040) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/types.go#L1029)) compares to decide whether a type satisfies an interface.

  Reproduced through the production topology — `localClient.QuerySync` → `BaseApp.Query` → `handleQueryCustom` → `vmHandler.queryEval` → `Machine.Eval` → `Preprocess` → `VerifyImplementedBy` → `BoundType` — with `tests/app_parallel_vmeval_race_test.go`. Four races, every pair two query goroutines, no consensus involvement. The same app queried through `abcicli.NewLocalClient(new(sync.Mutex), app)`, which is the pre-6082 shape, passes clean, so the exposure is this diff's.

  The ADR credits [#5811](https://github.com/gnolang/gno/pull/5811) with closing this class. It does not: `sealUverseTypes` seals uverse types only, and [`gnovm/pkg/gnolang/uverse.go:1930`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/uverse.go#L1930) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/uverse.go#L1930) states the assumption this PR invalidates: "Per-store types are unaffected (each is preprocessed by a single goroutine)." Types a realm declares are not covered.

  Fix: make the memo fields on `Type` write-once under `sync.Once` or an atomic, rather than check-then-set.
  </details>

- **[correctness — data race]** `tm2/pkg/sdk/helpers.go:56-64` — before the first block, concurrent simulates share one gas meter and one cache store, and the window is not the one the ADR describes.
  <details><summary>details</summary>

  When `getLastBlockHeader()` reports height below 1, `Simulate` falls back to `getContextForTx`, which copies `app.checkState.ctx` by value. `CacheContext()` forks the store but not the gas-meter pointer, so every concurrent simulate charges into one `infiniteGasMeter`. `tests/app_parallel_prefirstblock_race_test.go` reports 57 races, all on that meter, at [`tm2/pkg/store/types/gas.go:290`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/types/gas.go#L290) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/store/types/gas.go#L290) reading against [`tm2/pkg/store/types/gas.go:294`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/types/gas.go#L294) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/store/types/gas.go#L294) writing; the same fixture behind a real mutex passes.

  The window outlives genesis. `Commit()` republishes the header it was handed ([`tm2/pkg/sdk/baseapp.go:1026`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/sdk/baseapp.go#L1026) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/sdk/baseapp.go#L1026)), and after `InitChain` that is `initHeader`, built with no `Height` ([`tm2/pkg/sdk/baseapp.go:356`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/sdk/baseapp.go#L356) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/sdk/baseapp.go#L356)) — so the genesis commit stores a header of height 0 and the fallback stays live until the first real `BeginBlock`. `CreateEmptyBlocks` defaults to true, so `gnoland start` leaves the window quickly, but [`contribs/gnodev/pkg/dev/node.go:93`](https://github.com/gnolang/gno/blob/5c2227c96/contribs/gnodev/pkg/dev/node.go#L93) · [↗](../../../../../.worktrees/gno-review-6082/contribs/gnodev/pkg/dev/node.go#L93) and [`gno.land/pkg/integration/node_testing.go:194`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/integration/node_testing.go#L194) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/integration/node_testing.go#L194) both set it false, making `WaitForTxs()` true ([`tm2/pkg/bft/consensus/config/config.go:86-88`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/consensus/config/config.go#L86-L88) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/consensus/config/config.go#L86)), so those two idle in the window until a transaction arrives. A restarted node is exempt: `initFromMainStore` replays the persisted header.

  Fix: in the `height < 1` branch, build a fresh context over `app.checkState.ms.MultiCacheWrap()` with its own gas meter instead of copying `app.checkState.ctx`.
  </details>

## Warnings (should fix)

- **[resource bound]** `tm2/pkg/bft/proxy/client.go:52` — the mutex was the only thing bounding aggregate query work, and nothing replaced it.
  <details><summary>details</summary>

  Every query entry point allocates its own budget per call: `maxAllocQuery` is 1.5 GB and `maxGasQuery` is 3e9 ([`gno.land/pkg/sdk/vm/keeper.go:50-52`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/sdk/vm/keeper.go#L50-L52) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/sdk/vm/keeper.go#L50)), installed fresh at [`gno.land/pkg/sdk/vm/keeper.go:1390-1394`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/sdk/vm/keeper.go#L1390-L1394) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/sdk/vm/keeper.go#L1390), and `withQueryEvalMachine` takes two allocators, one for runtime and one for preprocess. Those cap one query and never the sum. Before this PR the sum was the per-call cap, because the lock admitted one caller. Now the only ceiling in the path is the HTTP listener: `MaxOpenConnections` defaults to 900 ([`tm2/pkg/bft/rpc/config/config.go:105`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/rpc/config/config.go#L105) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/rpc/config/config.go#L105)), applied via `netutil.LimitListener` ([`tm2/pkg/bft/rpc/lib/server/http_server.go:288-289`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/rpc/lib/server/http_server.go#L288-L289) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/rpc/lib/server/http_server.go#L288)), and `test4`, `test5` and `test11` all pin 900. There is no ABCI-level semaphore anywhere under `tm2/pkg/bft/rpc/`.

  The same shift applies to CPU. At most one query goroutine could previously be executing application code; now up to `MaxOpenConnections` can, sharing `GOMAXPROCS` with block execution. Consensus is never blocked by a query, which is what 5431 fixed, but it is now competed with.

  Fix: bound in-flight queries on the read-only connection with a semaphore sized from `GOMAXPROCS`, so the removed lock becomes a widened bound rather than no bound.
  </details>

- **[test coverage]** `.github/workflows/_ci-go.yml:124` — the assertion the change rests on cannot run in CI, and it is not a one-word fix.
  <details><summary>details</summary>

  The new test's header says "Run with -race; without it this only checks result stability and overlap" ([`gno.land/pkg/gnoland/app_parallel_query_test.go:22`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go#L22) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_parallel_query_test.go#L22)), and the ADR lists "no data race, under `-race`" as the second of its three assertions. CI runs `go test -covermode=set -timeout 30m ... ./...` and `grep -rn -- "-race" .github/workflows/` returns nothing. The two flags are mutually exclusive:

  ```
  $ go test -covermode=set -race -count=1 -run TestParallelQueries_NWaySimulate ./gno.land/pkg/gnoland/
  -covermode must be "atomic", not "set", when -race is enabled
  ```

  CI does run the test — no `-short` is passed — so overlap and gas stability are checked. The property the node's correctness now depends on is not.

  Fix: add a `-race -covermode=atomic` step over `gno.land/pkg/gnoland`, `gno.land/pkg/sdk/vm` and `tm2/pkg/bft/proxy` only, rather than `./...`.
  </details>

- **[decay]** `tm2/pkg/bft/proxy/client.go:16-18` — the interface contract still promises the mutex the implementation just deleted, in three places.
  <details><summary>details</summary>

  The PR rewrote the doc on the concrete method into a careful invariant statement but left the `ClientCreator` interface declarations that define the contract: [`tm2/pkg/bft/proxy/client.go:16-18`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L16-L18) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L16) and [`tm2/pkg/bft/appconn/multi_app_conn.go:24-26`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/appconn/multi_app_conn.go#L24-L26) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/appconn/multi_app_conn.go#L24) both read "It uses an independent mutex so query calls never block consensus", and the production call site repeats it at [`tm2/pkg/bft/appconn/multi_app_conn.go:74`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/appconn/multi_app_conn.go#L74) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/appconn/multi_app_conn.go#L74). `git diff 4ebaa9543 HEAD` touches neither `appconn` file.

  The stale wording is the smaller half. The missing half is the precondition: implementing `NewReadOnlyABCIClient` now requires the application be goroutine-safe for `Query` and `Info`, and that requirement is documented only on `localClientCreator`. An alternative implementation written against the interface would read "independent mutex", supply one, and silently revert the change.

  Fix: put the goroutine-safety precondition on both interface declarations and drop the mutex wording from all three sites.
  </details>

- **[correctness — ADR claim]** `tm2/adr/pr6082_lock_free_query_connection.md:78-84` — the reason given for leaving the pre-first-block race unfixed is that no node serves queries in that window, and the node's own startup order says otherwise.
  <details><summary>details</summary>

  [`tm2/pkg/bft/node/node.go:676-682`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/node/node.go#L676-L682) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/node/node.go#L676) starts the RPC listeners and [`tm2/pkg/bft/node/node.go:711`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/node/node.go#L711) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/node/node.go#L711) starts the P2P switch, so consensus cannot commit block 1 until after the query endpoint is live. `gnoland start -x-early-start` — "start RPC and P2P before genesis time, deferring only consensus" ([`gno.land/cmd/gnoland/start.go:172-177`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/cmd/gnoland/start.go#L172-L177) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/cmd/gnoland/start.go#L172)) — widens it deliberately. The repository's own [`tm2/adr/pr5937_bptree_fastindex_working_tree.md:146-157`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/adr/pr5937_bptree_fastindex_working_tree.md#L146-L157) · [↗](../../../../../.worktrees/gno-review-6082/tm2/adr/pr5937_bptree_fastindex_working_tree.md#L146) documents this same window as real and client-visible, and lists two pre-existing issues found inside it.

  This matters beyond the wording: the ADR uses the unreachability claim to justify shipping the branch with that path unlocked, and the PR's own test steers around it on the same grounds ([`gno.land/pkg/gnoland/app_parallel_query_test.go:205-210`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go#L205-L210) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_parallel_query_test.go#L205)).

  Fix: drop the unreachability claim and fix the branch, per the second Critical.
  </details>

- **[correctness — ADR claim]** `tm2/adr/pr6082_lock_free_query_connection.md:90-92` — the test the ADR argues from is described wrongly, and this PR falsifies that test's own comments without touching it.
  <details><summary>details</summary>

  The ADR says `TestQueryRace_FastIndexParity` "pits one query against one committer". It runs four concurrent query goroutines ([`gno.land/pkg/gnoland/app_query_race_test.go:212`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L212) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_query_race_test.go#L212)). The ADR's conclusion — that it cannot observe two queries overlapping — was true, but only because `queryMtx` serialised them, which is the mutex this PR removes.

  That test's orchestration is written against the same mutex. [`gno.land/pkg/gnoland/app_query_race_test.go:179-181`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L179-L181) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_query_race_test.go#L179) explains that the block loop "must never touch the query mutex once the hook can pause a query (the hooked query holds the query mutex while waiting for the loop to commit)". After this PR the paused hook holds nothing, so the other three hammers keep running against it and the freeze becomes best-effort. [`gno.land/pkg/gnoland/app_query_race_test.go:11-13`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L11-L13) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_query_race_test.go#L11) and [`gno.land/pkg/gnoland/app_query_race_test.go:134-135`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_query_race_test.go#L134-L135) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_query_race_test.go#L134) are now false as written.

  Fix: correct the ADR sentence and update that file's three comments in this PR, since this change is what invalidated them.
  </details>

- **[decay]** `tm2/pkg/sdk/baseapp.go:551` — a cost this repository accepted three PRs ago was accepted on the premise this diff removes.
  <details><summary>details</summary>

  Every custom query and every simulate builds a fresh immutable multistore per call, via `MultiImmutableCacheWrapWithVersion` → `immutableAtVersion` → `ims.LoadVersion` ([`tm2/pkg/store/rootmulti/store.go:408-447`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/rootmulti/store.go#L408-L447) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/store/rootmulti/store.go#L409)), which constructs every substore from scratch. The 6018 review measured that at 0.50 ms with 1,000 retained versions and 57.6 ms at 60,000, against gno.land's default retention of 705,600, and signed it off on the grounds that "the query connection serializes on one mutex, so the cost caps throughput on every query path, not just `.store`". Serialised, that is a throughput ceiling. Unserialised, N concurrent queries each pay it at once, and the same review named the deferred fix: one cached immutable multistore per snapshot generation.

  Fix: say in the pull request body whether the cached-per-generation immutable multistore is a prerequisite or a follow-up, and record the changed cost model in the ADR.
  </details>

## Missing Tests

- **[test coverage]** `tm2/pkg/bft/proxy/client.go:51` — the package whose contract changed has no test files at all, and a 1.4 s test pins the property structurally.
  <details><summary>details</summary>

  `tm2/pkg/bft/proxy`, `tm2/pkg/bft/abci/client` and `tm2/pkg/bft/appconn` contain zero `_test.go` files between them, so the only guard on this change is a 342-line integration test in another directory that needs a database, the VM and `gnoenv`. The 5431 round-1 review already asked for a unit test here; 6082 is the PR that deletes the mutex it would have pinned.

  The gauge assertion in the new test is statistical: it observes that overlap happened. A gate application proves it structurally — reintroducing any lock deadlocks the test rather than slowing it down. `tests/proxy_client_test.go` does that in 1.4 s under `-race` with no database and no VM, cheap enough to run under `-race` in CI even if the coverage-mode conflict above is not resolved.
  </details>

- **[test coverage]** `gno.land/pkg/gnoland/app_parallel_query_test.go:75` — the validation covers the one query path that does not reach the GnoVM.
  <details><summary>details</summary>

  `.app/simulate` of a `bank.NewMsgSend` ([`gno.land/pkg/gnoland/app_parallel_query_test.go:181`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go#L181) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_parallel_query_test.go#L181)) routes to the bank handler. `vm/qeval` and `vm/qrender` go through `handleQueryCustom` ([`tm2/pkg/sdk/baseapp.go:529`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/sdk/baseapp.go#L529) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/sdk/baseapp.go#L529)) into `gno.Machine`, preprocess and the shared type graph — which is precisely the machinery the ADR's invariant names, and where the first Critical lives. Nothing in the tree runs two concurrent VM queries: `gno.land/pkg/sdk/vm/` contains no `go func`, `errgroup` or `WaitGroup` in any file. Production already fans out — [`gno.land/pkg/gnoweb/handler_http.go:1082`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoweb/handler_http.go#L1082) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoweb/handler_http.go#L1082) is an `errgroup` issuing several VM queries per page render.

  `tests/app_parallel_vmeval_race_test.go` closes it, in the file's own style and reusing its `concurrencyProbeApp` shape.
  </details>

## Nits

- **[simplification]** [`tm2/pkg/bft/abci/client/local_client.go:31-34`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/abci/client/local_client.go#L31-L34) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/abci/client/local_client.go#L31) — the nil guard is dead code, and the doc paragraph apologising for it is four lines. No caller passes nil: [`tm2/pkg/bft/proxy/client.go:34`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L34) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L34) and [`tm2/pkg/bft/proxy/client.go:52`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L52) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L52) pass `&l.mtx` and `noLock`, and `consensus/common_test.go:284-285` passes `new(sync.Mutex)`. Deleting the guard removes three lines of code, the doc paragraph and an ADR Consequences entry, breaks nothing, and turns "a typed nil silently panics later" into "any nil panics at the call site".
- **[simplification]** [`tm2/pkg/bft/proxy/client.go:55-62`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L55-L62) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L55) — `noLock` is a package-level mutable `var` in the diff whose subject is removing shared mutable state. `noopMutex{}` is an empty struct, so inlining it at the one call site allocates nothing and cannot be reassigned.
- **[docs]** [`tm2/pkg/bft/proxy/client.go:45-46`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L45-L46) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L45) — "only the read-only subset of appconn.Query (Echo, Info, Query) is safe to use on it" reads as a safety claim about those three names, but the `*Async` forms nil-deref on the unset `Callback` ([`tm2/pkg/bft/abci/client/local_client.go:226`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/abci/client/local_client.go#L226) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/abci/client/local_client.go#L226)). Pre-existing and identical on the consensus client, and unreachable through `appconn.Query`, which exposes only the `*Sync` trio ([`tm2/pkg/bft/appconn/app_conn.go:34-42`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/appconn/app_conn.go#L34-L42) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/appconn/app_conn.go#L34)). Name the `*Sync` methods.
- **[docs]** `tm2/adr/pr6082_lock_free_query_connection.md:69-71` — Consequences names only `SetResponseCallback`, but every method on the read-only client lost its lock, the mutating ones included. Only `EchoSync`, `InfoSync` and `QuerySync` are reachable through the wrapper, but `NewReadOnlyABCIClient` is public and returns the raw `abcicli.Client`, as both new tests do.
- **[docs]** `tm2/adr/pr6082_lock_free_query_connection.md` — no `## Alternatives considered` section, which `AGENTS.md` asks for and 15 of the 20 files in `tm2/adr/` carry. The RWMutex and per-method alternatives are folded into Decision prose, and the PR body's own second alternative — a `readOnly` flag that panics on mutating methods — appears nowhere, though it is what would close the Nit above.
- **[test quality]** [`gno.land/pkg/gnoland/app_parallel_query_test.go:335-341`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app_parallel_query_test.go#L335-L341) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app_parallel_query_test.go#L335) — phase 2 compares each querier's readings against its own round 0, which is itself a parallel round, so a systematic shift caused by concurrency moves every round equally and passes. It proves stability under concurrency, not equality with serial execution. A serial baseline captured before the barrier would close it.

## Suggestions

- **[API surface]** [`tm2/pkg/bft/proxy/client.go:51`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/proxy/client.go#L51) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/proxy/client.go#L51) — `NewReadOnlyABCIClient` returns the full `abcicli.Client`, so the narrowing the ADR's safety argument relies on happens one layer away in `appconn` and does not protect direct callers. Returning a narrow interface carrying only `Error`, `EchoSync`, `InfoSync`, `QuerySync` and `service.Service` would move that paragraph into the type system. Not posted: the ADR already weighs the shape of this constructor, and the contract finding above asks for the change that matters.
- **[ops]** [`tm2/pkg/telemetry/metrics/metrics.go:42-99`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/telemetry/metrics/metrics.go#L42-L99) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/telemetry/metrics/metrics.go#L41) — the change removes the only backpressure with no metric for the quantity it makes unbounded and no way to put it back. There is no in-flight query counter and no query-latency histogram; the only instrumentation an `abci_query` passes is the unlabelled `HTTPRequestTime` ([`tm2/pkg/bft/rpc/lib/server/handlers.go:292`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/bft/rpc/lib/server/handlers.go#L292) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/bft/rpc/lib/server/handlers.go#L292)), so a slow simulate is indistinguishable from a slow `status`. The PR's own test needs a bespoke gauge to see overlap, which is the tell. `noLock` is unconditional, so reverting to serialised queries in production needs a rebuild. Not posted as its own comment: the actionable half, a knob beside the bound, rides with the resource-bound finding.

## Verified

- Two concurrent `vm/qeval` calls on the read-only connection race on `FuncType.BoundType` and `FuncType.TypeID`: `TestParallelVMEval_ReadOnlyConn` fails under `-race` with 4 races, every pair two query goroutines. The same app behind `abcicli.NewLocalClient(new(sync.Mutex), app)` passes with 0 races in 83.9 s. Fixture: `tests/app_parallel_vmeval_race_test.go`.
- Concurrent `.app/simulate` before the first block races on one `infiniteGasMeter`: `TestPreFirstBlockSimulate_ReadOnlyConn` fails with 57 races, all on that object; the serialised control passes. Fixture: `tests/app_parallel_prefirstblock_race_test.go`.
- The `.app/simulate` snapshot path at height >= 1 is genuinely safe, which is the ADR's core claim and it survives. A simulate never reaches `endTxHook`, because `runTx` returns at [`tm2/pkg/sdk/baseapp.go:962-964`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/sdk/baseapp.go#L962-L964) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/sdk/baseapp.go#L962) for any non-deliver mode, so `CommitGnoTransactionStore` ([`gno.land/pkg/gnoland/app.go:227`](https://github.com/gnolang/gno/blob/5c2227c96/gno.land/pkg/gnoland/app.go#L227) · [↗](../../../../../.worktrees/gno-review-6082/gno.land/pkg/gnoland/app.go#L227)) never runs and `transactionStore.Write()` — the only path into the shared `cacheNodes` ([`gnovm/pkg/gnolang/store.go:293-295`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/store.go#L293-L295) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/store.go#L293)) — is unreachable from a query. One query cannot change what a later query reads or is charged.
- Snapshot pinning is refcounted correctly under N-way concurrency: `refSnapshot` uses an `atomic.Int64` ([`tm2/pkg/store/rootmulti/store.go:30-47`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/rootmulti/store.go#L30-L47) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/store/rootmulti/store.go#L30)) and `snapshotMu.RLock` covers the `Load` and `acquire` as a unit against `refreshQuerySnapshot`'s write lock. `go test -race -run 'Snapshot|Immutable|Concurrent|QueryRace|Isolation' ./tm2/pkg/store/rootmulti/` passes in 146.5 s.
- The per-transaction allocator really does fork: `BeginTransaction` sets `alloc: ds.alloc.Fork().Reset()` ([`gnovm/pkg/gnolang/store.go:257`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/store.go#L257) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/store.go#L257)) and `Fork` returns a fresh allocator reading only the parent's `maxBytes` and `bytes` ([`gnovm/pkg/gnolang/alloc.go:301-309`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/alloc.go#L301-L309) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/alloc.go#L301)).
- Gas is charged before the shared amino cache is consulted ([`gnovm/pkg/gnolang/store.go:818`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/store.go#L818) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/store.go#L818) precedes [`gnovm/pkg/gnolang/store.go:824`](https://github.com/gnolang/gno/blob/5c2227c96/gnovm/pkg/gnolang/store.go#L824) · [↗](../../../../../.worktrees/gno-review-6082/gnovm/pkg/gnolang/store.go#L824)), and the cached value is copied out rather than shared, so a cache hit or miss cannot change `GasUsed`. Ristretto's probabilistic admission has no determinism consequence.
- `noopMutex` shared across every read-only client is correct: it is a stateless empty struct.
- The `sync.Locker` signature change is source-compatible and not a breaking API change — `*sync.Mutex` satisfies `sync.Locker`, and the constructor returns an unexported `*localClient`, so no external caller can name the type.
- Both merge commits carry zero conflict-resolution content: `git show 5c2227c96 --cc` and `git show b1c78cd7b --cc` print headers only.
- `TestParallelQueries_NWaySimulate` passes under `-race` at the reviewed head in 44.4 s, and its `peak > 1` assertion is not a flake: it held across nine runs at default, `GOMAXPROCS=1` and `GOMAXPROCS=2`.
- CI is green on every check run at 5c2227c96; the one red commit status is `Merge Requirements`, awaiting reviewer approval.

## Open questions

- `immutableAtVersion` acquires the snapshot reference at [`tm2/pkg/store/rootmulti/store.go:422`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/rootmulti/store.go#L422) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/store/rootmulti/store.go#L422) and calls `ims.LoadVersion` at [`tm2/pkg/store/rootmulti/store.go:442`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/rootmulti/store.go#L442) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/store/rootmulti/store.go#L442) with no `defer`, so a panic between them leaks the pin permanently while the RPC layer recovers and keeps serving. Proven only with an injected panic; the in-tree candidate, `nameToKey`'s `panic("Unknown name ")` reached through a store-set change, could not be constructed. Not posted: no demonstrated trigger, and it predates the diff.
- `multiStore.Close()` swaps `querySnapshot` to nil without `snapshotMu` ([`tm2/pkg/store/rootmulti/store.go:362-367`](https://github.com/gnolang/gno/blob/5c2227c96/tm2/pkg/store/rootmulti/store.go#L362-L367) · [↗](../../../../../.worktrees/gno-review-6082/tm2/pkg/store/rootmulti/store.go#L362)), after which an in-flight query silently downgrades to reading the live, closing database. Reaches a `pebble: closed` panic at shutdown. Predates the diff and the concurrency it needs is what the diff adds; left out because a crash during shutdown is the mildest form of it.
- The PR title capitalises after the colon, against 93% lowercase adoption over the last 200 master subjects. No linter enforces it — `.github/workflows/meta-gh-title.yml` lints the conventional-commit prefix only — so it is not posted.
