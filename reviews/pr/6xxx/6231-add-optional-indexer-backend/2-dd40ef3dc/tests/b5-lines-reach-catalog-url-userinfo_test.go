// Probe of indexer.Client.URL() (gno.land/pkg/gnoweb/indexer/client.go) as
// /search.json and the results page publish it, at gnolang/gno#6231 head dd40ef3dc.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6231/head && git checkout --detach dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp b5-lines-reach-catalog-url-userinfo_test.go gno.land/pkg/gnoweb/feature/omnisearch/b5_userinfo_probe_test.go
//	go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB5' -count=1 -v
//
// Fails (t.Errorf) when the indexer's basic-auth password appears in the
// public response.
package omnisearch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/indexer"
)

func TestB5IndexerURLUserinfoIsPublished(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "gnoweb" || pass != "hunter2" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), "latestBlockHeight") {
			_, _ = io.WriteString(w, `{"data":{"latestBlockHeight":7}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"getTransactions":[{"hash":"9f2a","block_height":5,"success":true}]}}`)
	}))
	defer srv.Close()

	// An indexer behind HTTP basic auth: Go's client turns the URL's userinfo
	// into the Authorization header, so this configuration works.
	pu, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	pu.User = url.UserPassword("gnoweb", "hunter2")
	pu.Path = "/graphql/query"
	idx := indexer.New(pu.String(), "")

	h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), idx)
	for _, raw := range []string{
		"/r/alice/blog$search&q=" + url.QueryEscape("tx:9f2a") + "&json",
		"/r/alice/blog$search&q=" + url.QueryEscape("tx:9f2a"),
	} {
		u := parseURL(t, raw)
		r := httptest.NewRequest(http.MethodGet, "/r/alice/blog", nil)
		w := httptest.NewRecorder()
		status, view := h.Handle(context.Background(), w, r, u)

		out := w.Body.String()
		if view != nil {
			var sb strings.Builder
			if err := view.Render(&sb); err != nil {
				t.Fatalf("render: %v", err)
			}
			out = sb.String()
		}
		i := strings.Index(out, "hunter2")
		t.Logf("%s: status %d, %d bytes, password at %d", raw, status, len(out), i)
		if i >= 0 {
			lo, hi := max(i-80, 0), min(i+40, len(out))
			t.Errorf("the indexer's basic-auth password is in the public response to %s:\n...%s...", raw, out[lo:hi])
		}
	}
}
