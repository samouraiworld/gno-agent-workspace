// Repro for the claim in main.go:195-198 ("Nothing else watches this process")
// and the PR body's "gnodev is watched, so a node that dies fails in seconds
// with the reason": the watch covers the boot only. Once waitReady has drained
// the died channel, a gnodev that dies mid-crawl is invisible — Crawler.Run
// logs each refused fetch to stderr and returns nil, so render() goes on to
// write the tree, the index and comment.md and exits 0 with a partial preview.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp b3-claims-r2-node-death-mid-crawl.go misc/gnopreview/zz_b3_claims_r2_test.go
//	cd misc/gnopreview && go test -run 'ClaimsR2' -v ./...
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

)

// TestClaimsR2NodeDeathMidCrawlIsSilent: the node answers the first seed and
// then goes away, as a gnodev that panics or is OOM-killed does.
func TestClaimsR2NodeDeathMidCrawlIsSilent(t *testing.T) {
	var (
		srv  *httptest.Server
		once sync.Once
		gone = make(chan struct{})
	)
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Answer this one, then stop listening and drop the connection: the
		// next fetch dials a port nothing holds, exactly as it does when the
		// gnodev process is gone.
		w.Header().Set("Connection", "close")
		io.WriteString(w, "<html><body>rendered</body></html>")
		once.Do(func() {
			_ = srv.Listener.Close()
			close(gone)
		})
	}))
	defer func() { recover() }()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/demo/a", "gno.land/r/demo/b", "gno.land/r/demo/c"},
		RenderOnly: true,
		MaxPages:   3,
		Live:       "https://gno.land",
	}
	seeds := len(c.Seeds())

	err := c.Run()
	<-gone

	t.Logf("seeds=%d captured=%d err=%v", seeds, len(c.pages), err)
	if err != nil {
		t.Fatalf("node death surfaced as an error (claim would hold): %v", err)
	}
	if len(c.pages) >= seeds {
		t.Fatalf("every seed was captured; the server did not die (captured %d of %d)", len(c.pages), seeds)
	}
	// Run returned nil with a partial capture: render() continues from here to
	// Write, Index and Comment, and the process exits 0.
	t.Logf("DEFECT: Run() == nil after the node died; %d of %d realm pages captured, "+
		"so the published snapshot is partial and the job is green", len(c.pages), seeds)
}
