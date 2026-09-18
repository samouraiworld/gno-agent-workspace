// b5-refactor-index-parity_test.go — misc/gnopreview, gnolang/gno PR 6194, head
// ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// A realm gnodev fails to render is skipped by Crawler.Run (crawl.go: a non-200
// or errored seed gets a stderr line and no entry in c.pages). Index filters
// those realms out of its list but still counts len(p.Realms) in its header,
// and Comment links every planned realm with no such filter. This test pins
// that state as it is at head: it PASSES today, showing the index header
// claiming 2 renders over 1 row and the PR comment linking a page the snapshot
// does not contain. Pruning plan.Realms to the captured set in main.go after
// c.Run() makes the last two checks fail, which is the fix.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/
//	cd misc/gnopreview && go test -run TestUnrenderedRealmParity -v ./...
package main

import (
	"strings"
	"testing"
)

func TestUnrenderedRealmParity(t *testing.T) {
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf", "gno.land/r/x/broken"},
		Realms:        []string{"gno.land/r/x/leaf", "gno.land/r/x/broken"},
	}
	c := &Crawler{}
	c.pages = map[string]*page{"/r/x/leaf": nil} // /r/x/broken returned HTTP 500

	idx := Index(p, c)
	if strings.Contains(idx, "r/x/broken") {
		t.Errorf("index lists the unrendered realm:\n%s", idx)
	}
	if n := strings.Count(idx, "<li>"); n != 1 {
		t.Errorf("index rows = %d; want 1", n)
	}
	if !strings.Contains(idx, "<p>2 realm(s) rendered") {
		t.Errorf("index header no longer over-counts — fixed:\n%s", idx)
	}
	com := Comment(p, "https://preview.test/pr-7", "7")
	if !strings.Contains(com, "https://preview.test/pr-7/r/x/broken/") {
		t.Errorf("comment no longer links the unrendered realm — fixed")
	}
	t.Log("head: index says 2 rendered and lists 1; the comment links /r/x/broken/, which 404s in the snapshot")
}
