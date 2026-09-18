// Adversarial checks for the claims misc/gnopreview/comment.go writes about
// itself (PR 6194, head ecf7af0f2). Not part of the branch.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/b5-claims-comment_test.go misc/gnopreview/zz_b5_claims_test.go
//	cd misc/gnopreview && go test ./... -run 'TestB5' -v
//
// Each test FAILS where the claim in the comment does not hold.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// b5Repo writes a miniature monorepo holding one realm per name (each name is a
// directory under examples/gno.land/r/) and returns the root plus the changed
// file list that touches every one of them.
func b5Repo(t *testing.T, names []string) (string, []string) {
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
	var changed []string
	for _, n := range names {
		dir := "examples/gno.land/r/" + n
		write(dir+"/gnomod.toml", "module = \"gno.land/r/"+n+"\"\n")
		write(dir+"/lib.gno", "package p\n")
		changed = append(changed, dir+"/lib.gno")
	}
	return root, changed
}

// Claim, comment.go:95: "the changed realms are always kept". BuildPlan
// truncates the combined realm list at -max-realms with no floor for the
// changed ones, so a pull request that changes more realms than the cap loses
// some of them — while the comment still lists every changed realm with a link
// into the snapshot that was never rendered, under a note saying they were kept.
func TestB5DroppedChangedRealmAgainstTheNote(t *testing.T) {
	root, changed := b5Repo(t, []string{"a", "b", "c"})
	plan, err := BuildPlan(root, changed, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dropped=%d", plan.ChangedRealms, plan.Realms, plan.Dropped)

	var lost []string
	for _, r := range plan.ChangedRealms {
		if !contains(plan.Realms, r) {
			lost = append(lost, r)
		}
	}
	if len(lost) == 0 {
		return // claim holds: the cap kept every changed realm
	}
	body := Comment(plan, "https://example.test/pr-1", "1")
	for _, r := range lost {
		link := "https://example.test/pr-1" + urlOf(r) + "/"
		if strings.Contains(body, link) {
			t.Errorf("changed realm %q was dropped by the cap, yet the comment links %s", r, link)
		}
	}
	if strings.Contains(body, "the changed realms are always kept") {
		t.Errorf("comment claims %q while %v were dropped", "the changed realms are always kept", lost)
	}
	t.Logf("comment:\n%s", body)
}

// Claim, comment.go:26-28 against comment.go:175-179: Index skips a realm whose
// page was never captured; Comment links every planned realm unconditionally,
// so a realm gnodev failed on is a dead link in the posted comment.
func TestB5CommentLinksUncapturedRealm(t *testing.T) {
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/ok", "gno.land/r/x/failed"},
		Realms:        []string{"gno.land/r/x/failed", "gno.land/r/x/ok"},
	}
	// Only one of the two realms actually rendered.
	c := &Crawler{pages: map[string]*page{
		"/r/x/ok": {URL: "/r/x/ok", File: "r/x/ok/index.html"},
	}}
	idx := Index(p, c)
	if strings.Contains(idx, "r/x/failed") {
		t.Errorf("index lists the uncaptured realm:\n%s", idx)
	}
	body := Comment(p, "https://example.test/pr-1", "1")
	if strings.Contains(body, "https://example.test/pr-1/r/x/failed/") {
		t.Errorf("comment links a realm that was never captured (index does not):\n%s", body)
	}
}

// Claim, comment.go:174-175: the index is "every captured render page", and its
// own paragraph says "%d realm(s) rendered" from len(p.Realms) while the list
// below it is the captured subset.
func TestB5IndexCountMatchesItsList(t *testing.T) {
	p := &Plan{Realms: []string{"gno.land/r/x/ok", "gno.land/r/x/failed"}}
	c := &Crawler{pages: map[string]*page{
		"/r/x/ok": {URL: "/r/x/ok", File: "r/x/ok/index.html"},
	}}
	idx := Index(p, c)
	rows := strings.Count(idx, "<li>")
	if !strings.Contains(idx, "<p>2 realm(s) rendered") {
		t.Skip("wording changed")
	}
	if rows != 2 {
		t.Errorf("index says 2 realm(s) rendered and lists %d", rows)
	}
}

// Claim, PR body: "The comment body is written by the untrusted job, so the
// publishing job passes it as a file and never interpolates it into a shell
// command. It cannot reach a secret." The body is still attacker-controlled
// markup: a realm's package path is its directory name in the fork's own tree,
// and Comment writes it raw into a markdown link and into an HTML alt=.
func TestB5CommentEscapesRealmNames(t *testing.T) {
	name := `evil)](https://attacker.example/)[x`
	root, changed := b5Repo(t, []string{name})
	plan, err := BuildPlan(root, changed, 25)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%q", plan.ChangedRealms)
	plan.Pairs = []ShotPair{{
		Realm: `x"><img src="https://attacker.example/pixel.png">`,
		After: "_shots/a.png", URL: "r/x/a/",
	}}
	body := Comment(plan, "https://example.test/pr-1", "1")
	if strings.Contains(body, "](https://attacker.example/)") {
		t.Errorf("a realm directory name closed the markdown link:\n%s", body)
	}
	if strings.Contains(body, `alt="x"><img src="https://attacker.example/pixel.png">"`) {
		t.Errorf("a realm name broke out of the alt attribute:\n%s", body)
	}
}
