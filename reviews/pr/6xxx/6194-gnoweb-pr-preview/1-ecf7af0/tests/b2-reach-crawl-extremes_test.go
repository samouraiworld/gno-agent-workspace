// Reach/extremes probes for misc/gnopreview/crawl.go at ecf7af0f2.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//   cp b2-reach-crawl-extremes_test.go misc/gnopreview/zz_reach_extremes_test.go
//   cd misc/gnopreview && go test -run 'TestReach' -v ./...
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestReachMaxPagesDiscardsEveryCapturedPage: Run() returns an error the moment
// the cap is hit, and main.go's `if err := c.Run(); err != nil { return err }`
// throws away every page already captured. The cap truncates nothing; it voids
// the whole preview and reddens the job.
func TestReachMaxPagesDiscardsEveryCapturedPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><head></head><body>%s</body></html>", r.URL.RequestURI())
	}))
	defer srv.Close()

	c := &Crawler{
		Base:     srv.URL,
		Realms:   []string{"gno.land/r/gnoland/home"},
		MaxPages: 2,
		Live:     "https://gno.land",
	}
	t.Logf("seeds: %q", c.Seeds())
	err := c.Run()
	if err == nil {
		t.Fatalf("want an error at the cap, got nil")
	}
	t.Logf("Run() error: %v", err)
	t.Logf("pages captured before the abort: %d, order: %v", len(c.pages), c.order)
	if len(c.pages) == 0 {
		t.Fatalf("expected pages to have been captured before the abort")
	}
	// The caller returns err without ever calling Write, so these captured
	// pages never reach the artifact.
}

