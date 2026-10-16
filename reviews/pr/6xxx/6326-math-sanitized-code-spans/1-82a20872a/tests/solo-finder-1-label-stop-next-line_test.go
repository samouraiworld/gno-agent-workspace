// Repro: the single-$ label stop ("[Send $10](...)") ends only the first-line
// search; Parse then searches the next line, so a $ closer there turns the
// link into a formula spanning both lines. Predates the diff: same output at
// head 82a20872a and at the merge base 21f9abd74. On one line the link stays.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6326/head && git checkout 82a20872a7e4eb89cae377e8ef46c5c578b1b639
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_label_stop_test.go
//   cd gno.land && go test ./pkg/gnoweb/markdown -run TestZZLabelStopNextLine -count=1 -v
//   # head and base: FAIL on "two lines", PASS on "one line"
//
// Observed TeX annotation at head: "10](/r/demo/bank$help&func=Send)\nBalance: 100"
package markdown

import (
	"strings"
	"testing"
)

func TestZZLabelStopNextLine(t *testing.T) {
	for _, c := range []struct{ name, src string }{
		{"one line", "[Send $10](/r/demo/bank$help&func=Send) Balance: 100$\n"},
		{"two lines", "[Send $10](/r/demo/bank$help&func=Send)\nBalance: 100$\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := renderMathMarkdown(t, c.src)
			if !strings.Contains(out, `<a href="/r/demo/bank$help&amp;func=Send"`) || strings.Contains(out, "<math") {
				t.Errorf("link turned into math: %s", out)
			}
		})
	}
}
