package main

/* Run: from a gno checkout:
gh pr checkout 6115 -R gnolang/gno && git checkout c7ac45512
curl -fsSL -o contribs/gpao/lograte_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6115-retry-startup-queries/1-c7ac45512/tests/lograte_test.go
(cd contribs/gpao && go test -count=1 -v -run TestPollLogRateOnAnUnreachableNode .)
rm contribs/gpao/lograte_test.go
*/

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	rpcclient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
	ctypes "github.com/gnolang/gno/tm2/pkg/bft/rpc/core/types"
	"github.com/gnolang/gno/tm2/pkg/commands"
)

// Counts what one poll writes to stderr while the node refuses every
// connection. At c7ac45512 twenty ticks write forty lines, one per query; with
// the ceiling query moved below the status error check they write twenty.

// deadRPC is a node nothing can reach. The embedded interface is nil on
// purpose, as in stubRPC: an unexpected call panics rather than passing
// silently.
type deadRPC struct{ rpcclient.Client }

func (deadRPC) Status(context.Context, *int64) (*ctypes.ResultStatus, error) {
	return nil, errors.New("dial tcp 127.0.0.1:26657: connect: connection refused")
}

func (deadRPC) ConsensusParams(context.Context, *int64) (*ctypes.ResultConsensusParams, error) {
	return nil, errors.New("dial tcp 127.0.0.1:26657: connect: connection refused")
}

func TestPollLogRateOnAnUnreachableNode(t *testing.T) {
	const ticks = 20

	var errBuf bytes.Buffer
	io := commands.NewTestIO()
	io.SetErr(commands.WriteNopCloser(&errBuf))

	o := newStubOracle(deadRPC{})
	o.io = io
	o.cfg.pollInterval = 10 * time.Millisecond
	o.cfg.startHeight = 0

	// Half a tick past the last one, so the count is not a race with the timer.
	ctx, cancel := context.WithTimeout(context.Background(), 205*time.Millisecond)
	defer cancel()
	_ = o.run(ctx)

	lines := strings.Split(strings.TrimSpace(errBuf.String()), "\n")
	ceiling, status := 0, 0
	for _, l := range lines {
		if strings.Contains(l, "block max gas query failed") {
			ceiling++
		}
		if strings.Contains(l, "status query failed") {
			status++
		}
	}
	t.Logf("%d ticks: total=%d ceiling=%d status=%d", ticks, len(lines), ceiling, status)

	if ceiling != 0 { // SHOULD: a tick that cannot reach the node asks once
		t.Errorf("the ceiling query ran %d times against a node no status call could reach; "+
			"a failed Status already says the same node will not answer ConsensusParams", ceiling)
	}
	// if ceiling != ticks { t.Error(...) } // IS at c7ac45512: one ceiling query per tick on top of the status one
}
