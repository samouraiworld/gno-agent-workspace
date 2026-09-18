// Repro for gnolang/gno#6194: render indexes plan.Realms[0] while the guard it
// sits behind, Plan.Empty, answers on plan.Gnoweb. A gnoweb-only change whose
// seed realms are absent from the previewed tree passes the guard with zero
// realms and panics.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_b3_refactor_test.go
//	cd misc/gnopreview && go test -run 'TestGnowebPlanWithNoRealms' -v ./...
//
// Both tests PASS while the defect is present; the second one fails with
// "render returned without panicking" once render guards the index. Observed at
// ecf7af0f2 with go1.25.9:
//
//	=== RUN   TestRenderPanicsOnGnowebPlanWithNoRealms
//	rendering 0 realm(s):
//	    render panicked: runtime error: index out of range [0] with length 0
//	--- PASS: TestRenderPanicsOnGnowebPlanWithNoRealms (0.00s)

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeRoot builds a monorepo root that holds one non-realm package and none of
// the gnowebSeedRealms, which is the state a gnoweb-only change lands in when
// the seed realms are not present in the tree being previewed.
func fakeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mk := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, p), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("examples/gnowork.toml", "gno = \"0.9\"\n")
	mk("examples/gno.land/p/x/foo/gnomod.toml", "module = \"gno.land/p/x/foo\"\n")
	mk("examples/gno.land/p/x/foo/foo.gno", "package foo\n")
	return root
}

// TestGnowebPlanWithNoRealmsIsNotEmpty pins the precondition gap: Plan.Empty
// answers on Gnoweb, render's first use of the plan indexes Realms[0].
func TestGnowebPlanWithNoRealmsIsNotEmpty(t *testing.T) {
	plan, err := BuildPlan(fakeRoot(t), []string{"gno.land/pkg/gnoweb/handler.go"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Gnoweb || len(plan.Realms) != 0 {
		t.Fatalf("want gnoweb plan with no realms, got gnoweb=%v realms=%v", plan.Gnoweb, plan.Realms)
	}
	if plan.Empty() {
		t.Fatal("Empty() true: render would not be reached")
	}
	// The CI gate in .github/workflows/pr-preview.yml is the same predicate:
	//   jq '.gnoweb or (.realms | length > 0)'
	// so this plan reaches `gnopreview render` in the job as well.
}

// TestRenderPanicsOnGnowebPlanWithNoRealms runs render on that plan with a stub
// gnodev, so nothing but the plan decides the outcome.
func TestRenderPanicsOnGnowebPlanWithNoRealms(t *testing.T) {
	root := fakeRoot(t)
	plan, err := BuildPlan(root, []string{"gno.land/pkg/gnoweb/handler.go"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config{root: root, out: t.TempDir(), gnodev: "/bin/true", port: 18899, live: defaultLive}

	defer func() {
		switch r := recover(); r {
		case nil:
			t.Fatal("render returned without panicking; the index guard is in place")
		default:
			t.Logf("render panicked: %v", r)
		}
	}()
	_ = render(cfg, plan)
}
