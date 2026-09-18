// b2-claims-comment-links-unrendered — PR 6194, misc/gnopreview.
//
// The claim: the PR body's "the comment lists the changed realms ... each with
// `source` and `help` shortcuts", and crawl.go:108 "Run ... returns the captured
// pages" with Run() skipping a non-200 page (crawl.go:134) and carrying on.
//
// Index(plan, c) drops a realm the crawl did not capture (comment.go:170), so
// the preview homepage is honest. Comment(plan, baseURL, pr) never sees the
// crawler at all (comment.go:17) and links every realm in the plan, so a realm
// gnodev failed to load is advertised in the sticky PR comment as an ordinary
// link — three 404s per realm (render, source, help) on the published site.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_comment_test.go
//	cd misc/gnopreview && go test -run TestCommentLinksRealms -v ./...
//
// Expected: PASS, which is the finding.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommentLinksRealmsTheCrawlNeverCaptured(t *testing.T) {
	// gnodev does not type-check before genesis: a realm whose deploy tx fails
	// is simply absent, and gnoweb answers 404 for it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, "/r/x/broken") {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `<html><head></head><body>ok</body></html>`)
	}))
	defer srv.Close()

	plan := &Plan{
		ChangedRealms: []string{"gno.land/r/x/ok", "gno.land/r/x/broken"},
		Realms:        []string{"gno.land/r/x/ok", "gno.land/r/x/broken"},
		Dirs:          []string{"examples/gno.land/r/x/ok", "examples/gno.land/r/x/broken"},
	}
	c := &Crawler{Base: srv.URL, Realms: plan.Realms, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.pages["/r/x/broken"]; ok {
		t.Fatal("the broken realm was captured; the fixture is wrong")
	}

	const base = "https://gnolang.github.io/gno-previews/pr-1"
	index := Index(plan, c)
	comment := Comment(plan, base, "1")

	if strings.Contains(index, "r/x/broken") {
		t.Error("Index listed the realm that was never captured")
	}
	if !strings.Contains(index, "r/x/ok") {
		t.Error("Index dropped the realm that was captured")
	}
	t.Log("index.html omits the realm the crawl never captured — comment.go:170")

	for _, want := range []string{
		base + "/r/x/broken/",
		base + "/r/x/broken/_t/source/",
		base + "/r/x/broken/_t/help/",
	} {
		if !strings.Contains(comment, want) {
			t.Errorf("comment is missing %q", want)
			continue
		}
		t.Logf("sticky comment links %s — nothing was written there", want)
	}
	for _, line := range strings.Split(comment, "\n") {
		if strings.Contains(line, "broken") {
			t.Logf("comment line: %s", line)
		}
	}
	if strings.Contains(comment, "not rendered") || strings.Contains(comment, "failed") {
		t.Error("the comment did warn about the realm; the finding does not hold")
	}
}
