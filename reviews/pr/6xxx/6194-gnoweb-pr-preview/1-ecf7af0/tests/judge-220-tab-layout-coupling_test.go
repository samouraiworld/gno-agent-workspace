// Asserts that the source/help links comment.go builds resolve to the files
// urlToFile writes. It PASSES at ecf7af0f2 — the finding is that no test in the
// package makes this assertion, so the two spellings of the `_t/<tab>/` layout
// can drift apart silently.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_judge220_test.go
//	cd misc/gnopreview && go test -run TestTabLinksMatchCrawlerLayout -v .   # PASS
//
//	# now rename the tab prefix the way its owner would, goldens included:
//	sed -i 's|"_t", slug(query)|"_tab", slug(query)|' crawl.go
//	sed -i 's|_t/|_tab/|g' crawl_test.go
//	go test .                                        # ok — the whole suite is green
//	go test -run TestTabLinksMatchCrawlerLayout -v .  # FAIL — every tab link 404s
//	git checkout -- crawl.go crawl_test.go && rm zz_judge220_test.go

package main

import (
	"path"
	"strings"
	"testing"
)

// The comment's source/help links must land on the files the crawler writes.
func TestTabLinksMatchCrawlerLayout(t *testing.T) {
	const base = "https://e.test/pr-1"
	for _, pkg := range []string{"gno.land/r/x/leaf", "gno.land/r/gnoland/home"} {
		u := urlOf(pkg)
		got := tabs(base, pkg)
		for _, q := range []string{"source", "help"} {
			// what the crawler wrote for this realm's tab page
			want := base + "/" + path.Dir(urlToFile(u+"$"+q)) + "/"
			if !strings.Contains(got, "("+want+")") {
				t.Errorf("%s: tabs() has no link to %q\n  tabs() = %s", pkg, want, got)
			}
		}
	}
}
