# PR #5231: fix(consensus): implement `RemovePeer` cleanup

URL: https://github.com/gnolang/gno/pull/5231
Author: davd-gzl | Base: master | Files: 2 | +92 -10
Reviewed by: davd-gzl | Model: claude-opus-4-7

Verdict: REQUEST CHANGES — `peer.Set(types.PeerStateKey, nil)` turns into a panic on any in-flight `Receive`/stats path that grabs the same peer between [`reactor.RemovePeer`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L190) and [`sw.peers.Remove`](../../../../../.worktrees/gno-review-5231/tm2/pkg/p2p/switch.go#L252); the PR author already agreed to postpone (tbruyelle, cometbft parity), but if it ever lands the nil-store needs replacing with a Delete or with nil-guarded readers.

## Summary

`ConsensusReactor.RemovePeer` was a noop. The PR makes it (1) close a per-`PeerState` `closer` channel so the three gossip loops bail early on `ps.IsDisconnected()` instead of waiting for `peer.IsRunning()`, and (2) clear `PeerStateKey` on the peer's CMap to break the `peer ↔ PeerState` reference cycle for prompt GC. The mechanism works, but step (2) stores literal `nil` (CMap has no Delete-by-`Set(nil)` semantics — see [`cmap.go:17-21`](../../../../../.worktrees/gno-review-5231/tm2/pkg/cmap/cmap.go#L17-L21)), which collides with three existing `ps, ok := peer.Get(...).(*PeerState); if !ok { panic(...) }` sites in `reactor.go` (lines 239, 831, 874). The motivation is also weak: tbruyelle pointed out cometbft still has `RemovePeer` as a noop, the author conceded it can be postponed, and the PR carries a `don't merge` label.

## Glossary

- `PeerState` — per-peer consensus state held in the peer's CMap under `PeerStateKey`; carries round, votes, stats.
- `closer chan struct{}` — new per-PeerState signal channel, closed once by `Disconnect()` via `sync.Once`.
- `statsMsgQueue` — buffered channel in `ConsensusState`; messages from peers are enqueued for the stats worker, which later re-looks-up the peer state in `reactor.go:820-832`.

## Fix

Before: `RemovePeer` returned without doing anything; goroutines lingered until `peer.IsRunning()` flipped, and the `PeerState` stayed reachable via the peer's CMap. After: `RemovePeer` calls `ps.Disconnect()` (closes `closer` exactly once via `sync.Once`), which the three gossip routines now check first on every iteration, then `peer.Set(PeerStateKey, nil)` to drop the cycle. The load-bearing constraint is that the readers of `PeerStateKey` panic on `ok == false`, and `peer.Set(key, nil)` produces exactly that case — see [`cmap.go:17-21`](../../../../../.worktrees/gno-review-5231/tm2/pkg/cmap/cmap.go#L17-L21) (`Set` stores literal nil) versus [`reactor.go:237-240`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L237-L240).

## Critical (must fix)

- **[receive panics on a recently-removed peer]** [`tm2/pkg/bft/consensus/reactor.go:206`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L206) — `peer.Set(PeerStateKey, nil)` makes `Receive` panic on any in-flight message from the same peer.
  <details><summary>details</summary>

  `peer.Set(PeerStateKey, nil)` calls `cmap.Set(key, nil)` which stores the literal `nil` value (the key is not deleted from the underlying `map[string]any` — see [`cmap.go:17-21`](../../../../../.worktrees/gno-review-5231/tm2/pkg/cmap/cmap.go#L17-L21)). On the next call to `Receive` for the same peer, [`reactor.go:237-240`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L237-L240) does `ps, ok := src.Get(PeerStateKey).(*PeerState)`; the assertion of a nil `any` to `*PeerState` returns `(nil, false)` and the next line panics with `Peer X has no state`. The same shape occurs at [`reactor.go:829-832`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L829-L832) (stats worker, reading `statsMsgQueue`) and [`reactor.go:872-875`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L872-L875) (`StringIndented`).

  The race window is real: `MultiplexSwitch.stopAndRemovePeer` ([`switch.go:221-258`](../../../../../.worktrees/gno-review-5231/tm2/pkg/p2p/switch.go#L221-L258)) calls `peer.Stop()`, then `reactor.RemovePeer(peer)`, **then** `sw.peers.Remove(peer.ID())`. Between the `RemovePeer` call (which now nils the key) and `peers.Remove`, the stats worker dequeues a previously-enqueued `msgInfo`, finds the peer via `Switch.Peers().Get(msg.PeerID)` (non-nil because `peers.Remove` has not run yet), and panics on the nil assertion. `Receive` is a smaller window because the mConnection read loop should be drained by `peer.Stop()` first, but it is not zero — and `StringIndented` ranges over `Switch.Peers().List()`, so the same race applies to any debug-RPC path that calls it.

  Repro (see [`tests/receive_after_remove_test.go`](tests/receive_after_remove_test.go)) confirms `Receive` panics on a removed peer.

  Fix: don't store nil. Either (a) add `Delete(key string)` to `PeerConn` and call it from `RemovePeer` so the entry vanishes (then `ok == false` correctly means "never had state"), or (b) keep `Set(key, nil)` and change the three panic sites to log+return — but option (a) is closer to the existing contract (`ok == false` already means "no state"), and option (b) silently masks the genuine "InitPeer never ran" bug that [`TestReactorReceivePanicsIfInitPeerHasntBeenCalledYet`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor_test.go#L179-L200) is explicitly asserting. Prefer (a).
  </details>

## Warnings (should fix)

- **[motivation thinner than the surface area]** [@tbruyelle](https://github.com/gnolang/gno/pull/5231#issuecomment-2762820555) PR body — cometbft upstream still has `RemovePeer` as a noop; PR author already agreed to postpone.
  <details><summary>details</summary>

  The PR description ("RemovePeer was a noop") and tbruyelle's question expose the same gap: there's no specific incident (no leak measured, no goroutine count, no profile) tying the unimplemented `RemovePeer` to an observed problem on gno.land. The author replied "Outside of the memory leak, there were no other motivations, this is not a priority. This can be postponed / closed." Combined with the `don't merge` label that's still on the PR, this should either (a) wait for an actual reproducer / pprof before re-opening, or (b) be closed in favor of mirroring whatever cometbft eventually does. Either way, a consensus-reactor change without a specific failure to point at is hard to justify before a stable mainnet.
  </details>

## Nits

- [`tm2/pkg/bft/consensus/reactor.go:897-898`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L897-L898) — `closerMu sync.Once` is misleadingly named; it is not a mutex.
  <details><summary>details</summary>

  `sync.Once` is not a mutex; the `Mu` suffix conflicts with [`reactor.go:900`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L900) (`mtx sync.Mutex`). Rename to `closeOnce` (the canonical name in the Go standard library and most consensus codebases that use this exact pattern, e.g. tendermint's own `quit chan` + `quitOnce`).
  </details>

- [`tm2/pkg/bft/consensus/reactor.go:188-189`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L188-L189) — comment says "goroutines terminate on their own via `peer.IsRunning()`" but the new code overrides that with `ps.IsDisconnected()` first.
  <details><summary>details</summary>

  The doc-comment describes the pre-PR behaviour, which doesn't match what the PR ships. Either drop the "will terminate on their own via the peer.IsRunning() check" half-sentence or rewrite it to say the new check (`ps.IsDisconnected()`) makes the bail prompt without waiting for `peer.Stop()` to flip `IsRunning`.
  </details>

## Missing Tests

- **[concurrent Receive ↔ RemovePeer]** [`tm2/pkg/bft/consensus/reactor_test.go:927`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor_test.go#L927) — `TestRemovePeerCleansUpState` only covers the single-threaded happy path.
  <details><summary>details</summary>

  The test calls `InitPeer → RemovePeer → assert peer.Get returns nil`. It doesn't exercise the scenario that the Critical above describes: a `Receive` (or stats-worker dequeue) crossing `RemovePeer`. The fix should be paired with a test that races `reactor.Receive` against `reactor.RemovePeer` (e.g. spawn a goroutine that calls `Receive` in a loop, call `RemovePeer` mid-loop, assert no panic). See [`tests/receive_after_remove_test.go`](tests/receive_after_remove_test.go) for a deterministic version (sequential Remove → Receive) that already fails.
  </details>

## Suggestions

- [`tm2/pkg/bft/consensus/reactor.go:929-936`](../../../../../.worktrees/gno-review-5231/tm2/pkg/bft/consensus/reactor.go#L929-L936) — return `<-chan struct{}` from a `Done()` accessor, drop the polling `IsDisconnected`.
  <details><summary>details</summary>

  The three gossip loops poll `ps.IsDisconnected()` once per iteration, but each loop has a `time.Sleep(PeerGossipSleepDuration)` (≈100ms) on the slow path, so a `RemovePeer` can sleep up to ~100ms before the loop notices. Exposing the channel directly (`func (ps *PeerState) Done() <-chan struct{} { return ps.closer }`) lets the loops `select` on it and react immediately, which is the whole point of having a channel-based signal. This is an optional follow-up — current polling is correct, just slow.
  </details>

## Questions for Author

- Given the maintainer's "let's postpone" and the `don't merge` label, is the plan to close this PR or to revive it once cometbft's analog lands?
- Was the "memory leak" in the PR description measured (pprof / runtime.NumGoroutine over time), or inferred from reading the code? If measured, attaching the numbers would justify the cost.
