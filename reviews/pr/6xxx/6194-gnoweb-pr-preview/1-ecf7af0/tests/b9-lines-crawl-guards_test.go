// Round 1 of gnolang/gno#6194, finder b9-lines. Two lines of misc/gnopreview/crawl.go
// that misc/gnopreview/crawl_test.go does not pin: one is unreachable, one carries
// the whole changed-file feature.
//
// Repro from a plain clone (go1.25.x; misc/gnopreview is its own module, so the
// test has to run from that directory):
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/b9_guards_test.go
//	cd misc/gnopreview && go test -vet=off -count=1 .          # ok, at head
//
// Finding 1 — the trailing-slash guard in inScope is unreachable, and
// crawl_test.go:95 {"/r/gnoland/home/", false} passes without it:
//
//	perl -0pi -e 's{\tif strings\.HasSuffix\(base, "/"\) \{\n\t\treturn false\n\t\}\n}{}' crawl.go
//	go test -vet=off -count=1 .     # => ok  github.com/gnolang/gno/misc/gnopreview
//	git checkout -- crawl.go
//
// Finding 2 — the ChangedFiles exemption in chargeFile is untested, and deleting
// it leaves the suite green while emptying the changed-file pages of every
// realms-only pull request (fileBudget(plan) is 0 when plan.Gnoweb is false):
//
//	perl -0pi -e 's{\t\tif _, listed := c\.ChangedFiles\[r\]; listed \{\n\t\t\treturn true[^\n]*\n\t\t\}\n}{}' crawl.go
//	go test -vet=off -count=1 .     # => ok  github.com/gnolang/gno/misc/gnopreview
//	git checkout -- crawl.go
//
// TestB9ChangedFilesSurviveAZeroBudget below is the case the suite is missing:
// it is green at head and red under the finding-2 mutation.
package main

import (
	"strings"
	"testing"
)

// TestB9TrailingSlashGuardIsRedundant shows why deleting the guard changes
// nothing: the realm match on the next line compares base to urlOf(r), and
// urlOf never yields a trailing slash, so a base that ends in "/" is already
// unmatchable. crawl_test.go:95 therefore pins the realm match, not the guard.
func TestB9TrailingSlashGuardIsRedundant(t *testing.T) {
	realms := []string{
		"gno.land/r/gnoland/home",
		"gno.land/r/demo/counter",
		"gno.land/r/gnoland/boards2/v0",
		"gno.land/p/nt/avl/v0",
		"gno.land", // degenerate: no slash at all
	}
	for _, r := range realms {
		if u := urlOf(r); strings.HasSuffix(u, "/") {
			t.Fatalf("urlOf(%q) = %q ends in a slash; the guard would be live", r, u)
		}
	}

	// The URL crawl_test.go uses, decomposed the way inScope decomposes it.
	base, _, _ := splitURL("/r/gnoland/home/")
	if base != "/r/gnoland/home/" {
		t.Fatalf("splitURL base = %q", base)
	}
	for _, r := range realms {
		if base == urlOf(r) {
			t.Fatalf("base %q matched realm %q after all", base, r)
		}
	}
	t.Logf("base %q matches no urlOf(realm); crawl.go's HasSuffix guard is dead", base)

	c := &Crawler{Realms: realms, FileBudget: GnowebFileBudget}
	if c.inScope("/r/gnoland/home/") {
		t.Error("inScope accepted the listing URL")
	}
}

// TestB9ChangedFilesSurviveAZeroBudget is the missing case. A realms-only pull
// request gets FileBudget 0 from fileBudget(plan), so every per-file page a
// listed realm asks for reaches chargeFile with a spent budget; only the
// ChangedFiles exemption keeps it. crawl_test.go exercises that exemption
// through inScope alone (TestWantFile) and never through chargeFile, so the
// line that keeps the feature alive is unpinned.
func TestB9ChangedFilesSurviveAZeroBudget(t *testing.T) {
	c := &Crawler{
		Realms:       []string{"gno.land/r/x/touched"},
		ChangedFiles: map[string][]string{"gno.land/r/x/touched": {"a.gno", "b.gno", "gnomod.toml"}},
		FileBudget:   0, // what fileBudget(plan) returns when plan.Gnoweb is false
	}
	for _, f := range []string{"a.gno", "b.gno", "gnomod.toml"} {
		u := "/r/x/touched$source&file=" + f
		if !c.inScope(u) {
			t.Errorf("inScope(%q) = false; a changed file must be crawled", u)
		}
		if !c.chargeFile(u) {
			t.Errorf("chargeFile(%q) = false; the changed file was dropped at capture", u)
		}
	}
	// The same realm's untouched files still go, because the realm is listed.
	if c.inScope("/r/x/touched$source&file=untouched.gno") {
		t.Error("an unchanged file of a listed realm was crawled")
	}
}
