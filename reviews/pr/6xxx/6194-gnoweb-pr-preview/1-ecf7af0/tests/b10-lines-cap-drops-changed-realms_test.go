// Repro for gnolang/gno#6194: when the realm cap is reached, BuildPlan drops
// changed realms, yet Comment still links every entry of ChangedRealms and
// tells the reader "the changed realms are always kept". The links point at
// pages the crawler never rendered, so they 404 inside the static preview.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/b10_cap_test.go
//	cd misc/gnopreview && go test -run TestB10CapDropsChangedRealms -v ./...
//
// Expected on a fixed tree: PASS. Observed at ecf7af0: FAIL, the comment links
// gno.land/r/x/lonely and gno.land/r/x/other, neither of which is in Realms.
package main

import (
	"strings"
	"testing"
)

func TestB10CapDropsChangedRealms(t *testing.T) {
	root := fakeRepo(t) // from plan_test.go: r/x/leaf, r/x/other, r/x/lonely

	changed := []string{
		"examples/gno.land/r/x/leaf/lib.gno",
		"examples/gno.land/r/x/other/lib.gno",
		"examples/gno.land/r/x/lonely/lib.gno",
	}
	p, err := BuildPlan(root, changed, 1) // maxRealms=1, as --max-realms would set it
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dropped=%d", p.ChangedRealms, p.Realms, p.Dropped)

	got := Comment(p, "https://example.test/pr-1", "1")

	var dead []string
	for _, r := range p.ChangedRealms {
		if !contains(p.Realms, r) && strings.Contains(got, "https://example.test/pr-1"+urlOf(r)+"/") {
			dead = append(dead, r)
		}
	}
	if len(dead) > 0 {
		t.Errorf("comment links %d changed realm(s) that were never rendered: %v", len(dead), dead)
	}
	if strings.Contains(got, "the changed realms are always kept") && len(dead) > 0 {
		t.Errorf("comment claims the changed realms are always kept while %d were dropped", len(dead))
	}
	if len(dead) > 0 {
		t.Logf("comment:\n%s", got)
	}
}
