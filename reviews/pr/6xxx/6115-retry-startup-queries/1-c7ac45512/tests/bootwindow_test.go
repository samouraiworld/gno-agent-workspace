package main

/* Run: from a gno checkout:
gh pr checkout 6115 -R gnolang/gno && git checkout c7ac45512
curl -fsSL -o contribs/gpao/bootwindow_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6115-retry-startup-queries/1-c7ac45512/tests/bootwindow_test.go
go test -count=1 -run 'TestStartHeightZeroReadsTheTipAtProcessStart' ./contribs/gpao/
rm contribs/gpao/bootwindow_test.go
*/

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	rpcclient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
	ctypes "github.com/gnolang/gno/tm2/pkg/bft/rpc/core/types"
)

// Asserts that -start-height 0 begins at the tip the chain had when the process
// started, not the tip one poll interval later. Measured with a node that is up
// from the first call and produces a block every 50ms against a 250ms poll: at
// c7ac45512 the first block read is 13, five blocks past the 8 the same run
// reads at 2ed70a202.
const (
	bootBlockTime = 50 * time.Millisecond
	bootPoll      = 250 * time.Millisecond
	bootStartTip  = int64(7)
)

// livechainRPC is a node that answers from the first call and keeps producing.
// Its tip depends on WHEN it is asked, not on how many times, which is what
// separates a tip read before the loop from one read on the first tick. The
// embedded interface is nil on purpose, as in stubRPC: an unexpected call
// panics rather than passing silently.
type livechainRPC struct {
	rpcclient.Client
	start      time.Time
	mu         sync.Mutex
	blockAsked *int64 // first height asked of Block, nil until then
	cancel     context.CancelFunc
}

func (s *livechainRPC) Status(context.Context, *int64) (*ctypes.ResultStatus, error) {
	res := &ctypes.ResultStatus{}
	res.SyncInfo.LatestBlockHeight = bootStartTip + int64(time.Since(s.start)/bootBlockTime)
	return res, nil
}

func (s *livechainRPC) ConsensusParams(context.Context, *int64) (*ctypes.ResultConsensusParams, error) {
	return nil, errors.New("the ceiling is not the subject here") // blockMaxGasFrom falls back
}

func (s *livechainRPC) Block(_ context.Context, height *int64) (*ctypes.ResultBlock, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blockAsked == nil {
		h := *height
		s.blockAsked = &h
		s.cancel() // the first height asked is the whole question; end the run
	}
	return blockWith(), nil
}

// TestStartHeightZeroReadsTheTipAtProcessStart pins that deferring the tip
// resolution into the polling loop does not skip the blocks produced while the
// oracle waits for its first tick.
//
// A MsgAddPackage in one of those blocks is never read, so it is never
// verified, never approved, and /status reports it "unknown" for the rest of
// the run -- with nothing in the log saying a height was passed over.
func TestStartHeightZeroReadsTheTipAtProcessStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpc := &livechainRPC{start: time.Now(), cancel: cancel}
	o := newStubOracle(rpc)
	o.cfg.pollInterval = bootPoll
	o.cfg.startHeight = 0 // the flag under test: begin at the tip

	require.NoError(t, o.run(ctx))

	require.NotNil(t, rpc.blockAsked, "the oracle never started following blocks")
	require.Equal(t, bootStartTip+1, *rpc.blockAsked, // SHOULD: the tip when the process started
		"-start-height 0 must begin at the tip the chain had when gpao started; every block between that tip and the first poll is read by nobody")
	// require.Equal(t, bootStartTip+6, *rpc.blockAsked) // IS at c7ac45512: the tip one 250ms poll later, blocks 8..12 skipped
}
