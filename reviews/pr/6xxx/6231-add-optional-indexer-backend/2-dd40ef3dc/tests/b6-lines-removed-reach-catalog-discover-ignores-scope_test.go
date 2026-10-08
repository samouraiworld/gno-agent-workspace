// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/6231/head && git checkout dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//   cp b6-lines-removed-reach-catalog-discover-ignores-scope_test.go gno.land/pkg/gnoweb/feature/omnisearch/zz_discover-ignores-scope_test.go
//   go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB6DiscoverIgnoresScope -v -count=1
// Expected at dd40ef3dc: FAIL for both URLs, logging "Scoped to /r/alice/blog: true, lists /r/bob/blog: true".
// A fix turns it green.

package omnisearch

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

// A free-text query carrying in:<path>, or typed on a realm page, reaches
// discover, which never reads q.PkgPath: the page heads the results
// "Scoped to <path>" and lists paths outside it.
func TestB6DiscoverIgnoresScope(t *testing.T) {
	for _, url := range []string{
		"/$search&q=blog+in:/r/alice/blog",
		"/r/alice/blog$search&q=blog",
	} {
		t.Run(url, func(t *testing.T) {
			h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), nil)
			_, view := h.Handle(context.Background(), httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil), parseURL(t, url))
			var buf bytes.Buffer
			if err := view.Component.Render(&buf); err != nil {
				t.Fatal(err)
			}
			html := buf.String()
			scoped := strings.Contains(html, "Scoped to")
			outside := strings.Contains(html, `href="/r/bob/blog"`)
			t.Logf("Scoped to /r/alice/blog: %v, lists /r/bob/blog: %v", scoped, outside)
			if scoped && outside {
				t.Errorf("page claims a scope its results ignore")
			}
		})
	}
}
