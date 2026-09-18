// Equivalence witness for deleting Crawler.order from misc/gnopreview/crawl.go.
//
// Claim: c.order (crawl.go:70, appended at :144, ranged at :327) duplicates
// c.pages' key set. It exists only to fix writePages' iteration order, and
// every URL it holds is a key of c.pages. Iterating the sorted keys instead
// deletes the field, the append and the reset that Run() forgets.
//
// This test runs a crawl and writes the tree, then pins every output path and
// the sha256 of every file. It passes identically on head and on the rewrite,
// which is the equivalence: write order is not observable in the result.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/order_redundant_test.go
//	cd misc/gnopreview && go test -run TestWriteTreeIsOrderIndependent -v ./...
//
// Then apply the rewrite and re-run the same command:
//
//	crawl.go: delete the `order []string` field and the `c.order = append(...)`
//	line; writePages iterates `slices.Sorted(maps.Keys(c.pages))`; add "maps"
//	to the imports.
//
// Both runs print the same file list and the same digests.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestWriteTreeIsOrderIndependent(t *testing.T) {
	body := map[string]string{
		"/r/x/y":        `<html><head></head><body><a href="/r/x/y$source">src</a><a href="/r/x">dir</a><a href="/r/x/y:p/about">args</a></body></html>`,
		"/r/x/y$source": `<html><head></head><body><a href="/r/x/y">back</a></body></html>`,
		"/r/x/y$help":   `<html><head></head><body></body></html>`,
		"/r/x/y:p/about": `<html><head></head><body><a href="/r/x/y$source">src</a></body></html>`,
		"/r/x":          `<html><head></head><body><a href="/r/x/y">y</a></body></html>`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := body[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(b))
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	dir := t.TempDir()
	// assets == "" writes the pages alone, which is the path under test.
	if err := c.Write(dir, ""); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var got []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		sum := sha256.Sum256(b)
		got = append(got, fmt.Sprintf("%s %s", filepath.ToSlash(rel), hex.EncodeToString(sum[:4])))
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	sort.Strings(got)
	t.Logf("tree:\n%s", strings.Join(got, "\n"))

	want := []string{
		"r/x/index.html",
		"r/x/y/_a/p-about-e71ffeb2/index.html",
		"r/x/y/_t/help/index.html",
		"r/x/y/_t/source/index.html",
		"r/x/y/index.html",
	}
	var paths []string
	for _, g := range got {
		paths = append(paths, strings.Fields(g)[0])
	}
	if strings.Join(paths, "|") != strings.Join(want, "|") {
		t.Fatalf("paths %v, want %v", paths, want)
	}
}
