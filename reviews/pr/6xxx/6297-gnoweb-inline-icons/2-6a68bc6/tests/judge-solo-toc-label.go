// judge-solo: the TOC title and ID an icon label gives its heading, and the
// alone-in-heading hint, at gnolang/gno#6297 head 6a68bc69f.
// Each subtest asserts the behaviour the rendered heading implies and logs
// what it got; c1 c2 c3 c5 c4 fail at 6a68bc69f.
//
//	cp judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go
//	go test -count=1 -run TestJudgeSoloToc -v ./gno.land/pkg/gnoweb/markdown
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func judgeSoloToc(t *testing.T, src string) (string, []string) {
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
	var got []string
	for _, it := range toc.Items {
		got = append(got, it.ID+"="+strings.TrimSpace(it.Title))
	}
	t.Logf("SRC %q\nHTML %s\nTOC %q", src, buf.String(), got)
	return buf.String(), got
}

func TestJudgeSoloToc(t *testing.T) {
	// candidates 1, 1002: a labeled icon that renders no svg titles its heading.
	t.Run("c1", func(t *testing.T) {
		for _, src := range []string{
			"## <gno-icon name=\"nope\" label=\"Top\" />\n",
			"## <gno-icon label=\"Top\" />\n",
			"## <gno-icon name=\"star\" label=\"Top\">\n",
		} {
			html, got := judgeSoloToc(t, src)
			assert.NotContains(t, html, "<svg")
			assert.Empty(t, got, "%q renders an empty heading", src)
		}
	})
	// candidates 1001, 2: character references stay raw in the TOC title.
	t.Run("c2", func(t *testing.T) {
		_, got := judgeSoloToc(t, "## <gno-icon name=\"star\" label=\"Q &amp; A\" />\n")
		assert.Equal(t, []string{"heading=Q & A"}, got)
		_, got = judgeSoloToc(t, "## <gno-icon name=\"star\" label=\"Say &quot;hi&quot;\" /> Picks\n")
		assert.Equal(t, []string{`picks=Say "hi" Picks`}, got)
	})
	// candidate 3: the label is glued to the text beside it.
	t.Run("c3", func(t *testing.T) {
		_, got := judgeSoloToc(t, "## Picks<gno-icon name=\"star\" label=\"Top\" />\n")
		assert.Equal(t, []string{"picks=Picks Top"}, got)
	})
	// candidate 5: a label-only heading's anchor is positional.
	t.Run("c5", func(t *testing.T) {
		_, alone := judgeSoloToc(t, "## <gno-icon name=\"star\" label=\"Top\" />\n")
		_, after := judgeSoloToc(t, "## <gno-icon name=\"heart\" />\n\n## <gno-icon name=\"star\" label=\"Top\" />\n")
		assert.Equal(t, alone, after, "adding an unlabeled icon heading above moves the Top anchor")
	})
	// candidate 4: an icon in an image's alt text takes the heading's hint.
	t.Run("c4", func(t *testing.T) {
		const hint = "<!-- gno-icon: alone in a link or heading"
		html, _ := judgeSoloToc(t, "## <gno-icon name=\"star\" />\n")
		assert.Contains(t, html, hint, "control")
		html, _ = judgeSoloToc(t, "## ![<gno-icon name=\"star\" />](i.png) <gno-icon name=\"star\" />\n")
		assert.Contains(t, html, hint)
	})
}
