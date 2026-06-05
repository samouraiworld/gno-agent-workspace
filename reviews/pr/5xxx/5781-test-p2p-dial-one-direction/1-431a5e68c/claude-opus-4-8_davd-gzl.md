# PR #5781: fix(tm2): make test p2p clusters dial each pair in one direction only

URL: https://github.com/gnolang/gno/pull/5781
Author: thehowl | Base: master | Files: 1 | +8 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8[1m] | Commit: 431a5e68c (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5781 431a5e68c`

**Verdict: APPROVE** — Correct, minimal fix for a real CI flake; the one-directional dial provably eliminates the duplicate-rejection teardown race, and every `PeerConnected` accounting invariant still holds. No blockers; one latent scaling note below the current test sizes.

## Summary
`MakeConnectedPeers` (test-only p2p cluster builder) had every switch concurrently dial every other switch. For any pair A,B both outbound dials could land before either accept loop ran, so each side registered its own outbound peer (firing `PeerConnected`, so setup reported success) then rejected the other's inbound as a duplicate and closed the TCP conn backing the already-registered peer. Test peers aren't persistent, so nothing redialed and the pair stayed split forever. For a 2-validator consensus net that is a total partition: the net never commits, and `TestReactorDoesNotResignKnownSelfPrevote` hangs 300s until `timeoutWaitGroup` panics. The fix dials only higher-indexed peers (`addrs[switchIndex+1:]`), so each pair connects in exactly one direction and the duplicate-rejection path can never fire. It also swaps the racy concurrent `append` into `sws` for an indexed write.

```
before (N=2):  s0 --dial--> s1      after (N=2):  s0 --dial--> s1   (only)
               s1 --dial--> s0
               both register outbound, each rejects     s0 outbound-adds s1, s1 inbound-adds s0
               the other's inbound dup -> both torn down  one link, no dup rejection -> stable
```

## Glossary
- `MakeConnectedPeers` — test helper in `tm2/pkg/internal/p2p` that spins up `Count` live switches and waits until all are mutually connected.
- `PeerConnected` — event fired by `addPeer` for both inbound and outbound peers; setup counts `Count-1` of these per switch to declare success.
- duplicate rejection — accept loop closes an inbound conn whose peer ID is already in the peer set (`switch.go:661`).

## Fix
Before, `connectPeers` called `multiplexSwitch.DialPeers(addrs...)` (every address, both directions) and appended switches concurrently into a slice. After, it dials only `addrs[switchIndex+1:]` and writes its switch at `sws[switchIndex]`. Because `addPeer` fires `PeerConnected` for inbound peers too ([`switch.go:721`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/p2p/switch.go#L721) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/p2p/switch.go#L721)), the lower-indexed switch still observes its `Count-1` events via outbound `addPeer` and the higher-indexed switch observes them via inbound `addPeer` in the accept loop ([`switch.go:673`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/p2p/switch.go#L673) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/p2p/switch.go#L673)). The success-counting invariant (`len(connectedPeers) == cfg.Count-1`, [`p2p.go:153`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/internal/p2p/p2p.go#L153) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/internal/p2p/p2p.go#L153)) is unchanged and still satisfied for every switch.

## Verification
Ran the headline test plus the other two `MakeConnectedPeers` consumers in the worktree at `431a5e68c`:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5781 -R gnolang/gno
go test -count=3 -run 'TestReactorDoesNotResignKnownSelfPrevote' ./tm2/pkg/bft/consensus/
go test -count=1 ./tm2/pkg/bft/mempool/ ./tm2/pkg/bft/blockchain/
```

```
ok  github.com/gnolang/gno/tm2/pkg/bft/consensus    0.199s   # was 300s panic on master
ok  github.com/gnolang/gno/tm2/pkg/bft/mempool      4.554s
ok  github.com/gnolang/gno/tm2/pkg/bft/blockchain   4.012s
```

The previously-hanging test now finishes in well under a second across repeated runs; no flake observed. CI on the PR is fully green (CodeQL, build, analyze, all txtar scenarios).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`p2p.go:135`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/internal/p2p/p2p.go#L135) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/internal/p2p/p2p.go#L135) — the new comment paragraph ends without a period; the surrounding comments in this file are full sentences.

## Missing Tests
None. This is itself a test-infra fix; the existing reactor suites are the regression coverage, and they were the symptom. A dedicated unit test asserting "every pair is connected after `MakeConnectedPeers`" would be nice but the file's own banner ([`p2p.go:1-7`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/internal/p2p/p2p.go#L1-L7) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/internal/p2p/p2p.go#L1-L7)) calls the whole package a stopgap slated for deletion, so adding scaffolding around it is hard to justify.

## Suggestions
- [`p2p.go:135`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/internal/p2p/p2p.go#L135) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/internal/p2p/p2p.go#L135) — latent scaling constraint, not a current bug.
  <details><summary>details</summary>

  With one-directional dialing, switch index 0 dials all `Count-1` peers as outbound, while the highest-index switch accepts `Count-1` inbound. The defaults are `MaxNumOutboundPeers: 10` and `MaxNumInboundPeers: 40` ([`config/config.go:61-62`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/p2p/config/config.go#L61-L62) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/p2p/config/config.go#L61-L62)). `DialPeers` silently drops dials once `NumOutbound >= maxOutboundPeers` ([`switch.go:564`](https://github.com/gnolang/gno/blob/431a5e68c/tm2/pkg/p2p/switch.go#L564) · [↗](../../../../../.worktrees/gno-review-5781/tm2/pkg/p2p/switch.go#L564)), so a cluster with `Count > 11` routed through this helper would leave switch 0 short of `Count-1` outbound links and `MakeConnectedPeers` would time out. The old all-to-all scheme spread the load (each switch dialed up to `Count-1` but the cap was rarely hit per side). Largest current consumer is `nPeers = 7` in `consensus/reactor_test.go:405` (so 6 outbound, well under 10); the `Count: 10` rpc/client tests do not use this helper. No action needed today; worth a one-line comment or a guard if someone later builds a bigger cluster through this path.
  </details>

## Questions for Author
None.
