// b9-tests-unpinned-crawl-invariants.go — the three assertions that
// misc/gnopreview/crawl_test.go leaves out, for gnolang/gno PR 6194 at
// ecf7af0f29abe4737a52803d672bc5a33c17cc60.
//
// Every test here passes at that head. Each of the three mutations below leaves
// the PR's own crawl_test.go entirely green and turns exactly one of these red,
// which is what makes the corresponding assertion missing rather than redundant.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno.git gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_unpinned_test.go
//	cd misc/gnopreview && go test ./...        # ok, everything green
//
// 1. maxSlugLen — crawl.go says the constant "keeps a long render argument from
//    producing a path the filesystem or the static host rejects", and
//    TestURLToFileIsSafe bounds each segment by maxSlugLen+16, so no value of
//    the constant can fail it:
//
//	sed -i 's/^const maxSlugLen = 64/const maxSlugLen = 400/' crawl.go
//	go test -run TestURLToFileIsSafe ./...          # ok   — the PR's test
//	go test -run TestSlugSegmentFitsNAMEMAX ./...   # FAIL — 400 > NAME_MAX 255
//	git checkout -- crawl.go
//
// 2. rewrite's ../ depth — TestMapURL hands mapURL a hand-written up string
//    ("../../../"), and rewrite is the only place that computes one:
//
//	sed -i 's|up := strings.Repeat("../", depth)|up := strings.Repeat("../", depth+1)|' crawl.go
//	go test ./...                                          # ok — whole PR suite
//	go test -run TestRewriteResolvesFromThePageDir ./...    # FAIL
//	git checkout -- crawl.go
//
// 3. setNoindex's placement — TestRewriteAddsNoindex counts the tag but never
//    says where it landed, so the <head> branch can be dead:
//
//	sed -i 's|if loc := headRe.FindStringIndex(body); loc != nil {|if loc := headRe.FindStringIndex(body); false \&\& loc != nil {|' crawl.go
//	go test -run TestRewriteAddsNoindex ./...          # ok   — the PR's test
//	go test -run TestNoindexLandsInsideHead ./...      # FAIL — tag precedes <!doctype>
//	git checkout -- crawl.go
package main

import (
	"path"
	"regexp"
	"strings"
	"testing"
)

// nameMax is POSIX NAME_MAX on ext4, tmpfs and every filesystem GitHub's
// runners and Pages use. It is a literal on purpose: an assertion written in
// terms of maxSlugLen cannot fail when maxSlugLen is what moved.
const nameMax = 255

func TestSlugSegmentFitsNAMEMAX(t *testing.T) {
	t.Parallel()
	for _, u := range []string{
		"/r/x/y:" + strings.Repeat("z", 400),
		"/r/x/y$file=" + strings.Repeat("q", 400) + ".gno&source",
		"/r/x/y:" + strings.Repeat("a/", 200),
	} {
		for seg := range strings.SplitSeq(urlToFile(u), "/") {
			if len(seg) > nameMax {
				t.Errorf("urlToFile(<%d-char url>) segment is %d bytes; NAME_MAX is %d, so MkdirAll returns ENAMETOOLONG and Crawler.Write fails the render",
					len(u), len(seg), nameMax)
			}
		}
	}
}

var hrefRe = regexp.MustCompile(`(?:href|src)="([^"]*)"`)

// TestRewriteResolvesFromThePageDir exercises the one computation the static
// tree depends on: every link rewrite() emits is relative to the directory
// holding that page's own index.html, so joining it back onto that directory
// has to land on the target's repo-relative path.
func TestRewriteResolvesFromThePageDir(t *testing.T) {
	t.Parallel()
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{
		"/r/a/b": {File: "r/a/b/index.html"},
	}}
	body := `<html><head></head><body>` +
		`<a href="/r/a/b">x</a><img src="/public/img/logo.svg"></body></html>`

	for _, file := range []string{
		"r/x/y/index.html",
		"r/x/y/_t/source/index.html",
		"r/x/y/_a/p-hello-13cc55eb/_t/source/index.html",
		"_root/index.html",
		"before/r/x/y/index.html",
	} {
		got := c.rewrite(&page{File: file, Body: body})
		dir := path.Dir(file)
		want := map[string]string{"/r/a/b": "r/a/b", "/public/img/logo.svg": "public/img/logo.svg"}
		var raws []string
		for _, m := range hrefRe.FindAllStringSubmatch(got, -1) {
			raws = append(raws, m[1])
		}
		if len(raws) != 2 {
			t.Fatalf("%s: rewrite emitted %d attributes, want 2: %s", file, len(raws), got)
		}
		for i, key := range []string{"/r/a/b", "/public/img/logo.svg"} {
			resolved := path.Clean(path.Join(dir, raws[i]))
			if resolved != want[key] {
				t.Errorf("page %s: %q rewritten to %q, which resolves to %q; want %q — the published link 404s",
					file, key, raws[i], resolved, want[key])
			}
		}
	}
}

// TestNoindexLandsInsideHead pins where setNoindex puts the tag. A meta before
// <!doctype html> is content before the doctype, which puts the whole published
// page into quirks mode.
func TestNoindexLandsInsideHead(t *testing.T) {
	t.Parallel()
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	for _, tc := range []struct{ name, body string }{
		{"normal head", `<!doctype html><html><head><title>x</title></head><body>hi</body></html>`},
		{"head with attrs", `<html><head lang="en"><title>x</title></head></html>`},
		{"gnoweb's index,follow", `<html><head><meta name="robots" content="index, follow" /><title>x</title></head></html>`},
	} {
		got := c.rewrite(&page{File: "r/x/index.html", Body: tc.body})
		low := strings.ToLower(got)
		open, closing := strings.Index(low, "<head"), strings.Index(low, "</head>")
		tag := strings.Index(got, noindexTag)
		if open < 0 || closing < 0 {
			t.Fatalf("%s: rewrite lost the head element: %s", tc.name, got)
		}
		if tag < open || tag > closing {
			t.Errorf("%s: noindex tag at %d is outside <head> (%d..%d): %s", tc.name, tag, open, closing, got)
		}
		if i := strings.Index(low, "<!doctype"); i > 0 {
			t.Errorf("%s: %d bytes precede the doctype, which puts the page in quirks mode: %s", tc.name, i, got)
		}
	}
}
