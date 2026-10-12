// Repro, from a plain clone of gnolang/gno:
//
//	git fetch origin pull/5593/head && git checkout --detach fa1ce31087d2ff53ae775bb71a1f52362543e45e
//	cp <this file> gno.land/pkg/gnoweb/markdown/solo_tab_padding_test.go
//	go test ./gno.land/pkg/gnoweb/markdown/ -run 'TestSoloDetails(TabPadding|DepthBalanced)' -v
//
// Claim: Open builds the summary segment from indices into PeekLine's
// line, which carries segment.Padding leading spaces when a container
// consumed part of a tab (`>\t`, a list item's tab continuation), and never
// subtracts that padding (goldmark's own atx heading parser does). The
// summary is then shifted right by the padding: its first bytes are lost
// and the bytes after the line end, the newline and the next line's prefix,
// are rendered inside <summary>.
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

func soloRender(t *testing.T, src string) (string, int) {
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
	return buf.String(), Get(pc)
}

func TestSoloDetailsTabPadding(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"control_quote_space", "> :::details Summary text\n> body\n> :::\n", "</svg>Summary text</summary>"},
		{"control_heading_quote_tab", ">\t# Heading text\n", "<h1>Heading text</h1>"},
		{"quote_tab", ">\t:::details Summary text\n>\tbody\n>\t:::\n", "</svg>Summary text</summary>"},
		{"alert_tab", "> [!NOTE]\n>\t:::details Summary text\n>\tbody\n>\t:::\n", "</svg>Summary text</summary>"},
		{"list_tab_continuation", "- item\n\t:::details Summary text\n\tbody\n\t:::\n", "</svg>Summary text</summary>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, _ := soloRender(t, c.src)
			t.Logf("output:\n%s", out)
			if !strings.Contains(out, c.want) {
				t.Errorf("want %q in output", c.want)
			}
		})
	}
}

// Every successful Push must be matched by one Pop: after a parse the
// shared gno-* depth is back to 0, whatever shape closed the blocks.
func TestSoloDetailsDepthBalanced(t *testing.T) {
	cases := map[string]string{
		"closed":        ":::details a\nb\n:::\n",
		"unclosed":      ":::details a\nb\n",
		"over_cap":      strings.Repeat(":::details a\n", 10),
		"in_quote_lazy": "> :::details a\n> b\n\nafter\n",
		"in_list_ends":  "- :::details a\n  b\n\npara\n",
		"sequential_6":  strings.Repeat(":::details a\nb\n:::\n", 6),
		"then_alerts":   strings.Repeat(":::details a\nb\n:::\n", 5) + "> [!NOTE]\n> x\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			out, d := soloRender(t, src)
			if d != 0 {
				t.Errorf("depth after parse = %d, want 0\n%s", d, out)
			}
			if name == "then_alerts" && !strings.Contains(out, "gno-alert") {
				t.Errorf("alert after 5 closed details refused:\n%s", out)
			}
		})
	}
}
