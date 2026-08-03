# PR [#6023](https://github.com/gnolang/gno/pull/6023): feat(p2p): wire config.P2P.Seeds into the switch

URL: https://github.com/gnolang/gno/pull/6023
Author: AviaOne | Base: master | Files: 5 | +388 -5
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 6324377f5 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6023 6324377f5`

**TL;DR:** A node's `config.toml` has a `seeds` setting that nothing reads, so operators who fill it in get no bootstrap peers at all. This wires it up: the seeds are dialed when the node starts, and again later whenever the node has nothing else left to dial.

**Verdict: REQUEST CHANGES** — the wiring works, but the startup dial and the steady-state loop both behave differently from what this PR's own README and design notes describe (2 Warnings, 2 Missing tests, 3 Nits, 1 Suggestion).

## Verify first

- [`tm2/pkg/bft/node/node.go:721`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/bft/node/node.go#L721) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L721) — confirm that dialing every seed in one call at startup is intended, given that [`dialSeed`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L574-L577) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L574-L577) deliberately dials one. Run the repro in the first Warning: 12 seeds queue under a limit of 10.
- [`tm2/pkg/p2p/switch.go:534`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L534-L538) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L534-L538) — confirm what `hasDialableItem` reports on a running node. [`runDialLoop`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L299-L301) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L299-L301) pops an item the moment its time arrives, so the queue an idle node presents to this check is empty.

## Summary

[`P2PConfig.Seeds`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/config/config.go#L30) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/config/config.go#L30) was declared and never read, so the only way to bootstrap a TM2 node was `persistent_peers`, which then redials that address forever with backoff. The PR adds [`WithSeeds`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch_option.go#L43) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch_option.go#L43), a `seeds` map on the switch kept apart from `persistentPeers` so the redial loop never sees it, and a fourth switch service that dials one random unconnected seed every 30 seconds when the dial queue holds nothing dialable. `node.go` parses the setting once, passes it to the switch only when `pex` is on, and dials the seeds itself at startup.

## Diagram

```
node.OnStart ──DialPeers(all seeds)──┐
                                     ├──> dialQueue ──runDialLoop(pops when ready)──> transport.Dial
runSeedDialLoop (30s) ──dialSeed()───┘                         │
        │                                                      │
        └── gate: hasDialableItem() == dialQueue.Peek() ready ──┘
                  ^ false on an idle node: the loop above already drained it
