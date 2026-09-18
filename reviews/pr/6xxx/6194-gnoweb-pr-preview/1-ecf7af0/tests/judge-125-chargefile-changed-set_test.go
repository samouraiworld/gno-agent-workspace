// Judge artifact for candidate #125 (misc/gnopreview/crawl_test.go:196).
//
// TestWantFile never names wantFile or chargeFile's changed-set exemption
// directly. wantFile's own branch is in fact pinned through inScope, but
// chargeFile's exemption — the line that keeps a changed file's $source page
// on a realms-only PR, where fileBudget(plan) is 0 — is asserted by no test.
// This file pins both: it calls wantFile directly and charges more changed
// files than any budget would allow.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_judge125_test.go
//	cd misc/gnopreview && go test -vet=off -count=1 .   # PASS at head
//
// The mutation the shipped suite does not catch — drop chargeFile's exemption:
//
//	perl -0pi -e 's{\t\tif _, listed := c\.ChangedFiles\[r\]; listed \{\n\t\t\treturn true[^\n]*\n\t\t\}\n}{}' misc/gnopreview/crawl.go
//	cd misc/gnopreview && go test -vet=off -count=1 .
//	  without this file: ok      with this file: FAIL TestChangedFilesEscapeTheBudget
//	git checkout -- . && rm -f misc/gnopreview/zz_judge125_test.go

package main

import "testing"

// wantFile is the budget question; no test calls it by name today.
func TestWantFileDirect(t *testing.T) {
	t.Parallel()
	c := &Crawler{
		Realms:       []string{"gno.land/r/x/touched", "gno.land/r/x/untouched"},
		ChangedFiles: map[string][]string{"gno.land/r/x/touched": {"a.gno", "b.gno", "c.gno"}},
		FileBudget:   0, // fileBudget(plan) for a realms-only PR (main.go:216)
	}
	for _, tc := range []struct {
		realm, name string
		want        bool
	}{
		{"gno.land/r/x/touched", "a.gno", true},      // listed: the changed set wins
		{"gno.land/r/x/touched", "other.gno", false}, // listed realm, unlisted file
		{"gno.land/r/x/untouched", "a.gno", false},   // unlisted realm, zero budget
	} {
		if got := c.wantFile(tc.realm, tc.name); got != tc.want {
			t.Errorf("wantFile(%q, %q) = %v; want %v", tc.realm, tc.name, got, tc.want)
		}
	}
}

// chargeFile must never spend a budget slot on a realm gated by the changed
// set: on a realms-only PR FileBudget is 0, so charging those pages would drop
// every source page the PR actually touched.
func TestChangedFilesEscapeTheBudget(t *testing.T) {
	t.Parallel()
	c := &Crawler{
		Realms:       []string{"gno.land/r/x/touched"},
		ChangedFiles: map[string][]string{"gno.land/r/x/touched": {"a.gno", "b.gno", "c.gno"}},
		FileBudget:   0,
	}
	for _, f := range []string{"a.gno", "b.gno", "c.gno"} {
		u := "/r/x/touched$source&file=" + f
		if !c.chargeFile(u) {
			t.Errorf("chargeFile(%q) refused a changed file under a zero budget", u)
		}
	}
}
