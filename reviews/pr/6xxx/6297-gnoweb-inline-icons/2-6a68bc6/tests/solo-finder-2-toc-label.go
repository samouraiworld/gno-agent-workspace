// solo-finder-2: the TOC title an icon label gives a heading (writeNodeText's
// *Icon case, gno.land/pkg/gnoweb/markdown/utils.go:205-206).
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6297/head && git checkout --detach 6a68bc69fc638c389d2308743e2fb1d11cc326bc
//	cp solo-finder-2-toc-label.go gno.land/pkg/gnoweb/markdown/zz_toclabel_test.go
//	go test -count=1 -run TestSoloFinder2TocLabel -v ./gno.land/pkg/gnoweb/markdown
//
// Both subtests assert what the rendered heading says and fail at 6a68bc69f:
//   - entity:  label="Q &amp; A" renders aria-label "Q & A", the TOC title is
//     the raw "Q &amp; A", which ui/toc.html's html/template {{ .Title }}
//     escapes again, so the TOC reads `Q &amp; A`.
//   - broken:  a labeled icon that renders only a comment (unknown name,
//     missing name, not self-closing) leaves an empty <h2>, yet the TOC lists
//     the label "Top" for it.
//
// At the merge base 84df2c459 an icon-only heading has no TOC entry at all,
// labeled or not (TestIconHeadingIDAndToc expected "status=Status" there).
package markdown

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func soloFinder2Toc(t *testing.T, src string) (html string, titles []string) {
	t.Helper()
	m := newProductionLikeMarkdown()
	b := []byte(src)
	doc := m.Parser().Parse(text.NewReader(b), parser.WithContext(NewGnoParserContext(GnoContext{})))
	var buf bytes.Buffer
	if err := m.Renderer().Render(&buf, b, doc); err != nil {
		t.Fatal(err)
	}
	toc, err := TocInspect(doc, b, TocOptions{MinDepth: 2, MaxDepth: 6})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range toc.Items {
		titles = append(titles, it.Title)
	}
	return buf.String(), titles
}

func TestSoloFinder2TocLabel(t *testing.T) {
	t.Run("entity", func(t *testing.T) {
		html, titles := soloFinder2Toc(t, "## <gno-icon name=\"star\" label=\"Q &amp; A\" />\n")
		assert.Contains(t, html, `aria-label="Q &amp; A"`, "the svg is named Q & A")
		assert.Equal(t, []string{"Q & A"}, titles, "the TOC names the heading as the svg does")
	})
	for _, src := range []string{
		"## <gno-icon name=\"unicorn\" label=\"Top\" />\n",
		"## <gno-icon label=\"Top\" />\n",
		"## <gno-icon name=\"star\" label=\"Top\">\n",
	} {
		t.Run("broken", func(t *testing.T) {
			html, titles := soloFinder2Toc(t, src)
			assert.NotContains(t, html, "<svg", "%q renders no icon", src)
			assert.Empty(t, titles, "%q: a heading that renders empty gets no TOC entry; html %s", src, html)
		})
	}
}
