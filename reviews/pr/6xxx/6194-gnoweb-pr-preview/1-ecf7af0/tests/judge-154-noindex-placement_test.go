package main

// Asserts where setNoindex puts the robots meta, not just that it is there.
// Measured at ecf7af0f: passes at head, fails once crawl.go:374's <head>
// branch is disabled, a mutation the PR's TestRewriteAddsNoindex reports `ok`.
//
/* Run: from a gno checkout:
gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f
curl -fsSL -o misc/gnopreview/judge-154-noindex-placement_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-154-noindex-placement_test.go
cd misc/gnopreview && go test -vet=off -count=1 -run TestJudge154 .
rm judge-154-noindex-placement_test.go
*/

import (
	"strings"
	"testing"
)

func TestJudge154NoindexLandsInsideHead(t *testing.T) {
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	for _, tc := range []struct{ name, body string }{
		{"normal head", `<!doctype html><html><head><title>x</title></head><body>hi</body></html>`},
		{"head with attrs", `<html><head lang="en"><title>x</title></head></html>`},
	} {
		got := c.rewrite(&page{File: "r/x/index.html", Body: tc.body})
		at := strings.Index(got, noindexTag)
		lower := strings.ToLower(got)
		open := strings.Index(lower, "<head")
		closed := strings.Index(lower, "</head>")
		if at < open || at > closed {
			t.Errorf("%s: robots meta at %d is outside <head> (%d..%d)", tc.name, at, open, closed)
		}
		// Anything before the doctype drops the page into quirks mode, so the
		// preview renders with different box sizing than gno.land itself.
		if d := strings.Index(lower, "<!doctype"); d > 0 {
			t.Errorf("%s: %d bytes precede the doctype: %.60s", tc.name, d, got)
		}
	}
}
