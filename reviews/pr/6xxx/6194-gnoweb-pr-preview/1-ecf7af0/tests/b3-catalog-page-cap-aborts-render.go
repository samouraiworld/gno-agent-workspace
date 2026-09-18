// Repro for: reaching -max-pages aborts the whole render instead of
// truncating it, so the pull request gets a red check and no preview.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_pagecap_test.go
//	cd misc/gnopreview && go test -run TestPageCap -v .
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestPageCapDiscardsAUsableSnapshot shows that Crawler.Run returns an error
// when MaxPages is reached, that the pages crawled up to that point are a
// complete, writable snapshot, and that main.go's `if err := c.Run(); err !=
// nil { return err }` therefore throws away work that could have been
// published with a "capped" note, the way plan.Dropped is.
func TestPageCapDiscardsAUsableSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head></head><body>" + r.URL.Path + "</body></html>"))
	}))
	defer srv.Close()

	realms := []string{"gno.land/r/demo/a", "gno.land/r/demo/b", "gno.land/r/demo/c"}
	c := &Crawler{Base: srv.URL, Realms: realms, MaxPages: 5, Live: "https://gno.land"}

	err := c.Run()
	if err == nil {
		t.Fatalf("Run() with MaxPages=5 over %d seeds returned nil; want the cap error", len(c.Seeds()))
	}
	t.Logf("seeds=%d captured=%d err=%v", len(c.Seeds()), len(c.pages), err)

	// The snapshot the caller is about to discard is complete enough to publish.
	dir := t.TempDir()
	if werr := c.Write(dir, ""); werr != nil {
		t.Fatalf("Write of the partial snapshot failed: %v", werr)
	}
	n := 0
	filepath.WalkDir(dir, func(p string, d os.DirEntry, e error) error {
		if e == nil && !d.IsDir() && filepath.Base(p) == "index.html" {
			n++
		}
		return nil
	})
	t.Logf("the discarded snapshot holds %d rendered page(s) under %s", n, dir)
	if n != len(c.pages) {
		t.Fatalf("wrote %d pages, captured %d", n, len(c.pages))
	}
}

// TestRealmCapDegrades is the contrast: the sibling cap records the overflow
// in Plan.Dropped and keeps going.
func TestRealmCapDegrades(t *testing.T) {
	p := &Plan{Realms: []string{"a", "b"}, Dropped: 7}
	if p.Empty() {
		t.Fatal("a capped plan must still render")
	}
	t.Logf("realm cap: Dropped=%d, render continues", p.Dropped)
}

// TestChangedFilesBypassTheFileBudget measures how the page count grows: for a
// realm listed in ChangedFiles every touched file gets its own $source page,
// with no per-realm budget, so a sweep over examples/ multiplies pages by the
// files it touched and walks into the 400-page cap above.
func TestChangedFilesBypassTheFileBudget(t *testing.T) {
	const files = 20
	realm := "gno.land/r/demo/a"
	var changed []string
	body := "<html><head></head><body>"
	for i := range files {
		name := fmt.Sprintf("f%02d.gno", i)
		changed = append(changed, name)
		body += fmt.Sprintf(`<a href="/r/demo/a$source&file=%s">%s</a>`, name, name)
	}
	body += "</body></html>"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := &Crawler{
		Base: srv.URL, Realms: []string{realm}, MaxPages: 0, Live: "https://gno.land",
		ChangedFiles: map[string][]string{realm: changed},
		FileBudget:   GnowebFileBudget,
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	t.Logf("one realm, %d changed files -> %d pages captured", files, len(c.pages))
	if len(c.pages) < files {
		t.Fatalf("expected at least %d pages, got %d", files, len(c.pages))
	}
}
