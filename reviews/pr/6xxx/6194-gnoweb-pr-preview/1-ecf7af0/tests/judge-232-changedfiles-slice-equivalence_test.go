// Pins BuildPlan's ChangedFiles output for a changed list carrying a duplicate
// entry, two .gno files in one package, a gnomod.toml and a _test.gno, so the
// map-of-maps and the slice+slices.Compact form can be compared.
// Measured: identical output both ways; plan.go 345 -> 343 lines after the rewrite.
// It PASSES at the reviewed head — it is an equivalence pin, not a bug repro.
//
// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_judge232_test.go
//	cd misc/gnopreview
//	go test -run TestJudge232ChangedFilesShape -v .        # head: map of maps
//	python3 - <<'PY'
//	import pathlib, subprocess
//	p = pathlib.Path("plan.go"); t = p.read_text()
//	for old, new in [
//	  ("\tchangedFiles := map[string]map[string]bool{}\n", "\tchangedFiles := map[string][]string{}\n"),
//	  ("\t\t\tif changedFiles[p.Path] == nil {\n\t\t\t\tchangedFiles[p.Path] = map[string]bool{}\n\t\t\t}\n\t\t\tchangedFiles[p.Path][path.Base(f)] = true\n",
//	   "\t\t\tchangedFiles[p.Path] = append(changedFiles[p.Path], path.Base(f))\n"),
//	  ("\t\t\tplan.ChangedFiles[r] = sortedKeys(f)\n",
//	   "\t\t\tsort.Strings(f)\n\t\t\tplan.ChangedFiles[r] = slices.Compact(f)\n"),
//	]:
//	    assert t.count(old) == 1, old[:50]
//	    t = t.replace(old, new)
//	p.write_text(t); subprocess.run(["gofmt","-w","plan.go"])
//	PY
//	wc -l plan.go                                          # 343
//	go test -run TestJudge232ChangedFilesShape -v . && go test ./...
//	git checkout -- plan.go && rm zz_judge232_test.go
//
// Observed, both trees:
//
//	ChangedFiles={"gno.land/r/demo/a":["b.gno","gnomod.toml","z.gno"]}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJudge232ChangedFilesShape(t *testing.T) {
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
	write("examples/gno.land/r/demo/a/gnomod.toml", "module = \"gno.land/r/demo/a\"\n")
	write("examples/gno.land/r/demo/a/z.gno", "package a\n")
	write("examples/gno.land/r/demo/a/b.gno", "package a\n")
	write("examples/gno.land/r/demo/a/b_test.gno", "package a\n")
	changed := []string{
		"examples/gno.land/r/demo/a/z.gno",
		"examples/gno.land/r/demo/a/b.gno",
		"examples/gno.land/r/demo/a/b.gno", // duplicate entry in the changed list
		"examples/gno.land/r/demo/a/gnomod.toml",
		"examples/gno.land/r/demo/a/b_test.gno", // dropped by renderRelevant
	}
	plan, err := BuildPlan(root, changed, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(plan.ChangedFiles)
	t.Logf("ChangedFiles=%s", b)
}
