// b10-reach-r2-seed-realms — two reachability checks on BuildPlan's gnoweb
// seed-realm branch (misc/gnopreview/plan.go:263-270), the branch plan_test.go
// leaves empty ("gnoweb alone" asserts wantRealms: []string{}).
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_b10_reach_seed_test.go
//	cd misc/gnopreview && go test -run 'TestSeed' -v ./...
//
// Both tests fail at this head. Delete the copied file afterwards.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// seedRepo builds a minimal monorepo. When seedIgnored is true the seed realm
// gno.land/r/gnoland/home carries `ignore = true`; when present is false no
// seed realm exists at all.
func seedRepo(t *testing.T, present, seedIgnored bool) string {
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
	// An unrelated realm, so examples/gno.land exists and LoadPkgs walks.
	write("examples/gno.land/r/x/leaf/gnomod.toml", "module = \"gno.land/r/x/leaf\"\n")
	write("examples/gno.land/r/x/leaf/lib.gno", "package leaf\n")
	if present {
		mod := "module = \"gno.land/r/gnoland/home\"\n"
		if seedIgnored {
			mod += "ignore = true\n"
		}
		write("examples/gno.land/r/gnoland/home/gnomod.toml", mod)
		write("examples/gno.land/r/gnoland/home/home.gno", "package home\n")
	}
	return root
}

// TestSeedRealmIgnoreIsSkipped: every other path in BuildPlan drops an
// ignore=true package (plan.go:220 for a direct change, plan.go:241 for an
// indirect one). The seed branch checks only `pkgs[r] != nil`, so an ignored
// seed realm is planned and its directory is handed to gnodev, which filters
// ignore=true extra-root packages before genesis deploy
// (contribs/gnodev/pkg/packages/loader.go:507, loader_test.go:475).
func TestSeedRealmIgnoreIsSkipped(t *testing.T) {
	root := seedRepo(t, true, true)
	plan, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/app.go"}, 25)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range plan.Realms {
		if r == "gno.land/r/gnoland/home" {
			t.Fatalf("ignore=true seed realm planned: Realms = %v, Dirs = %v", plan.Realms, plan.Dirs)
		}
	}
}

// TestSeedlessGnowebPlanIsEmpty: with no seed realm in the tree a gnoweb-only
// change yields Gnoweb=true and zero realms. Empty() is !Gnoweb && no realms,
// so render() walks past its guard (main.go:111) and indexes plan.Realms[0]
// at main.go:131. plan_test.go:111-121 pins exactly this shape as valid.
func TestSeedlessGnowebPlanIsEmpty(t *testing.T) {
	root := seedRepo(t, false, false)
	plan, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/app.go"}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Realms) != 0 {
		t.Fatalf("fixture built a realm: %v", plan.Realms)
	}
	// What render() does after the Empty() guard lets this plan through.
	if !plan.Empty() {
		panicked := func() (p any) {
			defer func() { p = recover() }()
			_ = urlOf(plan.Realms[0])
			return nil
		}()
		t.Fatalf("Empty() = false with 0 realms; render() then panics: %v", panicked)
	}
}
