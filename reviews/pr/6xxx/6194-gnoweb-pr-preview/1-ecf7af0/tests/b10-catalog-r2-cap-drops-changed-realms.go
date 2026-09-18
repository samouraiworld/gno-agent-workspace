// b10-catalog-r2-cap-drops-changed-realms.go
//
// gnolang/gno PR 6194 (feat(ci): publish a static gnoweb preview for every pull
// request), head ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// What it pins
//   1. TestCapDropsChangedRealmsWhileCommentDeniesIt — with 30 directly changed
//      realms and the default cap of 25 (the workflow never passes
//      -max-realms, so 25 is what CI runs), BuildPlan drops 5 *changed* realms,
//      while Comment still prints "the changed realms are always kept" and
//      still links all 30, so 5 links in the posted PR comment point at pages
//      the preview never rendered.
//   2. TestModFlagsInlineComment — modFlags is a hand-rolled TOML scan;
//      "ignore = true # quarantined" reports ignore=false.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno
//   cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//   git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//   cp <this file> misc/gnopreview/zz_catalog_test.go
//   cd misc/gnopreview && go test -count=1 -v \
//     -run 'TestCapDropsChangedRealmsWhileCommentDeniesIt|TestModFlagsInlineComment' ./...
//
// Observed at that head (go1.25.9):
//   changed=30 rendered=25 dropped=5 changed-but-not-rendered=5
//     [gno.land/r/x/a25 ... gno.land/r/x/a29]
//   comment says "the changed realms are always kept": true; links to never-rendered realms: 5
//   --- FAIL: TestCapDropsChangedRealmsWhileCommentDeniesIt
//   "ignore = true # quarantined" -> ignore=false
//   --- PASS: TestModFlagsInlineComment
//
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func capRepo(t *testing.T, n int) (string, []string) {
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
	for i := range n {
		dir := fmt.Sprintf("examples/gno.land/r/x/a%02d", i)
		write(dir+"/gnomod.toml", fmt.Sprintf("module = \"gno.land/r/x/a%02d\"\n", i))
		write(dir+"/lib.gno", fmt.Sprintf("package a%02d\n", i))
		changed = append(changed, dir+"/lib.gno")
	}
	return root, changed
}

// 30 directly changed realms under the default cap of 25.
func TestCapDropsChangedRealmsWhileCommentDeniesIt(t *testing.T) {
	root, changed := capRepo(t, 30)
	p, err := BuildPlan(root, changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	rendered := map[string]bool{}
	for _, r := range p.Realms {
		rendered[r] = true
	}
	var missing []string
	for _, r := range p.ChangedRealms {
		if !rendered[r] {
			missing = append(missing, r)
		}
	}
	t.Logf("changed=%d rendered=%d dropped=%d changed-but-not-rendered=%d %v",
		len(p.ChangedRealms), len(p.Realms), p.Dropped, len(missing), missing)

	c := Comment(p, "https://example.test/pr-1", "1")
	claims := strings.Contains(c, "the changed realms are always kept")
	linksMissing := 0
	for _, r := range missing {
		if strings.Contains(c, urlOf(r)+"/") {
			linksMissing++
		}
	}
	t.Logf("comment says %q: %v; links to never-rendered realms: %d",
		"the changed realms are always kept", claims, linksMissing)

	if len(missing) > 0 && claims {
		t.Errorf("comment claims the changed realms are always kept, but %d of them were dropped by the cap: %v", len(missing), missing)
	}
	if linksMissing > 0 {
		t.Errorf("comment links %d realm(s) the preview never rendered", linksMissing)
	}
}

// modFlags is a hand-rolled TOML scan; an inline comment defeats it.
func TestModFlagsInlineComment(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "gnomod.toml")
	for _, body := range []string{
		"module = \"gno.land/r/x/a\"\nignore = true\n",
		"module = \"gno.land/r/x/a\"\nignore = true # quarantined\n",
		"module = \"gno.land/r/x/a\"\nignore=true\n",
	} {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		_, ign := modFlags(p)
		t.Logf("%-46q -> ignore=%v", strings.ReplaceAll(body, "\n", "\\n"), ign)
	}
}
