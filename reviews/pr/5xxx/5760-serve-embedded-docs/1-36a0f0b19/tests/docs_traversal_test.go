// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5760 -R gnolang/gno
curl -fsSL -o gno.land/pkg/gnoweb/docs_traversal_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5760-serve-embedded-docs/1-36a0f0b19/tests/docs_traversal_test.go
go test -v -run 'TestDocsTraversal|TestDocsAssetTraversal|TestDocsResolveEscape' ./gno.land/pkg/gnoweb/
rm gno.land/pkg/gnoweb/docs_traversal_test.go

Exercises path traversal against DocsHandler.resolve and the raw-asset
http.FileServer mounted at /docs/. Calls the handler's ServeHTTP directly
(bypassing http.ServeMux path cleaning, which is the attacker's real lever
when gnoweb sits behind a proxy that forwards un-normalised paths) with
../ sequences, percent-encoded ../ (%2e%2e%2f), and absolute-looking paths.

Observed: resolve() rejects leading ".." via path.Clean + HasPrefix, and
http.FileServer rejects traversal, so all escape attempts return 404 / no
embedded-package source leaks. The "go-source-leak" case confirms the
embed scope does not expose docs/*.go. No bug — these assert the PR's
defenses hold; flipping any wantLeak to true would signal a regression.
*/
package gnoweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/log"
)

// newDocsHandlerForTest builds a DocsHandler with a nil renderer; the
// traversal cases never reach RenderRealm (they 404 first) or hit the
// asset server, so the renderer is unused.
func newDocsHandlerForTest(t *testing.T) *DocsHandler {
	t.Helper()
	return NewDocsHandler(log.NewTestingLogger(t), StaticMetadata{Domain: "gno.land"}, nil)
}

// TestDocsResolveEscape hits the unexported resolver with traversal inputs.
func TestDocsResolveEscape(t *testing.T) {
	h := newDocsHandlerForTest(t)
	escapes := []string{
		"../docs.go",
		"../../go.mod",
		"../sidebar.go",
		"..%2f..%2fgo.mod", // not decoded here; literal
		"....//go.mod",
		"resources/../../docs.go",
		"resources/../../../README.md",
		"/etc/passwd",
		"images/../../docs.go",
	}
	for _, in := range escapes {
		t.Run(in, func(t *testing.T) {
			_, _, ok := h.resolve(in)
			if ok {
				t.Errorf("resolve(%q) unexpectedly succeeded — path escape", in)
			}
		})
	}
}

// TestDocsTraversal drives the markdown branch via ServeHTTP directly,
// bypassing ServeMux cleaning. A 200 + embedded Go source in the body
// would mean the resolver leaked a non-doc file.
func TestDocsTraversal(t *testing.T) {
	h := newDocsHandlerForTest(t)
	cases := []string{
		"/docs/../docs.go",
		"/docs/../../go.mod",
		"/docs/resources/../../sidebar.go",
		"/docs/%2e%2e/docs.go",
	}
	for _, route := range cases {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://gno.land"+route, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			body := rec.Body.String()
			if strings.Contains(body, "package gnoweb") || strings.Contains(body, "package docs") || strings.Contains(body, "//go:embed") {
				t.Errorf("route %q leaked embedded Go source (status %d)", route, rec.Code)
			}
		})
	}
}

// TestDocsAssetTraversal drives the raw-asset http.FileServer branch
// (/docs/images/, /docs/_assets/) with traversal payloads. http.FileServer
// is expected to reject these; this asserts the StripPrefix wiring did not
// open a hole.
func TestDocsAssetTraversal(t *testing.T) {
	h := newDocsHandlerForTest(t)
	cases := []string{
		"/docs/images/../../docs.go",
		"/docs/_assets/../../sidebar.go",
		"/docs/images/../README.md",
		"/docs/_assets/../../../go.mod",
	}
	for _, route := range cases {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://gno.land"+route, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			body := rec.Body.String()
			if strings.Contains(body, "package docs") || strings.Contains(body, "//go:embed") || strings.Contains(body, "module github.com/gnolang/gno") {
				t.Errorf("asset route %q leaked source/module file (status %d)", route, rec.Code)
			}
		})
	}
}
