// Repro: gnodev's hot-reload socket refuses a page whose Origin host string
// differs from the -web-listener string. hotreload.js dials
// ws://<webListenerAddr>/_events (contribs/gnodev/app.go:492, default
// 127.0.0.1:8888), while docs/resources/gnodev.md:39, docs/builders/quickstart.md:22
// and contribs/gnodev/README.md:16 tell users to open http://localhost:8888.
// The browser then sends Origin: http://localhost:8888 with Host: 127.0.0.1:8888,
// and gorilla's same-origin default, now in force, refuses it.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/5258/head && git checkout dd05df75bc312f6bd064ba096225699f5e8fe320
//   cp <this file> contribs/gnodev/pkg/emitter/zz_solo_finder_2_origin_test.go
//   (cd contribs/gnodev && go test -count=1 -v -run TestSoloFinder2_PageOrigin ./pkg/emitter/)
//
// Head dd05df75b: the localhost row FAILS with 403. Merge base d25b308c1, same
// file and command: every row upgrades (101).
package emitter_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/contribs/gnodev/pkg/emitter"
	"github.com/gorilla/websocket"
)

func TestSoloFinder2_PageOrigin(t *testing.T) {
	srv := emitter.NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	s := httptest.NewServer(srv) // listens on 127.0.0.1:<port>, like the default -web-listener
	defer s.Close()
	port := s.URL[strings.LastIndex(s.URL, ":")+1:]
	target := "ws://127.0.0.1:" + port + "/_events" // what hotreload.js dials

	for _, page := range []string{"http://127.0.0.1:" + port, "http://localhost:" + port} {
		h := http.Header{}
		h.Set("Origin", page)
		c, resp, err := websocket.DefaultDialer.Dial(target, h)
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		if err == nil {
			c.Close()
		}
		t.Logf("page %-26s dials %s: status=%d", page, target, status)
		if status != http.StatusSwitchingProtocols {
			t.Errorf("page opened at %s gets no hot reload: status %d", page, status)
		}
	}
}
