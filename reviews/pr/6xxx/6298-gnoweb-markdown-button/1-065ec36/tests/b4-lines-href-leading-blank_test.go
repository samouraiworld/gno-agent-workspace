// b4-lines-href-leading-blank: isButtonHrefAllowed (ext_buttons.go:116) trims
// leading control and space bytes before its control-byte check, and
// newButtonLink (ext_buttons.go:72) tests the raw href for emptiness, so an
// href whose control byte leads, or which is blank only after entity decoding,
// is claimed as a button. The goldens cover an embedded control
// (hostile_control_chars, hostile_href_with_newline_entity) and a literal
// empty href (invalid_empty_attributes), never these two shapes.
//
// Repro from a plain clone of gnolang/gno at 065ec369b (PR #6298):
//
//	git fetch origin pull/6298/head && git checkout --detach 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//	cp <this file> gno.land/pkg/gnoweb/markdown/b4_lines_href_leading_blank_test.go
//	go test ./gno.land/pkg/gnoweb/markdown -run 'TestB4HrefLeadingBlank' -count=1 -v
//	rm gno.land/pkg/gnoweb/markdown/b4_lines_href_leading_blank_test.go
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

func TestB4HrefLeadingBlank(t *testing.T) {
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	for _, href := range []string{"/r/test&#10;x", "&#10;/r/test", "&#9;/r/test", "&#32;", ""} {
		src := []byte(`<gno-button href="` + href + `" label="L" />` + "\n")
		ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
		m := goldmark.New()
		NewGnoExtension().Extend(m)
		node := m.Parser().Parse(text.NewReader(src), ctx)
		var out bytes.Buffer
		if err := m.Renderer().Render(&out, src, node); err != nil {
			t.Fatal(err)
		}
		t.Logf("href %-16q allowed=%-5v -> %s", href, isButtonHrefAllowed([]byte(href)), strings.TrimSpace(out.String()))
	}
}
