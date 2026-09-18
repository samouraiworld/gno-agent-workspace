// Repro for: misc/gnopreview caps plan.Realms at maxRealms *after* the changed
// realms are counted, so a PR touching more than 25 realms gets a preview
// comment that links every changed realm while gnodev only ever rendered 25.
// The same comment states "the changed realms are always kept".
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin pull/6194/head:pr6194 && git checkout pr6194
//	cp <this file> misc/gnopreview/zz_b10_cap_test.go
//	cd misc/gnopreview
//	go test -run TestB10CapDropsDirectlyChangedRealms -v .
//	GNOROOT_FIXTURE=$(git rev-parse --show-toplevel) \
//	  go test -run TestB10RealRepoGnomodBump -v .
//
// Observed at ecf7af0f29abe4737a52803d672bc5a33c17cc60 with go1.25.9:
//
//	TestB10CapDropsDirectlyChangedRealms
//	  ChangedRealms=[gno.land/r/x/a gno.land/r/x/b gno.land/r/x/c]
//	  Realms(rendered)=[gno.land/r/x/a gno.land/r/x/b] Dropped=1
//	  comment links https://example.test/pr-1/r/x/c/, but r/x/c is absent from Dirs
//	TestB10RealRepoGnomodBump
//	  changed gnomod.toml files: 145
//	  ChangedRealms=58 rendered=25 Dropped=33
//	  changed realms linked but never rendered (404): 33
//	  comment length=13436 bytes, bullet lines=58
//	  comment claims 'changed realms are always kept': true
//
// Two further mutations, run in the same tree, show which plan_test.go cases
// pin nothing. Each leaves `go test -run TestBuildPlan .` green:
//
//	# "quarantined packages never appear" (plan_test.go:96)
//	perl -0pi -e 's/if !strings\.HasPrefix\(f, pkgRoot\+"\/"\) \|\| !renderRelevant\(f\) \{/if !renderRelevant(f) {/' plan.go
//	go test -run TestBuildPlan .   # ok
//
//	# the `&& !p.Ignore` guard on a directly changed file (plan.go:220)
//	perl -0pi -e 's/if p := byDir\[path\.Dir\(f\)\]; p != nil && !p\.Ignore \{/if p := byDir[path.Dir(f)]; p != nil {/' plan.go
//	go test -run TestBuildPlan .   # ok
//
// For contrast, removing the indirectExclude filter does turn
// "transitive dependency pulls both dependents" red, while
// "tests fixtures are not pulled in indirectly" (plan_test.go:84) stays green:
//
//	perl -0pi -e 's/\t\tif !changedPkgs\[p\] && hasPrefixAny\(p, indirectExclude\) \{\n\t\t\tcontinue\n\t\t\}\n//' plan.go
//	go test -run TestBuildPlan -v . 2>&1 | grep -- '--- '

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// capRepo writes n directly-renderable realms gno.land/r/x/a0..a<n-1>.
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
	for i := range n {
		name := string(rune('a' + i))
		dir := "examples/gno.land/r/x/" + name
		write(dir+"/gnomod.toml", "module = \"gno.land/r/x/"+name+"\"\n")
		write(dir+"/lib.gno", "package "+name+"\n")
		changed = append(changed, dir+"/lib.gno")
	}
	return root, changed
}

// A changed realm past the cap keeps its bullet in the comment and loses its page.
func TestB10CapDropsDirectlyChangedRealms(t *testing.T) {
	root, changed := capRepo(t, 3)
	p, err := BuildPlan(root, changed, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v", p.ChangedRealms)
	t.Logf("Realms(rendered)=%v Dropped=%d", p.Realms, p.Dropped)
	t.Logf("Dirs handed to gnodev = %v", p.Dirs)

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
	c := Comment(p, "https://example.test/pr-1", "1")
	if dead := "https://example.test/pr-1/r/x/c/"; strings.Contains(c, dead) {
		t.Errorf("comment links %s, but r/x/c is absent from Dirs so gnodev never renders it", dead)
	}
	if len(missing) > 0 && strings.Contains(c, "the changed realms are always kept") {
		t.Errorf("cap dropped directly-changed realms %v while the comment states they are always kept", missing)
	}
}

// The realistic trigger: a repo-wide gno version bump edits every gnomod.toml,
// and renderRelevant() counts gnomod.toml as a render-relevant change.
func TestB10RealRepoGnomodBump(t *testing.T) {
	root := os.Getenv("GNOROOT_FIXTURE")
	if root == "" {
		t.Skip("set GNOROOT_FIXTURE to the repo root")
	}
	var changed []string
	filepath.WalkDir(filepath.Join(root, "examples", "gno.land"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "gnomod.toml" {
			rel, _ := filepath.Rel(root, p)
			changed = append(changed, filepath.ToSlash(rel))
		}
		return nil
	})
	t.Logf("changed gnomod.toml files: %d", len(changed))
	p, err := BuildPlan(root, changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	rendered := map[string]bool{}
	for _, r := range p.Realms {
		rendered[r] = true
	}
	dead := 0
	for _, r := range p.ChangedRealms {
		if !rendered[r] {
			dead++
		}
	}
	c := Comment(p, "https://preview.example/pr-1", "1")
	t.Logf("ChangedRealms=%d rendered=%d Dropped=%d", len(p.ChangedRealms), len(p.Realms), p.Dropped)
	t.Logf("changed realms linked but never rendered (404): %d", dead)
	t.Logf("comment length=%d bytes, bullet lines=%d", len(c), strings.Count(c, "\n- "))
	t.Logf("comment claims 'changed realms are always kept': %v", strings.Contains(c, "the changed realms are always kept"))
	if dead > 0 {
		t.Errorf("%d changed realms get a preview link that 404s", dead)
	}
}
