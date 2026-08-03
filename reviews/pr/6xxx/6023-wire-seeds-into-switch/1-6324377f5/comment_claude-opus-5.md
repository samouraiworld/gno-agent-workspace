# Review: PR [#6023](https://github.com/gnolang/gno/pull/6023)
Event: REQUEST_CHANGES

## Body
Reproduced on 6324377f5: 12 seeds queue under a `max_num_outbound_peers` of 10, and six seed rounds leave all three configured seeds connected.

On the third undiscussed choice, dropping seeds when `pex` is off: a seed is still a reachable node, so the alternative is dialing it once as a plain peer and skipping only the discovery request. Was the full drop deliberate?

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6023-wire-seeds-into-switch/1-6324377f5/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## tm2/pkg/bft/node/node.go:721 [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L721)
Every configured seed is queued in one call here, so a node with more seeds than `max_num_outbound_peers` opens a bootstrap connection for each of them. [`DialPeers`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L666) re-reads `NumOutbound()` per address and no dial has completed yet, so the gate reads 0 for the whole batch. This is the state the [one-seed-per-round pacing](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L574-L576) avoids everywhere else, at the moment the node has no other peers and will not close a seed connection itself.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6023 -R gnolang/gno

cat > tm2/pkg/p2p/seed_startup_batch_test.go <<'EOF'
package p2p

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/p2p/types"
)

func TestSeedStartupBatch(t *testing.T) {
	seeds := generateNetAddr(t, 12)

	sw := NewMultiplexSwitch(&mockTransport{}, WithMaxOutboundPeers(10), WithSeeds(seeds))
	sw.peers = &mockSet{
		hasFn:         func(types.ID) bool { return false },
		numOutboundFn: func() uint64 { return 0 }, // nothing dialed yet
	}

	sw.DialPeers(seeds...) // what Node.OnStart does with n.seedAddrs

	queued := 0
	for sw.dialQueue.Pop() != nil {
		queued++
	}
	t.Logf("queued %d seeds under max_num_outbound_peers=10", queued)
}
EOF

go test -v -run TestSeedStartupBatch ./tm2/pkg/p2p/
rm tm2/pkg/p2p/seed_startup_batch_test.go
```

```
=== RUN   TestSeedStartupBatch
    seed_startup_batch_test.go:24: queued 12 seeds under max_num_outbound_peers=10
--- PASS: TestSeedStartupBatch (0.00s)
```
</details>

## tm2/pkg/p2p/switch.go:534-538 [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L534-L538)
On a healthy node this reads false, so every round dials another seed until all of them are connected and stay connected for the node's lifetime. The queue is not a backlog: [`runDialLoop`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L299-L301) pops an item the moment its time arrives, so a 30-second sample sees an empty queue or backed-off entries, both of which read as nothing to dial. [README.md:449](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/README.md?plain=1#L449) describes the opposite, a connection that lasts only until discovery has filled the queue.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6023 -R gnolang/gno

cat > tm2/pkg/p2p/seed_convergence_test.go <<'EOF'
package p2p

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/p2p/types"
)

func TestSeedConvergence(t *testing.T) {
	seeds := generateNetAddr(t, 3)

	sw := NewMultiplexSwitch(&mockTransport{}, WithSeeds(seeds))

	connected := make(map[types.ID]bool)
	sw.peers = &mockSet{
		hasFn:         func(id types.ID) bool { return connected[id] },
		numOutboundFn: func() uint64 { return uint64(len(connected)) },
	}

	// runDialLoop drains a ready item at once; stand in for it between rounds.
	for range 6 {
		sw.dialSeed()
		if item := sw.dialQueue.Pop(); item != nil {
			connected[item.Address.ID] = true
		}
	}
	t.Logf("seed connections held after 6 rounds: %d of %d configured", len(connected), len(seeds))
}
EOF

go test -v -run TestSeedConvergence ./tm2/pkg/p2p/
rm tm2/pkg/p2p/seed_convergence_test.go
```

```
=== RUN   TestSeedConvergence
    seed_convergence_test.go:27: seed connections held after 6 rounds: 3 of 3 configured
--- PASS: TestSeedConvergence (0.00s)
```
</details>

## tm2/pkg/p2p/switch_test.go:868-895 [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch_test.go#L868-L895)
Missing test: a tick of the loop reaching `dialSeed`. Only cancellation is covered, and [`seedDialInterval`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L29) is a package variable the loop reads directly, so no test can shorten it without racing the rest of the package. Moving it onto the switch behind an option makes the path testable in milliseconds.

## tm2/pkg/bft/node/node.go:576-582 [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L576-L582)
Missing test: `pex` off with seeds configured. The branch clears `seedAddrs` in one place and skips the switch option in another, and nothing asserts both halves.

<details><summary>test cases</summary>

A `NewNode` case with `PeerExchange` false and one entry in `P2P.Seeds`, asserting that `n.seedAddrs` is nil and that the switch's `seeds` map is empty.
</details>

## tm2/pkg/p2p/switch.go:589-600 [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/p2p/switch.go#L589-L600)
Nit: taking a modulo of a random uint64 skews the pick toward the low indices. [`discovery.go:108`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/discovery/discovery.go#L108-L111) picks a random peer with `rand.Int(rand.Reader, big.NewInt(n))`, which does not.

## tm2/pkg/bft/node/node.go:197 [↗](../../../../../.worktrees/gno-review-6023/tm2/pkg/bft/node/node.go#L197)
Nit: the node holds its own copy of the seed set while the switch holds the same addresses in [`sw.seeds`](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/p2p/switch.go#L79), and the two are kept in step by hand at [node.go:581](https://github.com/gnolang/gno/blob/6324377f5/tm2/pkg/bft/node/node.go#L581).
