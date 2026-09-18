// b5-reach-r2-index-both-mode_test.go — gnolang/gno PR 6194, head ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// Index() in misc/gnopreview/comment.go renders the preview homepage — the page
// the PR comment's "Open the preview homepage" link points at — and no test in
// the pull request executes a single statement of it. Same for the Mode()=="both"
// branch of Comment() (comment.go:47-53) and for isSeed's membership line (161).
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cd misc/gnopreview
//	# 1. the coverage gap this file is about:
//	go test -coverprofile=/tmp/cover.out ./... && grep comment.go /tmp/cover.out | awk '$NF==0'
//	#    -> comment.go:166.40,168.29 ... 178.2,195.33  (all of Index, count 0)
//	#    -> comment.go:47.15,49.18 / 49.18,51.5 / 52.4,53.42  (the "both" branch)
//	#    -> comment.go:161.2,161.44  (isSeed's seed-membership check)
//	# 2. this file, the tests the gap leaves out:
//	cp <this file> ./b5_reach_r2_test.go && go test -run 'B5Reach' -v ./...
//
// Observed at that head: TestB5ReachIndexCountMatchesRows,
// TestB5ReachIndexHonorsCrawlerPrefix and TestB5ReachIndexTabLinksAreCaptured
// fail; TestB5ReachIndexSkipsUncapturedRealm and TestB5ReachCommentBothMode pass.
package main

import (
	"strings"
	"testing"
)

// crawlerWith builds a Crawler whose page set is exactly the given URLs, the
// way Crawler.Run leaves it: key = crawled URL, File = Prefix + urlToFile(u).
func crawlerWith(prefix string, urls ...string) *Crawler {
	c := &Crawler{Prefix: prefix, pages: map[string]*page{}}
	for _, u := range urls {
		f := urlToFile(u)
		if prefix != "" {
			f = prefix + "/" + f
		}
		c.pages[u] = &page{URL: u, File: f}
		c.order = append(c.order, u)
	}
	return c
}

// A realm whose render page the crawl never captured (fetch error or non-200:
// crawl.go logs and continues) must not get a row.
func TestB5ReachIndexSkipsUncapturedRealm(t *testing.T) {
	p := &Plan{Realms: []string{"gno.land/r/x/leaf", "gno.land/r/x/gone"}}
	got := Index(p, crawlerWith("", "/r/x/leaf", "/r/x/leaf$source", "/r/x/leaf$help"))
	if strings.Contains(got, "gno.land/r/x/gone") {
		t.Errorf("index lists a realm that was never captured:\n%s", got)
	}
}

// The headline count is what the reader trusts; it must count the rows the page
// actually carries, not the realms the plan intended.
func TestB5ReachIndexCountMatchesRows(t *testing.T) {
	p := &Plan{Realms: []string{"gno.land/r/x/leaf", "gno.land/r/x/gone"}}
	got := Index(p, crawlerWith("", "/r/x/leaf"))
	if rows := strings.Count(got, "<li>"); !strings.Contains(got, ">"+itoa(rows)+" realm(s) rendered") {
		t.Errorf("index counts %d realm(s) but carries %d row(s):\n%s", len(p.Realms), rows, got)
	}
}

// Index recomputes the on-disk path with urlToFile instead of reading the page's
// own File, so it drops the crawler's Prefix. The head crawler's Prefix is "",
// which is the only reason the links resolve today.
func TestB5ReachIndexHonorsCrawlerPrefix(t *testing.T) {
	p := &Plan{Realms: []string{"gno.land/r/x/leaf"}}
	c := crawlerWith("_before", "/r/x/leaf")
	got := Index(p, c)
	if want := c.pages["/r/x/leaf"].File; !strings.Contains(got, `href="_before/r/x/leaf/"`) {
		t.Errorf("index links a path the crawler never wrote; page lives at %q:\n%s", want, got)
	}
}

// Every row carries a source and a help link unconditionally, though the crawl
// skips those two pages on any fetch error or non-200 and carries on.
func TestB5ReachIndexTabLinksAreCaptured(t *testing.T) {
	p := &Plan{Realms: []string{"gno.land/r/x/leaf"}}
	got := Index(p, crawlerWith("", "/r/x/leaf")) // render captured, $source/$help not
	for _, dead := range []string{"_t/source/", "_t/help/"} {
		if strings.Contains(got, dead) {
			t.Errorf("index links %s for a realm whose page was never captured:\n%s", dead, got)
		}
	}
}

// Mode()=="both": gnoweb plus realm sources. No test in the pull request reaches
// this branch, so nothing pins the sentence, the homepage link or the seed filter.
func TestB5ReachCommentBothMode(t *testing.T) {
	p := &Plan{
		Gnoweb:        true,
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/gnoland/home", "gno.land/r/x/leaf"},
		Pairs:         []ShotPair{{Realm: "gno.land/r/x/leaf", After: "_shots/a.png", URL: "r/x/leaf/"}},
	}
	got := Comment(p, "https://example.test/pr-3", "3")
	for _, want := range []string{
		"changes **gnoweb** and realm sources",
		"**[Open the preview homepage](https://example.test/pr-3/)**",
		"**Changed realms (1)**",
		"_shots/a.png",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("both-mode comment missing %q\n---\n%s", want, got)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
