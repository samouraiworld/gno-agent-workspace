/* Run: from a gno checkout:
gh pr checkout 5986 -R gnolang/gno && git checkout 223aea42e
curl -fsSL -o tm2/pkg/p2p/switch_accept_sentinel_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5986-errors-is-transport-sentinel/1-223aea42e/tests/switch_accept_sentinel_test.go
go test -v -run 'TestSwitchAcceptLoopUnrelatedSentinel' ./tm2/pkg/p2p/
rm tm2/pkg/p2p/switch_accept_sentinel_test.go
*/

// errTransportClosed is created by tm2/pkg/errors.New, whose static type is the
// errors.Error interface, so errors.As(err, &errTransportClosed) matched every
// sibling sentinel in that package and overwrote the shared variable with it.
// At the pinned hash the loop survives errDuplicateConnection and the sentinel
// keeps its own message; reverting switch.go:637 to errors.As turns both red.

package p2p

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitchAcceptLoopUnrelatedSentinel(t *testing.T) {
	// No t.Parallel: the assertions read the package-level sentinel.
	original := errTransportClosed
	t.Cleanup(func() { errTransportClosed = original })

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	var accepts atomic.Int64

	mockTransport := &mockTransport{
		acceptFn: func(ctx context.Context, _ PeerBehavior) (PeerConn, error) {
			// Block once the loop has proven it survives the unrelated sentinel.
			if accepts.Add(1) > 3 {
				<-ctx.Done()

				return nil, ctx.Err()
			}

			return nil, errDuplicateConnection
		},
	}

	sw := NewMultiplexSwitch(mockTransport)

	done := make(chan struct{})

	go func() {
		sw.runAcceptLoop(ctx)
		close(done)
	}()

	require.Eventually(
		t,
		func() bool { return accepts.Load() > 3 },
		2*time.Second,
		10*time.Millisecond,
		"accept loop exited on an error that is not errTransportClosed",
	)

	cancelFn()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "accept loop did not exit on context cancellation")
	}

	assert.Equal(t, "transport is closed", errTransportClosed.Error())
}
