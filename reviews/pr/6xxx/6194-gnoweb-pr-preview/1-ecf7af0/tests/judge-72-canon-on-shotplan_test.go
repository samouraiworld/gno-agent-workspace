// Asserts canonicalURL is the identity on every shotPlan URL, so Screenshot's
// c.pages[canonicalURL(s.url)] and Crawler.FileOf agree today: the reach into the
// unexported map is duplication, not a live keying bug. Measured: 4/4 identity,
// and canonicalURL("/r/x$b=2&a=1") = "/r/x$a=1&b=2". Passes at the reviewed head.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp path/to/judge-72-canon-on-shotplan_test.go misc/gnopreview/
//	cd misc/gnopreview && go test -run TestJudgeCanonOnShotPlan -v .
//	rm judge-72-canon-on-shotplan_test.go
package main

import "testing"

func TestJudgeCanonOnShotPlan(t *testing.T) {
	for _, s := range shotPlan {
		// Crawler.Run inserts under the raw URL (crawl.go:143); Screenshot and
		// FileOf both read canonicalURL(u). Identity means the two agree here.
		if got := canonicalURL(s.url); got != s.url {
			t.Errorf("canonicalURL(%q) = %q (differs)", s.url, got)
		} else {
			t.Logf("canonicalURL(%q) = identity", s.url)
		}
	}
	// The multi-arg query the finder names: sorting moves the key.
	t.Logf("multi-arg: canonicalURL(%q) = %q", "/r/x$b=2&a=1", canonicalURL("/r/x$b=2&a=1"))
}
