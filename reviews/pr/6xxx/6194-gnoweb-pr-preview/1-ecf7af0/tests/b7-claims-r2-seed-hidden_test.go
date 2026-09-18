// A realm that is genuinely affected by a changed package disappears from the
// published comment as soon as the same pull request also touches gnoweb.
//
// plan.go appends gnowebSeedRealms to Plan.Realms whenever Plan.Gnoweb is set,
// and comment.go then hides every realm in that fixed list (isSeed, comment.go:157)
// from the "Realms affected through a changed package" section. isSeed asks only
// whether the path is one of the four seeds, never why the realm is in
// Plan.Realms — so a realm that the reverse-import walk found is hidden too.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_seed_hidden_test.go
//	cd misc/gnopreview && go test -run TestSeedRealmHiddenFromComment -v ./...
//
// Expected on a fixed tree: PASS. Observed at ecf7af0: FAIL — gno.land/r/gnoland/home
// is in plan.Realms (it imports the changed gno.land/p/nt/ownable/v0) and is
// rendered, and the comment never links it.
package main

import (
	"strings"
	"testing"
)

// repoRoot is the monorepo root relative to misc/gnopreview.
const repoRoot = "../.."

func buildPlanOrFail(t *testing.T, changed []string) *Plan {
	t.Helper()
	// maxRealms 0: no cap, so the cap is not what removes anything here.
	p, err := BuildPlan(repoRoot, changed, 0)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSeedRealmHiddenFromComment(t *testing.T) {
	const (
		seed      = "gno.land/r/gnoland/home"                   // a gnowebSeedRealms entry
		pkgChange = "examples/gno.land/p/nt/ownable/v0/ownable.gno" // imported by that realm
		gnoweb    = "gno.land/pkg/gnoweb/app.go"
		baseURL   = "https://example.test/pr-1"
	)

	// Control: the package change alone. The realm is affected, rendered, and linked.
	p1 := buildPlanOrFail(t, []string{pkgChange})
	if !contains(p1.Realms, seed) {
		t.Fatalf("setup: %s is not an affected realm of %s", seed, pkgChange)
	}
	if c := Comment(p1, baseURL, "1"); !strings.Contains(c, seed) {
		t.Fatalf("control: comment omits %s\n---\n%s", seed, c)
	}

	// The case: the same package change, plus a gnoweb change.
	p2 := buildPlanOrFail(t, []string{pkgChange, gnoweb})
	if !contains(p2.Realms, seed) {
		t.Fatalf("setup: %s is not rendered in the gnoweb+package plan", seed)
	}
	if c := Comment(p2, baseURL, "1"); !strings.Contains(c, seed) {
		t.Errorf("%s is rendered (Plan.Realms) but absent from the comment:\n---\n%s", seed, c)
	}
}
