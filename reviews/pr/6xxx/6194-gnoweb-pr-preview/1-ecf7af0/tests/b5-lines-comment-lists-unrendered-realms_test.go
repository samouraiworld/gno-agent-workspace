package main

// Repro for two defects in misc/gnopreview/comment.go at gnolang/gno
// ecf7af0f29abe4737a52803d672bc5a33c17cc60 (PR 6194).
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/b5_lines_test.go
//	cd misc/gnopreview && go test ./... -run 'TestCommentLinksUnrenderedChangedRealms|TestCommentDropsAffectedSeedRealm' -v
//
// Both tests FAIL on this head: each t.Errorf below reports the live defect.
//
//  1. TestCommentLinksUnrenderedChangedRealms: BuildPlan caps Plan.Realms at
//     -max-realms (default 25) but never caps Plan.ChangedRealms, and
//     Comment() lists every ChangedRealm with a link. A PR changing more
//     realms than the cap gets comment links to pages the crawler never
//     rendered (404 on the published preview), under a footer claiming
//     "the changed realms are always kept".
//  2. TestCommentDropsAffectedSeedRealm: isSeed() excludes any gnoweb seed
//     realm from the "Realms affected through a changed package" list, even
//     when that realm is in the plan because a changed package reaches it.
//     The realm is rendered and indexed, but invisible in the comment.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func b5Repo(t *testing.T, pkgs map[string]string) string {
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
	for mod, body := range pkgs {
		dir := "examples/" + mod
		write(dir+"/gnomod.toml", "module = \""+mod+"\"\n")
		write(dir+"/lib.gno", body)
	}
	return root
}

func TestCommentLinksUnrenderedChangedRealms(t *testing.T) {
	root := b5Repo(t, map[string]string{
		"gno.land/r/x/a1": "package a1\n",
		"gno.land/r/x/a2": "package a2\n",
		"gno.land/r/x/a3": "package a3\n",
	})
	changed := []string{
		"examples/gno.land/r/x/a1/lib.gno",
		"examples/gno.land/r/x/a2/lib.gno",
		"examples/gno.land/r/x/a3/lib.gno",
	}
	p, err := BuildPlan(root, changed, 2) // stands in for the default cap of 25
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dropped=%d", p.ChangedRealms, p.Realms, p.Dropped)

	got := Comment(p, "https://example.test/pr-1", "1")
	rendered := map[string]bool{}
	for _, r := range p.Realms {
		rendered[r] = true
	}
	for _, r := range p.ChangedRealms {
		if rendered[r] {
			continue
		}
		url := "https://example.test/pr-1" + urlOf(r) + "/"
		if strings.Contains(got, url) {
			t.Errorf("comment links %s, but %s was dropped from Plan.Realms and never rendered: that URL 404s", url, r)
		}
	}
	if strings.Contains(got, "the changed realms are always kept") && len(p.ChangedRealms) > len(p.Realms) {
		t.Errorf("footer claims the changed realms are always kept, yet %d of %d changed realms were dropped",
			len(p.ChangedRealms)-len(p.Realms), len(p.ChangedRealms))
	}
}

func TestCommentDropsAffectedSeedRealm(t *testing.T) {
	root := b5Repo(t, map[string]string{
		"gno.land/p/x/base/v0":    "package base\n",
		"gno.land/r/gnoland/home": "package home\nimport \"gno.land/p/x/base/v0\"\n",
	})
	changed := []string{
		"examples/gno.land/p/x/base/v0/lib.gno",
		"gno.land/pkg/gnoweb/handler.go",
	}
	p, err := BuildPlan(root, changed, 25)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Mode=%s Gnoweb=%v ChangedPkgs=%v Realms=%v", p.Mode(), p.Gnoweb, p.ChangedPkgs, p.Realms)

	got := Comment(p, "https://example.test/pr-2", "2")
	for _, r := range p.Realms {
		if !strings.Contains(got, urlOf(r)+"/") {
			t.Errorf("comment never mentions %s, which the plan renders\n---\n%s", r, got)
		}
	}
}
