// Repro: a node whose cors_allowed_origins is narrowed (not "*") refuses every
// WebSocket client that sends no Origin header, i.e. every non-browser client
// (gorilla's Dialer and the in-tree tm2 rpc ws client send none), while a
// cross-site browser page is refused as intended.
//
// rs/cors v1.11.1 OriginAllowed reads Origin "" and compares it to the list;
// only allowedOriginsAll admits "". Its godoc ("No origin at all is also allowed")
// holds only for "*".
//
// From a plain clone of gnolang/gno at dd05df75bc312f6bd064ba096225699f5e8fe320:
//   cp <this file> tm2/pkg/bft/rpc/lib/server/solo_finder_1_noorigin_test.go
//   go test -count=1 -run TestSoloFinder1 -v ./tm2/pkg/bft/rpc/lib/server
// Head: no_origin_cli FAILS with 403; cross_site_page is refused (403) as intended.
// Merge base: NewWebsocketManager took no origins and CheckOrigin returned true.
package rpcserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	rs "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/server"
	types "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/types"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestSoloFinder1_NarrowedOriginsNoOrigin(t *testing.T) {
	funcMap := map[string]*rs.RPCFunc{
		"c": rs.NewWSRPCFunc(func(ctx *types.Context) (string, error) { return "foo", nil }, ""),
	}
	// What node.go passes for cors_allowed_origins = ["https://gno.land", "https://*.gno.land"].
	wm := rs.NewWebsocketManager(funcMap, []string{"https://gno.land", "https://*.gno.land"})
	wm.SetLogger(log.NewNoopLogger())
	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", wm.WebsocketHandler)
	s := httptest.NewServer(mux)
	defer s.Close()

	for _, tc := range []struct {
		name, origin string
		want         int
	}{
		{"no_origin_cli", "", http.StatusSwitchingProtocols},
		{"allowed_page", "https://app.gno.land", http.StatusSwitchingProtocols},
		{"cross_site_page", "https://evil.example", http.StatusForbidden},
		{"null_origin_page", "null", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.origin != "" {
				h.Set("Origin", tc.origin)
			}
			c, resp, err := websocket.DefaultDialer.Dial("ws://"+s.Listener.Addr().String()+"/websocket", h)
			got := 0
			if resp != nil {
				got = resp.StatusCode
			}
			if c != nil {
				c.Close()
			}
			t.Logf("origin %q -> status %d err %v", tc.origin, got, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
