// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6206/head && git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	cp <this file> gno.land/pkg/gnoweb/zz_claims_docs_test.go
//	go test ./gno.land/pkg/gnoweb -run TestClaimsDocsAliasBecomes404 -v
//
// Expect PASS: the /docs default alias answers 404 once the chain has no
// r/docs namespace and r/sys/users does not resolve "docs" — the gnoland-1
// shape, measured 2026-09-19 with the two gnokey queries quoted below.
//
package gnoweb_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaimsDocsAliasBecomes404 pins the PR's own note:
// "/docs becomes a 404 on gnoland-1".
//
// DefaultAliases maps "/docs" -> "/u/docs" (gno.land/pkg/gnoweb/app.go).
// After the gate, /u/docs is served only when "docs" is an address, holds a
// package, or resolves in r/sys/users. On gnoland-1 none of the three holds:
//
//	gnokey query vm/qpaths --data "@docs" --remote https://rpc.gno.land:443
//	  -> data: <empty>
//	gnokey query vm/qeval --data 'gno.land/r/sys/users.ResolveName("docs")' \
//	  --remote https://rpc.gno.land:443
//	  -> data: (nil *gno.land/r/sys/users.UserData)
//	           (false bool)
//
// The in-repo test node cannot catch this: it loads examples/, which ship
// gno.land/r/docs/*, so rule 2 keeps /docs at 200 there. This test stubs the
// live chain's answers instead.
func TestClaimsDocsAliasBecomes404(t *testing.T) {
	t.Parallel()

	client := &stubClient{
		// gnoland-1: no package under @docs (vm/qpaths returns one blank line).
		listPathsFunc: func(context.Context, string, int) ([]string, error) {
			return []string{""}, nil
		},
		// gnoland-1: ResolveName("docs") is (nil, false).
		evalFunc: func(_ context.Context, _, expr string) ([]byte, error) {
			assert.Equal(t, `ResolveName("docs")`, expr)
			return []byte("(nil *gno.land/r/sys/users.UserData)\n(false bool)"), nil
		},
	}

	cfg := newTestHandlerConfig(t, client)
	cfg.Aliases = gnoweb.DefaultAliases
	h, err := gnoweb.NewHTTPHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), cfg)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs", nil))

	assert.Equal(t, http.StatusNotFound, rr.Code,
		"the /docs default alias now 404s on a chain without an r/docs namespace")
	assert.Contains(t, rr.Body.String(), "user not found")
}
