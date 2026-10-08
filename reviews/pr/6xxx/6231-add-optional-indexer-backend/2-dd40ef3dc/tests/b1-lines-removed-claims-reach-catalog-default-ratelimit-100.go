// Repro for gnolang/gno#6231 at dd40ef3dc: the shipped default rate limit.
//
// The ADR (gno.land/adr/pr6231_gnoweb_indexer.md) says "the default moves to
// 1200/min to match the edge", and handler_http.go now declares
// defaultRateLimitPerMinute = 1200. But NewDefaultAppConfig, which
// cmd/gnoweb/main.go builds every AppConfig from, still sets
// StateRateLimitPerMinute: 100, and NewHTTPHandler only falls back to 1200
// when that field is <= 0. The binary therefore throttles at 100/min.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6231/head && git checkout dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp <this file> gno.land/pkg/gnoweb/zz_default_ratelimit_test.go
//	go test ./gno.land/pkg/gnoweb -run TestZZDefaultAppConfigRateLimit -count=1 -v
//
// Expected per the ADR: no 429 within 150 requests from one address.
// Observed: the 101st request is refused.
package gnoweb

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZZDefaultAppConfigRateLimit(t *testing.T) {
	cfg := NewDefaultAppConfig()
	t.Logf("NewDefaultAppConfig().StateRateLimitPerMinute = %d", cfg.StateRateLimitPerMinute)
	cfg.ChainID = "dev"            // skip the chain-id probe
	cfg.NodeRemote = "127.0.0.1:1" // closed port: every RPC fails fast; the limiter runs before any fetch

	router, err := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), cfg)
	require.NoError(t, err)

	limited, first := 0, 0
	for i := 1; i <= 150; i++ {
		req := httptest.NewRequest(http.MethodGet, "/r/demo/foo$search&q=zz&json", nil)
		req.RemoteAddr = "203.0.113.7:4444"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			limited++
			if first == 0 {
				first = i
			}
		}
	}
	t.Logf("$search&json from one address: %d of 150 refused with 429, first at request %d", limited, first)
	if limited > 0 {
		t.Errorf("default gnoweb config throttles at request %d; the ADR states a 1200/min default", first)
	}
}