```

## Warnings (should fix)

- **[the startup dial skips the limit the design agreed to]** `tm2/pkg/bft/node/node.go:721` — [`OnStart`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/bft/node/node.go#L721) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L721) hands every configured seed to `DialPeers` in one call, so a node with more seeds than `max_num_outbound_peers` queues all of them.
  <details><summary>details</summary>

  [`DialPeers`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L666) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L666) re-reads `NumOutbound()` per address, and at startup no dial has completed, so the gate reads 0 for every address in the batch and lets the whole list through. That is the state [`dialSeed`'s comment](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L574-L576) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L574-L576) says the one-per-round pacing exists to avoid, and it is the moment it matters most: the node has no other peers, and it never closes a seed connection itself. With the default `max_num_outbound_peers` of [10](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/config/config.go#L70) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/config/config.go#L70), 12 configured seeds all get queued; measured by [`seed_startup_batch_test.go`](tests/seed_startup_batch_test.go), [repro](comment_claude-opus-5.md). Fix: dial one seed at startup and let the loop take over, or state why startup is exempt.
  </details>

- **[the fallback converges on holding every seed]** `tm2/pkg/p2p/switch.go:534` — [`hasDialableItem`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L534-L538) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L534-L538) is false on a healthy idle node, so a seed is dialed every round until all of them are connected.
  <details><summary>details</summary>

  The queue is not a backlog: [`runDialLoop`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L299-L301) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L299-L301) pops an item as soon as its time arrives, so what a 30-second sample finds is either an empty queue or backed-off entries, both of which read as "nothing to dial". Each round therefore dials another seed, each connected seed is skipped from then on by the [`peers.Has`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L562) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L562) filter, and the node never drops one. The end state is a connection to every configured seed, held for the node's lifetime, in slots that are meant for peers carrying consensus and mempool traffic. That contradicts [README.md:449](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/README.md?plain=1#L449) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/README.md#L449), which says a seed connection exists only until discovery has filled the queue. Measured by [`seed_convergence_test.go`](tests/seed_convergence_test.go), [repro](comment_claude-opus-5.md). Fix: gate the round on the node still needing peers, not only on the queue.
  </details>

## Nits

- **[unbiased randomness is already in the package]** `tm2/pkg/p2p/switch.go:599` — [`randomIndex`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L589-L600) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L589-L600) takes a modulo of a random uint64, which skews toward the low indices. [`discovery.go:108`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/discovery/discovery.go#L108-L111) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/discovery/discovery.go#L108-L111) picks a random peer with `rand.Int(rand.Reader, big.NewInt(n))`, which does not.
- **[two copies of the seed set]** `tm2/pkg/bft/node/node.go:197` — the node keeps [`seedAddrs`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/bft/node/node.go#L197) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L197) while the switch holds the same addresses in [`sw.seeds`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L79) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L79), and the two are kept in step by hand at [node.go:581](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/bft/node/node.go#L581) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L581).
- **[file still ends mid-line]** `tm2/pkg/p2p/README.md:658` — the added [closing paragraph](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/README.md?plain=1#L656-L658) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/README.md#L656) leaves the file without a trailing newline, as before the PR. No enabled linter covers markdown here, so no change is needed; noted only because the diff already rewrites that line.

## Missing Tests

- **[the loop's only real path is uncovered]** `tm2/pkg/p2p/switch_test.go:868` — [`TestMultiplexSwitch_SeedDialLoop`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch_test.go#L868-L895) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch_test.go#L868-L895) asserts only that the loop exits on cancellation. Nothing asserts that a tick reaches `dialSeed`.
  <details><summary>details</summary>

  [`seedDialInterval`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L29) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L29) is a package-level variable read directly by the loop, so a test cannot shorten it without racing other tests in the package. Making it a switch field set by an option, defaulting to 30 seconds, gives the loop a test that ticks in milliseconds and asserts a seed reached the queue.
  </details>

- **[the ignore-seeds branch is uncovered]** `tm2/pkg/bft/node/node.go:578` — nothing exercises [the `pex` off path](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/bft/node/node.go#L576-L582) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L576-L582), where seeds are dropped from both the switch option and the startup dial.
  <details><summary>details</summary>

  The branch is what makes `pex = false` plus a filled `seeds` setting safe rather than a half-wired state, and it clears `seedAddrs` in one place while the switch is skipped in another. A `NewNode` test with `pex` off and one seed, asserting `sw.seeds` is empty and `n.seedAddrs` is nil, pins both halves.
  </details>

## Suggestions

- **[the interval belongs to the switch]** `tm2/pkg/p2p/switch.go:29` — [`seedDialInterval`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L29) · [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L29) sits beside `defaultDialTimeout` as a package variable.
  <details><summary>details</summary>

  Both the redial loop's 5-second tick and this one are fixed in code, so the pattern is consistent with the package. Moving this one onto the switch behind an option is what unblocks the missing loop test above, and it lets an operator running a node with few reachable peers slow the fallback down.
  </details>

## Verified

- The two Warnings each ship a test that fails at 6324377f5 and passes only under the corrected behavior: 12 of 12 seeds queued under a limit of 10, and 3 of 3 seeds connected after six rounds.
- `go test ./tm2/pkg/p2p/...` and `go test ./tm2/pkg/bft/node/...` are green at 6324377f5. Two tests in `tm2/pkg/p2p/conn` fail intermittently under `-count=5` (`TestMConnectionPingPongs`, `TestMConnectionPongTimeoutResultsInError`); the PR does not touch that package and the failures are timing-dependent, so they are not attributable here.

## Existing threads

- tbruyelle, on the re-dial gate and on slot accounting: settled, and the pushed commit implements what was agreed. No overlap with the two Warnings, which are about what the agreed design does at startup and in steady state, not about the choice itself. [thread](https://github.com/gnolang/gno/pull/6023#issuecomment-5144430236)

## Open questions

- The author flagged three choices as never discussed, one of which is dropping seeds entirely when `pex` is off. A seed is still a reachable node, so the alternative is dialing it once as a plain peer and skipping only the discovery request. Raised in the comment draft as a question, since it is a decision the author asked for and it changes what a `pex = false` operator gets.
