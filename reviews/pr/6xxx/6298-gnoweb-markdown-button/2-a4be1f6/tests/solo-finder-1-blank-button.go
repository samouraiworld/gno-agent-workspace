// solo-finder-1: a gno-button whose label or href renders blank is still
// emitted at a4be1f67b, although 256a0be71 says "a label needs at least one
// visible rune" and "an href blank only once decoded is no href".
//
// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/6298/head && git checkout --detach a4be1f67b066a1f0a28ed57d854c8d96c775965c
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_finder1_blank_test.go
//   cd gno.land && go test ./pkg/gnoweb/markdown -run TestFinder1BlankButton -v
// Each failing subtest is a blank button the fix lets through.

package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
)

func finder1Render(t *testing.T, src string) string {
	t.Helper()
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	require.NoError(t, err)
	m := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
	var buf bytes.Buffer
	ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
	require.NoError(t, m.Convert([]byte(src), &buf, ctx))
	return buf.String()
}

func TestFinder1BlankButton(t *testing.T) {
	for name, src := range map[string]string{
		"label U+2800 braille blank":         `<gno-button href="/r/test" label="&#x2800;" />`,
		"label U+034F grapheme joiner":       `<gno-button href="/r/test" label="&#x34F;" />`,
		"label U+FE0F variation selector":    `<gno-button href="/r/test" label="&#xFE0F;" />`,
		"label U+E0100 variation selector":   `<gno-button href="/r/test" label="&#xE0100;" />`,
		"label U+17B4 khmer inherent vowel":  `<gno-button href="/r/test" label="&#x17B4;" />`,
		"label U+180B mongolian fvs":         `<gno-button href="/r/test" label="&#x180B;" />`,
		"href U+00A0 no-break space":         `<gno-button href="&#xA0;" label="Go" />`,
		"href U+3000 ideographic space":      `<gno-button href="&#x3000;" label="Go" />`,
	} {
		t.Run(name, func(t *testing.T) {
			out := finder1Render(t, src)
			t.Logf("%q", out)
			if strings.Contains(out, `class="gno-button"`) {
				t.Errorf("blank button rendered: %s", out)
			}
		})
	}
}
