// Judge repro, candidate 1: detailsParser.Continue walks the last-descendant
// chain, closed blockquotes included, on every line. Three sizes, D and N
// doubling, details parse against the same bytes without the opener.
// Fails at c05644643 (details/plain grows with size); passes at 8ab095f49.
package markdown

import (
	"strings"
	"testing"
	"time"
)

func TestZZJudgeWalkGrowth(t *testing.T) {
	for _, s := range []struct{ d, n int }{{5000, 25000}, {10000, 50000}, {20000, 100000}} {
		deep := strings.Repeat(">", s.d) + "\n" + strings.Repeat("\n", s.n)
		t0 := time.Now()
		parseDetails(t, deep)
		plain := time.Since(t0)
		t0 = time.Now()
		parseDetails(t, ":::details a\n"+deep+":::\n")
		det := time.Since(t0)
		t.Logf("bytes=%d D=%d N=%d plain=%v details=%v ratio=%.1f", len(deep), s.d, s.n, plain, det, float64(det)/float64(plain))
		if det > 10*plain {
			t.Errorf("D=%d: details %v over 10x plain %v", s.d, det, plain)
		}
	}
}
