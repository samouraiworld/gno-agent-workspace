# PR #5495: fix(node): ensure halt_height triggers clean process shutdown

URL: https://github.com/gnolang/gno/pull/5495
Author: aeddi | Base: chain/gnoland1 | Files: 5 | +38 -5
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep, high) | Commit: `84664d7` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5495 84664d7`

**TL;DR:** A chain can be configured with a `halt_height` so every node stops at the same block for a coordinated upgrade. Today a halted node logs `CONSENSUS FAILURE!!!` and keeps running with consensus dead but P2P/RPC still answering. This PR makes the halt panic carry a typed error so the node can instead shut its process down cleanly (exit 0) and, on a restart where the halt height is already passed, fail with a readable message (exit 1) instead of an unrecovered panic (exit 2).

**Verdict: NEEDS DISCUSSION** — code is correct and tests pass; the blocker is an open maintainer design choice (this PR's exit-0 vs a full-zombie variant of #5334), unresolved in-thread with @tbruyelle's peer-relay rationale unanswered.

## Summary

PR #5334 added `halt_height`: `BeginBlock` panics once the chain passes the configured height ([`baseapp.go:534-536`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/sdk/baseapp.go#L534-L536) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/sdk/baseapp.go#L534-L536)). That panic is a raw string, so the consensus `receiveRoutine`'s blanket `recover()` swallows it, logs `CONSENSUS FAILURE!!!`, runs `onExit()` (stops the WAL, closes `cs.done`), and the goroutine returns. Consensus dies but the process lives on with P2P and RPC still serving: the "in-between" state. On a cold restart the same panic escapes through `Handshaker` block replay with no recover, crashing with an unrecovered-panic stack (exit 2).

The fix replaces the string panic with a typed `HaltHeightReachedError{Height}` and matches on it in two recover sites: the warm path (`receiveRoutine`) logs INFO and sends itself SIGTERM via `osm.Kill()`; the cold path (`doHandshake`) converts it to a clean returned error. Both rest on the panic value being a value type that all three sites agree on.

## Design state (the actual blocker)

The thread did not settle on closing this PR. Chronology:

- @tbruyelle (04-14): the zombie-but-alive behavior is intentional per cosmos-sdk; proposed closing.
- @aeddi (04-14): "you're probably right... I'll wait for moul input", then after talking to @moul posted a before/after table and asked, "Before we close this PR, do we want to stick to the original behavior for cold restarts too (exit 2 vs exit 1)?"
- @moul (04-14): reopened it. "Two valid approaches imo": exit 0 (this PR) or full-zombie (extend #5334 to also shut down P2P + RPC). "I don't have a strong preference between the two, but I'd avoid the in-between state where consensus is dead but P2P/RPC are still serving." If full-zombie wins, "we probably want a follow-up to #5334 rather than merging this as-is."
- @tbruyelle (04-14): prefers full-zombie (mirror SDK), but questioned why moul's option 2 kills the P2P reactor.
- @aeddi (04-15): "I'm fine with either" (mirror cosmos exactly, or exit 0).
- @tbruyelle (04-17, last word, unanswered): the SDK keeps connections to avoid the "waiting for peers" stall on upgrade, a graceful peer-relay during the binary swap.

Net: two of three maintainers are explicitly fine with this PR's exit-0 approach; @tbruyelle prefers full-zombie and posted the strongest argument for it last, with no reply. The PR is OPEN and MERGEABLE. The one option everyone rejects, the current consensus-dead-but-P2P-alive state, is exactly what this PR removes. This is a live design call, not a dead PR.

## Fix

`baseapp.go:535` panics `bft.HaltHeightReachedError{Height: app.haltHeight}` instead of `fmt.Sprintf(...)`. `receiveRoutine`'s recover gains a typed branch ([`state.go:611-618`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/consensus/state.go#L611-L618) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/consensus/state.go#L611-L618)) that logs INFO, runs `onExit()`, calls `osm.Kill()`, and `return`s, skipping the fallthrough `onExit()` at `state.go:633` so the WAL is closed exactly once. `doHandshake` becomes a named-return function with a deferred recover ([`node.go:256-265`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/node/node.go#L256-L265) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/node/node.go#L256-L265)) that turns the typed panic into an error and re-raises anything else.

The exit-0 outcome holds, but not by the mechanism the PR description states. gnoland never registers `osm.TrapSignal` (only `contribs/gnodev` and `tm2/pkg/autofile/cmd/logjack` do); it catches SIGTERM through `signal.NotifyContext` in [`gno.land/cmd/gnoland/root.go:16-21`](https://github.com/gnolang/gno/blob/84664d7/gno.land/cmd/gnoland/root.go#L16-L21) · [↗](../../../../../.worktrees/gno-review-5495/gno.land/cmd/gnoland/root.go#L16-L21), which cancels the context awaited at `start.go:296`, runs the graceful node stop, and returns nil for exit 0. So the self-sent SIGTERM funnels into the existing graceful-shutdown path. Worth a thought (Open questions) on whether cancelling that context directly is cleaner than a process-level self-signal, if the exit-0 direction is chosen.

## Critical (must fix)
None.

## Warnings (should fix)
None. No lens found a correctness, determinism, or panic-handling bug. Type assertions match (value type, not pointer) across [`baseapp.go:535`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/sdk/baseapp.go#L535) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/sdk/baseapp.go#L535), [`state.go:612`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/consensus/state.go#L612) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/consensus/state.go#L612), and [`node.go:259`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/node/node.go#L259) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/node/node.go#L259); the cold-restart panic propagates intact through the only ABCI client (`localClient`, in-process, deferred mutex unlock); `cs.done` is closed exactly once; no other code keyed off the old panic string.

## Nits

- `tm2/pkg/bft/consensus/state.go:616` — [`osm.Kill()`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/consensus/state.go#L616) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/consensus/state.go#L616) return value is dropped, unlike the sibling kill at [`state.go:1389`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/consensus/state.go#L1389) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/consensus/state.go#L1389) which logs on failure. If the self-SIGTERM fails to send, the node neither exits nor leaves a trace, silently contradicting the clean-shutdown intent. `osm.Kill()` on self essentially never fails, hence Nit. Fix: mirror line 1389 and log the error if non-nil.

- `tm2/pkg/bft/consensus/state.go:615-616` — [`onExit()` runs before `osm.Kill()`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/consensus/state.go#L615-L616) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/consensus/state.go#L615-L616). `onExit()` stops the WAL and closes `cs.done`; if it panics, the deferred recover has already fired (no second recover in the same defer), so the SIGTERM never sends and the process exits via crash rather than the intended clean path. Low likelihood. Fix: send SIGTERM before `onExit()`, or guard `onExit()` so `osm.Kill()` always runs.

- `tm2/pkg/bft/node/node.go:263` — [`panic(r)`](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/node/node.go#L263) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/node/node.go#L263) re-raises non-halt replay panics, which resets the stack to this line. A genuine replay/DB-corruption panic on restart, exactly when a good trace matters, now points at node.go:263 instead of its origin. Fix: capture the stack at recover time (`panic(fmt.Sprintf("%v\n%s", r, debug.Stack()))`) before re-raising.

## Missing Tests

- **[cold-restart recover is unit-testable and untested]** `tm2/pkg/bft/node/node.go:256-265` — the [`doHandshake` halt branch](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/node/node.go#L256-L265) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/node/node.go#L256-L265) has zero coverage (codecov flags it). `node_test.go` is `package node` (white-box), so `doHandshake` is callable directly.
  <details><summary>details</summary>

  Inject an `appconn.AppConns` whose `BeginBlockSync` panics `types.HaltHeightReachedError{Height: 5}` during replay, drive a `Handshaker` over a one-block store, and assert: returns non-nil error, message contains "halt height 5 already reached", and does not panic. Add the negative case: a plain-string panic must propagate (`require.Panics`). That negative case also locks the value-type contract below.
  </details>

- **[warm-halt recover untested, harder]** `tm2/pkg/bft/consensus/state.go:611-618` — the [warm branch](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/consensus/state.go#L611-L618) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/consensus/state.go#L611-L618) calls `osm.Kill()`, which SIGTERMs the test runner, so it can't be tested as-is.
  <details><summary>details</summary>

  Would need the exit call extracted behind an injectable hook (e.g. a `cs.exitFn` defaulting to `osm.Kill`) so a test can assert it fired and `onExit()` ran on a typed-error panic. Given the design is unsettled, defer this until the exit-0 direction is confirmed.
  </details>

## Suggestions

- `tm2/pkg/bft/types/errors.go:43-51` — the [contract that the panic is a value type](https://github.com/gnolang/gno/blob/84664d7/tm2/pkg/bft/types/errors.go#L43-L51) · [↗](../../../../../.worktrees/gno-review-5495/tm2/pkg/bft/types/errors.go#L43-L51) is duplicated across three sites. If a future edit changes the raise site to `&HaltHeightReachedError{}`, both `r.(types.HaltHeightReachedError)` assertions silently stop matching and the node reverts to the zombie/exit-2 behavior. The raise side is locked by `baseapp_test.go`'s `PanicsWithValue`, but the consumer assertions are not. A one-line comment on the type ("recovered by value, not pointer; state.go and node.go assert the value type") plus the negative cold-restart test above would close the gap.

## Open questions

- Should warm-halt cancel the `signal.NotifyContext` context directly (the path SIGTERM already funnels into) rather than self-signalling with `osm.Kill()`? Not posted: pure design refinement, moot unless exit-0 is the chosen direction.
- @aeddi's sub-question (keep cold-restart exit 2, or adopt this PR's exit 1) is still open and independent of the warm-halt direction. The cold-restart half (typed error + `doHandshake` recover) is correct and useful on its own; it could ship as a narrowed PR even if the warm-halt half is dropped for full-zombie.

## CI

`Run TM2 suite / Go Test / test` is red: `TestWALCrash/empty_block` times out (`WAL did not panic for 10 seconds`), a known-flaky timing test (`replay_test.go` carries an "XXX why so long?" note). The PR touches no WAL-crash code; that crash panics `WALWriteError`, not `HaltHeightReachedError`, so the new branch provably never intercepts it. Reproduced flaky on both base and head. `codecov/patch/tm2` red is the untested recover paths above, not a regression.
