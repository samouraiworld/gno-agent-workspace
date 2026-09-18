// Asserts that a gnoweb seed realm the diff genuinely affects through a changed
// package is named in the PR comment. Measured on the real examples/ tree: a diff
// touching gno.land/pkg/gnoweb/, examples/gno.land/p/moul/dynreplacer/v0/ (imported
// by r/gnoland/home) and one unrelated realm renders r/gnoland/home and never links
// it. Fails at ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// from a local clone of gnolang/gno:
//   gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f2
//   curl -fsSL -o misc/gnopreview/judge66_test.go \
//     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-66-seed-realm-affected-by-a-changed-package_test.go
//   cd misc/gnopreview && go test -run TestJudge66 -v .
//   rm misc/gnopreview/judge66_test.go

package main

import (
	"strings"
	"testing"
)

func TestJudge66SeedRealmAffectedByAChangedPackageIsUnlisted(t *testing.T) {
	// gnoweb + a package r/gnoland/home imports + one unrelated changed realm.
	// The changed realm keeps main.go off the Screenshot branch, so Shots is nil
	// and the seed realms have no other place in the comment.
	changed := []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/p/moul/dynreplacer/v0/gnomod.toml",
		"examples/gno.land/r/demo/counter/gnomod.toml",
	}
	plan, err := BuildPlan("../..", changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	const seed = "gno.land/r/gnoland/home"
	if !contains(plan.Realms, seed) {
		t.Fatalf("fixture stale: %s is not rendered; Realms=%v", seed, plan.Realms)
	}
	t.Logf("Mode=%s Gnoweb=%v ChangedRealms=%v ChangedPkgs=%v Realms=%v",
		plan.Mode(), plan.Gnoweb, plan.ChangedRealms, plan.ChangedPkgs, plan.Realms)

	body := Comment(plan, "https://example.test/pr-1", "1")
	if !strings.Contains(body, seed) {           // IS:     bug — the rendered realm is dropped from every list
		// if strings.Contains(body, seed) {     // SHOULD: an affected realm is always linked
		t.Errorf("%s is rendered but never named in the comment\n---\n%s", seed, body)
	}
}
