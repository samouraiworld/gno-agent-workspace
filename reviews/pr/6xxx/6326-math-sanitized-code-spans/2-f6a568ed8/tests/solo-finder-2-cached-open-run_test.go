// Repro: a backtick inside an AMS expression (or a raw HTML tag) before a $
// opener is carried into that opener's next-line search when the search is
// answered from the cache of an earlier, failed $ opener on the same line.
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6326/head && git checkout --detach f6a568ed830794fafe1ca4840786c30d4dc0c233
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_cached_open_run_test.go
//   go test ./gno.land/pkg/gnoweb/markdown -run TestZZCachedOpenRun -v -count=1
//
// Expected at f6a568ed8: FAIL on the second assertion — "$b c⏎d$" renders as
// math alone, but not once an unrelated unclosed "$a " precedes it on the line.
package markdown

import (
	"strings"
	"testing"
)

func TestZZCachedOpenRun(t *testing.T) {
	alone := renderMathMarkdown(t, "x \\\\(`\\\\) $b c\nd$ `e\n")
	withA := renderMathMarkdown(t, "$a \\\\(`\\\\) $b c\nd$ `e\n")
	t.Logf("alone: %d math, withA: %d math", strings.Count(alone, "<math"), strings.Count(withA, "<math"))
	if !strings.Contains(alone, ">b c\nd</annotation>") {
		t.Fatalf("control: $b c⏎d$ should be math:\n%s", alone)
	}
	if !strings.Contains(withA, ">b c\nd</annotation>") {
		t.Errorf("$b c⏎d$ is left as text once an unclosed $a precedes it:\n%s", withA)
	}
	html := renderMathMarkdown(t, "$a <x y=\"`\"> $b c\nd$ `e\n")
	if !strings.Contains(html, ">b c\nd</annotation>") {
		t.Errorf("raw HTML variant: $b c⏎d$ left as text:\n%s", html)
	}
}
