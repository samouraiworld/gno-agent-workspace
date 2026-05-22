package markdown

// Adversarial tests for PR #5585 heading-anchor extension.
// Drop into gno.land/pkg/gnoweb/markdown/ and run.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func renderHeading(t *testing.T, src string) string {
	t.Helper()
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	require.NoError(t, err)
	ctxOpts := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))

	ext := NewGnoExtension(WithImageValidator(func(uri string) bool { return true }))
	m := goldmark.New(goldmark.WithParserOptions(parser.WithAutoHeadingID()))
	ext.Extend(m)

	node := m.Parser().Parse(text.NewReader([]byte(src)), ctxOpts)
	var html bytes.Buffer
	require.NoError(t, m.Renderer().Render(&html, []byte(src), node))
	return html.String()
}

// TestMentionInHeading: a heading containing `@alice` produces an
// inline link via ExtMention. The mention's <a> sits as a sibling of
// the heading-anchor runs, never nested inside one.
func TestMentionInHeading(t *testing.T) {
	out := renderHeading(t, "## Hello @alice here\n")
	t.Logf("html: %s", out)
	// Three siblings inside the heading: anchor("Hello "), link(@alice), anchor(" here").
	require.Equal(t, 2, strings.Count(out, `class="heading-anchor"`))
	require.Contains(t, out, `<a href="/u/alice">`)
	// No nested <a>: closing tag of the mention immediately precedes the
	// next heading-anchor opening, but they are siblings, not nested.
	require.NotRegexp(t, `class="heading-anchor"[^>]*>[^<]*<a href="/u/`, out)
}

// TestBech32AddressInHeading: a heading containing a bech32 address
// produces an inline link. heading-anchor must not wrap it.
func TestBech32AddressInHeading(t *testing.T) {
	addr := "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"
	out := renderHeading(t, "## Send to "+addr+" today\n")
	t.Logf("html: %s", out)
	require.Contains(t, out, `href="/u/`+addr+`"`)
	// The two non-link runs must be wrapped, the address link must not be.
	require.Regexp(t, `class="heading-anchor"[^>]*>[^<]*Send to[^<]*</a>`, out)
}

// TestFootnoteInHeading: footnote refs inside a heading. Goldmark
// produces an <a> via the footnote extension when enabled. ExtHeading
// only knows Link/AutoLink/GnoLink — if footnotes were enabled and
// produced a different node kind, isLinkLike would miss it. Document
// that footnotes are not currently wired into GnoExtension, so the
// case is safe today, but it's a maintenance trap.
func TestFootnoteInHeadingNotEnabled(t *testing.T) {
	// GnoExtension does not enable goldmark's Footnote extension, so
	// `[^1]` in a heading is rendered as plain text. No issue.
	out := renderHeading(t, "## Title with [^1]\n")
	t.Logf("html: %s", out)
	require.Contains(t, out, `class="heading-anchor"`)
	require.NotContains(t, out, `class="footnote-ref"`)
}

// TestHeadingWithoutAutoID: when parser.WithAutoHeadingID is absent,
// headings have no id and the transformer must be a no-op.
func TestHeadingWithoutAutoID(t *testing.T) {
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	require.NoError(t, err)
	ctxOpts := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))

	ext := NewGnoExtension(WithImageValidator(func(uri string) bool { return true }))
	m := goldmark.New() // no WithAutoHeadingID
	ext.Extend(m)

	node := m.Parser().Parse(text.NewReader([]byte("## Plain Title\n")), ctxOpts)
	var html bytes.Buffer
	require.NoError(t, m.Renderer().Render(&html, []byte("## Plain Title\n"), node))
	out := html.String()
	t.Logf("html: %s", out)
	require.NotContains(t, out, `class="heading-anchor"`)
	require.NotContains(t, out, ` id=`)
}

// TestImageInHeadingClickHijack: an image in a heading is wrapped by
// the heading-anchor, so clicking the image navigates to the hash
// (the image is no longer just static content). Document the
// behavior — it might surprise users.
func TestImageInHeadingClickHijack(t *testing.T) {
	out := renderHeading(t, "## ![alt](/static/img.png)\n")
	t.Logf("html: %s", out)
	// The img is inside the heading-anchor. Clicking it sets URL hash.
	require.Regexp(t, `<a class="heading-anchor"[^>]*><img[^>]*></a>`, out)
}

// TestHeadingWithMultipleLinksAndText: A B link C D link E.
// Expect two heading-anchor runs (A B + C D + E) split by two links.
func TestHeadingWithMultipleLinksAndText(t *testing.T) {
	src := "## Start [one](/a) middle [two](/b) end\n"
	out := renderHeading(t, src)
	t.Logf("html: %s", out)
	// 3 anchor runs.
	require.Equal(t, 3, strings.Count(out, `class="heading-anchor"`),
		"expected 3 heading-anchor runs (Start /one/ middle /two/ end)")
}

// TestEmptyHeading: `##` with no text. Should not crash.
func TestEmptyHeading(t *testing.T) {
	out := renderHeading(t, "## \n")
	t.Logf("html: %s", out)
	// Just verify no panic and no malformed anchor.
	require.NotContains(t, out, `<a class="heading-anchor"></a>`)
}
