// Repro for: BuildPlan caps Plan.Realms (what gets rendered) but never caps
// Plan.ChangedRealms (what the PR comment links), so a pull request touching
// more realms than -max-realms publishes a comment full of links to pages the
// snapshot never rendered, under a sentence claiming the opposite.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/
//	cd misc/gnopreview && go test -run TestCapDropsChangedRealms -v ./...
//
// Observed at ecf7af0f2 with go1.25.9: FAIL — 2 of 4 changed realms are absent
// from Plan.Realms while the comment links all four and prints
// "the changed realms are always kept".
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func capRepo(t *testing.T, realms []string) string {
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

func TestCapDropsChangedRealms(t *testing.T) {
	t.Parallel()
	realms := []string{
		"gno.land/r/x/a", "gno.land/r/x/b", "gno.land/r/x/c", "gno.land/r/x/d",
	}
	root := capRepo(t, realms)

	changed := make([]string, 0, len(realms))
	for _, r := range realms {
		changed = append(changed, "examples/"+r+"/lib.gno")
	}

	const maxRealms = 2
	plan, err := BuildPlan(root, changed, maxRealms)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Realms        = %v (%d)", plan.Realms, len(plan.Realms))
	t.Logf("ChangedRealms = %v (%d)", plan.ChangedRealms, len(plan.ChangedRealms))
	t.Logf("Dropped       = %d", plan.Dropped)

	rendered := map[string]bool{}
	for _, r := range plan.Realms {
		rendered[r] = true
	}
	var orphan []string
	for _, r := range plan.ChangedRealms {
		if !rendered[r] {
			orphan = append(orphan, r)
		}
	}
	if len(orphan) > 0 {
		t.Errorf("changed realms not rendered: %v", orphan)
	}

	const base = "https://example.invalid/preview/6194"
	body := Comment(plan, base, "6194")
	for _, r := range orphan {
		want := fmt.Sprintf("(%s/%s/)", base, strings.TrimPrefix(urlOf(r), "/"))
		if strings.Contains(body, want) {
			t.Errorf("comment links %s, a page the snapshot does not contain: %s", r, want)
		}
	}
	if len(orphan) > 0 && strings.Contains(body, "the changed realms are always kept") {
		t.Errorf("comment claims %q while %d changed realm(s) were dropped",
			"the changed realms are always kept", len(orphan))
	}
}
