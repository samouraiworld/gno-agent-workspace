// b4-refactor-r2 — two redundancies in misc/gnopreview/shots.go at ecf7af0.
//
// Repro from a plain clone (no network beyond the clone; needs Chrome on PATH
// for the second test, which skips without one):
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_b4_refactor_test.go
//	cd misc/gnopreview && go test -run 'TestPairURLIsUrlOf|TestChromeShotRelativeDst' -v ./...
//
// Both pass at head, unmodified. That is the point: each pins a value the code
// computes a second way, so removing the second way changes nothing.
//
// 1. TestPairURLIsUrlOf — ShotPair.URL (shots.go:38, set at shots.go:77 as
//    path.Dir(head.FileOf(urlOf(r)))+"/") is always urlOf(Realm) trimmed of its
//    leading slash plus "/", which is what comment.go's link() and tabs()
//    already build for the same realm. plan_test.go's own fixtures spell it
//    that way by hand: URL: "r/x/leaf/" for gno.land/r/x/leaf.
//    Measured: dropping the field and calling urlOf(p.Realm) at pairGrid's two
//    sites takes shots.go from 213 to 212 lines, comment.go unchanged, and
//    `go test ./...` stays green with these two added href assertions in
//    TestCommentBeforeAfter:
//        `<a href="https://example.test/pr-5/r/x/leaf/">`
//        `<a href="https://example.test/pr-5/r/x/fresh/">`
//
// 2. TestChromeShotRelativeDst — chromeShot's filepath.Abs(dst) block
//    (shots.go:167-170, 4 lines) is a no-op. Nothing in the package calls
//    os.Chdir or sets cmd.Dir, so Chrome inherits the Go process's cwd and
//    resolves a relative --screenshot= against it. The test passes byte-for-
//    byte identically with the block present and with it deleted (5654 bytes
//    both ways, measured), and the caller's dst is relative in the default
//    run: cfg.out defaults to "_preview" (main.go:67) and is never absolutized.
package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestPairURLIsUrlOf(t *testing.T) {
	for _, pkg := range []string{
		"gno.land/r/gnoland/home",
		"gno.land/r/docs/security_patterns",
		"gno.land/r/demo/boards2/v0",
		"gno.land/p/nt/ownable",
		"gno.land/r/x/y_z.v1",
		"singleword",
	} {
		// what ScreenshotPairs stores in ShotPair.URL for the render page
		got := path.Dir(urlToFile(urlOf(pkg))) + "/"
		// what comment.go already builds for the same realm
		want := strings.TrimPrefix(urlOf(pkg), "/") + "/"
		if got != want {
			t.Errorf("%s: pair.URL=%q urlOf-derived=%q", pkg, got, want)
		}
	}
}

func TestChromeShotRelativeDst(t *testing.T) {
	bin := findChrome("")
	if bin == "" {
		t.Skip("no chrome")
	}
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd) //nolint:errcheck
	if err := chromeShot(bin, "data:text/html,<h1>hi</h1>", "zz-rel.png"); err != nil {
		t.Fatalf("relative dst: %v", err)
	}
	fi, err := os.Stat(filepath.Join(dir, "zz-rel.png"))
	if err != nil {
		t.Fatalf("relative dst did not land in cwd: %v", err)
	}
	t.Logf("relative dst landed at %s, %d bytes", filepath.Join(dir, "zz-rel.png"), fi.Size())
}
