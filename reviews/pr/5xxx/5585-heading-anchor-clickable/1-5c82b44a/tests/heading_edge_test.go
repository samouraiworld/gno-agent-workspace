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

// Render helper mirroring ext_test.go.
func render(t *testing.T, src string) string {
	t.Helper()
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	ctxOpts := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
	ext := NewGnoExtension(WithImageValidator(func(uri string) bool { return true }))
	m := goldmark.New(goldmark.WithParserOptions(parser.WithAutoHeadingID()))
	ext.Extend(m)
	node := m.Parser().Parse(text.NewReader([]byte(src)), ctxOpts)
	var buf bytes.Buffer
	if err := m.Renderer().Render(&buf, []byte(src), node); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// Heading with inline link -> nested <a> tags (invalid HTML).
func TestHeadingWithInlineLink_NestedAnchors(t *testing.T) {
	out := render(t, "## Title with [link](/r/foo)\n")
	t.Log(out)
	// Count <a> vs </a>. A well-formed heading has matched count but never nested.
	// Check: after `<a class="heading-anchor"`, before closing `</a>` that matches the heading-anchor, no other <a ...> should appear.
	idx := strings.Index(out, `<a class="heading-anchor"`)
	if idx < 0 {
		t.Fatal("no heading-anchor")
	}
	// naive: find next `<a ` after opening
	rest := out[idx+len(`<a class="heading-anchor"`):]
	if strings.Contains(rest[:strings.Index(rest, "</h2>")], "<a ") {
		t.Errorf("nested <a> tags inside heading-anchor (invalid HTML):\n%s", out)
	}
}

// Heading without AutoHeadingID parser option -> unbalanced </a>.
// Simulates using the extension standalone without parser.WithAutoHeadingID().
func TestHeadingWithoutAutoID_UnbalancedAnchor(t *testing.T) {
	gnourl, _ := weburl.Parse("https://gno.land/r/test")
	ctxOpts := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
	ext := NewGnoExtension()
	m := goldmark.New() // no AutoHeadingID
	ext.Extend(m)
	src := "# Hello\n"
	node := m.Parser().Parse(text.NewReader([]byte(src)), ctxOpts)
	var buf bytes.Buffer
	m.Renderer().Render(&buf, []byte(src), node)
	out := buf.String()
	t.Log(out)
	if strings.Contains(out, "</a></h1>") && !strings.Contains(out, "<a class=\"heading-anchor\"") {
		t.Errorf("unbalanced </a> emitted without matching <a>:\n%s", out)
	}
}

// Explicit attribute block on heading with id="" -> unbalanced </a>.
func TestHeadingEmptyID_UnbalancedAnchor(t *testing.T) {
	out := render(t, "# Hello {#}\n") // attribute syntax may vary
	t.Log(out)
	// Mainly observational; confirms behavior under degenerate IDs.
}
