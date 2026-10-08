// Repro, from a plain clone of gnolang/gno at dd40ef3dc9b92a3df888bb9e70dc21871dcc192a:
//
//	cp b2-lines-removed-reach-catalog-router-rate-default_test.go gno.land/pkg/gnoweb/zz_router_rate_default_test.go
//	go test ./gno.land/pkg/gnoweb -run TestRouterSearchRateLimitDefault -v -count=1
//
// The diff raises defaultRateLimitPerMinute from 100 to 1200 and documents it
// as the backstop "applied when no explicit value is configured", but
// NewDefaultAppConfig still sets StateRateLimitPerMinute: 100, which NewRouter
// forwards, so the gnoweb binary (main.go starts from NewDefaultAppConfig and
// has no flag for the rate) never sees 1200: the new search limiter and the
// state limiter both run at 100/min. Expect 1200 allowed requests; a red test
// shows how many the router actually allows.
package gnoweb

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouterSearchRateLimitDefault(t *testing.T) {
	cfg := NewDefaultAppConfig()
	cfg.ChainID = "dev"           // skip the chain-id RPC at construction
	cfg.NodeRemote = "127.0.0.1:1" // nothing listens; the limiter answers before any RPC
	t.Logf("NewDefaultAppConfig().StateRateLimitPerMinute = %d, defaultRateLimitPerMinute = %d",
		cfg.StateRateLimitPerMinute, defaultRateLimitPerMinute)

	router, err := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), cfg)
	require.NoError(t, err)

	allowed := 0
	for range defaultRateLimitPerMinute + 1 {
		rr := httptest.NewRecorder()
		// q shorter than omnisearch.MinTermLen: no path listing is needed.
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/r/demo/x$search&q=a&json", nil))
		if rr.Code == http.StatusTooManyRequests {
			break
		}
		allowed++
	}
	t.Logf("search requests from one IP before the first 429: %d", allowed)
	assert.GreaterOrEqual(t, allowed, defaultRateLimitPerMinute,
		"the router built from NewDefaultAppConfig should apply the documented %d/min default", defaultRateLimitPerMinute)
}
