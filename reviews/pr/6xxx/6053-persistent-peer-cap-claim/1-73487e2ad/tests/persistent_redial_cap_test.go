/* Run: from a gno checkout:
gh pr checkout 6053 -R gnolang/gno && git checkout 73487e2ad
curl -fsSL -o tm2/pkg/p2p/persistent_redial_cap_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6053-persistent-peer-cap-claim/1-73487e2ad/tests/persistent_redial_cap_test.go
go test -v -run TestPersistentRedialUnderOutboundCap ./tm2/pkg/p2p/
rm tm2/pkg/p2p/persistent_redial_cap_test.go
*/

// runRedialLoop queues a lost persistent peer through dialItems, which drops the
// item when NumOutbound() has reached max_num_outbound_peers. A persistent peer
// whose connection was inbound frees no outbound slot when it drops, so at the
// cap the redial never reaches the queue.
// At 73487e2ad the addr is absent from the dial queue while outbound sits at 2.
package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/p2p/types"
	"github.com/stretchr/testify/require"
)

func TestPersistentRedialUnderOutboundCap(t *testing.T) {
	t.Parallel()

	// newSwitch returns a switch holding one persistent peer that is not
	// connected, with the outbound counter pinned at numOutbound.
	newSwitch := func(t *testing.T, numOutbound uint64) (*MultiplexSwitch, *types.NetAddress) {
		t.Helper()

		addr := generateNetAddr(t, 1)[0]

		sw := NewMultiplexSwitch(
			&mockTransport{},
			WithMaxOutboundPeers(2),
			WithPersistentPeers([]*types.NetAddress{addr}),
		)
		sw.peers = &mockSet{
			hasFn:         func(types.ID) bool { return false }, // the persistent peer is gone
			numOutboundFn: func() uint64 { return numOutbound },
		}

		return sw, addr
	}

	// runRedialLoop calls redialFn once on entry, so one short run is enough.
	runOnce := func(sw *MultiplexSwitch) {
		ctx, cancelFn := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancelFn()

		sw.runRedialLoop(ctx)
	}

	t.Run("outbound slots free", func(t *testing.T) {
		t.Parallel()

		sw, addr := newSwitch(t, 1)
		runOnce(sw)

		require.True(t, sw.dialQueue.Has(addr))
	})

	t.Run("outbound cap reached", func(t *testing.T) {
		t.Parallel()

		sw, addr := newSwitch(t, 2)
		runOnce(sw)

		require.False(t, sw.dialQueue.Has(addr)) // IS:     the cap drops the redial
		// require.True(t, sw.dialQueue.Has(addr)) // README: the redial service brings the peer back
	})
}
