// Two claims PR 6194 writes about itself, checked against misc/gnopreview.
//
// 1. "Scope and bounds" in the PR body: "At most 25 realms and 400 pages, and
//    the comment says how many realms were dropped." The realm cap truncates
//    and reports Plan.Dropped; the page cap does not truncate at all --
//    Crawler.Run returns an error (crawl.go:126), render() propagates it
//    (main.go:143), the render step has no continue-on-error and the upload
//    step does not run, so the PR gets a red check and no preview.
// 2. main.go:140 indexes plan.Realms[0] after the plan.Empty() guard, and
//    Empty() is `!Gnoweb && len(Realms) == 0` (plan.go:91): a gnoweb-only
//    change whose four seed realms are absent from the tree reaches that index
//    with an empty slice.
//
// Repro from a plain clone (go1.25.9, no cgo, ~1 s):
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/zz_b3claims_test.go
//	cd misc/gnopreview && go test -run TestB3 -v .
//	rm zz_b3claims_test.go
//
// Observed at ecf7af0f2:
//
//	zz_b3claims_test.go:51: Run() with MaxPages=3 -> error "page cap 3 reached
//	    (queue still had 31)", 3 pages captured and discarded by the caller
//	zz_b3claims_test.go:58: Run() with MaxPages=0 -> nil, 19 pages
//	zz_b3claims_test.go:84: Gnoweb=true Realms=[] Empty()=false Mode()="gnoweb"
//	zz_b3claims_test.go:94: plan.Realms[0] panics: runtime error: index out of
//	    range [0] with length 0
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serveRealms answers every /r/... path with a page linking to the next realm,
// so a crawl started on realm 0 reaches all of them.
func serveRealms(t *testing.T, n int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString("<html><head></head><body>")
		for i := range n {
			fmt.Fprintf(&b, `<a href="/r/x/r%d">r%d</a>`, i, i)
		}
		b.WriteString("</body></html>")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(b.String()))
	})
	return httptest.NewServer(mux)
}

// TestB3PageCapIsFatalNotATruncation: the PR body's "Scope and bounds" says
// "At most 25 realms and 400 pages". The realm cap truncates and reports
// Plan.Dropped; the page cap does not truncate at all -- Crawler.Run returns an
// error (crawl.go:126), which render() propagates (main.go:143), so the whole
// preview is lost instead of being capped.
func TestB3PageCapIsFatalNotATruncation(t *testing.T) {
	const n = 6
	srv := serveRealms(t, n)
	defer srv.Close()

	realms := make([]string, n)
	for i := range n {
		realms[i] = fmt.Sprintf("gno.land/r/x/r%d", i)
	}

	capped := &Crawler{Base: srv.URL, Realms: realms, MaxPages: 3, Live: "https://gno.land"}
	err := capped.Run()
	if err == nil {
		t.Fatalf("MaxPages=3 over %d realms: want an error, got nil (%d pages kept)", n, len(capped.pages))
	}
	t.Logf("Run() with MaxPages=3 -> error %q, %d pages captured and discarded by the caller",
		err, len(capped.pages))

	uncapped := &Crawler{Base: srv.URL, Realms: realms, MaxPages: 0, Live: "https://gno.land"}
	if err := uncapped.Run(); err != nil {
		t.Fatalf("MaxPages=0 (documented as no cap): %v", err)
	}
	t.Logf("Run() with MaxPages=0 -> nil, %d pages", len(uncapped.pages))
}

// TestB3GnowebPlanWithNoSeedRealmsIsNotEmpty: Plan.Empty() is
// `!Gnoweb && len(Realms) == 0` (plan.go:91), so a gnoweb-only change whose
// seed realms are absent from the tree passes the Empty() guard in render()
// with an empty Realms slice -- and render() then indexes plan.Realms[0]
// unconditionally (main.go:140).
func TestB3GnowebPlanWithNoSeedRealmsIsNotEmpty(t *testing.T) {
	root := t.TempDir()
	// a tree with examples/gno.land but none of the four gnowebSeedRealms
	dir := filepath.Join(root, "examples", "gno.land", "r", "other", "thing")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gnomod.toml"), []byte("module = \"gno.land/r/other/thing\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "t.gno"), []byte("package thing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/app.go"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Gnoweb=%v Realms=%v Empty()=%v Mode()=%q", plan.Gnoweb, plan.Realms, plan.Empty(), plan.Mode())
	if plan.Empty() {
		t.Fatalf("plan is Empty(); the render path is unreachable in this shape")
	}
	if len(plan.Realms) != 0 {
		t.Fatalf("want no realms, got %v", plan.Realms)
	}
	// This is exactly what main.go:140 does with that plan.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("plan.Realms[0] panics: %v", r)
			return
		}
		t.Fatalf("no panic")
	}()
	_ = urlOf(plan.Realms[0])
}
