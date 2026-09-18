// b2-claims-page-cap-aborts — PR 6194, misc/gnopreview/crawl.go.
//
// Checks two claims the diff writes about itself:
//
//  1. README.md:118 "A `-max-pages` cap (400) backstops the rest." and the PR
//     body's "At most 25 realms and 400 pages". Crawler.Run returns an error
//     when the cap is reached, and main.render propagates it, so the whole
//     preview is discarded instead of being published truncated.
//  2. crawl.go:488 "A trailing slash is gnoweb's listing view ... give it its
//     own file" and urlToFile's _dir branch. No crawled URL ever carries a
//     trailing slash, because inScope rejects one and no seed has one, so the
//     branch is unreachable from Run.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_claims_test.go
//	cd misc/gnopreview && go test -run 'TestPageCap|TestListing' -v ./...
//
// Expected: both tests PASS, which is the finding — the cap aborts, and the
// listing branch never runs.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeGnoweb serves one realm whose render page links to many render-argument
// pages, the shape gnoweb produces for a realm with per-path content.
func fakeGnoweb(t *testing.T, args int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString(`<html><head><meta name="robots" content="index, follow" /></head><body>`)
		if r.URL.Path == "/r/x/y" || strings.HasPrefix(r.RequestURI, "/r/x/y") {
			for i := range args {
				fmt.Fprintf(&b, `<a href="/r/x/y:p/%d">p%d</a>`, i, i)
			}
			// gnoweb's breadcrumb links the listing view.
			b.WriteString(`<a href="/r/x/y/">listing</a>`)
		}
		b.WriteString(`</body></html>`)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, b.String())
	}))
}

func TestPageCapAbortsWholeRender(t *testing.T) {
	srv := fakeGnoweb(t, 40)
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, MaxPages: 5, Live: "https://gno.land"}
	err := c.Run()
	if err == nil {
		t.Fatalf("Run() returned nil with %d pages captured; the cap did not abort", len(c.pages))
	}
	t.Logf("Run() error: %v", err)
	t.Logf("pages captured before the abort: %d", len(c.pages))
	if !strings.Contains(err.Error(), "page cap") {
		t.Fatalf("unexpected error: %v", err)
	}
	// main.render returns this error, so nothing downstream runs: no Write, no
	// index.html, no comment.md. The pages already captured are thrown away.
	if len(c.pages) == 0 {
		t.Fatalf("expected pages to have been captured and then discarded")
	}
}

func TestListingBranchNeverRuns(t *testing.T) {
	srv := fakeGnoweb(t, 3)
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	for u, p := range c.pages {
		if strings.HasSuffix(u, "/") {
			t.Fatalf("captured a listing URL %q", u)
		}
		if strings.Contains(p.File, "/_dir/") {
			t.Fatalf("page %q written under _dir: %q", u, p.File)
		}
	}
	t.Logf("%d pages captured, none of them a listing: %v", len(c.pages), keysOf(c.pages))
	// The link to "/r/x/y/" was seen on every page and never followed.
	if !c.inScope("/r/x/y/") {
		t.Logf("inScope(%q) = false, which is why _dir is unreachable", "/r/x/y/")
	}
}

func keysOf(m map[string]*page) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
