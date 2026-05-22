# PR #5561: fix(tm2/rpc): don't panic when txDispatcher subscription closes on shutdown

**URL:** https://github.com/gnolang/gno/pull/5561
**Author:** thehowl | **Base:** master | **Files:** 4 | **+168 -2**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This is the "proper fix" for the `panic: txDispatcher subscription unexpectedly closed` flake that manifests during node teardown. A previous attempt (PR #5537 by moul) wrapped the panic with a `recover/IsRunning` check; this PR instead eliminates the panic entirely and addresses the root cause at three layers:

1. **mempool.go** — `listenRoutine` now treats a closed subscription channel as a clean shutdown signal (return + async `Stop()`) instead of panicking. This is correct because `SubscribeFilteredOn` (verified at `subscribe.go:55-56`) legitimately closes the channel when `evsw.Quit()` fires while the unbuffered send would block.

2. **pipe.go** — Adds `rpccore.Stop()` to tear down `gTxDispatcher`, and guards `SetEventSwitch` against leaking a previous dispatcher's goroutine. The NOTE comment on `SetEventSwitch` documents the known singleton limitation.

3. **node.go** — Calls `rpccore.Stop()` before `evsw.Stop()` in `Node.OnStop`, so the normal path exits via the dispatcher's own `Quit` channel rather than racing the event switch shutdown.

Two regression tests exercise both the direct-close path and the full event-switch-shutdown race (the latter by cleverly holding `td.mtx.Lock()` to stall `listenRoutine` inside `notifyTxEvent`, reproducing the exact CI failure sequence).

## Test Results

- **CI:** All checks pass (build, lint, test, analyze, e2e, CodeQL).
- **Codecov:** 75% patch coverage — 2 lines in `pipe.go` missing coverage (the `gTxDispatcher.Stop()` guard in `SetEventSwitch` and the nil-check in `Stop`). These are defensive paths for multi-node-in-process scenarios that the code explicitly says are unsupported.
- **Author verified:** Reverting the `listenRoutine` change makes `TestTxDispatcher_EventSwitchShutdown` fail with the original panic, confirming the test exercises the actual bug.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `tm2/pkg/bft/rpc/core/mempool.go:417` — **Async `Stop()` leaves a brief inconsistency window.** `go func() { _ = td.Stop() }()` means there's a period where the subscription is closed but `td.IsRunning()` still returns true. Any concurrent `BroadcastTxCommit` caller in `getTxResult` would see a running dispatcher but never receive a delivery event — it would timeout normally, which is acceptable. However, if `Stop()` could be called synchronously without deadlock (the old code did `td.Stop()` then `panic(...)` synchronously at the same call site), the async wrapper may be unnecessary. If async is kept for safety, a brief comment explaining why synchronous Stop could deadlock (e.g., "Stop may wait for this goroutine on some BaseService implementations") would clarify the intent.

- [ ] `tm2/pkg/bft/rpc/core/pipe.go:143-145` — **`Start()` is called on an already-started dispatcher.** `newTxDispatcher` (mempool.go:391) already calls `td.Start()`. Then `rpccore.Start()` calls `gTxDispatcher.Start()` again. This is a pre-existing issue, but since this PR is modifying the lifecycle, it would be a good moment to either make `rpccore.Start()` a no-op (remove the body, or add a guard) or remove the `td.Start()` from `newTxDispatcher` and rely on `rpccore.Start()` to start it. This would also make the `SetEventSwitch` → `Stop` → `newTxDispatcher` → `Start` sequence cleaner.

## Nits

- [ ] `tm2/pkg/bft/rpc/core/mempool_test.go:102` — The `defer func() { _ = recover() }()` comment says "tolerate preexisting SubscribeFilteredOn panics" but doesn't specify what panics. Consider a slightly more specific comment like "tolerate panics from SubscribeFilteredOn's callback when evsw.Quit() fires during an ongoing FireEvent" for future readers.

- [ ] `tm2/pkg/bft/rpc/core/mempool_test.go:107` — `time.Sleep(50 * time.Millisecond)` is a timing-dependent synchronization point. While pragmatic and well-commented, if the CI runners are heavily loaded, 50ms may be insufficient. Consider using a channel-based synchronization instead (e.g., have the second FireEvent callback signal that it's blocking before calling `evsw.Stop()`). This is a low-priority suggestion since the test already passed 30/30 runs.

## Missing Tests

- [ ] **`SetEventSwitch` goroutine-leak prevention** — The guard at `pipe.go:136-138` (stopping a previous dispatcher when `SetEventSwitch` is called again) has no test. A test that calls `SetEventSwitch` twice and verifies the first dispatcher's goroutine exits would cover the Codecov gap and validate the leak fix.

- [ ] **`rpccore.Stop()` idempotency** — No test verifies that calling `Stop()` twice, or calling `Stop()` when `gTxDispatcher` is nil, behaves correctly. A simple unit test would close the Codecov gap on the nil-check branch.

## Suggestions

- The `SetEventSwitch` NOTE comment (pipe.go:121-132) is excellent documentation of a known architectural limitation. Consider filing a tracking issue for the "threading per-node state through RPC handlers" proper fix, and referencing it in the comment.

- Compared to PR #5537's recover-based approach, this PR's solution is strictly better: it eliminates the panic at the source, adds the correct shutdown ordering, prevents goroutine leaks, and includes regression tests. This should be preferred over #5537.

## Questions for Author

- Why async `Stop()` in `listenRoutine`? The old code called `td.Stop()` synchronously at the same point. If there's a deadlock risk with the current BaseService implementation, a brief comment would help. If there isn't, calling `td.Stop()` synchronously before `return` would close the inconsistency window.

## Verdict

APPROVE — This is a thorough, well-reasoned fix that addresses the root cause at three layers rather than papering over it with a recover. The shutdown ordering fix in `Node.OnStop`, the goroutine-leak guard in `SetEventSwitch`, and the two regression tests (especially the clever `mtx.Lock` stalling technique) are all high quality. The async `Stop()` and double-`Start()` are minor concerns worth clarifying but not blocking.
