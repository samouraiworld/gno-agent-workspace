// PR 6194 @ ecf7af0f29abe4737a52803d672bc5a33c17cc60 — misc/gnopreview, page-cap and
// render-argument bounds. Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_pagecap_test.go
//	cd misc/gnopreview && go test -run TestArgPages -v -count=1 ./...
//
// Measured (go1.25.9):
//
//	zz_pagecap_test.go:83: captured: 50 arg pages, 2 per-file pages, 55 total
//	--- PASS: TestArgPagesHaveNoBudget
//	zz_pagecap_test.go:107: Run error: page cap 10 reached (queue still had 43)
//	zz_pagecap_test.go:108: 10 pages were crawled and are thrown away
//	--- PASS: TestArgPagesBlowThePageCapAndDiscardTheCrawl
//
// Both tests PASS, i.e. both facts hold at this head:
//   1. crawl.go:209 budgets $source&file= pages (GnowebFileBudget=2) and leaves
//      :args render-argument paths uncapped — 50 of them from one realm.
//   2. crawl.go:126 turns the -max-pages backstop into a fatal error: Run returns,
//      main.render (main.go:134) returns before Crawler.Write, and the 10 pages
//      already crawled are discarded. main.go:271 treats the same error from the
//      baseline crawl as a warning and carries on; plan.Dropped (plan.go:257)
//      truncates the realm cap and reports it in the comment (comment.go:86).
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func argServer(t *testing.T, nArgs, nFiles int) *httptest.Server {
	t.Helper()
	var b strings.Builder
	for i := range nArgs {
		fmt.Fprintf(&b, `<a href="/r/x/y:p/post%d">p%d</a>`, i, i)
	}
	for i := range nFiles {
		fmt.Fprintf(&b, `<a href="/r/x/y$source&amp;file=f%d.gno">f%d</a>`, i, i)
	}
	index := "<html><head></head><body>" + b.String() + "</body></html>"
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Path
		if r.URL.RawQuery != "" {
			u += "?" + r.URL.RawQuery
		}
		switch {
		case u == "/r/x/y":
			fmt.Fprint(w, index)
		case strings.HasPrefix(u, "/r/x/y"):
			fmt.Fprintf(w, "<html><head></head><body>%s</body></html>", u)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestArgPagesHaveNoBudget(t *testing.T) {
	srv := argServer(t, 50, 10)
	defer srv.Close()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/x/y"},
		Live:       "https://gno.land",
		FileBudget: GnowebFileBudget,
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	args, files := 0, 0
	for u := range c.pages {
		switch {
		case strings.Contains(u, ":p/post"):
			args++
		case strings.Contains(u, "file="):
			files++
		}
	}
	t.Logf("captured: %d arg pages, %d per-file pages, %d total", args, files, len(c.pages))
	if files > GnowebFileBudget {
		t.Errorf("per-file pages %d exceed GnowebFileBudget %d", files, GnowebFileBudget)
	}
	if args <= GnowebFileBudget {
		t.Errorf("arg pages %d were budgeted after all", args)
	}
}

func TestArgPagesBlowThePageCapAndDiscardTheCrawl(t *testing.T) {
	srv := argServer(t, 50, 0)
	defer srv.Close()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/x/y"},
		Live:       "https://gno.land",
		MaxPages:   10,
		FileBudget: GnowebFileBudget,
	}
	err := c.Run()
	if err == nil {
		t.Fatalf("Run returned nil with MaxPages=10 and 50 reachable pages")
	}
	t.Logf("Run error: %v", err)
	t.Logf("%d pages were crawled and are thrown away: main.render returns this error before Crawler.Write", len(c.pages))
	if len(c.pages) != 10 {
		t.Errorf("captured %d pages, want 10", len(c.pages))
	}
}
