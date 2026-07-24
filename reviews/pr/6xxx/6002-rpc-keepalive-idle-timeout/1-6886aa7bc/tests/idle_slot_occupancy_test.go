/* Run: from a gno checkout:
gh pr checkout 6002 -R gnolang/gno && git checkout 6886aa7bc
curl -fsSL -o tm2/pkg/bft/rpc/lib/server/idle_slot_occupancy_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6002-rpc-keepalive-idle-timeout/1-6886aa7bc/tests/idle_slot_occupancy_test.go
go test -v -run 'TestIdleConnsHoldMaxOpenConnectionSlots' ./tm2/pkg/bft/rpc/lib/server/
rm tm2/pkg/bft/rpc/lib/server/idle_slot_occupancy_test.go
*/

// Listen wraps the listener in netutil.LimitListener when MaxOpenConnections
// is set, and that semaphore is released only when a connection closes. An
// idle keep-alive connection therefore holds a slot for the whole IdleTimeout,
// so a new client waits IdleTimeout for a slot rather than being refused.
// At 6886aa7bc a fresh client waits the full IdleTimeout; with IdleTimeout
// unset it waits ReadTimeout.

package rpcserver

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// dialKeepAliveIdle opens a connection, completes one keep-alive request, and
// leaves the connection open and idle so it keeps holding its listener slot.
func dialKeepAliveIdle(t *testing.T, addr string) net.Conn {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	require.NoError(t, rawGet(conn, bufio.NewReader(conn)))

	return conn
}

// timeToFirstResponse measures how long a brand-new client waits for its
// first response once every connection slot is held by an idle connection.
func timeToFirstResponse(t *testing.T, addr string) time.Duration {
	t.Helper()

	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))
	require.NoError(t, rawGet(conn, bufio.NewReader(conn)))

	return time.Since(start)
}

func TestIdleConnsHoldMaxOpenConnectionSlots(t *testing.T) {
	t.Parallel()

	const (
		slots       = 2
		readTimeout = 200 * time.Millisecond
		idleTimeout = 3 * time.Second
	)

	tt := []struct {
		name        string
		idleTimeout time.Duration
		// The new client must not be served before this point, because a slot
		// only frees when an idle connection is reaped.
		minWait time.Duration
	}{
		{"idle_timeout unset", 0, readTimeout / 2},
		{"idle_timeout set", idleTimeout, idleTimeout - 500*time.Millisecond},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := DefaultConfig()
			config.MaxOpenConnections = slots
			config.ReadTimeout = readTimeout
			config.IdleTimeout = tc.idleTimeout
			addr := startIdleTestServer(t, config)

			for range slots {
				dialKeepAliveIdle(t, addr)
			}

			waited := timeToFirstResponse(t, addr)
			t.Logf("%s: new client served after %s", tc.name, waited)

			require.GreaterOrEqual(t, waited, tc.minWait,
				"a new client should have queued behind the idle connections")
		})
	}
}
