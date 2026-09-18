// misc/gnopreview/crawl.go:314 — the asset-relativization sentinel counts
// "/public/" across every .js and .css copyTree copies (crawl.go:573), while
// the comment beside it names one prefix in one file: js/index.js's hardcoded
// "/public/js/controller-". Measured on the shipped tree: exactly one
// "/public/" occurrence in gno.land/pkg/gnoweb/public, in js/index.js, so the
// sentinel tracks the invariant by coincidence. This fixture gives the tree a
// second, unrelated "/public/" reference and removes the controller prefix:
// the sentinel stays silent at the reviewed head.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp judge-180-asset-sentinel-scope_test.go misc/gnopreview/
//	cd misc/gnopreview && go test -run TestJudge180 -v .
//	# expect: FAIL — the controller prefix is gone and n != 0, so Write says nothing
//	cd ../.. && rm misc/gnopreview/judge-180-asset-sentinel-scope_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJudge180SentinelMissesControllerPrefixLoss builds an assets tree in which
// the invariant the comment names is broken (js/index.js no longer builds its
// import specifiers from "/public/js/controller-") while an unrelated
// stylesheet still carries a "/public/" URL.
func TestJudge180SentinelMissesControllerPrefixLoss(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "js", "index.js"),
		// the controller loader, rewritten to a bare relative specifier:
		// the exact regression the comment at crawl.go:311-313 promises to catch
		`await import("./js/controller-" + name + ".js");`)
	mustWrite(t, filepath.Join(src, "main.css"),
		// any stylesheet that gains one absolute asset URL keeps the count above zero
		`.logo { background: url(/public/imgs/gnoland.svg); }`)

	n, err := copyTree(src, filepath.Join(t.TempDir(), "public"))
	if err != nil {
		t.Fatalf("copyTree: %v", err)
	}

	// Write warns only on n == 0 (crawl.go:314-316), so n is the whole signal.
	t.Logf("copyTree rewrites=%d", n)
	if n != 0 {
		t.Errorf("IS:     rewrites=%d with js/index.js's %q gone — the n==0 sentinel stays silent and the snapshot ships without its interactive controls", n, "/public/js/controller-")
	}
	// SHOULD: the sentinel counts the string the comment names, in the file it
	// names, so losing the controller prefix is what makes the count zero.
	// if n != 0 { t.Errorf(...) }  // uncomment once the guard is scoped
}

// TestJudge180ShippedTreeHasOneOccurrence pins the measurement the claim rests
// on: the whole-tree count equals js/index.js's count today.
func TestJudge180ShippedTreeHasOneOccurrence(t *testing.T) {
	root := filepath.Join("..", "..", "gno.land", "pkg", "gnoweb", "public")
	total, perFile := 0, map[string]int{}
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		switch filepath.Ext(p) {
		case ".js", ".css":
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if c := strings.Count(string(b), "/public/"); c > 0 {
				rel, _ := filepath.Rel(root, p)
				perFile[filepath.ToSlash(rel)] = c
				total += c
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	t.Logf("total=%d perFile=%v", total, perFile)
	if total != perFile["js/index.js"] || total != 1 {
		t.Errorf("measurement moved: total=%d perFile=%v", total, perFile)
	}
}

func mustWrite(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
