# Review: PR [#6002](https://github.com/gnolang/gno/pull/6002)
Event: APPROVE

## Body
Verified on 6886aa7bc: a generated `config.toml` writes `idle_timeout = "0s"` and loads back as zero, so a node upgraded without touching its config file keeps the current behavior.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6002-rpc-keepalive-idle-timeout/1-6886aa7bc/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/rpc/config/config.go:71-74 [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L71)
Reaching `max_open_connections` does not just cap idle connections, it stalls new ones: the [`netutil.LimitListener`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/lib/server/http_server.go#L288-L290) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/lib/server/http_server.go#L288-L290) slot is [released only when a connection closes](https://github.com/golang/net/blob/v0.56.0/netutil/listen.go#L83-L86), so a new client waits instead of being refused. The wait is the whole `idle_timeout` rather than today's 10 seconds, so 10m20s at the default [900 slots](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/config/config.go#L111) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L111) with a 620s setting.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6002 -R gnolang/gno

cat > tm2/pkg/bft/rpc/lib/server/slot_repro_test.go <<'EOF'
package rpcserver

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIdleConnsHoldSlots(t *testing.T) {
	for _, tc := range []struct {
		name string
		idle time.Duration
	}{{"idle_timeout unset", 0}, {"idle_timeout 3s", 3 * time.Second}} {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultConfig()
			config.MaxOpenConnections = 2
			config.ReadTimeout = 200 * time.Millisecond
			config.IdleTimeout = tc.idle
			addr := startIdleTestServer(t, config)

			// Fill every slot with a connection that goes idle after one request.
			for range 2 {
				c, err := net.DialTimeout("tcp", addr, time.Second)
				require.NoError(t, err)
				defer c.Close()
				require.NoError(t, rawGet(c, bufio.NewReader(c)))
			}

			start := time.Now()
			c, err := net.DialTimeout("tcp", addr, 30*time.Second)
			require.NoError(t, err)
			defer c.Close()
			c.SetDeadline(time.Now().Add(30 * time.Second))
			require.NoError(t, rawGet(c, bufio.NewReader(c)))
			t.Logf("new client served after %s", time.Since(start))
		})
	}
}
EOF

go test -v -run TestIdleConnsHoldSlots ./tm2/pkg/bft/rpc/lib/server/
rm tm2/pkg/bft/rpc/lib/server/slot_repro_test.go
```

```
    slot_repro_test.go:38: new client served after 201.518143ms
    slot_repro_test.go:38: new client served after 3.000887879s
--- PASS: TestIdleConnsHoldSlots (3.20s)
    --- PASS: TestIdleConnsHoldSlots/idle_timeout_unset (0.20s)
    --- PASS: TestIdleConnsHoldSlots/idle_timeout_3s (3.00s)
```
</details>

## tm2/pkg/bft/rpc/config/config_test.go:16-19 [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config_test.go#L16)
Missing test: nothing asserts the 10s that the [comment quotes](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/config/config.go#L66-L67) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L66-L67) as the zero-value fallback, so changing [`rpcserver.DefaultConfig().ReadTimeout`](https://github.com/gnolang/gno/blob/6886aa7bc/tm2/pkg/bft/rpc/lib/server/http_server.go#L46) · [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/lib/server/http_server.go#L46) would leave the wrong number in every generated `config.toml`.

<details><summary>test cases</summary>

```go
// tm2/pkg/bft/rpc/config/config_test.go, in TestDefaultRPCConfig.
// Needs rpcserver "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/server" (no import cycle).

// The idle_timeout comment quotes this as the zero-value fallback.
assert.Equal(t, 10*time.Second, rpcserver.DefaultConfig().ReadTimeout)
```
</details>

## tm2/pkg/bft/rpc/config/config.go:69-70 [↗](../../../../../.worktrees/gno-review-6002/tm2/pkg/bft/rpc/config/config.go#L69)
Suggestion: the comment tells operators to exceed their proxy's idle timeout but never shows the written form, and an unquoted `idle_timeout = 620` in `config.toml` loads as 620 nanoseconds. Being non-zero it displaces the `net/http` fallback, so the server closes keep-alive connections immediately and nothing in the load path objects.
