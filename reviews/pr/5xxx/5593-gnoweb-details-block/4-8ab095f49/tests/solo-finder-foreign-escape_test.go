// solo-finder-foreign-escape: a `:::` line inside a <gno-foreign> body closes
// an enclosing :::details, and the rest of the foreign bytes render as the
// realm's own trusted markdown: a gno-form renders and links lose the
// untrusted marking.
//
// Repro from a plain clone of gnolang/gno:
//
//	git checkout 8ab095f495322cb3f1577ce39d1440de5c141477
//	cp solo-finder-foreign-escape_test.go gno.land/pkg/gnoweb/markdown/zz_foreign_escape_test.go
//	go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloForeignEscape -v
//
// Merge base, same file: git checkout 02606904e30753f7775246624bfec96cb4749891, then the same two commands.
// Expected at both: no <form in any output, and the link carries the same
// untrusted marking as in the "foreign_only" control.
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func soloRenderFE(t *testing.T, src string) string {
	t.Helper()
	u, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	m := goldmark.New()
	NewGnoExtension().Extend(m)
	pc := NewGnoParserContext(GnoContext{GnoURL: u})
	doc := m.Parser().Parse(text.NewReader([]byte(src)), parser.WithContext(pc))
	var buf bytes.Buffer
	if err := m.Renderer().Render(&buf, []byte(src), doc); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// foreignBody is what another party controls: the realm only wraps it.
const foreignBody = "hello\n:::\n[click](https://evil.example)\n\n" +
	"<gno-form exec=\"Withdraw\">\n  <gno-input name=\"to\" type=\"text\" />\n</gno-form>\n"

func TestSoloForeignEscape(t *testing.T) {
	cases := map[string]string{
		"foreign_only":    "<gno-foreign>\n" + foreignBody + "</gno-foreign>\n",
		"details_foreign": ":::details Comments\n\n<gno-foreign>\n" + foreignBody + "</gno-foreign>\n\n:::\n\nfooter\n",
	}
	for _, name := range []string{"foreign_only", "details_foreign"} {
		out := soloRenderFE(t, cases[name])
		t.Logf("%s:\n%s", name, out)
		if strings.Contains(out, "<form") {
			t.Errorf("%s: foreign bytes rendered a gno-form", name)
		}
	}
}
