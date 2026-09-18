// b1-lines-listpaths-limit-ignored.go
//
// Claim under test: gno.land/pkg/gnoweb/client.go:221 `(*rpcClient).ListPaths`
// never uses its `limit` argument, so the `MaxUserContributions = 200` cap that
// handler_http.go:505 asks for is dropped on the floor. The node then applies
// its own default, `pathsLimit` in gno.land/pkg/sdk/vm/handler.go:200, which is
// 1_000 — five times the cap the caller believes it set. The same is true of
// realm_directory.go:99, which passes `searchPathLimit`.
//
// The test captures the wire request rather than the reply: a fake JSON-RPC
// endpoint records what gnoweb asks the node for. If the limit were honoured it
// would have to appear either in the ABCI query path ("vm/qpaths?limit=200",
// which is the only shape `pathsLimit` parses) or in the data; the assertion is
// that two calls differing only in `limit` produce byte-identical requests.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	cp <this file> gno.land/pkg/gnoweb/zz_b1_listpaths_limit_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestB1ListPathsDropsItsLimit -v
//
// Expected on a fixed tree: FAIL (the two requests differ, one carrying the
// limit). Observed at 876762bdf: PASS — the requests are identical.

package gnoweb

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
)

func TestB1ListPathsDropsItsLimit(t *testing.T) {
	var captured []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = append(captured, string(body))
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"jsonrpc":"2.0","id":"","result":{"response":{}}}`)
	}))
	defer srv.Close()

	cli, err := client.NewHTTPClient(srv.URL)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	adapter := NewRPCClientAdapter(slog.New(slog.NewTextHandler(io.Discard, nil)), cli, "gno.land", 4)

	// Two calls that differ only in the limit the caller asks for.
	adapter.ListPaths(context.Background(), "@alice", 1)
	adapter.ListPaths(context.Background(), "@alice", MaxUserContributions)

	if len(captured) != 2 {
		t.Fatalf("captured %d requests, want 2: %q", len(captured), captured)
	}
	t.Logf("limit=1   -> %s", captured[0])
	t.Logf("limit=%d -> %s", MaxUserContributions, captured[1])

	for i, body := range captured {
		if !strings.Contains(body, `"vm/qpaths"`) {
			t.Fatalf("request %d does not query vm/qpaths: %s", i, body)
		}
		if strings.Contains(body, "limit") {
			t.Fatalf("request %d carries a limit, the finding is stale: %s", i, body)
		}
	}
	// Compare the params alone: every JSON-RPC request carries a fresh id.
	params := func(body string) string {
		_, after, _ := strings.Cut(body, `"params":`)
		return after
	}
	if params(captured[0]) != params(captured[1]) {
		t.Fatalf("requests differ, so the limit does reach the node:\n%s\n%s", captured[0], captured[1])
	}
	t.Log("both requests byte-identical: the caller's limit never reaches the node, " +
		"which falls back to pathsLimit's default of 1_000")
}
