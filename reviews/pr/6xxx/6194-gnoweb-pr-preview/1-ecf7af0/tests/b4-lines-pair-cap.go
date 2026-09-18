// Repro for the before/after cap in misc/gnopreview/shots.go:68.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f2
//	cp <this file> misc/gnopreview/shots_cap_test.go
//	cd misc/gnopreview && go test -run TestScreenshotPairsCap -v ./...
//
// ScreenshotPairs walks plan.ChangedRealms in the order plan.go builds it,
// which is sortedKeys(changedPkgs) -> plain alphabetical order, and stops at
// the first maxPairs (=2) realms it can photograph. A realm added by the pull
// request can never have a "before" half, yet it consumes a slot exactly like
// a modified one, so two alphabetically-early additions push the only realm
// with a real before/after out of the comment entirely.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeChrome stands in for the browser: it writes a non-empty file wherever
// --screenshot= points, which is all chromeShot checks.
func fakeChrome(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fakechrome")
	script := "#!/bin/sh\n" +
		"for a in \"$@\"; do case \"$a\" in --screenshot=*) out=\"${a#--screenshot=}\" ;; esac; done\n" +
		"printf 'PNG' > \"$out\"\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScreenshotPairsCapSpentOnAddedRealms(t *testing.T) {
	out := t.TempDir()

	// The head renders all three changed realms.
	head := &Crawler{pages: map[string]*page{
		"/r/a/added":    {URL: "/r/a/added", File: "r/a/added/index.html"},
		"/r/b/added":    {URL: "/r/b/added", File: "r/b/added/index.html"},
		"/r/z/modified": {URL: "/r/z/modified", File: "r/z/modified/index.html"},
	}}
	// The merge base only has the realm this PR modified; renderBase excludes
	// the two additions, which is why newRealms marks them.
	base := &Crawler{pages: map[string]*page{
		"/r/z/modified": {URL: "/r/z/modified", File: "_before/r/z/modified/index.html"},
	}}
	for _, p := range head.pages {
		mustWrite(t, filepath.Join(out, p.File))
	}
	for _, p := range base.pages {
		mustWrite(t, filepath.Join(out, p.File))
	}

	realms := []string{"gno.land/r/a/added", "gno.land/r/b/added", "gno.land/r/z/modified"}
	newRealms := map[string]bool{"gno.land/r/a/added": true, "gno.land/r/b/added": true}

	pairs := ScreenshotPairs(out, head, base, realms, newRealms, fakeChrome(t))

	if len(pairs) != maxPairs {
		t.Fatalf("pairs = %d, want %d", len(pairs), maxPairs)
	}
	for _, p := range pairs {
		t.Logf("pair realm=%q before=%q after=%q new=%v", p.Realm, p.Before, p.After, p.New)
	}
	var sawModified bool
	for _, p := range pairs {
		if p.Realm == "gno.land/r/z/modified" {
			sawModified = true
		}
		if p.Before != "" {
			t.Errorf("unexpected before half for %s", p.Realm)
		}
	}
	if sawModified {
		t.Fatal("the modified realm made it into the pairs; the cap ordering was fixed")
	}
	t.Log("the only realm with a merge-base rendering got no before/after: " +
		"both slots went to realms added by the PR, which carry a single image and " +
		`the note "New in this PR - nothing to compare against".`)

	// And no before/after image was ever produced for the modified realm.
	if _, err := os.Stat(filepath.Join(out, shotsDir)); err == nil {
		ents, _ := os.ReadDir(filepath.Join(out, shotsDir))
		for _, e := range ents {
			t.Logf("shot written: %s", e.Name())
		}
	}
}

func mustWrite(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("<html><head></head><body>x</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
}
