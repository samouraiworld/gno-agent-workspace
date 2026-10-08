package markdown

// Repro, from a plain clone of gnolang/gno at 56ff1770722eee73f51f207f97554ea1e2ea2b00:
//
//	cp b4-lines-removed-reach-catalog-stroke-head-paints-fills.go \
//	   gno.land/pkg/gnoweb/markdown/zz_b4_stroke_test.go
//	cd gno.land/pkg/gnoweb/markdown
//	go test -count=1 -run TestB4GeneratedGlyphPaint -v .
//
// Expect: every glyph in the generated table paints each shape as its source
// <symbol> does (TestBuildIconTableStrokeInheritance: "a value is dropped only
// where the element would inherit it anyway").
// Observe: icons whose source shapes are filled with no stroke gain the stroke
// that iconHeadStroke adds to the root (stroke="currentColor", 1.3, round), so
// fills grow by half a stroke on every side: battery bars close the gap to the
// outline, calculator keypad dots (r=1, 2 apart) overlap.

import (
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

var b4PaintDefault = map[string]string{
	"fill": "black", "stroke": "none", "stroke-width": "1",
	"stroke-linecap": "butt", "stroke-linejoin": "miter",
}

var b4Shapes = map[string]bool{
	"path": true, "circle": true, "ellipse": true, "line": true,
	"polyline": true, "polygon": true, "rect": true,
}

func b4Norm(v string) string {
	if v == "transparent" {
		return "none"
	}
	return v
}

// b4Paint returns each shape's tag and effective paint, in document order.
func b4Paint(root []html.Attribute, toks []html.Token) []string {
	base := maps.Clone(b4PaintDefault)
	for _, a := range root {
		if _, ok := b4PaintDefault[a.Key]; ok {
			base[a.Key] = b4Norm(a.Val)
		}
	}
	stack := []map[string]string{base}
	var out []string
	skip := 0
	for _, tok := range toks {
		switch tok.Type {
		case html.StartTagToken, html.SelfClosingTagToken:
			if skip > 0 || !iconElements[tok.Data] {
				if tok.Type == html.StartTagToken {
					skip++
				}
				continue
			}
			cur := maps.Clone(stack[len(stack)-1])
			for _, a := range tok.Attr {
				if _, ok := b4PaintDefault[a.Key]; ok {
					cur[a.Key] = b4Norm(a.Val)
				}
			}
			if b4Shapes[tok.Data] {
				s := fmt.Sprintf("%s fill=%s stroke=%s", tok.Data, cur["fill"], cur["stroke"])
				if cur["stroke"] != "none" {
					s += fmt.Sprintf(" width=%s cap=%s join=%s", cur["stroke-width"], cur["stroke-linecap"], cur["stroke-linejoin"])
				}
				out = append(out, s)
			}
			if tok.Type == html.StartTagToken {
				stack = append(stack, cur)
			}
		case html.EndTagToken:
			if skip > 0 {
				skip--
				continue
			}
			if iconElements[tok.Data] && len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return out
}

type b4Symbol struct {
	root []html.Attribute
	toks []html.Token
}

func TestB4GeneratedGlyphPaint(t *testing.T) {
	src := map[string]b4Symbol{}
	var sources [][]byte
	for _, s := range iconSources {
		data, err := fs.ReadFile(s.fsys, s.name)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, data)
		toks, err := ParseHTMLTokens(strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		var name string
		var cur b4Symbol
		for _, tok := range toks {
			if tok.Data == "symbol" && tok.Type == html.StartTagToken {
				id, _ := ExtractAttr(tok.Attr, "id")
				name, cur = strings.TrimPrefix(id, "ico-"), b4Symbol{root: tok.Attr}
				continue
			}
			if tok.Data == "symbol" && tok.Type == html.EndTagToken {
				src[name] = cur
				name = ""
				continue
			}
			if name != "" {
				cur.toks = append(cur.toks, tok)
			}
		}
	}
	table, err := buildIconTable(sources...)
	if err != nil {
		t.Fatal(err)
	}

	var bad []string
	for _, name := range slices.Sorted(maps.Keys(table)) {
		g := table[name]
		toks, err := ParseHTMLTokens(strings.NewReader("<svg " + g.head + ">" + g.body + "</svg>"))
		if err != nil {
			t.Fatal(err)
		}
		want := b4Paint(src[name].root, src[name].toks)
		got := b4Paint(toks[0].Attr, toks[1:len(toks)-1])
		if !slices.Equal(want, got) {
			for i := range min(len(want), len(got)) {
				if want[i] != got[i] {
					t.Logf("%s shape %d\n  source:    %s\n  generated: %s", name, i, want[i], got[i])
					break
				}
			}
			bad = append(bad, name)
		}
	}
	t.Logf("%d of %d glyphs paint a shape differently from their source: %s", len(bad), len(table), strings.Join(bad, " "))
	if len(bad) > 0 {
		t.Fail()
	}
}
