# Review: PR [#5258](https://github.com/gnolang/gno/pull/5258)
Event: REQUEST_CHANGES

## Body
The master-merge committed a 40 MB compiled binary at [`contribs/gnokeykc/gnokeykc`](https://github.com/gnolang/gno/blob/c8b2138a8/contribs/gnokeykc/gnokeykc) [↗](../../../../../.worktrees/gno-review-5258/contribs/gnokeykc/gnokeykc), unrelated to this change; it is not gitignored and would bloat the repo permanently. Drop it from the branch and gitignore the build output.

Exercised the gnodev emitter upgrade handler on c8b2138a8: a cross-origin dial is rejected with 403 while same-origin and no-Origin dials upgrade, and no committed test covers that path.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5258-validate-ws-origin/2-c8b2138a8/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/rpc/lib/server/handlers.go:970-988 [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L970)
A non-`*`, non-empty `CORSAllowedOrigins` rejects any WebSocket client that sends no Origin header with a 403. The in-tree Go WS client at [`client/ws/client.go:42`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/client/ws/client.go#L42) [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/client/ws/client.go#L42) dials with no Origin and backs the event-subscription client, so Go subscribers and curl clients break once an operator narrows the list. Treating an absent Origin as allowed matches gorilla's default and still blocks browser CSWSH, since a victim browser cannot omit Origin.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5258 -R gnolang/gno
cat > tm2/pkg/bft/rpc/lib/server/zz_origin_probe_test.go <<'EOF'
package rpcserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	rs "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/server"
	types "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/types"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gorilla/websocket"
)

func TestOriginProbe(t *testing.T) {
	cases := []struct {
		name    string
		origins []string
		origin  string
	}{
		{"wildcard+noOrigin", []string{"*"}, ""},
		{"restricted+noOrigin", []string{"http://good.com"}, ""},
		{"restricted+match", []string{"http://good.com"}, "http://good.com"},
	}
	for _, c := range cases {
		fm := map[string]*rs.RPCFunc{"c": rs.NewWSRPCFunc(func(ctx *types.Context, s string, i int) (string, error) { return "x", nil }, "s,i")}
		wm := rs.NewWebsocketManager(fm, c.origins)
		wm.SetLogger(log.NewNoopLogger())
		mux := http.NewServeMux()
		mux.HandleFunc("/websocket", wm.WebsocketHandler)
		s := httptest.NewServer(mux)
		h := http.Header{}
		if c.origin != "" {
			h.Set("Origin", c.origin)
		}
		d := websocket.Dialer{}
		conn, resp, err := d.Dial("ws://"+s.Listener.Addr().String()+"/websocket", h)
		st := 0
		if resp != nil {
			st = resp.StatusCode
		}
		if err == nil {
			conn.Close()
		}
		t.Logf("%-20s => status=%d", c.name, st)
		s.Close()
	}
}
EOF
go test -count=1 -v -run TestOriginProbe ./tm2/pkg/bft/rpc/lib/server/
rm tm2/pkg/bft/rpc/lib/server/zz_origin_probe_test.go
```

```
    zz_origin_probe_test.go:44: wildcard+noOrigin    => status=101
    zz_origin_probe_test.go:44: restricted+noOrigin  => status=403
    zz_origin_probe_test.go:44: restricted+match     => status=101
--- PASS: TestOriginProbe (0.01s)
```
</details>

## tm2/pkg/bft/rpc/lib/server/handlers_test.go:471 [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L471)
Missing test: no case pins the no-Origin reject under a non-empty `allowedOrigins`. The `no origin header is allowed` case uses an empty list, which takes the gorilla fallback, not the `rs/cors` path a narrowed list hits.

<details><summary>test cases</summary>

Add a row to `TestWebsocketManagerCheckOrigin` in [`handlers_test.go`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L471) [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L471) pinning whichever decision the restricted-list behavior lands on:

```go
{
	name:           "no origin header rejected when allowedOrigins set",
	allowedOrigins: []string{"http://good.com"},
	origin:         "",
	expectAllowed:  false, // 403 today; flip to true if the no-Origin fix lands
},
```
</details>
