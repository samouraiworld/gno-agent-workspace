// Asserts a heading whose only content renders nothing visible gets no
// permalink anchor. Measures the rendered <h2> and its ToC title over five
// headings, in the production parser config (auto heading IDs on).
// Fails at 2eff8be62: the omitted-HTML and unknown-icon headings carry a lone anchor.
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/components"
	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func judgeTitles(items []*components.TocItem, out *[]string) {
	for _, it := range items {
		*out = append(*out, it.Title)
		judgeTitles(it.Items, out)
	}
}

func TestJudgeLoneHeadingAnchor(t *testing.T) {
	u, _ := weburl.Parse("https://gno.land/r/test")
	for _, src := range []string{
		"## </gno-frame>\n",
		"## <gno-icon name=\"nope\" />\n",
		"## <span></span>\n",
		"##\n",
		"## Title\n",
	} {
		m := goldmark.New(goldmark.WithParserOptions(parser.WithAutoHeadingID()))
		NewGnoExtension().Extend(m)
		ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: u}))
		node := m.Parser().Parse(text.NewReader([]byte(src)), ctx)
		var buf bytes.Buffer
		if err := m.Renderer().Render(&buf, []byte(src), node); err != nil {
			t.Fatal(err)
		}
		toc, _ := TocInspect(node, []byte(src), TocOptions{MinDepth: 1, MaxDepth: 6})
		var titles []string
		judgeTitles(toc.Items, &titles)
		html := strings.TrimSpace(buf.String())
		t.Logf("%-32q html=%s toc=%q", src, html, titles)
		if strings.Contains(html, "heading-anchor") && len(titles) == 0 {
			t.Errorf("%q: anchor in a heading with no ToC title", src)
		}
	}
	// Sibling: the sanitize integration renderer (no auto IDs) keeps the bare <h1>
	// shape its htmlMustNot assertions match.
	if got := renderMarkdown(t, "Welcome to MyRealm\n===\n"); !strings.Contains(got, "<h1>Welcome to MyRealm</h1>") {
		t.Errorf("sanitize renderer heading shape changed: %s", got)
	}
}
