// Repro for: the realm cap drops directly changed realms, yet the pull request
// comment still links every changed realm and tells the reader the changed
// realms are always kept.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f2
//	cp <this file> misc/gnopreview/b10_reach_cap_test.go
//	cd misc/gnopreview && go test -run TestCapDropsChangedRealms ./...
//
// Expected at head: FAIL, the comment links gno.land/r/x/c (never rendered) and
// carries "the changed realms are always kept".
//
// The production cap is 25 (main.go defaultMaxRealms; .github/workflows/pr-preview.yml
// passes no -max-realms). Merged pull requests that change more realm directories
// than that, counted over the last 400 commits touching examples/gno.land/r:
//
//	git log --format="C %h %s" --name-only -n 400 -- 'examples/gno.land/r/*'
//	  #5669 123 realm dirs, #4461 119, #4331 116, #5726 102, #5220 95, #4040 87,
//	  #3374 63, #4316 62
//
// On #5669 the comment would list 123 changed realms, 98 of whose links 404.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func capRepo(t *testing.T, names ...string) string {
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
	for _, n := range names {
		dir := "examples/gno.land/r/x/" + n
		write(dir+"/gnomod.toml", "module = \"gno.land/r/x/"+n+"\"\n")
		write(dir+"/lib.gno", "package "+n+"\n")
	}
	return root
}

func TestCapDropsChangedRealms(t *testing.T) {
	names := []string{"a", "b", "c"}
	root := capRepo(t, names...)
	var changed []string
	for _, n := range names {
		changed = append(changed, "examples/gno.land/r/x/"+n+"/lib.gno")
	}

	const cap = 2
	plan, err := BuildPlan(root, changed, cap)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dropped=%d", plan.ChangedRealms, plan.Realms, plan.Dropped)

	if len(plan.ChangedRealms) != len(names) {
		t.Fatalf("ChangedRealms = %v; want all %d", plan.ChangedRealms, len(names))
	}
	if len(plan.Realms) != cap {
		t.Fatalf("Realms = %v; want %d", plan.Realms, cap)
	}

	base := "https://example.test/pr-1"
	c := Comment(plan, base, "1")

	for _, r := range plan.ChangedRealms {
		if contains(plan.Realms, r) {
			continue
		}
		link := base + urlOf(r) + "/"
		if strings.Contains(c, link) {
			t.Errorf("comment links %s, a changed realm the cap dropped: %s is never rendered and 404s", r, link)
		}
	}
	if strings.Contains(c, "the changed realms are always kept") {
		t.Errorf("comment claims the changed realms are always kept, but %d of %d were dropped",
			len(plan.ChangedRealms)-len(plan.Realms), len(plan.ChangedRealms))
	}
}
