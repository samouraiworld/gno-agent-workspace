// b2-lines-r2-help-origin_test.go — misc/gnopreview/crawl.go:40 (attrRe)
//
// Claim: the snapshot rewrite covers only href and src, so gnoweb's $help view
// ships the crawl-time gnodev origin into the published preview.
//   - components/views/action.html:104  <form ... action="{{ buildHelpURL $data . }}">
//   - components/views/action.html:84   data-copy-text-value="{{ buildHelpURL $data . }}"
//   - components/view_action.go:73      buildHelpURL = data.Origin + pkgPath + "$help&func=" + fn.Name
//   - handler_http.go:390               gnourl.Origin = requestOrigin(r)  (scheme+host of the request)
// crawl.go:30 seeds u+"$help" for every realm, so every previewed realm has one.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//   git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//   cp <this file> misc/gnopreview/zz_b2_lines_r2_test.go
//   cd misc/gnopreview && go test -run TestSnapshotKeepsCrawlTimeOrigin -v ./...
//
// Observed at ecf7af0f2 (go1.25.9): FAIL — the written page still carries
//   action="http://127.0.0.1:43095/r/x/y$help&amp;func=Foo"
//   data-copy-text-value="http://127.0.0.1:43095/r/x/y$help&func=Foo"
// while the sibling href on the same page became ../../../../../r/x/y/_t/source/.


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

// gnoweb's action.html ($help view) builds two attributes from buildHelpURL,
// which is components/view_action.go:73 -> data.Origin + pkgPath + "$help&func=".
// handler_http.go:390 sets Origin = requestOrigin(r), i.e. the scheme+host the
// crawler itself dialled. Neither attribute is href or src.
const helpBody = `<html><head><title>x</title></head><body>
<a href="/r/x/y$source">source</a>
<button data-controller="copy" data-copy-text-value="%[1]s/r/x/y$help&func=Foo">anchor</button>
<form class="params" method="GET" action="%[1]s/r/x/y$help&amp;func=Foo"></form>
</body></html>`

func TestSnapshotKeepsCrawlTimeOrigin(t *testing.T) {
	var origin string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/r/x/y$help":
			fmt.Fprintf(w, helpBody, origin)
		default:
			fmt.Fprintf(w, `<html><head></head><body><a href="/r/x/y$help">help</a></body></html>`)
		}
	}))
	defer srv.Close()
	origin = srv.URL

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := t.TempDir()
	if err := c.Write(out, ""); err != nil {
		t.Fatalf("Write: %v", err)
	}

	f, ok := c.FileOf("/r/x/y$help")
	if !ok {
		t.Fatal("the $help seed was not captured")
	}
	b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(f)))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	host := strings.TrimPrefix(srv.URL, "http://")
	t.Logf("snapshot file %s:\n%s", f, got)
	if strings.Contains(got, host) {
		t.Errorf("published snapshot still points at the crawl-time gnodev %s:\n%s", host, got)
	}
}
