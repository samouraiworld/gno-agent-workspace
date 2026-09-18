// Round 1, bundle 10 (misc/gnopreview/plan_test.go), tests angle.
//
// Four guards that misc/gnopreview/plan_test.go leaves unpinned at
// ecf7af0f29abe4737a52803d672bc5a33c17cc60. Each test below goes red when its
// guard is mutated; the shipped suite stays green under all four.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_untested_guards_test.go
//	cd misc/gnopreview && go test ./...          # PASS, this file included
//
// The four mutations, each applied alone to the pristine tree, each leaving the
// shipped suite green and this file red:
//
//	M1 plan.go:220   s/p != nil \&\& !p.Ignore/p != nil/
//	     shipped: ok      here: TestIgnoredRealmChangedDirectly
//	M2 plan.go:263   replace the `if plan.Gnoweb { for _, r := range gnowebSeedRealms ... }`
//	                 block with `if false { _ = gnowebSeedRealms }`
//	     shipped: ok      here: TestGnowebSeedRealmsAreSeeded
//	M3 comment.go:47 hoist `b.WriteString(shotGrid(p.Shots, base))` above the
//	                 `if p.Gnoweb {` of the default branch, so it always runs
//	     shipped: ok      here: TestRealmsModeDropsGnowebSample
//	M4 plan.go:274   s/plan.ChangedFiles\[r\] = sortedKeys\(f\)/_ = f/
//	     shipped: ok      here: TestPlanCarriesChangedFilesAndDirs
//
// Run one mutation, then:
//
//	cd misc/gnopreview && go test ./...
//	git checkout -- . && cp <this file> misc/gnopreview/zz_untested_guards_test.go

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// guardRepo builds the smallest examples/ tree each guard needs: a seed realm
// gnowebSeedRealms names, a realm carrying `ignore = true`, and one ordinary
// realm. fakeRepo in plan_test.go carries none of the seed realms, which is why
// its "gnoweb alone" case reaches the seeding block and observes nothing.
func guardRepo(t *testing.T) string {
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
	pkg("examples/gno.land/r/gnoland/home", "gno.land/r/gnoland/home", "package home\n")
	pkg("examples/gno.land/r/x/leaf", "gno.land/r/x/leaf", "package leaf\n")
	write("examples/gno.land/r/x/ignored/gnomod.toml",
		"module = \"gno.land/r/x/ignored\"\nignore = true\n")
	write("examples/gno.land/r/x/ignored/lib.gno", "package ignored\n")
	return root
}

// M1: BuildPlan must not preview a realm whose gnomod.toml says ignore = true,
// even when the PR edits that realm's own source. gnodev refuses to load an
// ignored package, so previewing one is a crawl of pages that will 404.
func TestIgnoredRealmChangedDirectly(t *testing.T) {
	t.Parallel()
	got, err := BuildPlan(guardRepo(t), []string{"examples/gno.land/r/x/ignored/lib.gno"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ChangedRealms, []string(nil)) {
		t.Errorf("ChangedRealms = %v; want none for an ignored realm", got.ChangedRealms)
	}
	if len(got.Realms) != 0 {
		t.Errorf("Realms = %v; want none for an ignored realm", got.Realms)
	}
	if got.Mode() != "none" {
		t.Errorf("Mode = %q; want %q", got.Mode(), "none")
	}
}

// M2: a gnoweb-only change has no changed realm to render, so the preview is
// worth nothing unless gnowebSeedRealms are added to the plan.
func TestGnowebSeedRealmsAreSeeded(t *testing.T) {
	t.Parallel()
	got, err := BuildPlan(guardRepo(t), []string{"gno.land/pkg/gnoweb/app.go"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Gnoweb {
		t.Fatal("Gnoweb = false; want true")
	}
	if !reflect.DeepEqual(got.Realms, []string{"gno.land/r/gnoland/home"}) {
		t.Errorf("Realms = %v; want the seed realm present in the repo", got.Realms)
	}
	if !reflect.DeepEqual(got.Dirs, []string{"examples/gno.land/r/gnoland/home"}) {
		t.Errorf("Dirs = %v; want the seed realm's dir", got.Dirs)
	}
}

// M3: a realm-only PR gets before/after pairs, never the gnoweb sample grid.
// TestCommentRealms asserts this with a Plan whose Shots are nil, so it holds
// whatever comment.go does with them.
func TestRealmsModeDropsGnowebSample(t *testing.T) {
	t.Parallel()
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/x/leaf"},
		Shots:         []Shot{{File: "_shots/home.png", Label: "Home — rendered markdown"}},
	}
	got := Comment(p, "https://example.test/pr-7", "7")
	if strings.Contains(got, "_shots/home.png") || strings.Contains(got, "Home — rendered markdown") {
		t.Errorf("realm-only comment carries the gnoweb sample:\n%s", got)
	}
}

// M4: Dirs is what gnodev is started on and ChangedFiles is what the crawler
// uses to pick the before/after pages. Neither is asserted by plan_test.go, so
// the BuildPlan side of that contract is unpinned; crawl_test.go builds its own
// ChangedFiles by hand.
func TestPlanCarriesChangedFilesAndDirs(t *testing.T) {
	t.Parallel()
	got, err := BuildPlan(guardRepo(t), []string{"examples/gno.land/r/x/leaf/lib.gno"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Dirs, []string{"examples/gno.land/r/x/leaf"}) {
		t.Errorf("Dirs = %v; want the changed realm's dir", got.Dirs)
	}
	want := map[string][]string{"gno.land/r/x/leaf": {"lib.gno"}}
	if !reflect.DeepEqual(got.ChangedFiles, want) {
		t.Errorf("ChangedFiles = %v; want %v", got.ChangedFiles, want)
	}
}
