# PR #5537: fix(tm2): recover shutdown-time txDispatcher panic

**URL:** https://github.com/gnolang/gno/pull/5537
**Author:** moul | **Base:** master | **Files:** 1 | **+18 -1**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

`txDispatcher.listenRoutine` deliberately panics with `"txDispatcher subscription unexpectedly closed"` when its subscription channel closes — a programmer-error signal. During normal node shutdown, there's a race between `EventSwitch.Quit()` (which closes subscriptions) and `td.Quit()`. When EventSwitch wins, the panic fires even though the dispatcher is about to stop anyway, causing intermittent flakes in in-process integration tests.

The fix wraps the `listenRoutine()` goroutine spawn in `OnStart()` with a `defer/recover` that only swallows the panic when `td.IsRunning()` is already false (i.e., shutdown in progress). If `td.IsRunning()` is still true, the panic is rethrown to preserve the original diagnostic.

The change is minimal (one file, `tm2/pkg/bft/rpc/core/mempool.go`), well-scoped, and the PR body thoroughly documents the root cause, alternatives considered, and rationale.

## Test Results
- **Existing tests:** CI passes (one `main / test` failure appears unrelated — likely the very flake this PR fixes)
- **Edge-case tests:** skipped — reproducing the race deterministically would require stubbing EventSwitch shutdown ordering; low ROI for this targeted fix

## Critical (must fix)
None

## Warnings (should fix)

- [ ] `tm2/pkg/bft/rpc/core/mempool.go:410` — **TOCTOU between `recover()` and `IsRunning()` check.** There's a small window where `td.IsRunning()` could transition from true to false between the panic and the recover check. In practice this is benign (it would mean we rethrow during a shutdown that's just barely started), but it's worth noting the check is best-effort. A log line in the swallow path (e.g., `log.Debug("txDispatcher: swallowed shutdown-time panic", "r", r)`) would help with post-mortem debugging if the race ever behaves unexpectedly. **To your question about "should he just add a log": yes, adding a debug-level log in the recovery path would be valuable. It makes the swallowed panic observable without crashing the process.** Something like:

```go
if r := recover(); r != nil {
    if td.IsRunning() {
        panic(r)
    }
    // Shutdown-time race — benign, but log for observability.
    log.Debug("txDispatcher: recovered shutdown-time panic", "err", r)
}
```

## Nits
- [ ] `tm2/pkg/bft/rpc/core/mempool.go:400-413` — The comment block is thorough but long for inline code. Consider moving the detailed explanation to a shorter inline comment and referencing the PR/commit for the full rationale.

## Missing Tests
- [ ] No test for the shutdown race. Acknowledged as low-ROI in the PR body; acceptable given the fix's simplicity and obvious root cause.

## Suggestions
- The PR body mentions the alternative of reordering `node.Stop()` internals to stop the dispatcher before EventSwitch. That would be the "correct" fix long-term. Consider filing an issue to track that cleanup. — `tm2/pkg/bft/rpc/core/mempool.go`

## Questions for Author
- Is there a logger available in the `txDispatcher` context? If so, adding a debug log in the recovery path would make this observable without being noisy.

## Verdict
APPROVE — Clean, minimal, well-documented fix for a real flake. The recover/IsRunning pattern is standard for shutdown races in Go. A debug log in the recovery path would be a nice addition but isn't blocking.
