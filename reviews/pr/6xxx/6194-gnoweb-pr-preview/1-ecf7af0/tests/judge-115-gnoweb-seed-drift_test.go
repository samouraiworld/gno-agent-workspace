// Asserts how far the missing Realms[0] guard actually reaches. Measured against the
// real examples/ tree: all 4 gnowebSeedRealms resolve at this head, so a gnoweb-only
// PR always renders 4 realms and never reaches the index; a renamed seed is dropped
// with no error and no comment line, and only losing all 4 panics render().
// The drop is live at ecf7af0f29abe4737a52803d672bc5a33c17cc60; the panic is latent.
//
// from a local clone of gnolang/gno:
//   gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f2
//   curl -fsSL -o misc/gnopreview/judge115_test.go \
//     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-115-gnoweb-seed-drift_test.go
//   cd misc/gnopreview && go test -run TestJudge115 -v .
//   rm misc/gnopreview/judge115_test.go

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// judge115Repo builds a throwaway monorepo holding exactly the realms named.
func judge115Repo(t *testing.T, realms []string) string {
	t.Helper()
	root := t.TempDir()
	for _, r := range realms {
		dir := filepath.Join(root, "examples", filepath.FromSlash(r))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		write := func(name, body string) {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		write("gnomod.toml", "module = \""+r+"\"\n")
		write("lib.gno", "package lib\n")
	}
	return root
}

// The baseline: every hardcoded seed still names a real package, which is why the
// index below is unreachable in CI today. A rename here is what arms it.
func TestJudge115SeedsStillResolve(t *testing.T) {
	pkgs, err := LoadPkgs("../..")
	if err != nil {
		t.Skipf("no examples/ tree beside the test: %v", err)
	}
	for _, r := range gnowebSeedRealms {
		if pkgs[r] == nil {
			t.Errorf("seed %s no longer resolves: BuildPlan drops it silently", r)
		}
	}
}

// A single renamed seed leaves no trace: no error, Dropped stays 0, and the realm
// simply stops appearing in the comment.
func TestJudge115RenamedSeedIsDroppedSilently(t *testing.T) {
	root := judge115Repo(t, gnowebSeedRealms[1:]) // seeds[0] renamed away
	p, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/app.go"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Realms=%v Dropped=%d", p.Realms, p.Dropped)
	if contains(p.Realms, gnowebSeedRealms[0]) {
		t.Fatalf("fixture wrong: %s should be absent", gnowebSeedRealms[0])
	}
	if p.Dropped != 0 {
		t.Fatalf("unexpected Dropped=%d", p.Dropped)
	}
	// IS:     a missing seed is invisible — nothing counts it, nothing reports it.
	t.Logf("seed %s vanished from the preview with Dropped=0 and no error", gnowebSeedRealms[0])
	// SHOULD: BuildPlan reports an unresolvable seed so the sample page is not lost.
	// t.Errorf("BuildPlan should surface the unresolvable seed %s", gnowebSeedRealms[0])
}

// Losing every seed is what render() cannot survive: Empty() says there is work,
// and main.go:131 then evaluates plan.Realms[0].
func TestJudge115AllSeedsGonePanics(t *testing.T) {
	root := judge115Repo(t, []string{"gno.land/p/x/unrelated"})
	p, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/app.go"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Gnoweb=%v Realms=%v Empty()=%v", p.Gnoweb, p.Realms, p.Empty())
	if p.Empty() {
		t.Fatalf("fixture wrong: plan is empty, render() returns early")
	}
	var rec any
	func() {
		defer func() { rec = recover() }()
		_ = urlOf(p.Realms[0]) // the expression render() runs at main.go:131
	}()
	// IS:     render() panics past the Empty() guard instead of reporting nothing to do.
	if rec == nil {
		t.Fatalf("expected the index to panic")
	}
	t.Logf("render() panics past Empty(): %v", rec)
	// SHOULD: Empty() also covers len(Realms)==0, so the job prints "nothing to preview".
	// t.Errorf("Empty() should be true when no realm resolved")
}
