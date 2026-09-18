// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_b7_test.go
//	cd misc/gnopreview && go test -run 'TestDotDirGuardStillDescends|TestSeedRealmDroppedFromComment|TestZZCountsDirectoryReads' -v .
//
// Both behaviour tests PASS at this head: each asserts what the code does today,
// and the t.Logf lines are the evidence. TestZZCountsDirectoryReads needs the
// counter patch below; without it, it is skipped.
//
// The two line-count rewrites are applied and run like this, from misc/gnopreview,
// each followed by `go test ./...` and `git checkout -- plan.go`:
//
//	# A: Pkg.Dir and byDir are derivable from Pkg.Path (345 -> 337 lines, green)
//	python3 - <<'PY'
//	import pathlib, subprocess
//	p = pathlib.Path("plan.go"); t = p.read_text()
//	for old, new in [
//	  ('\tDir     string   // examples/gno.land/r/gnoland/home (repo-relative, slash-separated)\n', ''),
//	  ('\t\t\tDir:   rel,\n', ''),
//	  ('\t// dir -> package, so a changed file maps back to its package.\n\tbyDir := map[string]*Pkg{}\n\tfor _, p := range pkgs {\n\t\tbyDir[p.Dir] = p\n\t}\n\n', ''),
//	  ('if p := byDir[path.Dir(f)]; p != nil && !p.Ignore {',
//	   'if p := pkgs[strings.TrimPrefix(path.Dir(f), examplesRel+"/")]; p != nil && !p.Ignore {'),
//	  ('plan.Dirs = append(plan.Dirs, pkgs[r].Dir)', 'plan.Dirs = append(plan.Dirs, examplesRel+"/"+r)'),
//	]:
//	    assert t.count(old) == 1, old[:40]
//	    t = t.replace(old, new)
//	p.write_text(t); subprocess.run(["gofmt", "-w", "plan.go"])
//	PY
//
//	# B: changedFiles as a slice instead of a map of maps (345 -> 343 lines, green)
//	#   changedFiles := map[string][]string{}
//	#   changedFiles[p.Path] = append(changedFiles[p.Path], path.Base(f))
//	#   sort.Strings(f); plan.ChangedFiles[r] = slices.Compact(f)
//
//	# D (REFUTED): sortedKeys -> slices.Sorted(maps.Keys(m)), 8 -> 3 lines, turns
//	# four TestBuildPlan subtests red — "Realms = []; want []" — because the
//	# iterator form returns nil for an empty map and reflect.DeepEqual separates
//	# nil from the pre-allocated empty slice. plan.json carries "realms": [] on
//	# that accident.
//
//	# Counter patch for TestZZCountsDirectoryReads, in plan.go:
//	#   var zzDirs, zzEntries int          (above LoadPkgs)
//	#   zzEntries++                        (first line of the WalkDir callback body)
//	#   zzDirs++                           (immediately before `ents, err := os.ReadDir(p)`)
//	# Measured at this head: packages=145 walk-callback-entries=1531 callback ReadDir calls=304.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LoadPkgs' dotted-directory guard returns nil, not fs.SkipDir: it drops the
// dotted directory's own package and still descends into its children.
func TestDotDirGuardStillDescends(t *testing.T) {
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
	write("examples/gno.land/r/.hidden/deep/gnomod.toml", "module = \"gno.land/r/.hidden/deep\"\n")
	write("examples/gno.land/r/.hidden/deep/lib.gno", "package deep\n")
	write("examples/gno.land/r/.self/gnomod.toml", "module = \"gno.land/r/.self\"\n")
	write("examples/gno.land/r/.self/lib.gno", "package self\n")

	pkgs, err := LoadPkgs(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pkgs["gno.land/r/.hidden/deep"]; !ok {
		t.Errorf("package under a dotted directory was pruned")
	} else {
		t.Logf("LOADED gno.land/r/.hidden/deep: the walk descended past the dotted directory")
	}
	if _, ok := pkgs["gno.land/r/.self"]; ok {
		t.Errorf("dotted directory's own package was loaded")
	} else {
		t.Logf("SKIPPED gno.land/r/.self: the dotted directory's own package is dropped")
	}
}

// isSeed classifies by the package-level gnowebSeedRealms list instead of by
// what BuildPlan actually appended, so in mode "both" a realm pulled in through
// a changed package disappears from the comment when it is one of the seeds.
func TestSeedRealmDroppedFromComment(t *testing.T) {
	p := &Plan{
		Gnoweb:      true,
		ChangedPkgs: []string{"gno.land/p/x/base/v0"},
		Realms:      []string{"gno.land/r/gnoland/home", "gno.land/r/x/other"},
	}
	got := Comment(p, "https://example.test/pr-1", "1")
	t.Logf("Mode()=%q", p.Mode())
	if strings.Contains(got, "gno.land/r/gnoland/home") {
		t.Errorf("seed realm listed; isSeed no longer hides it")
	} else {
		t.Logf("ABSENT gno.land/r/gnoland/home: in Plan.Realms, so rendered and crawled, and linked nowhere in the comment")
	}
	if !strings.Contains(got, "gno.land/r/x/other") {
		t.Errorf("non-seed affected realm missing from the comment too")
	}
}
