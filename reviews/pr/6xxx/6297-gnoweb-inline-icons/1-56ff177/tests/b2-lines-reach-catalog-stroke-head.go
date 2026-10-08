// Probe: iconHead swaps a System UIcons root for iconHeadStroke, whose
// stroke="currentColor" (and round caps/joins) is inherited by shapes the
// upstream root left unstroked: filled dots and fills gain a 1.3 outline.
//
// Repro from a plain clone of gnolang/gno:
//
//	git checkout 56ff1770722eee73f51f207f97554ea1e2ea2b00
//	cp b2-lines-reach-catalog-stroke-head.go gno.land/pkg/gnoweb/markdown/b2_stroke_head_test.go
//	cd gno.land/pkg/gnoweb && go test ./markdown -run 'TestB2StrokeHead' -v -count=1
//
// Expected at the head: both tests fail. TestB2StrokeHeadMinimal shows the
// upstream menu-horizontal shape (three fill="currentColor" circles, no stroke
// anywhere) coming out under iconHeadStroke with nothing setting stroke="none";
// TestB2StrokeHeadVendored lists every shipped icon with a shape whose
// effective stroke is none in its source and currentColor in the registry.
package markdown

import (
	"bytes"
	"os"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

var b2Shapes = map[string]bool{
	"path": true, "circle": true, "ellipse": true, "line": true,
	"polyline": true, "polygon": true, "rect": true,
}

// b2Strokes returns the effective stroke paint of every shape leaf, in
// document order, for markup whose first start tag is the root.
func b2Strokes(t *testing.T, markup string) []string {
	t.Helper()
	toks, err := ParseHTMLTokens(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}
	var stack []string
	var out []string
	for _, tok := range toks {
		switch tok.Type {
		case html.StartTagToken, html.SelfClosingTagToken:
			cur := "none" // SVG initial value of stroke
			if len(stack) > 0 {
				cur = stack[len(stack)-1]
			}
			if v, ok := ExtractAttr(tok.Attr, "stroke"); ok && !strings.Contains(v, "url(") {
				cur = v
			}
			if b2Shapes[tok.Data] {
				out = append(out, cur)
			}
			if tok.Type == html.StartTagToken {
				stack = append(stack, cur)
			}
		case html.EndTagToken:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return out
}

func TestB2StrokeHeadMinimal(t *testing.T) {
	// Upstream src/images/icons/menu_horizontal.svg body, wrapped as
	// tools/cmd/iconset writes it.
	const src = `<symbol id="ico-m" viewBox="0 0 21 21" stroke-width="1.3">` +
		`<g fill="currentColor" fill-rule="evenodd"><circle cx="10.5" cy="10.5" r="1"/>` +
		`<circle cx="5.5" cy="10.5" r="1"/><circle cx="15.5" cy="10.5" r="1"/></g></symbol>`
	table, err := buildIconTable([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g := table["m"]
	before := b2Strokes(t, src)
	after := b2Strokes(t, "<svg "+g.head+">"+g.body+"</svg>")
	t.Logf("head=%s", g.head)
	t.Logf("body=%s", g.body)
	t.Logf("source strokes=%v registry strokes=%v", before, after)
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("shape %d: stroke %q in the source, %q in the registry", i, before[i], after[i])
		}
	}
}

func TestB2StrokeHeadVendored(t *testing.T) {
	var changed []string
	shapes := 0
	for _, f := range []string{"icons/vendored.svg", "icons/drawn.svg"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sym := range bytes.Split(data, []byte("<symbol "))[1:] {
			s := "<symbol " + string(sym[:bytes.Index(sym, []byte("</symbol>"))]) + "</symbol>"
			table, err := buildIconTable([]byte(s))
			if err != nil {
				t.Fatal(err)
			}
			for name, g := range table {
				before := b2Strokes(t, s)
				after := b2Strokes(t, "<svg "+g.head+">"+g.body+"</svg>")
				if len(before) != len(after) {
					t.Fatalf("%s: %d shapes in, %d out", name, len(before), len(after))
				}
				n := 0
				for i := range before {
					if before[i] == "none" && after[i] != "none" {
						n++
					}
				}
				if n > 0 {
					shapes += n
					changed = append(changed, name)
				}
			}
		}
	}
	sort.Strings(changed)
	if len(changed) > 0 {
		t.Errorf("%d icons, %d shapes: unstroked in the source, stroked in the registry: %s",
			len(changed), shapes, strings.Join(changed, " "))
	}
}
