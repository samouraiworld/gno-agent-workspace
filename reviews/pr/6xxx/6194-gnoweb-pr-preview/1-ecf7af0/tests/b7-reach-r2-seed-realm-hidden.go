package main

// b7-reach-r2-seed-realm-hidden-from-comment — FAILS at PR 6194 head ecf7af0f2.
//
// BuildPlan appends the four gnowebSeedRealms into Plan.Realms (plan.go:264-268)
// with nothing marking them as seeds. The only downstream test for "this realm is
// a gnoweb sample, not a consequence of the diff" is isSeed (comment.go:157),
// which is membership in gnowebSeedRealms. So a seed realm that IS genuinely
// affected by a changed package is classified as a sample and dropped from the
// comment's "Realms affected through a changed package (N)" list, and out of N.
//
// This is live on the tree at this head, not hypothetical:
// gno.land/r/gnoland/home imports gno.land/p/nt/ownable/v0.
//
//	go build -o /tmp/gnopreview ./misc/gnopreview
//	printf 'examples/gno.land/p/nt/ownable/v0/ownable.gno\n' > /tmp/ch2.txt
//	/tmp/gnopreview plan -root . -changed /tmp/ch2.txt | jq -c '{n:(.realms|length),realms}'
//	# {"n":7,"realms":[... "gno.land/r/gnoland/home" ...]}   <- home is affected
//
//	printf 'gno.land/pkg/gnoweb/app.go\nexamples/gno.land/p/nt/ownable/v0/ownable.gno\n' > /tmp/ch.txt
//	/tmp/gnopreview plan -root . -changed /tmp/ch.txt | jq -c '{n:(.realms|length),realms}'
//	# {"n":10,...}  gnoweb=true, so isSeed() now hides home, blog, boards2/v0
//	#               and docs/security_patterns from the affected list.
//
// The unit repro below pins the same thing in one package-local test.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/b7-reach-r2-seed-realm-hidden.go \
//	   misc/gnopreview/zz_b7_seed_hidden_test.go
//	cd misc/gnopreview && go test -run TestSeedRealmAffectedIsHiddenFromComment -v .
//
// Observed at this head:
//
//	**Realms affected through a changed package (1)**
//	_changed: `gno.land/p/x/base/v0`_
//	- [`gno.land/r/x/other`](...)
//	BUG: the comment never names gno.land/r/gnoland/home, which the plan renders as affected
//	BUG: the comment counts 1 affected realm; the plan rendered 2
//	--- FAIL: TestSeedRealmAffectedIsHiddenFromComment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedRepo(t *testing.T) string {
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
	// gno.land/r/gnoland/home is a seed realm AND, here, a real importer of the
	// changed package — the shape measured on the real tree above.
	pkg("examples/gno.land/r/gnoland/home", "gno.land/r/gnoland/home",
		"package home\nimport \"gno.land/p/x/base/v0\"\n")
	// The control: an ordinary realm affected in exactly the same way.
	pkg("examples/gno.land/r/x/other", "gno.land/r/x/other",
		"package other\nimport \"gno.land/p/x/base/v0\"\n")
	return root
}

func TestSeedRealmAffectedIsHiddenFromComment(t *testing.T) {
	root := seedRepo(t)
	// A pull request that touches gnoweb and a package two realms import.
	changed := []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/p/x/base/v0/lib.gno",
	}
	plan, err := BuildPlan(root, changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Gnoweb || plan.Mode() != "both" {
		t.Fatalf("precondition: gnoweb=%v mode=%q", plan.Gnoweb, plan.Mode())
	}
	if !contains(plan.Realms, "gno.land/r/gnoland/home") || !contains(plan.Realms, "gno.land/r/x/other") {
		t.Fatalf("precondition: plan.Realms = %v", plan.Realms)
	}

	got := Comment(plan, "https://example.test/pr-1", "1")
	t.Logf("plan.Realms=%v ChangedPkgs=%v\n--- comment ---\n%s", plan.Realms, plan.ChangedPkgs, got)

	if !strings.Contains(got, "gno.land/r/x/other") {
		t.Errorf("control realm gno.land/r/x/other missing from the comment")
	}
	if !strings.Contains(got, "gno.land/r/gnoland/home") {
		t.Errorf("BUG: the comment never names gno.land/r/gnoland/home, which the plan renders as affected by %v", plan.ChangedPkgs)
	}
	if strings.Contains(got, "changed package (1)") {
		t.Errorf("BUG: the comment counts 1 affected realm; the plan rendered 2 (%v)", plan.Realms)
	}
}
