// judge-33-empty-crawl-publishes — Run has no floor on misc/gnopreview/crawl.go
// at gnolang/gno PR 6194, head ecf7af0f2.
//
// Asserts that a crawl in which every fetch fails still returns nil with zero
// pages, and that the index and the sticky comment are produced anyway.
// PASSes at the reviewed head; the PASS is the defect.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_judge33_test.go
//	cd misc/gnopreview && go test -run 'TestJudge33' -v .
//	rm misc/gnopreview/zz_judge33_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gnodev answered waitReady, then wedged: every crawl fetch 500s.
func TestJudge33EmptyCrawlStillPublishes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Crawler{
		Base:   srv.URL,
		Realms: []string{"gno.land/r/x/y"},
		Live:   "https://gno.land",
	}
	err := c.Run()
	if err != nil {
		t.Fatalf("head behaviour changed: Run() = %v, want nil", err)
	}
	if len(c.pages) != 0 {
		t.Fatalf("head behaviour changed: captured %d page(s), want 0", len(c.pages))
	}
	t.Logf("Run() = %v with %d page(s) captured", err, len(c.pages))

	// render() gates nothing on len(c.pages): Index and Comment run next.
	plan := &Plan{
		Realms:        []string{"gno.land/r/x/y"},
		ChangedRealms: []string{"gno.land/r/x/y"},
		Dirs:          []string{"examples/gno.land/r/x/y"},
	}
	idx := Index(plan, c)
	if strings.Contains(idx, "<li>") {
		t.Fatalf("head behaviour changed: the index lists a realm\n%s", idx)
	}
	for _, line := range strings.Split(idx, "\n") {
		if strings.Contains(line, "realm(s) rendered") {
			t.Logf("index says: %s", strings.TrimSpace(line))
		}
	}

	cm := Comment(plan, "https://example.github.io/preview/6194", "6194")
	if cm == "" {
		t.Fatal("head behaviour changed: no comment for an empty crawl")
	}
	t.Logf("comment.md is still written, %d bytes; first line: %s",
		len(cm), strings.SplitN(cm, "\n", 2)[0])

	// SHOULD: a crawl that captured nothing fails the render step instead.
	// if err == nil && len(c.pages) == 0 {
	// 	t.Error("Run() returned nil after capturing no pages")
	// }
}
