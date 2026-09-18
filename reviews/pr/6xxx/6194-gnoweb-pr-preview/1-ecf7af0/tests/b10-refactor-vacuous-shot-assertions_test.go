// b10-refactor-vacuous-shot-assertions_test.go
//
// What it shows: the two negative assertions plan_test.go makes about
// screenshots — "realm-only comment should carry no screenshots" (line 219)
// and "realm change should not carry the gnoweb sample" (line 266) — cannot
// fail, because both fixtures leave Plan.Shots empty and shotGrid returns ""
// for an empty slice. This file re-states the same two assertions over a Plan
// that actually carries a Shot, so the guard they claim to protect is exercised.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head:pr6194 && git checkout pr6194   # ecf7af0
//	cp <this file> misc/gnopreview/
//	cd misc/gnopreview
//	go test ./... -run 'Vacuous|TestComment'          # green: head is correct
//
//	# now delete the `if p.Gnoweb {` guard around
//	#   b.WriteString(shotGrid(p.Shots, base))
//	# in comment.go's default branch, so shots render for a realm-only plan:
//	go test ./... -run TestComment                    # STILL GREEN: the PR's
//	                                                  # own assertions pin nothing
//	go test ./... -run Vacuous                        # RED: this file catches it
//
// Fix: add `Shots: []Shot{{File: "_shots/home.png", Label: "Home — rendered markdown"}}`
// to the fixtures of TestCommentRealms and TestCommentBeforeAfter; both tests
// then stay green on head and turn red on the mutation above.

package main

import (
	"strings"
	"testing"
)

// gnowebSampleLabel is the caption shots.go gives the sample page that a
// realm-only comment must never show.
const gnowebSampleLabel = "Home — rendered markdown"

func TestVacuousRealmOnlyCommentDropsShots(t *testing.T) {
	t.Parallel()
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		ChangedPkgs:   []string{"gno.land/p/x/base/v0"},
		Realms:        []string{"gno.land/r/x/leaf", "gno.land/r/x/other"},
		Dropped:       3,
		Shots:         []Shot{{File: "_shots/home.png", Label: gnowebSampleLabel}},
	}
	got := Comment(p, "https://example.test/pr-7/", "7")
	if got == "" {
		t.Fatal("Comment returned empty for a non-empty plan")
	}
	if strings.Contains(got, "<table>") {
		t.Errorf("realm-only comment carries a screenshot grid:\n%s", got)
	}
	if strings.Contains(got, gnowebSampleLabel) {
		t.Errorf("realm-only comment carries the gnoweb sample:\n%s", got)
	}
}

func TestVacuousBeforeAfterDropsGnowebSample(t *testing.T) {
	t.Parallel()
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/x/leaf"},
		Pairs: []ShotPair{
			{Realm: "gno.land/r/x/leaf", Before: "_shots/b.png", After: "_shots/a.png", URL: "r/x/leaf/"},
		},
		Shots: []Shot{{File: "_shots/home.png", Label: gnowebSampleLabel}},
	}
	got := Comment(p, "https://example.test/pr-5", "5")
	if !strings.Contains(got, "_shots/b.png") {
		t.Fatalf("before/after pair missing:\n%s", got)
	}
	if strings.Contains(got, gnowebSampleLabel) {
		t.Errorf("before/after comment carries the gnoweb sample too:\n%s", got)
	}
}
