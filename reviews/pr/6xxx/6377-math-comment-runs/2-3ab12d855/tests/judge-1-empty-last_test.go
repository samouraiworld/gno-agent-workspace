package mathml

// Times ConvertDisplay on runs of plain and indented `%` lines at three sizes,
// for comparing the branch's Empty with the rewrite in judge-1-empty-last.patch.
// Copy into gno.land/pkg/gnoweb/markdown/mathml and run with -run TestJudgeTiming -v.

import (
	"strings"
	"testing"
	"time"
)

func TestJudgeTiming(t *testing.T) {
	for _, line := range []string{"%\n", "  %\n"} {
		for _, n := range []int{1 << 14, 1 << 16, 1 << 18} {
			tex := "x" + strings.Repeat(line, n) + "y"
			best := time.Duration(1 << 62)
			for range 3 {
				start := time.Now()
				if _, err := NewMathMLConverter().ConvertDisplay(tex); err != nil {
					t.Fatal(err)
				}
				best = min(best, time.Since(start))
			}
			t.Logf("line=%q lines=%d best=%v", line, n, best)
		}
	}
}
