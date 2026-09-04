# Review: [#6115](https://github.com/gnolang/gno/pull/6115)
Event: REQUEST_CHANGES

## Body
The ceiling settles for the chain's life once the first answer lands: tm2 changes consensus params only from [`abciResponses.EndBlock.ConsensusParams`](https://github.com/gnolang/gno/blob/c7ac45512/tm2/pkg/bft/state/execution.go#L427-L430), and the one `ResponseEndBlock` [gno.land's `EndBlocker`](https://github.com/gnolang/gno/blob/c7ac45512/gno.land/pkg/gnoland/app.go#L1228-L1230) fills carries `ValidatorUpdates` alone.

## contribs/gpao/oracle.go:272 [gh](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L272) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L272)
`-start-height 0` resolves the tip on the first poll rather than at process start, so a `MsgAddPackage` landing in between is never read and stays [`unknown`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/status.go#L25) on `/status` for the rest of the run.

```suggestion
	height := o.cfg.startHeight
	if height <= 0 {
		// A node already up resolves the tip here, so the blocks it produces
		// while the first tick is pending are still read.
		if status, err := o.client.RPCClient.Status(ctx, nil); err == nil {
			height = status.SyncInfo.LatestBlockHeight + 1
		}
	}
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6115 -R gnolang/gno

cat > contribs/gpao/bootwindow_test.go <<'EOF'
package main

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
// started, not the tip one poll interval later. The node is up from the first
// call and produces a block every 50ms, against a 250ms poll.
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
	// require.Equal(t, bootStartTip+6, *rpc.blockAsked) // IS: the tip one 250ms poll later, blocks 8..12 skipped
}
EOF

echo "== at the branch head =="
(cd contribs/gpao && go test -count=1 -run TestStartHeightZeroReadsTheTipAtProcessStart .)

echo "== with the merge base's contribs/gpao/oracle.go =="
base=$(git merge-base origin/master HEAD)
mv contribs/gpao/run_test.go /tmp/run_test.go.bak
git checkout "$base" -- contribs/gpao/oracle.go contribs/gpao/endtoend_test.go
(cd contribs/gpao && go test -count=1 -run TestStartHeightZeroReadsTheTipAtProcessStart .)

git checkout HEAD -- contribs/gpao/oracle.go contribs/gpao/endtoend_test.go
mv /tmp/run_test.go.bak contribs/gpao/run_test.go
rm contribs/gpao/bootwindow_test.go
```

The first run asks for block 13 rather than 8, skipping the five blocks produced while the first tick was pending, and the same file passes once the merge base's `oracle.go` is in place.

```
== at the branch head ==
--- FAIL: TestStartHeightZeroReadsTheTipAtProcessStart (0.50s)
        	            	expected: 8
        	            	actual  : 13
FAIL	github.com/gnolang/gno/contribs/gpao	0.608s
== with the merge base's contribs/gpao/oracle.go ==
ok  	github.com/gnolang/gno/contribs/gpao	0.303s
```
</details>

## contribs/gpao/oracle.go:304-315 [gh](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L304-L315) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L304)
Suggestion: the ceiling query runs before the status query, so an unreachable node pays two failed round trips and two stderr lines per poll instead of one.

```suggestion
		status, err := o.client.RPCClient.Status(ctx, nil)
		if err != nil {
			o.errf("gpao: status query failed: %v", err)
			continue
		}

		if !ceilingKnown {
			if maxGas, ok := o.queryBlockMaxGas(ctx); ok {
				o.blockMaxGas.Store(maxGas)
				ceilingKnown = true
			}
		}
```

## contribs/gpao/oracle.go:591 [gh](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L591) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L591)
Suggestion: the unanswered path returns `0` rather than the `defaultBlockMaxGas` its log names, so a caller ignoring `answered` clamps every gas amount to zero through [`gasWantedFor`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L756-L770).

```suggestion
		return defaultBlockMaxGas, false
```
