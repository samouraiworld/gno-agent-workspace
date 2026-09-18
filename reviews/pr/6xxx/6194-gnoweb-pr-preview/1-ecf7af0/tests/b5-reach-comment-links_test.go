// b5-reach-comment-links_test.go — two reachability checks on misc/gnopreview/comment.go
// at gnolang/gno PR 6194, head ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/b5_reach_comment_links_test.go
//	cd misc/gnopreview && go test -run 'TestReachB5' -v ./...
//
// Both tests FAIL on this head; each failure is the finding.
//
//  1. TestReachB5CommentLinksUnrenderedChangedRealms: Plan.ChangedRealms is never
//     truncated, only Plan.Realms is, so Comment links changed realms the crawl
//     never rendered while the cap note says "the changed realms are always kept".
//  2. TestReachB5CommentHidesIndirectSeedRealm: isSeed() drops every seed realm
//     from the "Realms affected through a changed package" list whenever
//     Plan.Gnoweb is set, including seed realms that are genuinely affected by
//     the changed package.
package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// 1. The render cap drops changed realms; the comment still links them.
func TestReachB5CommentLinksUnrenderedChangedRealms(t *testing.T) {
	root := fakeRepo(t) // from plan_test.go
	changed := []string{
		"examples/gno.land/r/x/leaf/lib.gno",
		"examples/gno.land/r/x/lonely/lib.gno",
		"examples/gno.land/r/x/other/lib.gno",
	}
	p, err := BuildPlan(root, changed, 1) // -max-realms 1; the real default is 25
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dropped=%d", p.ChangedRealms, p.Realms, p.Dropped)

	rendered := map[string]bool{}
	for _, r := range p.Realms {
		rendered[r] = true
	}

	// Index() only lists realms the crawler actually fetched: build the crawler
	// the way render() does, with a page per rendered realm.
	c := &Crawler{}
	c.pages = map[string]*page{}
	for _, r := range p.Realms {
		c.pages[urlOf(r)] = &page{URL: urlOf(r), File: urlToFile(urlOf(r))}
	}
	index := Index(p, c)
	got := Comment(p, "https://example.test/pr-1", "1")

	for _, r := range p.ChangedRealms {
		if rendered[r] {
			continue
		}
		link := "https://example.test/pr-1" + urlOf(r) + "/"
		if strings.Contains(got, link) {
			t.Errorf("comment links %s, which the cap dropped from the render set: %s 404s", r, link)
		}
		if strings.Contains(index, urlOf(r)+"/") {
			t.Errorf("index.html links unrendered realm %s", r)
		}
	}
	if strings.Contains(got, "the changed realms are always kept") && len(p.ChangedRealms) > len(p.Realms) {
		t.Errorf("cap note claims every changed realm was kept, but %d of %d were dropped\n---\n%s",
			len(p.ChangedRealms)-len(p.Realms), len(p.ChangedRealms), got)
	}
}

// 1b. The same on the real tree, at the cap the workflow actually uses (the
// default 25, pr-preview.yml passes no -max-realms): a sweep over every realm's
// gnomod.toml, the shape of a version bump, leaves most changed realms unrendered.
func TestReachB5TreeWideChangeOutrunsTheCap(t *testing.T) {
	root, err := findRoot("")
	if err != nil {
		t.Skip(err)
	}
	var changed []string
	err = filepath.WalkDir(filepath.Join(root, "examples", "gno.land", "r"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "gnomod.toml" {
			changed = append(changed, filepath.ToSlash(mustRel(root, p)))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildPlan(root, changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	got := Comment(p, "https://gnolang.github.io/gno-preview/pr-6194", "6194")
	t.Logf("changed files=%d ChangedRealms=%d Realms=%d Dropped=%d comment=%d bytes",
		len(changed), len(p.ChangedRealms), len(p.Realms), p.Dropped, len(got))

	rendered := map[string]bool{}
	for _, r := range p.Realms {
		rendered[r] = true
	}
	var dead []string
	for _, r := range p.ChangedRealms {
		if !rendered[r] {
			dead = append(dead, r)
		}
	}
	if len(dead) > 0 {
		t.Errorf("%d of %d changed realms are linked by the comment but never rendered, e.g. %v",
			len(dead), len(p.ChangedRealms), dead[:min(3, len(dead))])
	}
}

// 2. gnoweb + a package imported only by seed realms: the comment names no realm.
func TestReachB5CommentHidesIndirectSeedRealm(t *testing.T) {
	root, err := findRoot("") // the repo this test file sits in
	if err != nil {
		t.Skip(err)
	}
	changed := []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/p/leon/svgbtn/v0/svgbtn.gno",
	}
	p, err := BuildPlan(root, changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Mode=%q Gnoweb=%v ChangedPkgs=%v Realms=%v", p.Mode(), p.Gnoweb, p.ChangedPkgs, p.Realms)
	if !contains(p.Realms, "gno.land/r/gnoland/home") {
		t.Fatalf("fixture drifted: r/gnoland/home no longer imports p/leon/svgbtn/v0; Realms=%v", p.Realms)
	}

	got := Comment(p, "https://example.test/pr-2", "2")
	for _, r := range p.Realms {
		if strings.Contains(got, urlOf(r)+"/") {
			continue
		}
		t.Errorf("comment never names %s, a realm the changed package %v affects and the preview rendered\n---\n%s",
			r, p.ChangedPkgs, got)
	}
}
