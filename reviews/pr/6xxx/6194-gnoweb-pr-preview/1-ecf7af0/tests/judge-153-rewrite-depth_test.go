package main

// Asserts rewrite()'s own `up` derivation: a published link must resolve from
// the page's own directory. Measured at ecf7af0f: passes at head, fails on
// crawl.go:351 depth -> depth+1, a mutation the PR's suite reports as `ok`.
//
/* Run: from a gno checkout:
gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f
curl -fsSL -o misc/gnopreview/judge-153-rewrite-depth_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-153-rewrite-depth_test.go
cd misc/gnopreview && go test -vet=off -count=1 -run TestJudge153 .
rm judge-153-rewrite-depth_test.go
*/

import (
	"path"
	"strings"
	"testing"
)

func TestJudge153RewriteResolvesFromThePageDir(t *testing.T) {
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{
		"/r/a/b": {File: "r/a/b/index.html"},
	}}
	for _, tc := range []struct{ file, href, resolves string }{
		{"r/x/y/index.html", "/r/a/b", "r/a/b"},
		{"r/x/y/index.html", "/public/css/main.css", "public/css/main.css"},
		{"r/x/y/_t/source/index.html", "/r/a/b", "r/a/b"},
		{"r/x/y/_t/source/index.html", "/public/js/controller-abc.js", "public/js/controller-abc.js"},
	} {
		body := `<!doctype html><html><head></head><body><a href="` + tc.href + `">x</a></body></html>`
		got := c.rewrite(&page{File: tc.file, Body: body})
		i := strings.Index(got, `href="`)
		if i < 0 {
			t.Fatalf("%s: no href survived the rewrite: %s", tc.file, got)
		}
		rel := got[i+len(`href="`):]
		rel = rel[:strings.Index(rel, `"`)]
		// Where the published page actually points, from its own directory.
		resolved := path.Join(path.Dir(tc.file), rel)
		if resolved != tc.resolves {
			t.Errorf("page %s: %q rewritten to %q, which resolves to %q; want %q — the published link 404s",
				tc.file, tc.href, rel, resolved, tc.resolves)
		}
	}
}
