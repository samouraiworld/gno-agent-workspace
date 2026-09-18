// Repro for gnolang/gno#6194, round 1, bundle 9 (misc/gnopreview/crawl_test.go),
// angle reach: what the crawl's two caps actually bound.
//
// From a plain clone, at head ecf7af0f29abe4737a52803d672bc5a33c17cc60:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f2
//	cp <this file> misc/gnopreview/b9_reach_crawl_caps_test.go
//	cd misc/gnopreview && go test -run B9Reach -v ./...
//
// Observed at that head with go1.25.9, all four PASS (each asserts the current
// behaviour, so a fix turns the first three red):
//
//	B9REACH captured=6 served=3004 maxPages=10 fileBudget=2
//	B9REACH cap err="page cap 2 reached (queue still had 2)" captured=2
//	B9REACH defaultMaxPages=400 err=page cap 400 reached (queue still had 120998) captured=400
//	B9REACH seeds=[/r/gnoland/home /r/gnoland/home$source /r/gnoland/home$help /r/gnoland]
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// The per-realm file budget is charged at capture, so every per-file link a
// realm's $source page carries is fetched before it is dropped, and -max-pages
// never sees those fetches: it counts captured pages only.
func TestB9ReachMaxPagesDoesNotBoundFetches(t *testing.T) {
	const fanout = 3000
	var hits atomic.Int64
	var body strings.Builder
	for i := range fanout {
		fmt.Fprintf(&body, `<a href="/r/x/y$source&amp;file=f%d.gno">f%d</a>`+"\n", i, i)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/r/x/y$source" {
			fmt.Fprint(w, "<html><head></head><body>"+body.String()+"</body></html>")
			return
		}
		fmt.Fprint(w, "<html><head></head><body>ok</body></html>")
	}))
	defer srv.Close()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/x/y"},
		MaxPages:   10,
		FileBudget: GnowebFileBudget,
		Live:       "https://gno.land",
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Logf("B9REACH captured=%d served=%d maxPages=%d fileBudget=%d",
		len(c.pages), hits.Load(), c.MaxPages, c.FileBudget)
	if hits.Load() <= int64(c.MaxPages) {
		t.Fatalf("only %d fetches; the cap would have bounded them", hits.Load())
	}
}

// Hitting -max-pages aborts the whole render instead of truncating it: Run
// returns an error, main's render() propagates it, and nothing is written.
func TestB9ReachMaxPagesAbortsTheRender(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head></head><body>ok</body></html>")
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, MaxPages: 2, Live: "https://gno.land"}
	err := c.Run()
	if err == nil {
		t.Fatal("no error at the cap")
	}
	t.Logf("B9REACH cap err=%q captured=%d (all discarded, Write is never reached)", err, len(c.pages))
}

// The default cap is reached by a realm that renders 400+ of its own :arg
// links — each one is in scope and each one is captured.
func TestB9ReachArgLinksReachTheDefaultCap(t *testing.T) {
	var body strings.Builder
	for i := range 500 {
		fmt.Fprintf(&body, `<a href="/r/x/y:p/post%d">p%d</a>`+"\n", i, i)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head></head><body>"+body.String()+"</body></html>")
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, MaxPages: defaultMaxPages, Live: "https://gno.land"}
	err := c.Run()
	t.Logf("B9REACH defaultMaxPages=%d err=%v captured=%d", defaultMaxPages, err, len(c.pages))
	if err == nil {
		t.Fatal("no error at the default cap")
	}
}

// urlToFile's _dir and _root branches answer URLs no crawl can reach: inScope
// refuses any base ending in "/", and Seeds() never emits one.
func TestB9ReachListingBranchesAreUnreachable(t *testing.T) {
	c := &Crawler{Realms: []string{"gno.land/r/gnoland/home"}, FileBudget: GnowebFileBudget}
	for _, u := range []string{"/", "/r/", "/r/gnoland/", "/r/gnoland/home/"} {
		if c.inScope(u) {
			t.Errorf("inScope(%q) = true", u)
		}
	}
	for _, s := range c.Seeds() {
		if strings.HasSuffix(s, "/") {
			t.Errorf("Seeds() emits %q", s)
		}
	}
	t.Logf("B9REACH seeds=%v; urlToFile(\"/r/\")=%q urlToFile(\"/\")=%q are dead",
		c.Seeds(), urlToFile("/r/"), urlToFile("/"))
}
