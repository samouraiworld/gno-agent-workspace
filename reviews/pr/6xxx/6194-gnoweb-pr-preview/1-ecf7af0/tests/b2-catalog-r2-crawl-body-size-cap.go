// b2-catalog-r2-crawl-body-size-cap.go — invariant catalog, allocation-bound class.
//
// Claim under test: misc/gnopreview/crawl.go reads every gnoweb response with a
// bare io.ReadAll (crawl.go:444) and keeps every captured body in c.pages until
// writePages runs at the end of the job (crawl.go:326). The only bound the tool
// declares is a page COUNT (-max-pages 400, README "Bounds"), so peak memory of
// the render job is page count x page size, and page size is whatever the
// crawled realms render -- content the pull request under preview supplies.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_bodycap_test.go
//	cd misc/gnopreview && go test -run 'TestCrawlHasNoBodySizeCap' -v ./...
//	rm misc/gnopreview/zz_bodycap_test.go
//
// Expected if a byte cap existed: the retained total stops at the cap.
// Observed: the retained total is realms x served bytes, exactly.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestCrawlHasNoBodySizeCap(t *testing.T) {
	const pageBytes = 8 << 20 // 8 MiB of realm-rendered HTML per page
	const realms = 4

	filler := strings.Repeat("x", pageBytes)
	served := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><head></head><body>%s</body></html>", filler)
	}))
	defer srv.Close()

	var rs []string
	for i := range realms {
		rs = append(rs, fmt.Sprintf("gno.land/r/x/y%d", i))
	}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	c := &Crawler{
		Base:       srv.URL,
		Realms:     rs,
		MaxPages:   400, // the only cap the tool declares
		RenderOnly: true,
		Live:       "https://gno.land",
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	retained := 0
	for _, u := range c.order {
		retained += len(c.pages[u].Body)
	}
	runtime.ReadMemStats(&after)

	t.Logf("served responses=%d captured pages=%d retained body bytes=%d heap delta=%d",
		served, len(c.pages), retained, int64(after.HeapAlloc)-int64(before.HeapAlloc))

	if want := realms * (pageBytes + len("<html><head></head><body></body></html>")); retained != want {
		t.Fatalf("retained %d bytes, want %d: a byte cap truncated something", retained, want)
	}
	if len(c.pages) != realms {
		t.Fatalf("captured %d pages, want %d", len(c.pages), realms)
	}
	t.Logf("no byte cap: %d pages x %d MiB were read and retained whole under -max-pages 400",
		len(c.pages), pageBytes>>20)
}
