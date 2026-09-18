// Claims probes for gnolang/gno#6194, bundle misc/gnopreview/crawl_test.go.
// Each probe runs one claim the diff writes about itself and prints what the
// code actually does. Nothing here asserts: the log lines are the evidence.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_probe_claims_test.go
//	cd misc/gnopreview && go test -run TestProbe -v .
//
// Recorded at ecf7af0f2 (go1.25.9):
//
//	head.html ships: <meta name="robots" content="index, follow" />
//	as shipped today       -> 1 robots meta(s), index,follow survives: false
//	attributes reordered   -> 2 robots meta(s), index,follow survives: true
//	single quotes          -> 2 robots meta(s), index,follow survives: true
//	newline between attrs  -> 1 robots meta(s), index,follow survives: false
//	seeds: [/r/gnoland/home /r/gnoland/home$source /r/gnoland/home$help
//	        /r/demo/counter /r/demo/counter$source /r/demo/counter$help /r/demo /r/gnoland]
//	inScope("/r/gnoland/home/") = false, urlToFile -> "r/gnoland/home/_dir/index.html"
//	inScope("/r/") = false, urlToFile -> "r/_dir/index.html"
//	inScope("/") = false, urlToFile -> "_root/index.html"
//	urlToFile("/r/x/y/../../../../etc/passwd") = "../etc/passwd/index.html"
//	urlToFile("/r/x/y/../z") = "r/x/z/index.html"
//	"/r/x/dep$file=&source" inScope=true chargeFile=true -> "r/x/dep/_t/file-source-5e3ba8d6/index.html"
//	budget after: map[]
//	$source overview -> "r/x/dep/_t/source/index.html"
package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var anyRobots = regexp.MustCompile(`(?is)<meta[^>]*robots[^>]*>`)

// A. TestRewriteAddsNoindex asserts the replacement against a hand-typed copy
// of gnoweb's tag ("gnoweb's own layout ships this on every page"). Nothing
// ties that copy to components/layouts/head.html, so the claim holds only for
// the exact spelling that file carries today.
func TestProbeRobotsShapeCoupling(t *testing.T) {
	b, err := os.ReadFile("../../gno.land/pkg/gnoweb/components/layouts/head.html")
	if err != nil {
		t.Fatal(err)
	}
	live := ""
	for _, l := range strings.Split(string(b), "\n") {
		if strings.Contains(l, "robots") {
			live = strings.TrimSpace(l)
		}
	}
	t.Logf("head.html ships: %s", live)
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	for _, tc := range []struct{ name, tag string }{
		{"as shipped today", live},
		{"attributes reordered", `<meta content="index, follow" name="robots" />`},
		{"single quotes", `<meta name='robots' content='index, follow' />`},
		{"newline between attrs", "<meta name=\"robots\"\n    content=\"index, follow\" />"},
	} {
		body := `<html><head>` + tc.tag + `<title>x</title></head></html>`
		got := c.rewrite(&page{File: "r/x/index.html", Body: body})
		t.Logf("%-22s -> %d robots meta(s), index,follow survives: %v",
			tc.name, len(anyRobots.FindAllString(got, -1)), strings.Contains(got, "index, follow"))
	}
}

// B. splitURL, canonicalURL and urlToFile all keep the trailing slash so the
// listing page cannot overwrite the render page. inScope drops every
// trailing-slash path and no seed carries one, so the _dir branch never runs.
func TestProbeListingBranchUnreachable(t *testing.T) {
	c := &Crawler{
		Realms:     []string{"gno.land/r/gnoland/home", "gno.land/r/demo/counter"},
		FileBudget: GnowebFileBudget,
	}
	seeds := c.Seeds()
	t.Logf("seeds: %v", seeds)
	for _, s := range seeds {
		if strings.HasSuffix(s, "/") {
			t.Errorf("seed %q ends in a slash", s)
		}
	}
	for _, u := range []string{"/r/gnoland/home/", "/r/gnoland/", "/r/", "/"} {
		t.Logf("inScope(%q) = %v, urlToFile -> %q", u, c.inScope(u), urlToFile(u))
	}
}

// C. TestURLToFileIsSafe's header claims no output may escape its realm
// directory. urlToFile slugs the render arguments and the tab query; the base
// is path.Join'd unchecked, so the guarantee lives in inScope's exact realm
// match rather than in urlToFile.
func TestProbeURLToFileBaseNotCleaned(t *testing.T) {
	for _, u := range []string{"/r/x/y/../../../../etc/passwd", "/r/x/y/../z", "/r/x/.."} {
		t.Logf("urlToFile(%q) = %q", u, urlToFile(u))
	}
}

// D. view_directory.go emits "<pkgpath>$source&file=" with an empty value.
// inScope admits it, chargeFile calls it "not a per-file page" and skips the
// budget, and gnoweb serves the source overview for it: a second copy of the
// $source page under a file-shaped slug.
func TestProbeEmptyFileValue(t *testing.T) {
	c := &Crawler{Realms: []string{"gno.land/r/x/dep"}, FileBudget: GnowebFileBudget}
	u := canonicalURL("/r/x/dep$source&file=")
	t.Logf("%q inScope=%v chargeFile=%v -> %q", u, c.inScope(u), c.chargeFile(u), urlToFile(u))
	t.Logf("budget after: %v", c.fileBudget)
	t.Logf("$source overview -> %q", urlToFile("/r/x/dep$source"))
}
