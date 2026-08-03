/* Run: from a gno checkout:
gh pr checkout 6023 -R gnolang/gno && git checkout 6324377f5
curl -fsSL -o tm2/pkg/p2p/seed_startup_batch_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6023-wire-seeds-into-switch/1-6324377f5/tests/seed_startup_batch_test.go
go test -v -run TestSeedStartupBatchIgnoresOutboundLimit ./tm2/pkg/p2p/
rm tm2/pkg/p2p/seed_startup_batch_test.go
*/

// Node.OnStart hands every configured seed to DialPeers in one call. DialPeers
// re-reads NumOutbound per address, and no dial has completed yet at that point,
// so the max_num_outbound_peers gate lets the whole batch through.
// At 6324377f5 all 12 seeds are queued under a limit of 10.
package p2p

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/p2p/types"
	"github.com/stretchr/testify/assert"
)

func TestSeedStartupBatchIgnoresOutboundLimit(t *testing.T) {
	t.Parallel()

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

	// SHOULD: the startup dial respects the limit the design agreed to.
	assert.LessOrEqual(t, queued, 10)
}
