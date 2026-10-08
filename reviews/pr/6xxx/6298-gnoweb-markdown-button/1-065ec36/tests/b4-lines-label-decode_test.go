// b4-lines-label-decode: buttonLabel (ext_buttons.go:96) decodes the label with
// html.UnescapeString, which applies the text-content rules (attribute=false):
// a legacy entity name with no ';' is decoded even when an alphanumeric or '='
// follows it. HTML decodes attribute text the other way and leaves those
// literal, which is what the godoc at ext_buttons.go:89 and the ADR
// ("decoded as HTML decodes attribute text") promise. The x/net/html
// tokenizer behind ParseHTMLTokens, used by the sibling block gno-* tags, is
// the attribute-mode reference here.
//
// Second half: a label made only of invisible format characters outside the
// StripBidiAndZeroWidth set (U+2060, U+00AD, U+3164) is not "no label".
//
// Repro from a plain clone of gnolang/gno at 065ec369b (PR #6298):
//
//	git fetch origin pull/6298/head && git checkout --detach 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//	cp <this file> gno.land/pkg/gnoweb/markdown/b4_lines_label_decode_test.go
//	go test ./gno.land/pkg/gnoweb/markdown -run 'TestB4Label' -count=1 -v
//	rm gno.land/pkg/gnoweb/markdown/b4_lines_label_decode_test.go
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

func b4LabelRender(t *testing.T, input string) string {
	t.Helper()
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatal(err)
	}
	ctx := parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
	m := goldmark.New()
	NewGnoExtension().Extend(m)
	src := []byte(input)
	node := m.Parser().Parse(text.NewReader(src), ctx)
	var out bytes.Buffer
	if err := m.Renderer().Render(&out, src, node); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out.String())
}

func TestB4LabelDecodeAttributeMode(t *testing.T) {
	for _, raw := range []string{"email&notifications", "Lock&amplify", "x&copy=1", "Buy&regular"} {
		tag := `<gno-button href="/r/x" label="` + raw + `" />`
		toks, err := ParseHTMLTokens(strings.NewReader(tag))
		if err != nil || len(toks) != 1 {
			t.Fatalf("tokenize %q: %v %d", tag, err, len(toks))
		}
		htmlAttr, _ := ExtractAttr(toks[0].Attr, "label")
		got := buttonLabel([]byte(raw))
		mark := "same"
		if got != htmlAttr {
			mark = "DIFFERS"
		}
		t.Logf("raw %-22q buttonLabel %-22q html attribute %-22q %s", raw, got, htmlAttr, mark)
	}
	t.Logf("rendered: %s", b4LabelRender(t, `<gno-button href="/r/x" label="email&notifications" />`+"\n"))
}

func TestB4LabelInvisibleOnly(t *testing.T) {
	for _, label := range []string{"&#x2060;", "&shy;", "ㅤ", "&#x200B;"} {
		t.Logf("label %-10q -> %s", label, b4LabelRender(t, `<gno-button href="/r/x" label="`+label+`" />`+"\n"))
	}
}
