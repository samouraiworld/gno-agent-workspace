/* Run: from a gno checkout:
gh pr checkout 6023 -R gnolang/gno && git checkout 6324377f5
curl -fsSL -o tm2/pkg/p2p/seed_convergence_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6023-wire-seeds-into-switch/1-6324377f5/tests/seed_convergence_test.go
go test -v -run TestSeedConvergence ./tm2/pkg/p2p/
rm tm2/pkg/p2p/seed_convergence_test.go
*/

// runDialLoop pops a ready item as soon as its time arrives, so hasDialableItem
// is false on an idle node and every seed round dials another seed.
// The pop-and-connect below stands in for that loop; at 6324377f5 all 3 seeds
// end up connected and are never dropped.
package p2p

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/p2p/types"
	"github.com/stretchr/testify/assert"
)

func TestSeedConvergence(t *testing.T) {
	t.Parallel()

	seeds := generateNetAddr(t, 3)

	sw := NewMultiplexSwitch(&mockTransport{}, WithSeeds(seeds))

	connected := make(map[types.ID]bool)
	sw.peers = &mockSet{
		hasFn:         func(id types.ID) bool { return connected[id] },
		numOutboundFn: func() uint64 { return uint64(len(connected)) },
	}

	for range 6 {
		sw.dialSeed()

		if item := sw.dialQueue.Pop(); item != nil {
			connected[item.Address.ID] = true
		}
	}

	t.Logf("seed connections held after 6 rounds: %d of %d configured", len(connected), len(seeds))

	// SHOULD: bootstrap connections stay a fallback, not every seed at once.
	assert.Less(t, len(connected), len(seeds))
}
