// Probe: which /u/ shapes reach the new GetUserView gate, and which bypass it.
//
// Repro from a plain clone of gnolang/gno at 876762bdf2ea6635b27e2b0a42f26f9fdc54af24:
//   cp <this file> gno.land/pkg/gnoweb/zz_reach_b1_test.go
//   go test ./gno.land/pkg/gnoweb/ -run TestReachB1UserGateShapes -v
package gnoweb

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestReachB1UserGateShapes(t *testing.T) {
	logger := log.NewTestingLogger(t)
	cfg := NewDefaultAppConfig()
	cfg.NodeRemote = sharedNodeRemote(t)

	router, err := NewRouter(logger, cfg)
	require.NoError(t, err)

	for _, route := range []string{
		"/u/zzznotauser",  // the gate's own case
		"/u/zzznotauser/", // trailing slash: IsDir() wins over IsUser() in the dispatch
		"/u/zzznotauser$source",
		"/u/zzznotauser$help",
		"/u/zzznotauser:someargs",
		"/u/moul001",  // registered in genesis_txs.jsonl, no packages
		"/u/moul001/", // same, trailing slash
		"/u/std",      // qpaths special-cases @std to the standard library
		"/u/stdlibs",
		"/u/",
		"/u/" + strings.Repeat("a", 64), // at maxUsernameLen
		"/u/" + strings.Repeat("a", 65), // over it
		"/u/g1manfred47kzduec920z88wfr64ylksmdcedlf5",
		"/u/g1manfred47kzduec920z88wfr64ylksmdcedlf5/",
	} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		body := rr.Body.String()
		marker := "-"
		switch {
		case strings.Contains(body, "user not found"):
			marker = "user-not-found"
		case strings.Contains(body, "Gnome "):
			marker = "USER PAGE SERVED"
		}
		fmt.Printf("REACH %-48s status=%d body=%s\n", route, rr.Code, marker)
	}
}
