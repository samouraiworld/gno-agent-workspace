// Equivalence harness for the crawl.go refactor candidates on PR 6194.
//
// It carries the ORIGINAL bodies of Seeds, links, inScope and chargeFile as
// orig*, and asserts the package's own versions agree with them. Dropped into
// the unmodified head it passes trivially; dropped into a tree carrying the
// proposed rewrites it is what proves the shorter forms behave identically.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//	cp <this file> misc/gnopreview/
//	cd misc/gnopreview && go test ./... -run Refactor -v   # green: baseline
//	# then apply the rewrites and re-run:
//	patch -p1 < <b2-refactor-crawl.patch>
//	gofmt -l . && go vet ./... && go test ./... -run Refactor -v
//
// Mutation that must turn it red (proves the harness is not vacuous): delete
// the sort from links' return, e.g. return the map keys unsorted.
package main

import (
	"fmt"
	"html"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// --- original bodies, verbatim from ecf7af0 --------------------------------

func origSeeds(c *Crawler) []string {
	seen := map[string]bool{}
	var out []string
	add := func(u string) {
		if !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	dirs := map[string]bool{}
	for _, r := range c.Realms {
		u := urlOf(r)
		add(u)
		if c.RenderOnly {
			continue
		}
		add(u + "$source")
		add(u + "$help")
		for d := path.Dir(u); strings.Count(d, "/") >= 2; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	if !c.RenderOnly {
		for _, d := range sortedKeys(dirs) {
			add(d)
		}
	}
	return out
}

func origLinks(body string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range attrRe.FindAllStringSubmatch(body, -1) {
		raw := html.UnescapeString(m[2])
		if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/public/") {
			continue
		}
		p, _, _ := strings.Cut(raw, "#")
		p = canonicalURL(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func origInScope(c *Crawler, p string) bool {
	if c.RenderOnly {
		return false
	}
	base, args, query := splitURL(p)
	if args != "" && query != "" {
		return false
	}
	if strings.HasSuffix(base, "/") {
		return false
	}
	realm := ""
	for _, r := range c.Realms {
		if base == urlOf(r) {
			realm = r
			break
		}
	}
	if realm == "" {
		return false
	}
	for part := range strings.SplitSeq(query, "&") {
		key, val, _ := strings.Cut(part, "=")
		if explosiveArgs[key] {
			return false
		}
		if key == "file" && !c.wantFile(realm, val) {
			return false
		}
	}
	return true
}

func origChargeFile(c *Crawler, u string) bool {
	base, _, query := splitURL(u)
	name := ""
	for part := range strings.SplitSeq(query, "&") {
		if k, v, _ := strings.Cut(part, "="); k == "file" {
			name = v
		}
	}
	if name == "" {
		return true
	}
	for _, r := range c.Realms {
		if base != urlOf(r) {
			continue
		}
		if _, listed := c.ChangedFiles[r]; listed {
			return true
		}
		if c.fileBudget == nil {
			c.fileBudget = map[string]int{}
		}
		if c.fileBudget[r] >= c.FileBudget {
			return false
		}
		c.fileBudget[r]++
		return true
	}
	return true
}

// --- equivalence -----------------------------------------------------------

// crawlers covers the shapes the tool builds: the head pass, the RenderOnly
// "before" pass, a realm nested under another realm (the only input for which
// the original's dedupe closure does any work), and a budgeted pass.
func crawlers() []*Crawler {
	return []*Crawler{
		{Realms: []string{"gno.land/r/gnoland/home", "gno.land/r/demo/counter"}, FileBudget: GnowebFileBudget},
		{Realms: []string{"gno.land/r/gnoland/home"}, RenderOnly: true},
		{Realms: []string{"gno.land/r/x", "gno.land/r/x/y"}},
		{Realms: []string{"gno.land/r/x/touched", "gno.land/r/x/untouched"},
			ChangedFiles: map[string][]string{"gno.land/r/x/touched": {"a.gno"}},
			FileBudget:   GnowebFileBudget},
		{Realms: []string{"gno.land/r/deep/a/b/c/d"}},
	}
}

func TestRefactorSeedsEquivalent(t *testing.T) {
	t.Parallel()
	for i, c := range crawlers() {
		got, want := c.Seeds(), origSeeds(c)
		// Seeds only ever feeds Run's queue, which is deduped by `visited` and
		// by seedSet; compare as sets, then report an order change separately.
		if !reflect.DeepEqual(set(got), set(want)) {
			t.Errorf("crawler %d: Seeds() = %v; original = %v", i, got, want)
		}
		if !reflect.DeepEqual(dedupe(got), want) {
			t.Errorf("crawler %d: Seeds() order/dedupe differs: %v vs %v", i, dedupe(got), want)
		}
	}
}

func TestRefactorLinksEquivalent(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`<a href="/r/x/y$source&amp;file=a.gno">s</a>
		 <a href="/r/x/y#frag">f</a>
		 <a href="https://gno.land/out">ext</a>
		 <a href="#local">l</a>
		 <a href="/">root</a>
		 <a href="/#top">roothash</a>
		 <a href="//cdn.example/x">proto-rel</a>
		 <link href="/public/main.css?v=1">
		 <script src="/public/js/index.js"></script>`,
		`<a href="/r/x/y$file=a.gno&amp;source">a</a><a href="/r/x/y$source&amp;file=a.gno">b</a>`,
		``,
		`<a href="">empty</a><a href="/">slash</a>`,
	} {
		// eq, not DeepEqual: sortedKeys returns an allocated empty slice where
		// the original returned nil. Every caller only ranges over the result.
		if got, want := links(body), origLinks(body); !eq(got, want) {
			t.Errorf("links(%q) = %v; original = %v", body, got, want)
		}
	}
}

// urls covers every branch inScope and chargeFile take: render, tabs, the
// listing slash, render args, the explosive keys, per-file pages inside and
// outside the changed set, a realm that is not selected, and a repeated file=.
func urls() []string {
	return []string{
		"/r/gnoland/home", "/r/gnoland/home/", "/r/gnoland/home$source",
		"/r/gnoland/home$help", "/r/gnoland/home:p/x", "/r/gnoland/home:p/x$source",
		"/r/gnoland/home$state", "/r/gnoland/home$state&oid=deadbeef",
		"/r/gnoland/home$download&file=home.gno", "/r/gnoland/home$help&func=Admin",
		"/r/gnoland/home$source&file=home.gno", "/r/gnoland/home$source&file=b.gno",
		"/r/gnoland/home$source&file=a.gno&file=b.gno",
		"/r/x/touched$source&file=a.gno", "/r/x/touched$source&file=z.gno",
		"/r/x/untouched$source&file=f0.gno", "/r/x/untouched$source&file=f1.gno",
		"/r/x/untouched$source&file=f2.gno", "/r/x/y", "/r/x/y$source&file=q.gno",
		"/r/other/realm$source&file=a.gno", "/r/", "/", "/u/g1abc",
	}
}

func TestRefactorInScopeEquivalent(t *testing.T) {
	t.Parallel()
	for i := range crawlers() {
		// Fresh pair per crawler: inScope reads the budget map, so the two must
		// never share one.
		a, b := crawlers()[i], crawlers()[i]
		for _, u := range urls() {
			if got, want := a.inScope(u), origInScope(b, u); got != want {
				t.Errorf("crawler %d: inScope(%q) = %v; original = %v", i, u, got, want)
			}
		}
	}
}

func TestRefactorChargeFileEquivalent(t *testing.T) {
	t.Parallel()
	for i := range crawlers() {
		a, b := crawlers()[i], crawlers()[i]
		// Three passes: the budget is stateful, so equivalence has to hold
		// after it is exhausted too.
		for pass := range 3 {
			for _, u := range urls() {
				if got, want := a.chargeFile(u), origChargeFile(b, u); got != want {
					t.Errorf("crawler %d pass %d: chargeFile(%q) = %v; original = %v", i, pass, u, got, want)
				}
			}
		}
		if !reflect.DeepEqual(a.fileBudget, b.fileBudget) {
			t.Errorf("crawler %d: budget map %v; original %v", i, a.fileBudget, b.fileBudget)
		}
	}
}

func TestRefactorMapURLEmptyHref(t *testing.T) {
	t.Parallel()
	// The `raw == ""` clause the rewrite drops: HasPrefix("", "/") is false, so
	// the next clause already returns "" untouched.
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	if got := c.mapURL("", "../"); got != "" {
		t.Errorf(`mapURL("") = %q; want ""`, got)
	}
}

func eq(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func set(s []string) map[string]bool {
	m := map[string]bool{}
	for _, v := range s {
		m[v] = true
	}
	return m
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

var _ = fmt.Sprintf
