// judge-124-slug-length-bound_test.go — gnolang/gno#6194, misc/gnopreview/crawl_test.go:276.
//
// Asserts the exact longest path segment slug() can emit: maxSlugLen+9, being
// 64 truncated chars + "-" + 8 hex. TestURLToFileIsSafe bounds each segment by
// maxSlugLen+16 instead, 7 chars of slack, so a digest widened from sha256[:4]
// to sha256[:6] (77 chars) still passes it. Both tests pass at ecf7af0f; the
// second is the assertion the in-tree bound fails to make.
//
/* Run: from a gno checkout:
gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f
curl -fsSL -o misc/gnopreview/judge-124-slug-length-bound_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-124-slug-length-bound_test.go
(cd misc/gnopreview && go test -run TestJudge124 -v .)
rm misc/gnopreview/judge-124-slug-length-bound_test.go
*/
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// longestSegment is TestURLToFileIsSafe's own measurement, over the same input.
func longestSegment(p string) (string, int) {
	worst, n := "", 0
	for _, seg := range strings.Split(p, "/") {
		if len(seg) > n {
			worst, n = seg, len(seg)
		}
	}
	return worst, n
}

func TestJudge124SlugSegmentBoundIsExact(t *testing.T) {
	t.Parallel()
	seg, n := longestSegment(urlToFile("/r/x/y:" + strings.Repeat("z", 400)))
	t.Logf("maxSlugLen=%d longest segment=%d (%q)", maxSlugLen, n, seg)
	// SHOULD: the bound the code can actually reach, red the moment it moves.
	if n != maxSlugLen+9 {
		t.Errorf("longest segment is %d chars; slug() emits at most maxSlugLen+9 = %d", n, maxSlugLen+9)
	}
	// IS: crawl_test.go:276, which accepts 7 chars slug() can never emit.
	// if n > maxSlugLen+16 {
	// 	t.Errorf("segment %q is %d chars", seg, n)
	// }
}

// slugWide is slug() with the digest widened from sha256[:4] to sha256[:6] —
// the one-token change the in-tree bound is meant to catch and does not.
func slugWide(s string) string {
	out := strings.Trim(unsafeSeg.ReplaceAllString(s, "-"), "-.")
	if len(out) > maxSlugLen {
		out = out[:maxSlugLen]
	}
	if out == s && out != "" {
		return out
	}
	sum := sha256.Sum256([]byte(s))
	if out == "" {
		return hex.EncodeToString(sum[:6])
	}
	return out + "-" + hex.EncodeToString(sum[:6])
}

func TestJudge124WiderDigestClearsTheInTreeBound(t *testing.T) {
	t.Parallel()
	n := len(slugWide(strings.Repeat("z", 400)))
	if n > maxSlugLen+16 {
		t.Fatalf("a sha256[:6] digest is %d chars, over the in-tree bound %d — nothing to report", n, maxSlugLen+16)
	}
	if n <= maxSlugLen+9 {
		t.Fatalf("a sha256[:6] digest is %d chars, within the exact bound %d — nothing to report", n, maxSlugLen+9)
	}
	t.Logf("a sha256[:6] digest gives a %d-char segment: over the exact bound %d, under crawl_test.go:276's %d",
		n, maxSlugLen+9, maxSlugLen+16)
}
