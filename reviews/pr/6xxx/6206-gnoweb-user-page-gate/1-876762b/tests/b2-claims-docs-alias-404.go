// b2-claims: /docs, a shipped alias that serves 200 on gno.land today, is a
// 404 under the new gate.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6206/head && git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	cp <this file> gno.land/pkg/gnoweb/zz_b2_claims_docs_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestB2Claims_DocsAliasIs404 -count=1 -v
//
// The stub answers exactly what rpc.gno.land answers for "docs" (2026-09-19):
//
//	gnokey query vm/qpaths --data '@docs' --remote https://rpc.gno.land:443
//	  -> data: (empty)
//	gnokey query vm/qeval --data 'gno.land/r/sys/users.ResolveName("docs")' --remote https://rpc.gno.land:443
//	  -> (nil *gno.land/r/sys/users.UserData)
//	     (false bool)
//
// And the page the gate replaces is live:
//
//	curl -s -o /dev/null -w '%{http_code}\n' https://gno.land/docs   -> 200
//	  (body title: "gno.land - /u/docs")
//
// Result at 876762bdf: PASS — the handler answers 404 for /u/docs.

package gnoweb_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The gate 404s /u/docs, which is where the shipped /docs alias points.
func TestB2Claims_DocsAliasIs404(t *testing.T) {
	alias, ok := gnoweb.DefaultAliases["/docs"]
	require.True(t, ok, "/docs alias is gone")
	require.Contains(t, fmt.Sprintf("%v", alias), "/u/docs")

	rr := getUserPage(t, &stubClient{
		listPathsFunc: func(context.Context, string, int) ([]string, error) {
			return []string{""}, nil
		},
		evalFunc: func(context.Context, string, string) ([]byte, error) {
			return []byte("(nil *gno.land/r/sys/users.UserData)\n(false bool)"), nil
		},
		realmFunc: func(context.Context, string, string) ([]byte, error) {
			t.Error("home realm fetched for a gated name")
			return nil, nil
		},
	}, "/u/docs")

	assert.Equal(t, http.StatusNotFound, rr.Code,
		"https://gno.land/docs answers 200 today and renders /u/docs")
}
