// Repro for PR 6194 (misc/gnopreview), finder b3-reach.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/b3-reach-waitready_test.go
//	cd misc/gnopreview && go test -run 'TestWaitReady|TestCrawler|TestRunPageCap' -v ./...
//
// Expected on this head:
//   - TestWaitReadyProbeNon200 FAILS: waitReady burns the whole timeout and
//     returns "gnoweb not ready after 3s" although the server is up and every
//     other page answers 200.
//   - TestCrawlerTolerates500 PASSES: the crawl itself logs the 500 and keeps
//     going, which is the asymmetry the finding is about.
//   - TestRunPageCapAbortsEverything FAILS: Run returns an error instead of
//     stopping at the cap, so render() aborts and writes no snapshot.

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// gnoweb answers 500 for a realm whose Render panics: handler_http.go's
// clientErrorMessage maps every error that is not PackageNotFound / Timeout /
// BadRequest to http.StatusInternalServerError.
func brokenRealmServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/r/a/broken", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "internal error")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "<html><head></head><body>ok</body></html>")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// The readiness probe is one realm page, plan.Realms[0] (main.go:140). A realm
// whose Render panics answers 500 forever, so the probe never succeeds and the
// whole render fails after -timeout with a message blaming gnodev.
func TestWaitReadyProbeNon200(t *testing.T) {
	srv := brokenRealmServer(t)
	died := make(chan error, 1)

	start := time.Now()
	err := waitReady(srv.URL, "/r/a/broken", 3*time.Second, died)
	elapsed := time.Since(start).Round(100 * time.Millisecond)

	t.Logf("waitReady returned after %s: %v", elapsed, err)
	if err != nil {
		t.Fatalf("gnodev is up and serving, yet waitReady waited %s and failed: %v", elapsed, err)
	}
}

// Contrast: the crawler treats the same 500 as a skipped page and finishes.
func TestCrawlerTolerates500(t *testing.T) {
	srv := brokenRealmServer(t)
	c := &Crawler{
		Base:     srv.URL,
		Realms:   []string{"gno.land/r/a/broken", "gno.land/r/b/ok"},
		MaxPages: 50,
		Live:     "https://gno.land",
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := c.pages["/r/a/broken"]; ok {
		t.Fatal("the 500 page should have been skipped")
	}
	if _, ok := c.pages["/r/b/ok"]; !ok {
		t.Fatal("the healthy realm should have been crawled")
	}
	t.Logf("crawl finished with %d page(s) despite the 500", len(c.pages))
}

// -max-pages is documented as a cap. Overflowing it returns an error from
// Run, which render() propagates, so nothing at all is written: no
// preview.json, no comment.md, no index.html, and the job goes red.
func TestRunPageCapAbortsEverything(t *testing.T) {
	const realm = "gno.land/r/a/ok"
	var body strings.Builder
	body.WriteString("<html><head></head><body>")
	for i := range 20 {
		fmt.Fprintf(&body, `<a href="/r/a/ok:p/%d">post %d</a>`, i, i)
	}
	body.WriteString("</body></html>")

	mux := http.NewServeMux()
	mux.HandleFunc("/r/a/ok", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, body.String())
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "<html><head></head><body>leaf</body></html>")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{realm}, MaxPages: 5, Live: "https://gno.land"}
	err := c.Run()
	t.Logf("Run with MaxPages=5 over %d linked pages: err=%v, pages kept=%d", 20, err, len(c.pages))
	if err != nil {
		t.Fatalf("hitting the page cap should truncate the crawl, not fail the preview: %v", err)
	}
}
