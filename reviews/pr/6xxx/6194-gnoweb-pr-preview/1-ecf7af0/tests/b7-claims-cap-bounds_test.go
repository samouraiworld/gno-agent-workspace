// b7-claims-cap-bounds_test.go — three claims misc/gnopreview/plan.go writes
// about itself, each with the run that settles it.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_b7_claims_test.go
//	cd misc/gnopreview && go test -run TestB7 -v ./...
//
// Observed at ecf7af0f2 with go1.25.9 — all three fail:
//
//	ChangedRealms=[gno.land/r/x/a gno.land/r/x/b gno.land/r/x/c] Realms=[gno.land/r/x/a] Dropped=2
//	changed realm gno.land/r/x/b   rendered=false linked in comment=true
//	comment asserts the changed realms are always kept while 2 of them were dropped
//	maxRealms=1 -> len(Realms)=5 [.../security_patterns .../blog .../boards2/v0 .../home .../r/x/a] Dropped=1
//	Gnoweb=true Realms=[] Empty()=false -> main.go:131 evaluates plan.Realms[0]
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// b7Repo writes a miniature monorepo whose realm set is given by the caller.
func b7Repo(t *testing.T, realms []string) string {
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
	for _, r := range realms {
		dir := "examples/" + r
		write(dir+"/gnomod.toml", "module = \""+r+"\"\n")
		write(dir+"/lib.gno", "package x\n")
	}
	return root
}

// Claim, plan.go:251 "the cap never drops the realm the PR is actually about",
// echoed by comment.go:86 "the changed realms are always kept" and by the README
// "Directly changed realms are always kept". When the pull request changes more
// realms than the cap, changed realms are dropped and the comment still links
// every one of them.
func TestB7CapDropsChangedRealms(t *testing.T) {
	all := []string{"gno.land/r/x/a", "gno.land/r/x/b", "gno.land/r/x/c"}
	root := b7Repo(t, all)
	var changed []string
	for _, r := range all {
		changed = append(changed, "examples/"+r+"/lib.gno")
	}
	plan, err := BuildPlan(root, changed, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dropped=%d", plan.ChangedRealms, plan.Realms, plan.Dropped)
	if len(plan.ChangedRealms) != 3 || len(plan.Realms) != 1 {
		t.Fatalf("setup: ChangedRealms=%d Realms=%d", len(plan.ChangedRealms), len(plan.Realms))
	}
	body := Comment(plan, "https://preview.test/pr-1", "1")
	for _, r := range plan.ChangedRealms {
		rendered := false
		for _, got := range plan.Realms {
			rendered = rendered || got == r
		}
		linked := strings.Contains(body, "https://preview.test/pr-1"+urlOf(r)+"/")
		t.Logf("changed realm %-16s rendered=%-5v linked in comment=%v", r, rendered, linked)
		if !rendered && linked {
			t.Errorf("comment links %s, which the cap dropped: the link 404s on the published snapshot", r)
		}
	}
	if strings.Contains(body, "the changed realms are always kept") {
		t.Errorf("comment asserts the changed realms are always kept while %d of them were dropped", 2)
	}
}

// Claim, plan.go:71 "capped at MaxRealms" and plan.go:192 "maxRealms caps how
// many realms are rendered". The gnoweb seed realms are appended after the cap.
func TestB7SeedsBypassCap(t *testing.T) {
	realms := append([]string{"gno.land/r/x/a", "gno.land/r/x/b"}, gnowebSeedRealms...)
	root := b7Repo(t, realms)
	plan, err := BuildPlan(root, []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/r/x/a/lib.gno",
		"examples/gno.land/r/x/b/lib.gno",
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("maxRealms=1 -> len(Realms)=%d %v Dropped=%d", len(plan.Realms), plan.Realms, plan.Dropped)
	if len(plan.Realms) > 1 {
		t.Errorf("Realms holds %d realms under a cap of 1", len(plan.Realms))
	}
}

// Claim, plan.go:91 Empty "reports whether there is nothing worth previewing".
// A gnoweb-only change whose seed realms are all absent yields Gnoweb=true with
// no realms; Empty is false, and main.go:131 then indexes plan.Realms[0].
func TestB7GnowebWithNoRealmsIsNotEmpty(t *testing.T) {
	root := b7Repo(t, []string{"gno.land/r/x/a"})
	plan, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/app.go"}, 25)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Gnoweb=%v Realms=%v Empty()=%v -> main.go:131 evaluates plan.Realms[0]",
		plan.Gnoweb, plan.Realms, plan.Empty())
	if plan.Gnoweb && len(plan.Realms) == 0 && !plan.Empty() {
		t.Errorf("render() would index plan.Realms[0] on an empty realm list")
	}
}
