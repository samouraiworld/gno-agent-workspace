// Round 1 of the review of gnolang/gno#6194, finder b4-reach-r2.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/zz_b4_reach_test.go
//	cd misc/gnopreview && go test -run 'TestShotPlan|TestBeforeTree' -v .
//
// Both tests pass at ecf7af0. They pin two couplings shots.go leaves unpinned:
// the shotPlan/gnowebSeedRealms agreement, and the fact that nothing the
// published preview exposes ever links the _before/ tree the render writes.
package main

import (
	"strings"
	"testing"
)

// TestShotPlanReachesCrawledPages pins the coupling between shots.go's shotPlan
// and plan.go's gnowebSeedRealms. Screenshot() looks a shotPlan URL up in the
// crawler's page map and skips it silently on a miss, so a seed realm renamed,
// dropped from gnowebSeedRealms, or removed from examples/ costs the gnoweb
// preview one of its four sample shots with no error anywhere.
func TestShotPlanReachesCrawledPages(t *testing.T) {
	c := &Crawler{Realms: gnowebSeedRealms}
	seeds := map[string]bool{}
	for _, s := range c.Seeds() {
		seeds[s] = true
	}
	for _, s := range shotPlan {
		key := canonicalURL(s.url)
		if !seeds[key] {
			t.Errorf("shotPlan %q -> key %q is not a seed of gnowebSeedRealms: this shot can never be taken", s.url, key)
		}
	}
}

// TestBeforeTreeUnreferenced records that the _before/ tree renderBase writes
// into the published snapshot is linked by neither the sticky comment nor the
// snapshot index: only the two capped PNGs under _shots/ are ever read, while
// every before page ships and counts against the publish step's size cap.
func TestBeforeTreeUnreferenced(t *testing.T) {
	p := &Plan{
		Realms:        []string{"gno.land/r/demo/boards"},
		ChangedRealms: []string{"gno.land/r/demo/boards"},
		Pairs: []ShotPair{{
			Realm:  "gno.land/r/demo/boards",
			Before: shotsDir + "/r-demo-boards-e78ddc59-before.png",
			After:  shotsDir + "/r-demo-boards-e78ddc59-after.png",
			URL:    "r/demo/boards/",
		}},
	}
	c := &Crawler{}
	c.pages = map[string]*page{"/r/demo/boards": {URL: "/r/demo/boards", File: "r/demo/boards/index.html"}}
	c.order = []string{"/r/demo/boards"}

	body := Comment(p, "https://example.com/pr-1", "1")
	if !strings.Contains(body, shotsDir+"/") {
		t.Fatal("comment does not embed the _shots images; the fixture is wrong")
	}
	if strings.Contains(body, beforeDir+"/") {
		t.Errorf("comment links %s/: the before tree is reachable after all", beforeDir)
	}
	if idx := Index(p, c); strings.Contains(idx, beforeDir+"/") {
		t.Errorf("index links %s/: the before tree is reachable after all", beforeDir)
	}
}
