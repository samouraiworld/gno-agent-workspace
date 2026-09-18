// Repro for PR 6194, misc/gnopreview/shots.go: chromeShot() runs the browser
// with no deadline, while every other subprocess the tool starts is bounded by
// -timeout. A browser that never exits blocks the render step forever, and
// .github/workflows/pr-preview.yml sets no timeout-minutes, so the job runs to
// GitHub's 360-minute default.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_catalog_test.go
//	cd misc/gnopreview && go test -run 'TestChromeShotHasNoDeadline|TestSlugStaysInsideShotsDir' -v ./...
//
// Expected on a fixed tree: TestChromeShotHasNoDeadline fails (chromeShot
// returns a deadline error). On ecf7af0f2 it passes: nothing bounds the child.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChromeShotHasNoDeadline(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "hung-chrome")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- chromeShot(stub, "http://127.0.0.1:1/never", filepath.Join(dir, "out.png"))
	}()
	select {
	case err := <-done:
		t.Fatalf("chromeShot returned on a hung browser: %v (a deadline now bounds it)", err)
	case <-time.After(6 * time.Second):
		t.Log("chromeShot still blocked after 6s on a browser that never exits: no deadline bounds exec.Command")
	}
}

func TestSlugStaysInsideShotsDir(t *testing.T) {
	// ScreenshotPairs builds filepath.Join(outDir, shotsDir, slug(realmURL)+"-after.png").
	for _, s := range []string{"..", "../..", "r/../../../etc/passwd", "r/gnoland/home", "/"} {
		got := slug(s)
		t.Logf("slug(%q) = %q", s, got)
		if strings.ContainsAny(got, `/\`) || got == ".." || strings.HasPrefix(got, "../") {
			t.Errorf("slug(%q) = %q escapes %s/", s, got, shotsDir)
		}
	}
}
