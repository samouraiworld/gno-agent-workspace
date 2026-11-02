// Repro: a $$, \\( or \\[ closer after a link whose destination or title the
// scanner misreads (quoted title holding "(", pointy destination holding "(",
// "<" never closed) is hidden, so a formula master rendered stays text.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/6326/head && git checkout f6a568ed830794fafe1ca4840786c30d4dc0c233
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_dest_hides_closer_test.go
//   go test ./gno.land/pkg/gnoweb/markdown -run TestZZDestHidesCloser -v
// Same at the merge base 1cae57ed72430444a2386e6052d93a2fc282482d for comparison.
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestZZDestHidesCloser(t *testing.T) {
	for _, src := range []string{
		`$$ [a](x "(") b $$` + "\n",
		`\\( [a](x "(") b \\)` + "\n",
		`\\[ [a](x "(") b \\]` + "\n",
		`\\( [a](<x(>) b \\)` + "\n",
		`\\( [a](<x b \\)` + "\n",
		`$$ [a](x 'it(s') b $$` + "\n",
		`$$ [a](x "t) $$ q") c $$` + "\n",
	} {
		var plain bytes.Buffer
		_ = goldmark.New().Convert([]byte(src), &plain)
		out := renderMathMarkdown(t, src)
		t.Logf("%q\n  math=%d\n  plain goldmark: %s  gnoweb: %s", src, strings.Count(out, "<math"), plain.String(), out)
	}
}

// The hidden closer then pairs with one on the next line, or a later $$ line
// opens a display block: the link and the text between become TeX.
func TestZZDestHidesCloserPairsLater(t *testing.T) {
	for _, src := range []string{
		`$$ [a](x "t) $$ q") c $$` + "\n",
		`Pay $$ [a](x "(") b $$ then` + "\nsee $$ below.\n",
		`$$ [a](x "(") b $$` + "\nsome text\n$$\n",
	} {
		out := renderMathMarkdown(t, src)
		t.Logf("%q\n  math=%d links=%d\n  gnoweb: %s", src, strings.Count(out, "<math"), strings.Count(out, "<a "), out)
	}
}
