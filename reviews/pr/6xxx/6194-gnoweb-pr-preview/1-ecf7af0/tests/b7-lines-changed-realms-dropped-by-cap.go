// Repro for two BuildPlan cap defects in misc/gnopreview/plan.go at
// gnolang/gno PR 6194, head ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/plan_cap_repro_test.go
//	cd misc/gnopreview && go test ./... -run 'Cap' -v
//
// Both tests fail at this head.
//
// TestChangedRealmsSurviveTheCap: BuildPlan truncates Realms (and so Dirs, the
// list gnodev is started with) to -max-realms while ChangedRealms keeps every
// changed realm, so Comment() links preview pages that were never rendered and
// prints "the changed realms are always kept". Real shape: 09a7530a6 (#6162)
// touched 35 distinct realm package dirs against a cap of 25.
//
// TestGnowebSeedsRespectTheCap: the four gnowebSeedRealms are appended after
// the cap is applied, so Realms can exceed -max-realms by up to four.

package main

import (
	"os"
	"path"
	"path/filepath"
	"testing"
)

func capRepo(t *testing.T, pkgPaths []string) string {
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
	for _, p := range pkgPaths {
		dir := "examples/" + p
		write(dir+"/gnomod.toml", "module = \""+p+"\"\n")
		write(dir+"/lib.gno", "package "+path.Base(p)+"\n")
	}
	return root
}

func TestChangedRealmsSurviveTheCap(t *testing.T) {
	var pkgs, changed []string
	for i := 0; i < 30; i++ {
		p := "gno.land/r/x/r" + string(rune('a'+i/26)) + string(rune('a'+i%26))
		pkgs = append(pkgs, p)
		changed = append(changed, "examples/"+p+"/lib.gno")
	}
	root := capRepo(t, pkgs)

	plan, err := BuildPlan(root, changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	inRealms := map[string]bool{}
	for _, r := range plan.Realms {
		inRealms[r] = true
	}
	var missing []string
	for _, r := range plan.ChangedRealms {
		if !inRealms[r] {
			missing = append(missing, r)
		}
	}
	t.Logf("changed=%d rendered=%d dropped=%d", len(plan.ChangedRealms), len(plan.Realms), plan.Dropped)
	if len(missing) > 0 {
		t.Errorf("%d changed realm(s) are listed by Comment() but never rendered: %v", len(missing), missing)
	}
}

func TestGnowebSeedsRespectTheCap(t *testing.T) {
	pkgs := append([]string{"gno.land/r/x/leaf"}, gnowebSeedRealms...)
	root := capRepo(t, pkgs)

	plan, err := BuildPlan(root, []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/r/x/leaf/lib.gno",
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("max-realms=1 rendered=%d dropped=%d realms=%v", len(plan.Realms), plan.Dropped, plan.Realms)
	if len(plan.Realms) > 1 {
		t.Errorf("Realms = %d realm(s) with -max-realms=1", len(plan.Realms))
	}
}
