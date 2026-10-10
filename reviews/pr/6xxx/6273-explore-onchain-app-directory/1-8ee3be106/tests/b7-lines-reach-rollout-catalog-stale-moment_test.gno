// Repro: the store's activity carries no height, so the seed's "listed"
// events stay the pulse moment for good (seed.gno:35, view.go:454).
// Realm half: api/v1/home serves the same seed events a year later.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp <this file> examples/gno.land/r/gnoland/store/v0/zz_b7_moment_test.gno
//   cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/gnoland/store/v0 -run TestB7StaleSeedMoment -v
//
// Expected at 8ee3be106: PASS, printing the same activity at seed and a year
// later: [{"kind":"listed","slug":"valopers","stars":0},...].
// gnoweb half: b7-lines-reach-rollout-catalog-stale-moment_test.go phrases
// it "Valopers just joined" beside Block 6307200.
package store

import (
	"strings"
	"testing"
)

// The seed's "listed" events stay the store's activity until another event
// replaces them: a year after the seed, api/v1/home still serves them, with
// no height, and gnoweb phrases the first as "<title> just joined".
func TestB7StaleSeedMoment(t *testing.T) {
	start := Render("api/v1/home")
	testing.SetHeight(365 * blocksPerDay)
	later := Render("api/v1/home")
	_, a0, _ := strings.Cut(start, `"activity":`)
	a0, _, _ = strings.Cut(a0, `,"listings"`)
	_, a1, _ := strings.Cut(later, `"activity":`)
	a1, _, _ = strings.Cut(a1, `,"listings"`)
	println("at seed:   ", a0)
	println("year later:", a1)
	if a0 != a1 || !strings.HasPrefix(a1, `[{"kind":"listed","slug":"valopers"`) {
		t.Errorf("activity changed or not the seed's: %s", a1)
	}
}
