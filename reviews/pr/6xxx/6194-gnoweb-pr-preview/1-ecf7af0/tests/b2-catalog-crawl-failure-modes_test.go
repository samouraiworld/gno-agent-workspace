// Repro for two crawl.go failure modes on gnolang/gno PR 6194 (head ecf7af0f2).
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_catalog_test.go
//	cd misc/gnopreview && go test -run 'TestCatalog' -v ./...
//
// Both tests PASS on the head commit, and each PASS documents the defect:
//   - TestCatalogFailedPageSilentlyFallsBackToLive: a realm whose page does not
//     return 200 is dropped, Run reports success, and every link to it is
//     rewritten to the production origin.
//   - TestCatalogPageCapDiscardsTheWholeCrawl: render-argument pages are followed
//     without any per-realm bound, and hitting MaxPages returns an error that
//     main.go propagates, throwing away the pages already captured.
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogFailedPageSilentlyFallsBackToLive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/r/demo/a":
			io.WriteString(w, `<html><head></head><body><a href="/r/demo/b">b</a></body></html>`)
		case "/r/demo/b":
			// what gnoweb serves when the realm's Render panics under the PR
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "internal error")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/demo/a", "gno.land/r/demo/b"},
		MaxPages:   50,
		Live:       "https://gno.land",
		RenderOnly: true,
	}

	err := c.Run()
	t.Logf("Run() error = %v", err)
	if err != nil {
		t.Fatalf("head behaviour changed: Run reported the broken realm (%v)", err)
	}
	if _, ok := c.pages["/r/demo/b"]; ok {
		t.Fatal("head behaviour changed: the 500 page was captured")
	}
	t.Logf("captured %d of 2 requested realms, Run() == nil", len(c.pages))

	body := c.rewrite(c.pages["/r/demo/a"])
	if !strings.Contains(body, `href="https://gno.land/r/demo/b"`) {
		t.Fatalf("head behaviour changed: link not sent to the live origin:\n%s", body)
	}
	t.Log(`link to the realm that failed now reads href="https://gno.land/r/demo/b" ` +
		`- the preview shows the deployed realm in place of the PR's broken one`)
}

func TestCatalogPageCapDiscardsTheWholeCrawl(t *testing.T) {
	const records = 6
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() == "/r/demo/dir" {
			var b strings.Builder
			for i := range records {
				// one render-argument page per record, the shape a directory
				// realm emits: /r/demo/dir:alice
				fmt.Fprintf(&b, `<a href="/r/demo/dir:u%d">u%d</a>`, i, i)
			}
			io.WriteString(w, "<html><head></head><body>"+b.String()+"</body></html>")
			return
		}
		io.WriteString(w, "<html><head></head><body>record</body></html>")
	}))
	defer srv.Close()

	// no cap: every :arg link is followed, so the page count is set by what the
	// realm renders, not by the size of the diff.
	free := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/demo/dir"}, Live: "https://gno.land"}
	if err := free.Run(); err != nil {
		t.Fatalf("uncapped crawl failed: %v", err)
	}
	t.Logf("uncapped crawl captured %d pages for one realm with %d records", len(free.pages), records)
	if len(free.pages) < records {
		t.Fatalf("head behaviour changed: render-argument pages are no longer followed (%d pages)", len(free.pages))
	}

	capped := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/demo/dir"}, MaxPages: 3, Live: "https://gno.land"}
	err := capped.Run()
	t.Logf("capped Run() error = %v; pages already captured = %d", err, len(capped.pages))
	if err == nil {
		t.Fatal("head behaviour changed: the cap no longer aborts the crawl")
	}
	if len(capped.pages) == 0 {
		t.Fatal("head behaviour changed: nothing was captured before the cap")
	}
	t.Log("main.go:134 returns that error, so the render step fails and the " +
		"pages already captured are never written")
}
