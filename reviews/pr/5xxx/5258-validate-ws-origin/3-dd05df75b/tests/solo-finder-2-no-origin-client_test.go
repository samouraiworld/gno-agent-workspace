// Repro: a client sending no Origin header, the in-tree Go WebSocket client
// (tm2/pkg/bft/rpc/lib/client/ws, behind rpcclient.NewWSClient, used by
// contribs/tx-archive backup and contribs/gnohealth timestamp), is refused
// on /websocket once cors_allowed_origins is narrowed, while a plain HTTP
// POST with no Origin is still served by the same node.
//
// The server is wired the way tm2/pkg/bft/node/node.go startRPC wires it:
// the WebSocket manager gets CORSAllowedOrigins only when CORS is enabled,
// and the mux is wrapped in rs/cors with the same list.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/5258/head && git checkout dd05df75bc312f6bd064ba096225699f5e8fe320
//   cp <this file> tm2/pkg/bft/rpc/lib/server/zz_solo_finder_2_no_origin_test.go
//   go test -count=1 -v -run TestSoloFinder2_NoOriginClient ./tm2/pkg/bft/rpc/lib/server/
//
// Head dd05df75b: the two narrowed rows FAIL (ws: bad handshake, 403) while
// their HTTP POST answers 200. At the merge base d25b308c1 the constructor has
// no origins argument: drop `wsOrigins` from the NewWebsocketManager call and
// every row connects.
package rpcserver_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/client/ws"
	rs "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/server"
	types "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/types"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/rs/cors"
)

func TestSoloFinder2_NoOriginClient(t *testing.T) {
	for _, origins := range [][]string{nil, {"*"}, {"https://gno.land"}, {"https://*.gno.land"}} {
		funcs := map[string]*rs.RPCFunc{
			"status": rs.NewRPCFunc(func(*types.Context) (string, error) { return "ok", nil }, ""),
		}
		var wsOrigins []string
		if len(origins) != 0 { // node.go: n.config.RPC.IsCorsEnabled()
			wsOrigins = origins
		}
		wm := rs.NewWebsocketManager(funcs, wsOrigins)
		wm.SetLogger(log.NewNoopLogger())
		mux := http.NewServeMux()
		mux.HandleFunc("/websocket", wm.WebsocketHandler)
		rs.RegisterRPCFuncs(mux, funcs, log.NewNoopLogger(), 0)
		var root http.Handler = mux
		if len(origins) != 0 {
			root = cors.New(cors.Options{AllowedOrigins: origins}).Handler(mux)
		}
		s := httptest.NewServer(root)

		httpStatus := 0
		resp, err := http.Post(s.URL, "application/json",
			bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"status"}`))
		if err == nil {
			httpStatus = resp.StatusCode
			resp.Body.Close()
		}

		c, wsErr := ws.NewClient("ws" + strings.TrimPrefix(s.URL, "http") + "/websocket")
		if wsErr == nil {
			c.Close()
		}
		t.Logf("cors_allowed_origins=%-22q HTTP POST=%d ws.NewClient err=%v", origins, httpStatus, wsErr)
		if wsErr != nil {
			t.Errorf("cors_allowed_origins=%q: a client sending no Origin cannot open /websocket: %v", origins, wsErr)
		}
		s.Close()
	}
}
