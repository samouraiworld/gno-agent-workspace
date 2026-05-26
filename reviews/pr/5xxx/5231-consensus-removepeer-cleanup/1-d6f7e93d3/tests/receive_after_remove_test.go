// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5231 -R gnolang/gno && git checkout d6f7e93d3
curl -fsSL -o tm2/pkg/bft/consensus/receive_after_remove_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5231-consensus-removepeer-cleanup/1-d6f7e93d3/tests/receive_after_remove_test.go
go test -timeout 60s -run 'TestReactor_ReceiveAfterRemovePeer_Panics' -v ./tm2/pkg/bft/consensus/
rm tm2/pkg/bft/consensus/receive_after_remove_test.go
*/

// The mechanism: ConsensusReactor.RemovePeer sets PeerStateKey to nil
// (peer.Set(types.PeerStateKey, nil)). The CMap stores the literal nil.
// A subsequent Receive on the same peer does
// `src.Get(PeerStateKey).(*PeerState)` which returns (nil, false) on the
// type-assertion, and the next line panics with "Peer X has no state".
//
// Result observed: assert.Panics succeeds — Receive panics after RemovePeer.
// To assert the fix, flip the inner assertion to assert.NotPanics: Receive
// should be a no-op (or log+drop) when the peer state has already been
// cleared by RemovePeer, because the message can be in-flight (e.g. queued
// onto statsMsgQueue) when the reactor concurrently tears the peer down.

package consensus

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	p2pTesting "github.com/gnolang/gno/tm2/pkg/internal/p2p"
	"github.com/gnolang/gno/tm2/pkg/log"

	"github.com/stretchr/testify/assert"
)

func TestReactor_ReceiveAfterRemovePeer_Panics(t *testing.T) {
	t.Parallel()

	N := 1
	css, cleanup := randConsensusNet(N, "consensus_receive_after_remove_test", newMockTickerFunc(true), newCounter)
	defer cleanup()
	reactors, _, eventSwitches, p2pSwitches := startConsensusNet(t, css, N)
	defer stopConsensusNet(log.NewTestingLogger(t), reactors, eventSwitches, p2pSwitches)

	reactor := reactors[0]
	peer := p2pTesting.NewPeer(t)
	msg := amino.MustMarshalAny(&HasVoteMessage{Height: 1, Round: 1, Index: 1, Type: types.PrevoteType})

	// Normal lifecycle: Init → Add → Receive works.
	reactor.InitPeer(peer)
	reactor.AddPeer(peer)
	assert.NotPanics(t, func() {
		reactor.Receive(StateChannel, peer, msg)
	})

	// RemovePeer clears the PeerStateKey to nil via peer.Set(key, nil).
	reactor.RemovePeer(peer, "test removal")

	// IS:     bug — Receive panics ("Peer X has no state") because
	//                peer.Set(key, nil) stores nil, and the assertion to *PeerState
	//                fails, hitting the existing panic at reactor.go:239.
	assert.Panics(t, func() {
		reactor.Receive(StateChannel, peer, msg)
	}, "Receive on a removed peer panics — a real race window via statsMsgQueue and any in-flight reactor work")

	// SHOULD: post-fix — Receive should be a no-op (or log+drop). Flip the
	//                    assertion to NotPanics to validate the fix:
	// assert.NotPanics(t, func() {
	// 	reactor.Receive(StateChannel, peer, msg)
	// })
}
