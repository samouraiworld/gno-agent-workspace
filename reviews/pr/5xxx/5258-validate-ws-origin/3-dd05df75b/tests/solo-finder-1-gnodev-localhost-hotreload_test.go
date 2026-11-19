// Repro: gnodev hot reload is refused for a page opened at http://localhost:<port>
// while gnodev listens on its default 127.0.0.1:<port> (docs/resources/gnodev.md:39
// tells users to open http://localhost:8888; the default -web-listener is 127.0.0.1:8888).
//
// The handler wiring copies contribs/gnodev/app.go:492-494: the middleware injects
// hotreload.js dialing ws://<webListenerAddr>/_events, so the page at
// http://localhost:<port> dials ws://127.0.0.1:<port>/_events with
// Origin: http://localhost:<port>, and gorilla's same-origin check compares
// "localhost:<port>" against Host "127.0.0.1:<port>".
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/5258/head && git checkout dd05df75bc312f6bd064ba096225699f5e8fe320
//   cp <this file> contribs/gnodev/pkg/emitter/solo_finder_1_origin_test.go
//   (cd contribs/gnodev && go test -count=1 -run TestSoloFinder1 -v ./pkg/emitter)
// Head: localhost_page FAILS (403). Merge base d25b308c1: both subtests pass (101).
package emitter

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestSoloFinder1_HotReloadOrigin(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	webListenerAddr := fmt.Sprintf("127.0.0.1:%d", port) // gnodev default host shape

	page := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>realm</body></html>"))
	})
	mux := http.NewServeMux()
	mux.Handle("/_events", NewServer(log.NewTestingLogger(t)))
	mux.Handle("/", NewMiddleware(fmt.Sprintf("%s/_events", webListenerAddr), page))
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	for _, tc := range []struct{ name, pageOrigin string }{
		{"127_page", "http://" + webListenerAddr},
		{"localhost_page", fmt.Sprintf("http://localhost:%d", port)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			h.Set("Origin", tc.pageOrigin) // what a browser sends for the page's own origin
			c, resp, err := websocket.DefaultDialer.Dial("ws://"+webListenerAddr+"/_events", h)
			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			t.Logf("page origin %s -> ws://%s/_events status %d err %v", tc.pageOrigin, webListenerAddr, status, err)
			require.NoError(t, err, "hot reload websocket refused")
			c.Close()
		})
	}
}
