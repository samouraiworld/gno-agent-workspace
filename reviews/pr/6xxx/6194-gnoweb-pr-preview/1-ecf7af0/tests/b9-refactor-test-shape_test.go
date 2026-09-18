package main

/* Run: from a plain gno clone:
gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
curl -fsSL -o misc/gnopreview/b9-refactor-test-shape_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/b9-refactor-test-shape_test.go
(cd misc/gnopreview && go test -count=1 -v -run 'TestB9' .)
rm misc/gnopreview/b9-refactor-test-shape_test.go

Both tests below pass at ecf7af0, and each passing is the finding.

1. TestB9SeedsAssertionIsOneSided: crawl_test.go's TestSeedsSkipSingleSegmentDirs
   asserts only that "/r" and "/p" are absent. It stays green on a Seeds() that
   emits no directory page at all. Confirmed in the worktree too: flipping
   crawl.go's `strings.Count(d, "/") >= 2` to `>= 3` drops "/r/gnoland" from
   Seeds() and `go test ./...` still reports ok.

2. TestB9TrailingSlashTestIsRedundant: TestSplitURLKeepsTrailingSlash (10 lines)
   catches no mutation TestSplitURL's own table rows miss. Confirmed in the
   worktree: with that function deleted, trimming the trailing slash inside
   splitURL turns TestSplitURL, TestCanonicalURL, TestCrawlerInScope,
   TestURLToFile and TestURLToFileIsSafe red.
*/

import (
	"reflect"
	"strings"
	"testing"
)

// asWritten is TestSeedsSkipSingleSegmentDirs' assertion, verbatim from
// crawl_test.go, lifted so it can be applied to a seed list of our choosing.
func asWritten(t *testing.T, seeds []string) {
	t.Helper()
	for _, s := range seeds {
		if s == "/r" || s == "/p" {
			t.Errorf("Seeds() includes %q, which gnoweb answers with 400", s)
		}
	}
}

// proposed is the shorter, two-sided form: one equality, no loop. It pins the
// directory page the crawler needs as well as the single-segment one it must
// not seed.
func proposed(seeds []string) bool {
	return reflect.DeepEqual(seeds, []string{
		"/r/gnoland/home", "/r/gnoland/home$source", "/r/gnoland/home$help", "/r/gnoland",
	})
}

func TestB9SeedsAssertionIsOneSided(t *testing.T) {
	c := &Crawler{Realms: []string{"gno.land/r/gnoland/home"}}
	real := c.Seeds()
	if !proposed(real) {
		t.Fatalf("the proposed assertion does not hold at this head: Seeds() = %#v", real)
	}

	// What Seeds() returns once the directory walk stops seeding anything —
	// the `>= 3` mutation, reproduced here without touching crawl.go.
	var noDirs []string
	for _, s := range real {
		if strings.HasPrefix(s, "/r/gnoland/home") {
			noDirs = append(noDirs, s)
		}
	}
	if len(noDirs) == len(real) {
		t.Fatalf("no directory page in Seeds() to drop: %#v", real)
	}

	// The finding: the assertion the PR wrote sees nothing wrong here.
	asWritten(t, noDirs) // green — this is the defect the test cannot catch
	if proposed(noDirs) {
		t.Errorf("the proposed assertion also missed it: %#v", noDirs)
	}
	t.Logf("Seeds() with no directory page = %#v — TestSeedsSkipSingleSegmentDirs passes on it", noDirs)
}

// splitURLTrimmed is splitURL with the trailing slash trimmed: the one mutation
// TestSplitURLKeepsTrailingSlash exists to catch.
func splitURLTrimmed(p string) (base, args, query string) {
	p = strings.TrimSuffix(p, "/")
	if i := strings.Index(p, "$"); i >= 0 {
		p, query = p[:i], p[i+1:]
	}
	if i := strings.Index(p, ":"); i >= 0 {
		p, args = p[:i], p[i+1:]
	}
	return p, args, query
}

func TestB9TrailingSlashTestIsRedundant(t *testing.T) {
	// TestSplitURL's table already carries both spellings with distinct wanted
	// bases, so it rejects any implementation the dedicated test rejects.
	tableCatches := false
	for _, tc := range []struct{ in, base string }{
		{"/r/x/y", "/r/x/y"},
		{"/r/x/y/", "/r/x/y/"},
	} {
		if base, _, _ := splitURLTrimmed(tc.in); base != tc.base {
			tableCatches = true
		}
	}

	// The dedicated test's whole body: a != b.
	a, _, _ := splitURLTrimmed("/r/x/y")
	b, _, _ := splitURLTrimmed("/r/x/y/")
	dedicatedCatches := a == b

	if dedicatedCatches && !tableCatches {
		t.Fatalf("the dedicated test catches a mutation the table misses — not redundant")
	}
	t.Logf("table rows catch the trim: %v; dedicated test catches it: %v — the 10-line test adds nothing",
		tableCatches, dedicatedCatches)

	// And at this head both hold, so deleting the function loses no coverage.
	if base, _, _ := splitURL("/r/x/y/"); base != "/r/x/y/" {
		t.Errorf("splitURL(%q) = %q at head", "/r/x/y/", base)
	}
}
