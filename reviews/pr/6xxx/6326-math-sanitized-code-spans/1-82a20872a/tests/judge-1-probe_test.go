// Judge probe: prints gnoweb's HTML for each input, at whichever tree it runs in.
// Run: cp into gno.land/pkg/gnoweb/markdown/, then from gno.land:
//   go test ./pkg/gnoweb/markdown -run TestZZJudgeProbe -count=1 -v
package markdown

import "testing"

func TestZZJudgeProbe(t *testing.T) {
	for _, src := range []string{
		`\\( [a,b](x \\) and [l](/r/y) then \\(z\\)` + "\n",
		"$$[0,1](x $$ and [l](/r/y) z\nw $$\n",
		"$$[0,1](x$$ then $$y$$\n",
		"$ [a,b](x $ and [l](/r/y) then $z$\n",
		"$[a,b](x$ and [l](/r/y)\nw$\n",
		"$$ a `$$` b\nc\n$$\n",
		"\\\\(a `b\nc\\\\)d` e\\\\)\n",
		"[Send $10](/r/demo/bank$help&func=Send)\nBalance: 100$\n",
	} {
		t.Logf("SRC %q\nOUT %s", src, renderMathMarkdown(t, src))
	}
}
