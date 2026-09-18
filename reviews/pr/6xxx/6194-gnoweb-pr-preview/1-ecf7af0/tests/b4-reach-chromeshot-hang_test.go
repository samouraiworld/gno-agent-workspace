package main

// Finder b4-reach, PR 6194, head ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//   cp <this file> misc/gnopreview/
//   cd misc/gnopreview && go test -run 'TestChromeShot' -v -timeout 5m ./...
//
// Needs a chrome/chromium on PATH (ubuntu-latest, where pr-preview.yml runs,
// ships google-chrome-stable). TestChromeShotPlainPage is the control and
// passes; TestChromeShotStalledSubresource fails at this head.
//
// What it shows: chromeShot (misc/gnopreview/shots.go:166) runs Chrome with no
// deadline. --virtual-time-budget=4000 does not bound the process: one
// subresource whose server accepts the connection and never answers holds
// Chrome open past 60s (raw chromium probe: still alive at 90s, no PNG
// written). The screenshotted page is realm HTML from the pull request, and
// gnoweb's image policy (gno.land/pkg/gnoweb/markdown/ext_imgvalidator.go,
// AllowSvgDataImage) rejects only non-svg data: URIs, so `![](http://host/x.png)`
// in a fork's realm markdown reaches the snapshot verbatim (Crawler.mapURL
// leaves external URLs alone). pr-preview.yml sets no timeout-minutes, so the
// job runs to the 360-minute default.

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// stallOrigin accepts and answers nothing until the client goes away.
func stallOrigin(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { //nolint:errcheck
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Minute):
		}
	}))
	return ln.Addr().String()
}

func snapshotDir(t *testing.T, body string) (dir, origin string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, origin, err := serve(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.Close() })
	return dir, origin
}

func runShot(t *testing.T, origin, dir string, budget time.Duration) (time.Duration, error, bool) {
	t.Helper()
	bin := findChrome("")
	if bin == "" {
		t.Skip("no chrome/chromium on PATH")
	}
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- chromeShot(bin, origin+"/index.html", filepath.Join(dir, "shot.png")) }()
	select {
	case err := <-done:
		return time.Since(start), err, true
	case <-time.After(budget):
		exec.Command("pkill", "-f", "--virtual-time-budget=4000").Run() //nolint:errcheck
		return budget, nil, false
	}
}

func TestChromeShotPlainPage(t *testing.T) {
	dir, origin := snapshotDir(t, `<html><body><h1>realm</h1></body></html>`)
	d, err, returned := runShot(t, origin, dir, 60*time.Second)
	if !returned {
		t.Fatalf("control: chromeShot did not return within %s", d)
	}
	if err != nil {
		t.Fatalf("control: %v", err)
	}
	t.Logf("control returned in %s", d)
}

func TestChromeShotStalledSubresource(t *testing.T) {
	stall := stallOrigin(t)
	dir, origin := snapshotDir(t, fmt.Sprintf(
		`<html><body><h1>realm</h1><img src="http://%s/x.png"></body></html>`, stall))
	d, err, returned := runShot(t, origin, dir, 60*time.Second)
	if !returned {
		t.Fatalf("chromeShot still running after %s: --virtual-time-budget=4000 (4s) does not bound Chrome, "+
			"and exec.Command carries no deadline, so one stalled subresource in a PR's realm page "+
			"pins the preview job until the runner's 360-minute default", d)
	}
	t.Logf("returned in %s: %v — finding refuted", d, err)
}
