// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5649 -R gnolang/gno && git checkout 805b940b5
curl -fsSL -o gno.land/pkg/gnoweb/feature/state/search_pagination_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5649-state-explorer-frontend/3-805b940b5/tests/search_pagination_repro_test.go
go test -run 'TestSearchPaginationKeepsFilter' ./gno.land/pkg/gnoweb/feature/state/
rm gno.land/pkg/gnoweb/feature/state/search_pagination_repro_test.go
*/
//
// statePageHref stamps offset/limit/view into the pagination links but not the
// active search= filter, so with >500 matches the Next/Last links exit the search.
// FAILS at 805b940b5 (links render as `/r/demo$offset=500&state`, no search=);
// passes once the filter is threaded through buildPagination/statePageHref.
package state

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// 600 int variables named item0..item599, so searching "item" matches more than one page.
func buildSearchableDeclsFixture(n int) []byte {
	var b strings.Builder
	b.WriteString(`{"names":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"item%d"`, i)
	}
	b.WriteString(`],"values":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"T":{"@type":"/gno.PrimitiveType","value":"32"},"N":"AAAAAAAAAAA="}`)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

func TestSearchPaginationKeepsFilter(t *testing.T) {
	h := newPageHandler(&pageMockClient{pkgBytes: buildSearchableDeclsFixture(600)})
	rec := servePageReq(t, h, url.Values{"search": {"item"}}, "/r/demo")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	navStart := strings.Index(body, `<nav class="b-state-pagination"`)
	if navStart == -1 {
		t.Fatal("pagination footer missing")
	}
	nav := body[navStart:]
	if end := strings.Index(nav, "</nav>"); end != -1 {
		nav = nav[:end]
	}

	if !strings.Contains(nav, "search=item") {
		t.Errorf("pagination hrefs dropped the active search filter:\n%s", nav)
	}
}
