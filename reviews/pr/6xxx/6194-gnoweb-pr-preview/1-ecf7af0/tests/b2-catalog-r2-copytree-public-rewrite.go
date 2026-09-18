// b2-catalog-r2-copytree-public-rewrite.go — invariant catalog, unproven-bound class.
//
// Claim under test: copyTree (misc/gnopreview/crawl.go:547) relativises the
// assets by a plain strings.ReplaceAll of the literal "/public/", and Write
// (crawl.go:314) only prints a stderr line when that replaced nothing. Nothing
// in crawl_test.go exercises copyTree or Write at all, so the rewrite the whole
// preview's CSS and controller loading depends on is pinned by no test.
//
// This measures the margin: how many "/public/" literals the shipped asset tree
// actually carries, and what the one in public/js/index.js becomes.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_copytree_test.go
//	cd misc/gnopreview && go test -run TestCopyTreePublicRewrite -v ./...
//	rm misc/gnopreview/zz_copytree_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyTreePublicRewrite(t *testing.T) {
	src := filepath.Join("..", "..", "gno.land", "pkg", "gnoweb", "public")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("asset tree not found: %v", err)
	}

	// How many literals the rewrite has to find for Write's canary to stay quiet.
	before := 0
	var carriers []string
	_ = filepath.Walk(src, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if n := strings.Count(string(b), "/public/"); n > 0 {
			before += n
			rel, _ := filepath.Rel(src, p)
			carriers = append(carriers, rel)
		}
		return nil
	})
	t.Logf("assets carrying an absolute /public/ reference: %v (total literals=%d)", carriers, before)

	dst := filepath.Join(t.TempDir(), "public")
	n, err := copyTree(src, dst)
	if err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	t.Logf("copyTree reported %d rewrite(s)", n)
	if n != before {
		t.Fatalf("copyTree rewrote %d of %d literals", n, before)
	}

	// The single carrier today: public/js/index.js loads controllers by
	// dynamic import(), so the rewritten specifier resolves against the
	// module's own URL, <out>/public/js/index.js.
	b, err := os.ReadFile(filepath.Join(dst, "js", "index.js"))
	if err != nil {
		t.Fatalf("read rewritten index.js: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "/public/") {
		t.Fatal("an absolute /public/ survived the rewrite")
	}
	i := strings.Index(s, "js/controller-")
	if i < 0 {
		t.Fatal("controller path literal disappeared")
	}
	got := s[max(0, i-6) : i+len("js/controller-")]
	t.Logf("controller loader literal after rewrite: %q", got)
	if !strings.Contains(got, "../js/controller-") {
		t.Fatalf("controller path is %q, not ../js/controller-", got)
	}
	// ../js/controller-X.js resolved against <out>/public/js/index.js
	// is <out>/public/js/controller-X.js, which exists:
	if _, err := os.Stat(filepath.Join(dst, "js", "controller-copy.js")); err != nil {
		t.Fatalf("resolved controller target missing: %v", err)
	}
	t.Log("rewrite is correct today, and no test in the package pins it")
}
