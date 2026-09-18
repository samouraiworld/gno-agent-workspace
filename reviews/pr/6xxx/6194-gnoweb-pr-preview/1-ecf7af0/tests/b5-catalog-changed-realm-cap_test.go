// Repro for the changed-realm cap. A pull request that changes more realm
// packages than -max-realms (default 25) has changed realms silently dropped
// from the render plan, while the posted comment keeps linking every one of
// them and states that the changed realms are always kept.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp b5-catalog-changed-realm-cap_test.go misc/gnopreview/
//	cd misc/gnopreview && go test -run TestCatalogChangedRealmsSurviveTheCap -v .
//
// 15 of the last 400 gnolang/gno commits touching examples/gno.land/r/ touch
// more than 25 realm packages, so the case is ordinary, not a corner.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// capRepo writes n realm packages under examples/gno.land/r/x and returns the
// root plus the changed-file list naming every one of them.
func capRepo(t *testing.T, n int) (root string, changed []string) {
	t.Helper()
	root = t.TempDir()
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
	for i := 0; i < n; i++ {
		dir := fmt.Sprintf("examples/gno.land/r/x/r%02d", i)
		write(dir+"/gnomod.toml", fmt.Sprintf("module = \"gno.land/r/x/r%02d\"\n", i))
		write(dir+"/lib.gno", fmt.Sprintf("package r%02d\n", i))
		changed = append(changed, dir+"/lib.gno")
	}
	return root, changed
}

func TestCatalogChangedRealmsSurviveTheCap(t *testing.T) {
	t.Parallel()
	root, changed := capRepo(t, defaultMaxRealms+5)
	p, err := BuildPlan(root, changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("changed realms %d, rendered realms %d, dropped %d",
		len(p.ChangedRealms), len(p.Realms), p.Dropped)

	var missing []string
	for _, r := range p.ChangedRealms {
		if !contains(p.Realms, r) {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d changed realm(s) never rendered: %v", len(missing), missing)
	}

	c := Comment(p, "https://example.test/pr-1", "1")
	for _, r := range missing {
		if strings.Contains(c, "https://example.test/pr-1"+urlOf(r)+"/") {
			t.Errorf("comment links %s, which was never rendered (404)", r)
			break
		}
	}
	if len(missing) > 0 && strings.Contains(c, "the changed realms are always kept") {
		t.Errorf("comment claims the changed realms are always kept; %d were dropped", len(missing))
	}
}
