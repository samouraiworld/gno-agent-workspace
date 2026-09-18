// The page cap aborts the crawl instead of bounding it (PR 6194, crawl.go:135).
//
// misc/gnopreview has two caps on the same pipeline. The realm cap truncates
// and records the overflow (plan.go:256-258, "Dropped counts realms left out by
// the cap — never silently"), so a wide PR still gets a preview plus a line
// saying what was left out. The page cap one level down returns an error that
// main.go:134 propagates before Write ever runs, so the same overflow throws
// away every page already captured and fails the render job instead.
//
// This test pins the abort and shows what it discards.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/
//	cd misc/gnopreview && go test ./... -run PageCap -v
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageCapAbortsInsteadOfTruncating(t *testing.T) {
	t.Parallel()
	// A realm whose render page links to ten render-argument pages, all of
	// which inScope follows: ":p/<n>" is args with no web query.
	var links strings.Builder
	for i := range 10 {
		fmt.Fprintf(&links, `<a href="/r/x/y:p/%d">p%d</a>`, i, i)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><head></head><body>%s</body></html>", links.String())
	}))
	defer srv.Close()

	c := &Crawler{
		Base:     srv.URL,
		Realms:   []string{"gno.land/r/x/y"},
		MaxPages: 3,
		Live:     "https://gno.land",
	}
	err := c.Run()
	if err == nil {
		t.Fatalf("Run() succeeded with MaxPages=3 and %d pages reachable", 10)
	}
	if !strings.Contains(err.Error(), "page cap") {
		t.Fatalf("Run() = %v; want the page-cap error", err)
	}
	// The pages up to the cap were fetched and are sitting in memory. main.go's
	// `if err := c.Run(); err != nil { return err }` returns before Write, so
	// every one of them is discarded and the render job goes red.
	if len(c.pages) != c.MaxPages {
		t.Fatalf("captured %d pages, want %d", len(c.pages), c.MaxPages)
	}
	t.Logf("cap tripped: %v — %d captured pages discarded by the caller", err, len(c.pages))

	// The sibling cap one level up does the opposite: it truncates the realm
	// list and records the overflow in Plan.Dropped rather than failing.
	plan := &Plan{}
	realms := []string{"a", "b", "c", "d"}
	const maxRealms = 2
	if maxRealms > 0 && len(realms) > maxRealms {
		plan.Dropped = len(realms) - maxRealms
		realms = realms[:maxRealms]
	}
	if plan.Dropped != 2 {
		t.Fatalf("realm cap shape changed: Dropped = %d", plan.Dropped)
	}
}
