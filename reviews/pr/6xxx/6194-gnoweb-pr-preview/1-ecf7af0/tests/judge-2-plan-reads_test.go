// judge-2-plan-reads_test.go — one printout per judged candidate on
// misc/gnopreview/plan.go. Asserts nothing; every line is a measurement read
// off BuildPlan/modFlags at the reviewed head (ecf7af0).
//
// Measured at ecf7af0: a change to p/nt/ufmt/v0 affects 31 realms, the cap
// keeps the alphabetic head, and the 6 it cuts are every gno.land/r/sys/*.
//
/* Run: from a gno checkout:
gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0
curl -fsSL -o misc/gnopreview/judge-2-plan-reads_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-2-plan-reads_test.go
cd misc/gnopreview && go test -count=1 -run TestJudge -v .
rm judge-2-plan-reads_test.go
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// repoRoot is the monorepo root: misc/gnopreview is two levels down.
func repoRoot(t *testing.T) string {
	if v := os.Getenv("GNOPREVIEW_ROOT"); v != "" {
		return v
	}
	r, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// C241: among indirectly affected realms the cap cuts by alphabet, not by
// anything about the change.
func TestJudge241Cap(t *testing.T) {
	head := repoRoot(t)
	changed := []string{"examples/gno.land/p/nt/ufmt/v0/gnomod.toml"}
	capped, err := BuildPlan(head, changed, 25)
	if err != nil {
		t.Fatal(err)
	}
	all, err := BuildPlan(head, changed, 0)
	if err != nil {
		t.Fatal(err)
	}
	in := map[string]bool{}
	for _, r := range capped.Realms {
		in[r] = true
	}
	var dropped []string
	for _, r := range all.Realms {
		if !in[r] {
			dropped = append(dropped, r)
		}
	}
	fmt.Printf("C241 affected=%d kept=%d Dropped=%d\nC241 keptFirst=%s\nC241 dropped=%v\n",
		len(all.Realms), len(capped.Realms), capped.Dropped, capped.Realms[0], dropped)
}

// C228: gnoweb seed realms are appended after the truncation, so the rendered
// count runs past the cap the flag names.
func TestJudge228Seeds(t *testing.T) {
	p, err := BuildPlan(repoRoot(t), []string{
		"examples/gno.land/p/nt/ufmt/v0/gnomod.toml",
		"gno.land/pkg/gnoweb/render.go",
	}, 25)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("C228 gnoweb=%v len(Realms)=%d cap=25 Dropped=%d len(Dirs)=%d\n",
		p.Gnoweb, len(p.Realms), p.Dropped, len(p.Dirs))
	// C225: Dirs is index-aligned with Realms today; nothing states or tests it.
	fmt.Printf("C225 len(Realms)=%d len(Dirs)=%d aligned=%v\n",
		len(p.Realms), len(p.Dirs), len(p.Realms) == len(p.Dirs))
}

// C224: a file the pull request deleted lands in ChangedFiles, and wantFile
// then rejects every file that is still on disk.
func TestJudge224Deleted(t *testing.T) {
	head := repoRoot(t)
	p, err := BuildPlan(head, []string{"examples/gno.land/r/gnoland/home/deleted.gno"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("C224 ChangedRealms=%v ChangedFiles=%v\n", p.ChangedRealms, p.ChangedFiles)
	ents, err := os.ReadDir(filepath.Join(head, "examples/gno.land/r/gnoland/home"))
	if err != nil {
		t.Fatal(err)
	}
	c := &Crawler{Realms: p.Realms, ChangedFiles: p.ChangedFiles, fileBudget: map[string]int{}}
	for _, e := range ents {
		if filepath.Ext(e.Name()) == ".gno" {
			fmt.Printf("C224 wantFile(home, %s) = %v   <- on disk\n",
				e.Name(), c.wantFile("gno.land/r/gnoland/home", e.Name()))
		}
	}
	fmt.Printf("C224 wantFile(home, deleted.gno) = %v   <- gnoweb serves no such page\n",
		c.wantFile("gno.land/r/gnoland/home", "deleted.gno"))
}

// C247: a realm the pull request deletes outright has no directory in the head
// tree, so byDir misses it and the plan records nothing.
func TestJudge247DeletedRealm(t *testing.T) {
	p, err := BuildPlan(repoRoot(t), []string{"examples/gno.land/r/does/not/exist/lib.gno"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("C247 Empty=%v Realms=%v ChangedRealms=%v Dropped=%d\n",
		p.Empty(), p.Realms, p.ChangedRealms, p.Dropped)
}

// C252: Pkg.Path comes from the directory while the import graph is keyed by
// the module path, so a gnomod.toml whose module differs drops its dependents.
func TestJudge252ModuleMismatch(t *testing.T) {
	root := t.TempDir()
	w := func(p, body string) {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w("examples/gnowork.toml", "")
	// the module line disagrees with the directory it sits in
	w("examples/gno.land/p/x/base/v2/gnomod.toml", "module = \"gno.land/p/x/base\"\n")
	w("examples/gno.land/p/x/base/v2/lib.gno", "package base\n")
	w("examples/gno.land/r/x/leaf/gnomod.toml", "module = \"gno.land/r/x/leaf\"\n")
	w("examples/gno.land/r/x/leaf/lib.gno", "package leaf\nimport \"gno.land/p/x/base\"\n")
	p, err := BuildPlan(root, []string{"examples/gno.land/p/x/base/v2/lib.gno"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("C252 mismatch: ChangedPkgs=%v Realms=%v   <- leaf imports gno.land/p/x/base\n",
		p.ChangedPkgs, p.Realms)
	// the same shape with module == directory
	w("examples/gno.land/p/x/base/v2/gnomod.toml", "module = \"gno.land/p/x/base/v2\"\n")
	w("examples/gno.land/r/x/leaf/lib.gno", "package leaf\nimport \"gno.land/p/x/base/v2\"\n")
	q, err := BuildPlan(root, []string{"examples/gno.land/p/x/base/v2/lib.gno"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("C252 aligned:  ChangedPkgs=%v Realms=%v\n", q.ChangedPkgs, q.Realms)
}

// C254: modFlags strips spaces and compares the whole line, so a trailing
// comment or a tab defeats it.
func TestJudge254ModFlags(t *testing.T) {
	d := t.TempDir()
	for _, body := range []string{
		"module = \"gno.land/r/x/y\"\nignore = true\n",
		"module = \"gno.land/r/x/y\"\nignore = true # quarantined until X\n",
		"module = \"gno.land/r/x/y\"\ndraft = true # not ready\n",
		"module = \"gno.land/r/x/y\"\nignore\t=\ttrue\n",
	} {
		p := filepath.Join(d, "gnomod.toml")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		draft, ignore := modFlags(p)
		fmt.Printf("C254 %-56q -> draft=%v ignore=%v\n", body, draft, ignore)
	}
}