// TestReachCapturedDirectoryIsLinkedOffsite: Seeds() captures the directory page
// at "/r/gnoland" (no trailing slash), but every gnoweb link to a directory
// carries one — views/directory.html emits `href="{{ .Link }}/"`. mapURL keys on
// the exact string, so the link leaves the snapshot for gno.land even though the
// page is sitting in the artifact.
func TestReachCapturedDirectoryIsLinkedOffsite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head></head><body><a href="/r/gnoland/">Browse</a></body></html>`)
	}))
	defer srv.Close()

	c := &Crawler{
		Base:   srv.URL,
		Realms: []string{"gno.land/r/gnoland/home"},
		Live:   "https://gno.land",
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.pages["/r/gnoland"]; !ok {
		t.Fatalf("directory seed not captured; pages=%v", c.order)
	}
	f, _ := c.FileOf("/r/gnoland")
	t.Logf("captured directory page file: %s", f)

	got := c.mapURL("/r/gnoland/", "../../")
	t.Logf("mapURL(%q) = %q", "/r/gnoland/", got)
	if strings.HasPrefix(got, "https://gno.land") {
		t.Errorf("trailing-slash link to a CAPTURED page maps offsite: %s", got)
	}

	// The rendered page proves it end to end.
	body := c.rewrite(c.pages["/r/gnoland/home"])
	t.Logf("rewritten realm page: %s", body)
	if strings.Contains(body, `href="https://gno.land/r/gnoland/"`) {
		t.Errorf("Browse link in the snapshot points at the live site")
	}
}

// TestReachOverBudgetFilePagesAreFetchedThenThrownAway: wantFile gates at
// discovery on a counter only chargeFile increments, so every file link on a
// page is enqueued and fetched before the first one is charged. With
// FileBudget=2 and six file links, six pages are downloaded and four discarded.
func TestReachOverBudgetFilePagesAreFetchedThenThrownAway(t *testing.T) {
	var mu sync.Mutex
	fileHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := r.URL.RequestURI()
		if strings.Contains(uri, "file=") {
			mu.Lock()
			fileHits++
			mu.Unlock()
			fmt.Fprint(w, "<html><head></head><body>src</body></html>")
			return
		}
		var b strings.Builder
		b.WriteString("<html><head></head><body>")
		for _, f := range []string{"a.gno", "b.gno", "c.gno", "d.gno", "e.gno", "f.gno"} {
			fmt.Fprintf(&b, `<a href="/r/gnoland/home$source&amp;file=%s">%s</a>`, f, f)
		}
		b.WriteString("</body></html>")
		fmt.Fprint(w, b.String())
	}))
	defer srv.Close()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/gnoland/home"},
		Live:       "https://gno.land",
		FileBudget: GnowebFileBudget,
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	kept := 0
	for u := range c.pages {
		if strings.Contains(u, "file=") {
			kept++
		}
	}
	t.Logf("file pages fetched from gnodev: %d, kept: %d, budget: %d", fileHits, kept, c.FileBudget)
	if fileHits <= kept {
		t.Errorf("expected over-fetching: fetched %d, kept %d", fileHits, kept)
	}
}

// TestReachSecondRunNilDerefs: Run() resets c.pages but not c.order, so a
// Crawler reused for a second crawl carries dead keys into writePages.
func TestReachSecondRunNilDerefs(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := r.URL.RequestURI()
		mu.Lock()
		first := !seen[uri]
		seen[uri] = true
		mu.Unlock()
		if !first && strings.Contains(uri, "help") {
			http.Error(w, "gone", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, "<html><head></head><body>x</body></html>")
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/gnoland/home"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	n1 := len(c.order)
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	t.Logf("order after one run: %d, after two: %d, pages: %d", n1, len(c.order), len(c.pages))

	defer func() {
		if r := recover(); r != nil {
			t.Logf("writePages panicked on the stale order entry: %v", r)
			return
		}
		t.Errorf("expected a nil-pointer panic from the stale c.order entry")
	}()
	_ = c.writePages(t.TempDir())
}

// TestReachColonArgsAreUngated: explosiveArgs gates the "$query" key space, but
// the ":args" space has no gate at all — inScope accepts every ":arg" under a
// realm it follows. A realm whose Render emits nested ":arg" links (boards2:
// ":board/1/reply", ":board/1/edit", ":board/1/flag", ...) is crawled until the
// page cap, and the cap then throws the whole preview away.
func TestReachColonArgsAreUngated(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		uri := r.URL.RequestURI()
		arg := ""
		if i := strings.Index(uri, ":"); i >= 0 {
			arg = uri[i+1:]
		}
		fmt.Fprintf(w, `<html><head></head><body>
			<a href="/r/gnoland/boards2/v0:%s/a">a</a>
			<a href="/r/gnoland/boards2/v0:%s/b">b</a>
			</body></html>`, arg, arg)
	}))
	defer srv.Close()

	c := &Crawler{
		Base:     srv.URL,
		Realms:   []string{"gno.land/r/gnoland/boards2/v0"},
		MaxPages: 400, // main.go's defaultMaxPages
		Live:     "https://gno.land",
	}
	if c.inScope("/r/gnoland/boards2/v0:test-board/1/reply") != true {
		t.Fatal("expected a nested :arg link to be in scope")
	}
	err := c.Run()
	t.Logf("Run() error: %v", err)
	t.Logf("gnodev requests served: %d, pages captured then discarded: %d", hits, len(c.pages))
	if err == nil {
		t.Fatalf("expected the page cap to abort")
	}
	if len(c.pages) < 400 {
		t.Fatalf("expected the cap to be reached, captured %d", len(c.pages))
	}
}

// TestReachDirListingBranchIsDead: urlToFile has a "_dir" branch and two tests
// pin that "/r/x/y/" and "/r/x/y" must not collide, but inScope refuses every
// trailing-slash URL and Seeds never produces one, so no page is ever written
// through that branch.
func TestReachDirListingBranchIsDead(t *testing.T) {
	c := &Crawler{Realms: []string{"gno.land/r/gnoland/home"}}
	if c.inScope("/r/gnoland/") {
		t.Fatal("trailing-slash URL unexpectedly in scope")
	}
	for _, s := range c.Seeds() {
		if strings.HasSuffix(s, "/") {
			t.Fatalf("Seeds() produced a trailing-slash URL: %q", s)
		}
	}
	t.Logf("no crawled URL can end in '/', so urlToFile's _dir branch (%q) is unreachable", urlToFile("/r/gnoland/"))
}
