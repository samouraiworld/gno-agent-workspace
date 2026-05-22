# PR #5495: fix(node): ensure halt_height triggers clean process shutdown

**URL:** https://github.com/gnolang/gno/pull/5495
**Author:** aeddi | **Base:** chain/gnoland1 | **Files:** 5 | **+38 -5**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes the halt-height shutdown behavior introduced in PR #5334. The original implementation panicked with a raw string in `BeginBlock`, which was caught by the consensus `receiveRoutine`'s blanket `recover()` handler, logging `CONSENSUS FAILURE!!!` but leaving the process in a zombie state (consensus dead, P2P/RPC still running). On cold restart, the panic propagated unrecovered through `Handshaker.replayBlock`, crashing with exit code 2.

The fix introduces a typed error `HaltHeightReachedError` and handles it in two places:

1. **Warm halt** (`tm2/pkg/bft/consensus/state.go:611-618`): The `receiveRoutine` recovery handler detects `HaltHeightReachedError`, logs an INFO message, calls `onExit()` to close the WAL, then sends SIGTERM via `osm.Kill()` to trigger the normal graceful shutdown path (exit code 0).

2. **Cold restart** (`tm2/pkg/bft/node/node.go:256-264`): `doHandshake` gets a `defer/recover` that catches `HaltHeightReachedError` during block replay and converts it to a regular error with an actionable message, propagating cleanly through `NewNode` (exit code 1).

3. **Typed error** (`tm2/pkg/bft/types/errors.go:43-51`): New `HaltHeightReachedError` struct implementing `error`, placed alongside existing domain errors.

4. **baseapp.go** (`tm2/pkg/sdk/baseapp.go:535`): The panic value changes from `fmt.Sprintf(...)` string to `bft.HaltHeightReachedError{...}`.

5. **baseapp_test.go** (`tm2/pkg/sdk/baseapp_test.go:1303-1308`): Test updated from `require.Panics` to `require.PanicsWithValue` to verify the typed error.

**Active discussion:** There is ongoing discussion about whether to adopt this "exit 0" approach vs. the cosmos-sdk "full zombie" approach (keeping node alive but shutting down consensus + P2P + RPC). The maintainers (@moul, @tbruyelle, @aeddi) have not reached consensus yet; the PR may be reworked or closed depending on that decision.

## Test Results
- **Existing tests:** `TestHaltHeight` in `tm2/pkg/sdk` — PASS (all 4 subtests pass)
- **CI:** TM2 test suite has a failure in `TestWALCrash/empty_block` (timeout after 30m), which is a known flaky test unrelated to this PR. All other checks pass.
- **Edge-case tests:** skipped

## Critical (must fix)
None

## Warnings (should fix)

- [ ] `tm2/pkg/bft/consensus/state.go:616` — `osm.Kill()` return value is silently discarded. The existing usage at `state.go:1389` checks the error and logs if `Kill()` fails. This new call site should do the same for consistency and debuggability:
  ```go
  if err := osm.Kill(); err != nil {
      cs.Logger.Error("Failed to kill process after halt height", "err", err)
  }
  ```

- [ ] `tm2/pkg/bft/consensus/state.go:614-616` — `onExit()` is called before `osm.Kill()`. If `onExit()` panics (e.g., WAL stop fails), `osm.Kill()` will never execute and the node will be left in an undefined state. The existing code at line 633 also calls `onExit()` in the deferred function's fallthrough path, meaning if the halt-height branch is taken, `onExit()` will NOT be called twice (the `return` on line 617 prevents reaching line 633), which is correct. However, wrapping `onExit()` in a separate recovery or moving `osm.Kill()` before `onExit()` would be more defensive.

## Nits

- [ ] `tm2/pkg/bft/types/errors.go:45` — The `Height` field is `uint64`, matching `app.haltHeight`. Consistent, good.

- [ ] `tm2/pkg/bft/node/node.go:260` — The error message says "remove or increase halt_height in config before restarting" which is actionable and clear. Good UX.

## Missing Tests

- [ ] No integration/unit test for the warm-halt path in `receiveRoutine`. The consensus `state.go` recovery handler detecting `HaltHeightReachedError` and calling `osm.Kill()` is untested. This is understandable given the complexity of testing consensus lifecycle, but worth noting — Codecov flags 5 missing lines + 1 partial in `state.go`.

- [ ] No unit test for the cold-restart `doHandshake` recovery path. The `node.go` defer/recover converting `HaltHeightReachedError` to an error is untested (4 missing lines + 1 partial per Codecov). A test could mock a block store that triggers the panic during replay.

## Suggestions

- Consider adding a `// TODO` comment noting that the warm-halt and cold-restart paths need integration tests, even if they can't be easily unit-tested today.

- The decision between "exit 0" vs "full zombie" should be explicitly documented (e.g., in a code comment or ADR) once the maintainers converge. The cosmos-sdk rationale for keeping P2P alive (preventing connectivity loss for peers) is non-obvious and worth recording regardless of which approach is chosen.

## Questions for Author

- Has the interaction between `osm.Kill()` (which sends SIGTERM) and the `TrapSignal` handler (which calls `os.Exit(0)`) been verified when the signal is sent from within the same process? Specifically: the SIGTERM is delivered asynchronously via a channel, so there's a brief window where the `receiveRoutine`'s deferred function has returned but `TrapSignal`'s goroutine hasn't yet called `os.Exit(0)`. Could any other goroutine observe partial state during this window?

- The PR description mentions "exit code 0" for warm halt. Does the `Node.OnStop` cleanup (called by `TrapSignal`'s callback) properly handle the case where `receiveRoutine` has already called `onExit()` (WAL stopped, `cs.done` closed)? Could double-close of `cs.done` panic?

## Verdict
NEEDS DISCUSSION — The code is well-structured and the typed-error approach is sound, but the PR's fate depends on the ongoing maintainer discussion about exit-0 vs. zombie behavior. The `osm.Kill()` error handling inconsistency should be fixed regardless of which approach wins.
