// Asserts that every changed realm the PR comment links was actually rendered, and
// that the cap note matches what the cap did. Measured on the real examples/ tree:
// 58 realm directories, cap 25 — a tree-wide sweep drops 33 changed realms from the
// render while the comment links all 58 and says the changed realms are always kept.
// Fails at ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// from a local clone of gnolang/gno:
//   gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f2
//   curl -fsSL -o misc/gnopreview/judge67_test.go \
//     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-67-cap-drops-changed-realms_test.go
//   cd misc/gnopreview && go test -run TestJudge67 -v .
//   rm misc/gnopreview/judge67_test.go

package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// judge67Sweep is a gnomod.toml bump over every realm in examples/, the shape of a
// tree-wide migration or a gno fmt run.
func judge67Sweep(t *testing.T) []string {
	t.Helper()
	var changed []string
	err := filepath.WalkDir("../../examples/gno.land/r", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "gnomod.toml" {
			changed = append(changed, "examples/"+filepath.ToSlash(strings.TrimPrefix(p, "../../examples/")))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func TestJudge67CapDropsChangedRealmsTheCommentStillLinks(t *testing.T) {
	changed := judge67Sweep(t)
	plan, err := BuildPlan("../..", changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("changed files=%d ChangedRealms=%d Realms=%d Dropped=%d cap=%d",
		len(changed), len(plan.ChangedRealms), len(plan.Realms), plan.Dropped, defaultMaxRealms)

	base := "https://example.test/pr-1"
	body := Comment(plan, base, "1")
	var linkedButUnrendered []string
	for _, r := range plan.ChangedRealms {
		if contains(plan.Realms, r) {
			continue
		}
		// urlOf gives the path the comment links; nothing wrote that page.
		if strings.Contains(body, base+urlOf(r)+"/") {
			linkedButUnrendered = append(linkedButUnrendered, r)
		}
	}
	if len(linkedButUnrendered) > 0 {       // IS:     bug — dead links into the snapshot
		// if len(linkedButUnrendered) == 0 { // SHOULD: only rendered realms are linked
		t.Errorf("%d changed realm(s) linked but never rendered, e.g. %s%s/ for %s",
			len(linkedButUnrendered), base, urlOf(linkedButUnrendered[0]), linkedButUnrendered[0])
	}
	_ = path.Dir
}

func TestJudge67CapNoteContradictsTheDrop(t *testing.T) {
	changed := judge67Sweep(t)
	plan, err := BuildPlan("../..", changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	body := Comment(plan, "https://example.test/pr-1", "1")
	const note = "the changed realms are always kept"
	dropped := 0
	for _, r := range plan.ChangedRealms {
		if !contains(plan.Realms, r) {
			dropped++
		}
	}
	t.Logf("changed realms dropped by the cap=%d Dropped=%d", dropped, plan.Dropped)
	if dropped > 0 && strings.Contains(body, note) {       // IS:     bug — the note states the opposite
		// if dropped == 0 || !strings.Contains(body, note) { // SHOULD: the note holds
		t.Errorf("cap dropped %d of %d changed realms, yet the comment says %q",
			dropped, len(plan.ChangedRealms), note)
	}
}
