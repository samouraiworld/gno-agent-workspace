// Asserts that LoadPkgs enumerates examples/gno.land twice: filepath.WalkDir
// delivers every entry to the callback, which throws them away and re-reads the
// same directory with os.ReadDir.
// Measured at this head: packages=145 walk-callback-entries=1531 callback ReadDir calls=304.
// The counters do not exist in plan.go; the patch below adds them.
//
// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_count_test.go
//	cd misc/gnopreview
//	python3 - <<'PY'
//	import pathlib
//	p = pathlib.Path("plan.go"); t = p.read_text()
//	for old, new in [
//	  ("// LoadPkgs walks examples/", "var zzDirs, zzEntries int\n\n// LoadPkgs walks examples/"),
//	  ("\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif !d.IsDir()",
//	   "\t\tzzEntries++\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif !d.IsDir()"),
//	  ("\t\tents, err := os.ReadDir(p)", "\t\tzzDirs++\n\t\tents, err := os.ReadDir(p)"),
//	]:
//	    assert t.count(old) == 1, old[:50]
//	    t = t.replace(old, new)
//	p.write_text(t)
//	PY
//	go test -run TestZZCountsDirectoryReads -v .
//	git checkout -- plan.go && rm zz_count_test.go
//
// Observed:
//
//	zz_count_test.go:38: packages=145 walk-callback-entries=1531 callback ReadDir calls=304
//
// Cross-check, no patch needed: os.walk over examples/gno.land counts
// 304 directories and 1530 entries, so the walk already read all 304 before the
// callback re-read them.

package main

import "testing"

func TestZZCountsDirectoryReads(t *testing.T) {
	zzDirs, zzEntries = 0, 0
	pkgs, err := LoadPkgs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// zzEntries is callback invocations (1 + every entry the walk read);
	// zzDirs is the os.ReadDir calls the callback issues on top of that.
	t.Logf("packages=%d walk-callback-entries=%d callback ReadDir calls=%d", len(pkgs), zzEntries, zzDirs)
}
