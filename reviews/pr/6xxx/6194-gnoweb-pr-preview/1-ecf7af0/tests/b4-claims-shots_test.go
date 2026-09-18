// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/b4_claims_shots_test.go
//	cd misc/gnopreview && go test -run 'TestChromeShot|TestMixedMode' -v ./...
//
// Two claims misc/gnopreview/shots.go writes about itself, each with the run
// that settles it.
//
//	1. chromeShot's comment says --virtual-time-budget=4000 lets the page
//	   "settle before the frame is grabbed". That flag bounds the page's
//	   *virtual* clock, not the browser process, and chromeShot gives
//	   exec.Command no context and no deadline. Neither .github/workflows/
//	   pr-preview.yml nor pr-preview-publish.yml carries timeout-minutes, so a
//	   browser that never exits holds the runner to GitHub's 6 h default.
//	   TestChromeShotHasNoWallClockBound FAILS on this head: the call is still
//	   blocked after 6 s against a binary that never exits.
//
//	2. shotPlan's godoc calls it "the fixed sample photographed when gnoweb
//	   itself changed", and comment.go:53 embeds shotGrid(p.Shots, base) in the
//	   "gnoweb + realm sources" branch. main.go:147-151 fills Plan.Shots only in
//	   the else-if arm (no changed realm), so in that branch p.Shots is always
//	   nil and the call renders nothing.
//	   TestMixedModeCommentHasNoGnowebSample PASSES on this head; it is the
//	   proof of the gap, and the sibling assertion shows the grid renders fine
//	   once Shots is non-empty, i.e. the branch is dead only for lack of a
//	   caller.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestChromeShotHasNoWallClockBound stands in a browser that starts, writes
// nothing and never exits. chromeShot must not outlive the caller's patience.
func TestChromeShotHasNoWallClockBound(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-chrome")
	// Ignores every flag chromeShot passes and never returns.
	script := "#!/bin/sh\nwhile true; do sleep 3600; done\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- chromeShot(fake, "http://127.0.0.1:1/x", filepath.Join(dir, "out.png")) }()

	const patience = 6 * time.Second
	select {
	case err := <-done:
		t.Logf("chromeShot returned within %s: %v", patience, err)
	case <-time.After(patience):
		t.Fatalf("chromeShot still blocked after %s: exec.Command carries no context "+
			"and no deadline, and the preview workflows set no timeout-minutes", patience)
	}
}

// TestMixedModeCommentHasNoGnowebSample pins that the gnoweb+realm branch of
// Comment emits no fixed sample, and that the grid itself works — so the gap is
// in who fills Plan.Shots, not in comment.go.
func TestMixedModeCommentHasNoGnowebSample(t *testing.T) {
	base := "https://example.test/pr-1"
	mixed := func(shots []Shot) string {
		p := &Plan{
			Gnoweb:        true,
			ChangedRealms: []string{"gno.land/r/x/y"},
			Realms:        []string{"gno.land/r/x/y"},
			Shots:         shots,
		}
		if got := p.Mode(); got == "gnoweb" {
			t.Fatalf("Mode() = %q, want the mixed (default) branch", got)
		}
		return Comment(p, base, "1")
	}

	// What main.go actually produces in this mode: Shots is never assigned.
	out := mixed(nil)
	if !strings.Contains(out, "changes **gnoweb** and realm sources") {
		t.Fatalf("not the mixed branch:\n%s", out)
	}
	if strings.Contains(out, "_shots/") {
		t.Fatalf("unexpected sample image in the mixed branch:\n%s", out)
	}

	// The same branch with Shots filled in renders the sample, so comment.go:53
	// is reachable only if some caller populates it — none does.
	out = mixed([]Shot{{File: "_shots/home.png", Label: "Home — rendered markdown"}})
	if !strings.Contains(out, "_shots/home.png") {
		t.Fatalf("shotGrid did not render with Shots set:\n%s", out)
	}
}
