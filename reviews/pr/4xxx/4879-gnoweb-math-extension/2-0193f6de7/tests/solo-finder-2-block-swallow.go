// Repro: hasClosingLine accepts any later line containing `$$` within 8 KiB,
// across blank lines, headings and fenced code, so one stray `$$` line turns
// everything up to a later inline `$$x$$` into a single math block.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/4879/head && git checkout --detach 0193f6de7fc5823e3c01d4a5e580e377cde98659
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_swallow_repro_test.go
//   go test -v -count=1 -run TestBlockSwallow ./gno.land/pkg/gnoweb/markdown/
//   rm gno.land/pkg/gnoweb/markdown/zz_swallow_repro_test.go
//
// Observed at 0193f6de7 (whitespace collapsed):
//   case 1 OUT: <math class="math-displaystyle" ...> <semantics> <mrow> <mi>f</mi> <mi>o</mi> ...
//     <merror title="Unexpanded macro argument">?#</merror> <mi>H</mi> <mi>e</mi> ... <mo>`</mo> ...
//     <annotation encoding="application/x-tex"> forgot to close # Heading A paragraph. ```go code() ``` Later, inline </annotation>
//     </semantics> </math> <p>x$$ display.</p>
//   (no <h1>, no <pre>; the heading, paragraph and code block are rendered as math identifiers)
//   case 2 (no later `$$`) OUT: <p>$$ forgot to close</p> <h1>Heading</h1> <p>A paragraph with no closer.</p>
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestBlockSwallow(t *testing.T) {
	for _, src := range []string{
		"$$\nforgot to close\n\n# Heading\n\nA paragraph.\n\n```go\ncode()\n```\n\nLater, inline $$x$$ display.\n",
		"$$\nforgot to close\n\n# Heading\n\nA paragraph with no closer.\n",
	} {
		var buf bytes.Buffer
		_ = goldmark.New(goldmark.WithExtensions(NewGnoExtension())).Convert([]byte(src), &buf)
		t.Logf("IN : %q\nOUT: %s", src, strings.Join(strings.Fields(buf.String()), " "))
	}
}
