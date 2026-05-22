# PR #5574: refactor(tm2/rpc): move rpc/core globals into per-node Environment

**URL:** https://github.com/gnolang/gno/pull/5574
**Author:** thehowl | **Base:** master | **Files:** 20 | **+753 -1599**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR eliminates the 13 package-level globals in `tm2/pkg/bft/rpc/core` (e.g. `stateDB`, `blockStore`, `consensusState`, `mempool`, `evsw`, `gTxDispatcher`) by introducing a per-node `*core.Environment` struct. Every RPC handler becomes a method on `*Environment`; `Node` owns one Environment per instance. This fixes the core bug where two in-process nodes (integration tests with `INMEMORY_TS=1`, parallel tests, multiple `gnodev` instances) would clobber each other's globals — whoever called `Set*` last owned every handler.

Stacked on #5561 which added defensive scaffolding around the `txDispatcher` shutdown race; this PR removes that scaffolding and fixes the underlying design. 4 commits total; the author asks reviewers to focus on the top commit (`4be1a4a5`).

### Changes by area

**Core refactor (pipe.go, routes.go):** New `Environment` struct with 13 dep fields + internal `txDispatcher`/`started`/`stopped` state. `Start()` creates the txDispatcher iff EventSwitch is non-nil; `Stop()` tears it down. `Routes(unsafe bool)` builds a fresh route map from bound method values. Lifecycle is a one-way door: Start-after-Stop panics.

**Handler conversion (abci.go, blocks.go, consensus.go, dev.go, health.go, mempool.go, net.go, status.go, tx.go):** All 23 handlers converted from package-level functions to `(env *Environment)` methods. Large HTTP-example comment blocks (swagger-style docs) removed from every handler — net -1093 lines of comment deletion, which accounts for most of the -1599 delta.

**Node integration (node.go):** `configureRPC()` now builds an `Environment` literal instead of 13 `Set*` calls. `OnStart()` calls `env.Start()`. `OnStop()` calls `env.Stop()` before `evsw.Stop()` (fixing the shutdown race). New `RPCEnvironment()` accessor for Local client binding.

**Local client (client/local.go):** `NewLocal(env *core.Environment)` replaces `NewLocal()`. All 22 handler calls dispatch through `c.env.Method()` instead of `core.Function()`.

**gnodev callers (accounts.go, dev/node.go):** `accounts.go` now creates a `client.NewLocal(devNode.Node.RPCEnvironment())` per call. `dev/node.go` creates the client in `rebuildNode()` after the node is ready, instead of during `NewDevNode()` construction.

**Tests:** New `environment_test.go` with isolation, StartStop idempotency, and BroadcastTxCommit-rejects-unstarted tests. Existing `blocks_test.go`, `status_test.go`, `tx_test.go` converted to `Environment` literals. `mempool_test.go` (from #5561) unchanged — still exercises txDispatcher directly.

## Test Results

- **Existing tests:** PASS — `go test ./tm2/pkg/bft/rpc/core/... -race -short` (1.1s), `go test ./tm2/pkg/bft/rpc/client/... -race -short` (1.1s), `go test ./tm2/pkg/bft/node/... -short` (7.2s). CI full matrix green.
- **Edge-case tests:** Skipped

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `tm2/pkg/bft/rpc/core/dev.go:39` — `UnsafeStopCPUProfiler` nil-dereferences `profFile` if called without a prior `UnsafeStartCPUProfiler`. Pre-existing issue, but now it's a method on `Environment` and more discoverable. Should guard with `if profFile == nil { return nil, errors.New("profiler not running") }`.

- [ ] `tm2/pkg/bft/node/node.go:685` — `_ = n.rpcEnv.Stop()` silently discards the return value. This is safe today because `Environment.Stop()` panics on real errors (so the discard only ever receives `nil`), but the intent is not obvious to future readers. Consider either logging the error or adding a comment explaining why the discard is safe.

## Nits

- [ ] `tm2/pkg/bft/rpc/core/pipe.go:63` — `Consensus` field naming (vs the `ConsensusState` handler method) is documented in the PR body but not in the code. A one-line comment like `// Named Consensus to avoid collision with the ConsensusState handler method` would prevent future confusion.

## Missing Tests

- [ ] `Environment.Routes(unsafe bool)` is untested — no test verifies the route map is correctly built or that unsafe routes are included/excluded.
- [ ] `client.NewLocal(env)` with the new signature has no test — the existing `local_test.go` (if any) likely tests the old `NewLocal()` zero-arg form.
- [ ] Handler tests for `ConsensusState`, `DumpConsensusState`, `ConsensusParams`, `Validators`, `NetInfo`, `Genesis` were not converted or added — these handlers have zero unit test coverage per Codecov (0% on consensus.go, net.go).
- [ ] `UnsafeStopCPUProfiler` nil-deref path (`profFile == nil`) is untested.

## Suggestions

- Add a nil guard in `UnsafeStopCPUProfiler` (`dev.go:38-39`): `if profFile == nil { return nil, errors.New("CPU profiler not running") }`. This is a 2-line defensive fix that prevents a clear nil-deref on a publicly accessible unsafe route.
- Document the `Consensus` field naming choice inline at `pipe.go:63`.
- The 86 lines of missing coverage per Codecov (0% on `local.go`, 36% on `mempool.go`, 0% on `consensus.go`, `net.go`, `abci.go`, `dev.go`) are mostly pre-existing but this PR is a good opportunity to add at least smoke tests for the method-binding on `local.go`.

## Questions for Author

- The PR is stacked on #5561. Is the plan that #5561 lands first and this PR rebases to a single commit, or will both be merged together? If #5561 doesn't land first, the defensive scaffolding from that PR would still be present in the diff.
- Was there consideration of making `Environment` fields unexported with a constructor function (e.g., `NewEnvironment(...)`) to prevent post-Start field mutation? The current public fields mean callers can nil-out `Consensus` or `BlockStore` after `Start()`.

## Verdict
APPROVE — Clean, well-scoped refactor that eliminates a real concurrency bug (global state clobbering between in-process nodes). The design is sound: one Environment per Node, method-value route binding, one-way lifecycle with panic-on-misuse. Critical paths (txDispatcher lifecycle, shutdown ordering, isolation) are tested. Warnings are minor defensive improvements, not correctness issues.
