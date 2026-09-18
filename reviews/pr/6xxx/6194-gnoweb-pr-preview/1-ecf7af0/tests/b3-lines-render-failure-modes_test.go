// Three failure modes of misc/gnopreview's render path, PR 6194 at ecf7af0.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/b3_lines_render_failure_modes_test.go
//	cd misc/gnopreview && go test -run 'TestReadinessProbe|TestCrawlKeepsGoing|TestPageCapIsAHardError' -v ./...
//
// Each test asserts what the code does today; a green run is the finding.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// main.go:131 — waitReady probes one page, urlOf(plan.Realms[0]), and accepts
// only HTTP 200. gnodev renders realms lazily, so a single realm whose Render
// fails answers 500 forever: the probe burns the whole -timeout (5m by default)
// and render returns "gnoweb not ready", losing the preview of every other
// realm in the plan, which was serving 200 the entire time.
func TestReadinessProbeTiedToOneRealm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/r/demo/broken" {
			http.Error(w, "panic in Render", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "<html><head></head><body>ok</body></html>")
	}))
	defer srv.Close()

	// every other realm in the plan is up
	if resp, err := http.Get(srv.URL + "/r/demo/ok"); err != nil || resp.StatusCode != 200 {
		t.Fatalf("healthy realm: %v %v", err, resp)
	}

	start := time.Now()
	err := waitReady(srv.URL, urlOf("gno.land/r/demo/broken"), 2*time.Second, make(chan error, 1))
	if err == nil {
		t.Fatal("waitReady returned nil; the probe is no longer tied to one realm")
	}
	t.Logf("waitReady error after %s: %v", time.Since(start).Round(100*time.Millisecond), err)
	if !strings.Contains(err.Error(), "not ready") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// main.go:134 and main.go:159 — Crawler.Run never returns an error for a page
// that failed; it prints one stderr line and moves on. Comment() then links the
// realm from the plan, not from what was captured, so the posted comment points
// at a directory the snapshot does not contain, and Index() claims a realm count
// it does not list.
func TestCrawlKeepsGoingAndTheCommentStillLinksTheLostRealm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/r/demo/broken") {
			http.Error(w, "panic in Render", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "<html><head></head><body>ok</body></html>")
	}))
	defer srv.Close()

	realm := "gno.land/r/demo/broken"
	c := &Crawler{
		Base:     srv.URL,
		Realms:   []string{realm},
		MaxPages: 400,
		Live:     "https://gno.land",
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run returned an error; failures are no longer silent: %v", err)
	}
	if _, ok := c.pages[urlOf(realm)]; ok {
		t.Fatal("the realm page was captured; rewrite the fixture")
	}
	t.Logf("crawl finished with err=nil and %d page(s); the realm page is not among them", len(c.pages))

	plan := &Plan{
		ChangedRealms: []string{realm},
		Realms:        []string{realm},
		Dirs:          []string{"examples/gno.land/r/demo/broken"},
		ChangedFiles:  map[string][]string{},
	}
	md := Comment(plan, "https://preview.example/pr/1", "1")
	want := "https://preview.example/pr/1/r/demo/broken/"
	if !strings.Contains(md, want) {
		t.Fatalf("comment lost the dead link:\n%s", md)
	}
	t.Logf("comment.md still links the page that was never captured: %s", want)

	idx := Index(plan, c)
	if !strings.Contains(idx, "1 realm(s) rendered") {
		t.Fatalf("index no longer counts from the plan:\n%s", idx)
	}
	if strings.Contains(idx, "r/demo/broken") {
		t.Fatal("index listed the missing realm; only the count is plan-derived")
	}
	t.Log(`index.html says "1 realm(s) rendered" and lists none`)
}

// main.go:126 — MaxPages is wired straight from -max-pages (400) into a Crawler
// whose Run() returns an error the moment the cap is reached. render() returns
// that error, the pr-preview job goes red and the upload-artifact step is
// skipped, so a broad pull request gets no preview at all. The sibling cap,
// -max-realms, truncates instead and says so in the comment (plan.Dropped).
func TestPageCapIsAHardError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html><head></head><body>ok</body></html>")
	}))
	defer srv.Close()

	c := &Crawler{
		Base:     srv.URL,
		Realms:   []string{"gno.land/r/demo/a", "gno.land/r/demo/b"},
		MaxPages: 1, // stands in for the 400 the flag defaults to
		Live:     "https://gno.land",
	}
	err := c.Run()
	if err == nil {
		t.Fatal("Run truncated instead of failing; the cap is no longer fatal")
	}
	t.Logf("render() propagates this and the job is red: %v", err)

	// the realm cap, by contrast, degrades: it drops realms and reports the count
	p := &Plan{Realms: []string{"gno.land/r/demo/a"}, Dropped: 3}
	if !strings.Contains(Comment(p, "https://preview.example/pr/1", "1"), "not** rendered (cap reached)") {
		t.Fatal("the realm cap no longer degrades gracefully")
	}
	t.Log("-max-realms drops realms and warns in the comment; -max-pages fails the job")
}
