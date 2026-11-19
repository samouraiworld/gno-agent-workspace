// Repro: with cors_allowed_origins = [] node.go hands NewWebsocketManager nil, so
// gorilla's same-origin check compares Origin's host with the Host header. A reverse
// proxy that rewrites Host to its upstream (nginx's proxy_pass default) makes a page
// served from the node's own public origin fail the upgrade (403); base: 101.
//
// From a plain clone of gnolang/gno at the PR head:
//   cp <this file> tm2/pkg/bft/rpc/lib/server/zz_judge_r1_test.go
//   go test -count=1 -v -run TestJudgeR1_ProxyHost ./tm2/pkg/bft/rpc/lib/server/
// Head: the rewritten-Host row FAILS with 403. Merge base: drop the nil argument, both rows 101.
package rpcserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rs "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/server"
	"github.com/gorilla/websocket"
)

func TestJudgeR1_ProxyHost(t *testing.T) {
	wm := rs.NewWebsocketManager(map[string]*rs.RPCFunc{}, nil)
	srv := httptest.NewServer(http.HandlerFunc(wm.WebsocketHandler))
	defer srv.Close()
	u := "ws" + strings.TrimPrefix(srv.URL, "http")
	for _, row := range []struct{ name, host string }{
		{"proxy_keeps_host", "rpc.example.com"},
		{"proxy_rewrites_host", ""},
	} {
		h := http.Header{"Origin": {"https://rpc.example.com"}}
		if row.host != "" {
			h.Set("Host", row.host)
		}
		c, resp, err := websocket.DefaultDialer.Dial(u, h)
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		t.Logf("%s: page https://rpc.example.com, Host %q -> status %d err %v", row.name, row.host, code, err)
		if err != nil {
			t.Errorf("%s: same-origin page refused: %v", row.name, err)
			continue
		}
		c.Close()
	}
}
