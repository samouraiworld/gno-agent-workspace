// Probe of Handler.scope + Handler.Search (gno.land/pkg/gnoweb/feature/omnisearch/handler.go)
// at gnolang/gno#6231 head dd40ef3dc: the page lists `in:/r/<path>` as
// "Narrow to a package", scope() parses it into q.PkgPath, and unknownFilter()
// accepts it, but a query with no selector goes to discover(), which never
// reads q.PkgPath. The answer is the unscoped listing, with no error.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6231/head && git checkout --detach dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp b3-lines-reach-catalog-in-qualifier-ignored_test.go gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_inqual_test.go
//	go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3InQualifier' -count=1 -v
//
// Expected at dd40ef3dc: FAIL, "/r/bob/blog" listed for `blog in:/r/alice/blog`.
package omnisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestB3InQualifierNarrowsDiscovery(t *testing.T) {
	h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), nil)
	u := parseURL(t, "/$search&q="+url.QueryEscape("blog in:/r/alice/blog")+"&json")
	w := httptest.NewRecorder()
	status, _ := h.Handle(context.Background(), w, httptest.NewRequest(http.MethodGet, "/", nil), u)

	var resp jsonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	t.Logf("status=%d pkg_path=%q unknown_filter=%q", status, resp.PkgPath, resp.UnknownFilter)
	for _, g := range resp.Groups {
		for _, r := range g.Results {
			t.Logf("group=%s result=%s", g.Label, r.Title)
			if g.Label != "Users" && r.Title != "/r/alice/blog" {
				t.Errorf("in:/r/alice/blog answered with %q (group %s)", r.Title, g.Label)
			}
		}
	}
}
