// b1-lines: TestButtonHostileBudget never exercises the length cap.
//
// Every shape in TestButtonHostileBudget stops each scan attempt at the next
// '<' (the `case c == '<'` abort), so maxButtonTagLen is not what bounds them.
// A '<' inside an attribute name is not an abort: `<gno-button x` repeated makes
// every attempt read until the cap. Moving the cap from a pre-scan truncation to
// a post-scan size check keeps every PR test green and turns this shape
// quadratic; this test goes red on that mutation and is green at the head.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6298/head && git checkout 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//   cp <this file> gno.land/pkg/gnoweb/markdown/b1_lines_budget_cap_test.go
//   go test -count=1 -run 'TestButtonHostileBudgetCapShape' ./gno.land/pkg/gnoweb/markdown/   # PASS (~0.2s)
// Mutation (cap after scanning instead of truncating first), in utils.go:
//   delete `src = src[:min(len(src), maxLen)]` from scanGnoTag, rename its body
//   scanGnoTagUncapped(src, prefix, attr), and make scanGnoTag call it and
//   return 0, false when size > maxLen.
//   go test -count=1 -run 'TestButtonHostileBudget$|TestParseButtonTagBound|TestScanGnoTag|TestParseButtonTagNoAlloc' ./gno.land/pkg/gnoweb/markdown/  # still PASS
//   go test -count=1 -run 'TestButtonHostileBudgetCapShape' ./gno.land/pkg/gnoweb/markdown/  # FAIL, over budget
package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestButtonHostileBudgetCapShape(t *testing.T) {
	const budget = 5 * time.Second
	for name, src := range map[string]string{
		// a '<' inside an attribute name does not end the attempt
		"one line, 40k tags, '<' in a key":  "a " + strings.Repeat(`<gno-button x`, 40_000),
		"one line, 30k tags, quote ladder": "a " + strings.Repeat(`<gno-button a='"`, 30_000),
	} {
		start := time.Now()
		renderMarkdown(t, src)
		require.Less(t, time.Since(start), budget, name)
	}
}
