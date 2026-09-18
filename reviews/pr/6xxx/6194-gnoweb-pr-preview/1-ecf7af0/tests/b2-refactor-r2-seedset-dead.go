// Equivalence witness for the `seedSet` rewrite in misc/gnopreview/crawl.go.
//
// Claim: the seedSet map Run() builds (crawl.go:112-115) and the
// `seedSet[link] ||` half of the link filter (crawl.go:148) admit nothing the
// crawl would not reach anyway — every seed is in the queue before the loop
// starts, and a dequeued seed is marked visited, so a link equal to a seed is
// always either visited or still queued.
//
// This test captures a realm page that links to the seeded directory page
// /r/x — the exact URL inScope() rejects and seedSet exists to admit — and
// pins the captured set. It passes on head and on the rewrite, which is the
// equivalence.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/seedset_dead_test.go
//	cd misc/gnopreview && go test -run TestSeedSetAdmitsNothingNew -v ./...
//
// Then apply the rewrite and re-run the same command:
//
//	python3 - <<'EOF'
//	import pathlib
//	p = pathlib.Path("crawl.go"); s = p.read_text()
//	s = s.replace("""	c.pages = map[string]*page{}
//		seeds := c.Seeds()
//		seedSet := map[string]bool{}
//		for _, s := range seeds {
//			seedSet[s] = true
//		}
//
//		queue := append([]string(nil), seeds...)
//	""", """	c.pages = map[string]*page{}
//
//		queue := c.Seeds()
//	""")
//	s = s.replace("if !visited[link] && (seedSet[link] || c.inScope(link)) {",
//	              "if !visited[link] && c.inScope(link) {")
//	p.write_text(s)
//	EOF
//
// Both runs print the same PASS and the same captured set.
package main

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

func TestSeedSetAdmitsNothingNew(t *testing.T) {
	body := map[string]string{
		// The render page links to its own $source tab, to the seeded directory
		// page above it, and to a realm outside the selection.
		"/r/x/y":        `<html><head></head><body><a href="/r/x/y$source">src</a><a href="/r/x">dir</a><a href="/r/other">out</a></body></html>`,
		"/r/x/y$source": `<html><head></head><body><a href="/r/x">dir</a></body></html>`,
		"/r/x/y$help":   `<html><head></head><body></body></html>`,
		"/r/x":          `<html><head></head><body><a href="/r/x/y">y</a></body></html>`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := body[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(b))
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := append([]string(nil), c.order...)
	sort.Strings(got)
	want := []string{"/r/x", "/r/x/y", "/r/x/y$help", "/r/x/y$source"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("captured %v, want %v", got, want)
	}
	t.Logf("captured %v", got)
}
