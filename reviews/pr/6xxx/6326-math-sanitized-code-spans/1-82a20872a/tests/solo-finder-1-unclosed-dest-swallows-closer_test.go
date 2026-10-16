// Repro: a $$, \\( or \\[ formula holding "[..](" with no balancing ")" before
// its closer misses that closer at head 82a20872a and runs on to a later one,
// swallowing a following link; the merge base 21f9abd74 closes at the first.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6326/head && git checkout 82a20872a7e4eb89cae377e8ef46c5c578b1b639
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_unclosed_dest_test.go
//   cd gno.land && go test ./pkg/gnoweb/markdown -run TestZZUnclosedDest -count=1 -v
//   # head: FAIL on all three cases
//   git checkout 21f9abd744ded2ac2e925d54a0aa3e765c991927 && go test ./pkg/gnoweb/markdown -run TestZZUnclosedDest -count=1 -v
//   # base: PASS
//
// Observed at head (TeX annotation of the single formula produced):
//   \\( case: " [a,b](x \\) and [l](/r/y) then \\(z"  (link /r/y gone)
//   $$ case:  "[0,1](x $$ and [l](/r/y) z\nw "       (link and next line gone)
//   ADR's own example "$$[0,1](x$$ then $$y$$": plain text, no formula
package markdown

import (
	"strings"
	"testing"
)

func TestZZUnclosedDest(t *testing.T) {
	for _, c := range []struct {
		name, src, want string
		maths           int
	}{
		{"\\\\( closer eaten as destination paren", `\\( [a,b](x \\) and [l](/r/y) then \\(z\\)` + "\n", `<a href="/r/y"`, 2},
		{"$$ runs to the next line's closer", "$$[0,1](x $$ and [l](/r/y) z\nw $$\n", `<a href="/r/y"`, 1},
		{"ADR known-limit example", "$$[0,1](x$$ then $$y$$\n", "", 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := renderMathMarkdown(t, c.src)
			if c.want != "" && !strings.Contains(out, c.want) {
				t.Errorf("link lost: %s", out)
			}
			if n := strings.Count(out, "<math"); n != c.maths {
				t.Errorf("got %d formulas, want %d: %s", n, c.maths, out)
			}
		})
	}
}
