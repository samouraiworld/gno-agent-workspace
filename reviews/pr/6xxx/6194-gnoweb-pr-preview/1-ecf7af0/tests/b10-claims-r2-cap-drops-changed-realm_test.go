// Repro: the realm cap drops directly changed realms, and the comment then
// links pages the snapshot never rendered while telling the reviewer the
// opposite.
//
// Three places claim the cap cannot touch a directly changed realm:
//   - misc/gnopreview/plan.go:251  "the cap never drops the realm the PR is actually about"
//   - misc/gnopreview/comment.go:86 prints "— the changed realms are always kept" to the PR
//   - misc/gnopreview/README.md:29 "Directly changed realms are always kept"
//
// All three hold only while len(plan.ChangedRealms) <= maxRealms. The sort at
// plan.go:253 orders changed realms first, but the truncation at plan.go:256
// cuts the list whatever it holds, and plan.ChangedRealms is never truncated —
// so Comment lists every changed realm with `source` and `help` links, of which
// the ones past the cap 404 on the published snapshot.
//
// Run from a plain clone (head ecf7af0f29abe4737a52803d672bc5a33c17cc60):
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/zz_cap_repro_test.go
//	cd misc/gnopreview && go test ./... -run TestCapDropsAChangedRealm -v
//
// Observed at that head (go1.25.9), the test FAILS with:
//
//	ChangedRealms = [gno.land/r/x/leaf gno.land/r/x/other]
//	Realms (rendered) = [gno.land/r/x/leaf]
//	Dropped = 1
//	changed realm gno.land/r/x/other   listed in comment=true rendered=false
//	comment links gno.land/r/x/other, which the snapshot never rendered
//	comment asserts the changed realms are always kept while 1 was dropped
//
// It uses fakeRepo from plan_test.go, so it must live in misc/gnopreview.
package main

import (
	"strings"
	"testing"
)

func TestCapDropsAChangedRealm(t *testing.T) {
	root := fakeRepo(t)
	changed := []string{
		"examples/gno.land/r/x/leaf/lib.gno",
		"examples/gno.land/r/x/other/lib.gno",
	}
	// Two directly changed realms, a cap of one. In production the same shape is
	// 26 changed realms against the default -max-realms=25: a sweep over
	// examples/ (a gnomod.toml field migration, a formatting pass) reaches it,
	// 58 realm modules being in the tree at this head.
	p, err := BuildPlan(root, changed, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms = %v", p.ChangedRealms)
	t.Logf("Realms (rendered) = %v", p.Realms)
	t.Logf("Dirs = %v", p.Dirs)
	t.Logf("Dropped = %d", p.Dropped)

	got := Comment(p, "https://example.test/pr-1", "1")
	for _, r := range p.ChangedRealms {
		listed := strings.Contains(got, "https://example.test/pr-1"+urlOf(r)+"/")
		rendered := contains(p.Realms, r)
		t.Logf("changed realm %-20s listed in comment=%v rendered=%v", r, listed, rendered)
		if listed && !rendered {
			t.Errorf("comment links %s, which the snapshot never rendered", r)
		}
	}
	if strings.Contains(got, "the changed realms are always kept") {
		t.Errorf("comment asserts the changed realms are always kept while %d was dropped", p.Dropped)
	}
	t.Logf("comment:\n%s", got)
}
