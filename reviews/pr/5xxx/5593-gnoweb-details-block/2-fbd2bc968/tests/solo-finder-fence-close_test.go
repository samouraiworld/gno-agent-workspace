// Repro: detailsParser.Continue runs on the outer block before its children,
// so a `:::` line inside a fenced code block closes the details. The fence is
// left open and everything after it, footer included, renders as one <pre>.
//
// From a plain clone of gnolang/gno at fbd2bc9687ddebeef9f09c16032068fcca159696:
//
//	cp <this file> gno.land/pkg/gnoweb/markdown/solo_fence_close_test.go
//	go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloDetails -v -count=1
//
// At head fbd2bc968:
//
//	TestSoloDetailsFenceKeepsColons   FAIL: output is
//	  <details ...><pre><code></code></pre></div></details>
//	  <pre><code>after\n:::\n## Footer\n</code></pre>
//	TestSoloDetailsHTMLLineThenClose  PASS (footer after </details>)
//
// With the fix suggested in review (Continue returns Continue|HasChildren when
// the last opened block is KindFencedCodeBlock or KindHTMLBlock) the fence test
// passes and the HTML test FAILS: an HTML block of type 6/7 runs to the next
// blank line, so `<div>x</div>` followed directly by the closer swallows the
// closer and folds `## Footer` into the details. Exempting fenced code only
// passes both.
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

func soloFenceRender(t *testing.T, src string) string {
	t.Helper()
	u, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: u}))
	m := goldmark.New()
	NewGnoExtension().Extend(m)
	in := []byte(src)
	node := m.Parser().Parse(text.NewReader(in), ctx)
	var b bytes.Buffer
	if err := m.Renderer().Render(&b, in, node); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestSoloDetailsFenceKeepsColons(t *testing.T) {
	out := soloFenceRender(t, ":::details A\n```\n:::\n```\nafter\n:::\n\n## Footer\n")
	t.Logf("\n%s", out)
	if !strings.Contains(out, "<pre><code>:::\n</code></pre>") {
		t.Errorf("the `:::` line inside the fence was not kept as code")
	}
	if strings.Contains(out, "<pre><code>after") || !strings.Contains(out, "</details>\n<h2>Footer</h2>") {
		t.Errorf("text after the fence did not stay in the details, footer not rendered as a heading")
	}
}

func TestSoloDetailsHTMLLineThenClose(t *testing.T) {
	out := soloFenceRender(t, ":::details A\n<div>x</div>\n:::\n\n## Footer\n")
	t.Logf("\n%s", out)
	if !strings.Contains(out, "</details>\n<h2>Footer</h2>") {
		t.Errorf("closer after an HTML line was swallowed: footer folded into the details")
	}
}
