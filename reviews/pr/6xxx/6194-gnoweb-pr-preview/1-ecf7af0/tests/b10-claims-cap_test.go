// Claims check for PR 6194, bundle 10 (misc/gnopreview/plan_test.go).
//
// Three claims the diff writes about itself, each run here:
//
//  1. plan.go Plan.Realms godoc: "changed realms plus every realm that
//     (transitively) imports a changed package, capped at MaxRealms", and the
//     PR body's "At most 25 realms". main.go registers the same bound as a
//     flag: -max-realms "cap on rendered realms; 0 for no cap".
//     TestClaimSeedRealmsBypassCap runs BuildPlan with maxRealms=1 on a diff
//     that also touches gnoweb.
//
//  2. plan.go BuildPlan: "Directly changed realms first, so the cap never
//     drops the realm the PR is actually about", echoed to the reviewer by
//     comment.go: "the changed realms are always kept".
//     TestClaimChangedRealmsAreDropped runs BuildPlan with three changed
//     realms and maxRealms=1.
//
//  3. comment.go isSeed(): a realm that is genuinely affected through a
//     changed package is dropped from "Realms affected through a changed
//     package" whenever it also happens to be a gnoweb seed realm.
//     TestClaimAffectedSeedRealmIsUnlisted runs Comment on that shape.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_claims_test.go
//	cd misc/gnopreview && go test -run TestClaim -v ./...
//
// Every one of the three tests fails on the head commit; each t.Errorf prints
// the observed value beside the claim it contradicts.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// claimRepo writes a miniature monorepo that, unlike plan_test.go's fakeRepo,
// also carries the four realms in gnowebSeedRealms, so the seed-append branch
// of BuildPlan (plan.go, `if plan.Gnoweb { ... }`) is reachable.
func claimRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(p, body string) {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("examples/gnowork.toml", "")
	pkg := func(dir, module, body string) {
		write(dir+"/gnomod.toml", "module = \""+module+"\"\n")
		write(dir+"/lib.gno", body)
	}
	pkg("examples/gno.land/p/x/base/v0", "gno.land/p/x/base/v0", "package base\n")
	// two realms that import the changed package
	pkg("examples/gno.land/r/x/leaf", "gno.land/r/x/leaf",
		"package leaf\nimport \"gno.land/p/x/base/v0\"\n")
	pkg("examples/gno.land/r/x/other", "gno.land/r/x/other",
		"package other\nimport \"gno.land/p/x/base/v0\"\n")
	// the fixed gnoweb sample, present so BuildPlan can append it
	for _, r := range gnowebSeedRealms {
		pkg("examples/"+r, r, "package seed\n")
	}
	return root
}

// TestClaimSeedRealmsBypassCap: the gnoweb seed realms are appended after the
// cap is applied, so -max-realms N renders up to N+len(gnowebSeedRealms).
func TestClaimSeedRealmsBypassCap(t *testing.T) {
	root := claimRepo(t)
	changed := []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/p/x/base/v0/lib.gno",
	}
	got, err := BuildPlan(root, changed, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Realms) > 1 {
		t.Errorf("max-realms=1 rendered %d realms: %v (Dirs=%d, Dropped=%d)",
			len(got.Realms), got.Realms, len(got.Dirs), got.Dropped)
	}
}

// TestClaimChangedRealmsAreDropped: with more changed realms than the cap, the
// cap drops realms the PR changed, while the comment tells the reviewer the
// changed realms are always kept.
func TestClaimChangedRealmsAreDropped(t *testing.T) {
	root := claimRepo(t)
	changed := []string{
		"examples/gno.land/r/x/leaf/lib.gno",
		"examples/gno.land/r/x/other/lib.gno",
		"examples/gno.land/r/gnoland/home/lib.gno",
	}
	got, err := BuildPlan(root, changed, 1)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, r := range got.ChangedRealms {
		if !contains(got.Realms, r) {
			missing = append(missing, r)
		}
	}
	body := Comment(got, "https://example.test/pr-1", "1")
	says := strings.Contains(body, "the changed realms are always kept")
	if len(missing) > 0 && says {
		t.Errorf("changed realms dropped by the cap: %v (rendered %v, Dropped=%d); comment still says %q",
			missing, got.Realms, got.Dropped, "the changed realms are always kept")
	}
}

// TestClaimAffectedSeedRealmIsUnlisted: in "both" mode a realm affected
// through a changed package is silently omitted from the comment's affected
// list, and from its count, when it is also a gnoweb seed realm.
func TestClaimAffectedSeedRealmIsUnlisted(t *testing.T) {
	seed := gnowebSeedRealms[0]
	p := &Plan{
		Gnoweb:      true,
		ChangedPkgs: []string{"gno.land/p/x/base/v0"},
		Realms:      []string{seed, "gno.land/r/x/other"},
	}
	got := Comment(p, "https://example.test/pr-1", "1")
	if !strings.Contains(got, seed) {
		t.Errorf("rendered realm %q appears nowhere in the comment:\n%s", seed, got)
	}
	if !strings.Contains(got, "**Realms affected through a changed package (2)**") {
		t.Errorf("affected count excludes %q; comment says:\n%s", seed, got)
	}
}
