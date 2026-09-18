// b7-reach-plan-caps: three reachability facts about misc/gnopreview/plan.go
// at gnolang/gno PR 6194, head ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_b7_reach_test.go
//	cd misc/gnopreview && go test -run TestB7 -v ./...
//
// All three tests FAIL at this head; each failure is the finding.
//
//	TestB7CapDropsChangedRealms  a changed realm past -max-realms is never
//	                             rendered, yet Comment links it and prints
//	                             "the changed realms are always kept"
//	TestB7NonEmptyPlanWithNoRealms  Empty() reports a plan with work while
//	                             Realms is empty; render() indexes Realms[0]
//	                             (main.go:131) and panics
//	TestB7SeedsOvershootCap      gnoweb seed realms are appended after the cap,
//	                             so -max-realms 2 renders 3 realms
//
// Observed output at this head (go1.25.9):
//
//	ChangedRealms=[gno.land/r/x/a gno.land/r/x/b gno.land/r/x/c]
//	Realms(rendered)=[gno.land/r/x/a gno.land/r/x/b] Dirs=[...a ...b] Dropped=1
//	comment links a realm that was never rendered: [`gno.land/r/x/c`](https://preview.example/pr-1/r/x/c/)
//	Gnoweb=true Realms=[] Empty=false Mode="gnoweb"
//	render() panics on a non-empty plan with no realms: runtime error: index out of range [0] with length 0
//	max-realms=2 Realms=[gno.land/r/gnoland/home gno.land/r/x/a gno.land/r/x/b] (3) Dirs=3

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// b7repo builds a minimal monorepo holding n realms named r/x/a, r/x/b, ...
func b7repo(t *testing.T, n int) string {
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
	for i := 0; i < n; i++ {
		name := string(rune('a' + i))
		dir := "examples/gno.land/r/x/" + name
		write(dir+"/gnomod.toml", "module = \"gno.land/r/x/"+name+"\"\n")
		write(dir+"/lib.gno", "package "+name+"\n")
	}
	return root
}

// A changed realm past the cap is dropped from Realms, never rendered, and
// still printed as a live link by Comment, under a line claiming it was kept.
func TestB7CapDropsChangedRealms(t *testing.T) {
	root := b7repo(t, 3)
	changed := []string{
		"examples/gno.land/r/x/a/lib.gno",
		"examples/gno.land/r/x/b/lib.gno",
		"examples/gno.land/r/x/c/lib.gno",
	}
	p, err := BuildPlan(root, changed, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v", p.ChangedRealms)
	t.Logf("Realms(rendered)=%v Dirs=%v Dropped=%d", p.Realms, p.Dirs, p.Dropped)

	var missing []string
	for _, r := range p.ChangedRealms {
		if !contains(p.Realms, r) {
			missing = append(missing, r)
		}
	}
	if len(missing) == 0 {
		t.Fatalf("no changed realm was dropped; the cap did not bite")
	}
	t.Logf("changed realms never rendered: %v", missing)

	c := Comment(p, "https://preview.example/pr-1", "1")
	for _, r := range missing {
		link := fmt.Sprintf("[`%s`](https://preview.example/pr-1%s/)", r, urlOf(r))
		if strings.Contains(c, link) {
			t.Errorf("comment links a realm that was never rendered: %s", link)
		}
	}
	if strings.Contains(c, "the changed realms are always kept") {
		t.Errorf("comment claims every changed realm was kept, while %v were dropped", missing)
	}
}

// BuildPlan returns a plan that Empty() calls non-empty and whose Realms is
// empty, which render() indexes at main.go:131 (urlOf(plan.Realms[0])).
func TestB7NonEmptyPlanWithNoRealms(t *testing.T) {
	root := b7repo(t, 1)
	p, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/app.go"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Gnoweb=%v Realms=%v Empty=%v Mode=%q", p.Gnoweb, p.Realms, p.Empty(), p.Mode())
	if p.Empty() || len(p.Realms) > 0 {
		t.Skip("plan carries a realm; no index to reach")
	}
	var rec any
	func() {
		defer func() { rec = recover() }()
		_ = urlOf(p.Realms[0]) // the expression render() runs at main.go:131
	}()
	if rec == nil {
		t.Fatalf("expected a panic from plan.Realms[0]")
	}
	t.Errorf("render() panics on a non-empty plan with no realms: %v", rec)
}

// Seed realms are appended after the cap, so a gnoweb PR renders more realms
// than -max-realms allows.
func TestB7SeedsOvershootCap(t *testing.T) {
	root := b7repo(t, 3)
	seedDir := "examples/" + gnowebSeedRealms[0]
	full := filepath.Join(root, filepath.FromSlash(seedDir))
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(full, "gnomod.toml"), []byte("module = \""+gnowebSeedRealms[0]+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(full, "lib.gno"), []byte("package home\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed := []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/r/x/a/lib.gno",
		"examples/gno.land/r/x/b/lib.gno",
	}
	p, err := BuildPlan(root, changed, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("max-realms=2 Realms=%v (%d) Dirs=%d", p.Realms, len(p.Realms), len(p.Dirs))
	if len(p.Realms) > 2 {
		t.Errorf("-max-realms 2 rendered %d realms", len(p.Realms))
	}
}
