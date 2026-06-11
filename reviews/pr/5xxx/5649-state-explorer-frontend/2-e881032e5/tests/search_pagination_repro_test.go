// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5649 -R gnolang/gno && git checkout e881032e5
curl -fsSL -o gno.land/pkg/gnoweb/feature/state/search_pagination_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5649-state-explorer-frontend/2-e881032e5/tests/search_pagination_repro_test.go
go test -run 'TestSearchPaginationKeepsFilter' ./gno.land/pkg/gnoweb/feature/state/
rm gno.land/pkg/gnoweb/feature/state/search_pagination_repro_test.go
*/
//
// Mechanism: servePackagePage builds the pagination footer via
// buildPagination -> statePageHref, which stamps offset/limit/view but not
// the active `search=` filter. The address bar keeps the filter (the htmx
// search path pushes canonicalStateURL, which does carry search), so the
// two URL builders disagree. With >maxTopLevelDecls (500) matches, the
// footer renders and its Next/Last hrefs silently drop the filter: the
// click lands on the unfiltered realm at offset=500.
//
// Observed at e881032e5: the pagination <nav> contains
// `/r/demo$state&offset=500` with no `search=` param — test FAILS, pinning
// the bug. After the fix (thread search through buildPagination /
// statePageHref, mirroring canonicalStateURL), the hrefs carry
// `search=item` and the test passes.
//
// Flip check: drop the searchQuery argument from the assertion below and
// assert the href equals the unfiltered form to watch the current
// (buggy) behavior pass instead.
package state

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// buildSearchableDeclsFixture mirrors buildManyTopLevelDeclsFixture but
// names decls item0..item(n-1) so one substring matches every decl.
func buildSearchableDeclsFixture(n int) []byte {
	var b strings.Builder
	b.WriteString(`{"names":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `"item%d"`, i)
	}
	b.WriteString(`],"values":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		nbuf := make([]byte, 8)
		for j := 0; j < 8; j++ {
			nbuf[j] = byte(i >> (8 * j))
		}
		fmt.Fprintf(&b,
			`{"T":{"@type":"/gno.PrimitiveType","value":"32"},"N":"%s"}`,
			base64.StdEncoding.EncodeToString(nbuf),
		)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

// TestSearchPaginationKeepsFilter — when a search filter matches more
// than one page of decls, the pagination footer must preserve the filter:
// every page href carries `search=<query>` alongside `offset=`.
func TestSearchPaginationKeepsFilter(t *testing.T) {
	// 600 decls named item0..item599; search=item matches all of them,
	// so setTotal=600 > maxTopLevelDecls=500 and the footer renders.
	h := newPageHandler(&pageMockClient{pkgBytes: buildSearchableDeclsFixture(600)})

	rec := servePageReq(t, h, url.Values{"search": {"item"}}, "/r/demo")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()

	// The footer must be present (600 filtered matches > 500 page size).
	navStart := strings.Index(body, `<nav class="b-state-pagination"`)
	if navStart == -1 {
		t.Fatalf("pagination footer missing — expected 600 search matches to paginate")
	}
	nav := body[navStart:]
	if navEnd := strings.Index(nav, "</nav>"); navEnd != -1 {
		nav = nav[:navEnd]
	}

	// Next/Last hrefs must keep the active filter. Today they render as
	// `/r/demo$state&offset=500` (no search=) and this assertion fails.
	if !strings.Contains(nav, "search=item") {
		t.Errorf("pagination hrefs dropped the active search filter:\n%s", nav)
	}
}
