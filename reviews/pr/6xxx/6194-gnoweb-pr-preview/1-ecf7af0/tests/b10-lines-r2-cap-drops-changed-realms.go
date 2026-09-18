// b10-lines-r2-cap-drops-changed-realms: the realm cap truncates DIRECTLY
// changed realms too, while Comment tells the reader "the changed realms are
// always kept" and links to every one of them -- including the ones that were
// never rendered.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/zz_cap_test.go
//	cd misc/gnopreview && go test ./... -run 'TestCapDropsChangedRealms' -v
//
// Expected if the claim held: every realm in plan.ChangedRealms is also in
// plan.Realms (i.e. rendered), or the comment says which changed realms were
// dropped.  Observed: with 2 changed realms and -max-realms 1, ChangedRealms
// has both, Realms has one, and the comment links to the dropped one under
// "Changed realms (2)" while the footer claims changed realms are always kept.
package main

import (
	"strings"
	"testing"
)

func TestCapDropsChangedRealms(t *testing.T) {
	root := fakeRepo(t)
	changed := []string{
		"examples/gno.land/r/x/leaf/lib.gno",
		"examples/gno.land/r/x/other/lib.gno",
	}
	p, err := BuildPlan(root, changed, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dirs=%v Dropped=%d", p.ChangedRealms, p.Realms, p.Dirs, p.Dropped)

	for _, r := range p.ChangedRealms {
		if !contains(p.Realms, r) {
			t.Errorf("changed realm %q was dropped by the cap but is not rendered", r)
		}
	}

	got := Comment(p, "https://example.test/pr-1", "1")
	t.Logf("---comment---\n%s", got)
	if strings.Contains(got, "the changed realms are always kept") {
		t.Errorf("comment claims changed realms are always kept while %v were dropped", p.Dropped)
	}
	for _, r := range p.ChangedRealms {
		if contains(p.Realms, r) {
			continue
		}
		if strings.Contains(got, strings.TrimPrefix(r, "gno.land")+"/") {
			t.Errorf("comment links to %q, a realm gnodev was never asked to load", r)
		}
	}
}
