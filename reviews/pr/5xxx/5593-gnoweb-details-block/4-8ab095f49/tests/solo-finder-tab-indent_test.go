// solo-finder-tab-indent: the round-2 Padding fix holds across 2368 container
// and indentation shapes (no panic, no mangled summary, cap == len), while a
// tab inside the <=3-column indent after a container is missed: goldmark's
// fenced code opens on the same line and :::details does not, and a `> \t:::`
// closer is ignored, so the rest of the quote folds into the block.
//
// Repro from a plain clone of gnolang/gno at 8ab095f495322cb3f1577ce39d1440de5c141477:
//
//	cp solo-finder-tab-indent_test.go gno.land/pkg/gnoweb/markdown/zz_tab_indent_test.go
//	go test ./gno.land/pkg/gnoweb/markdown/ -run 'TestProbe(TabMatrix|TabCloser)' -v | grep -E 'cases=|DIVERGE|SRC|OUT|<p>'
//
// Expected at head: cases=2368 panics=0 mangled=0 diverge=92; TestProbeTabCloser
// shows "body\n:::\nafter" in one paragraph for the `> \t:::` and `  \t:::` closers.
package markdown

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func probeParse(src []byte) (doc ast.Node, out string, depth int, perr any) {
	defer func() { perr = recover() }()
	u, _ := weburl.Parse("https://gno.land/r/test")
	m := goldmark.New()
	NewGnoExtension().Extend(m)
	pc := NewGnoParserContext(GnoContext{GnoURL: u})
	doc = m.Parser().Parse(text.NewReader(src), parser.WithContext(pc))
	var buf bytes.Buffer
	_ = m.Renderer().Render(&buf, src, doc)
	return doc, buf.String(), Get(pc), nil
}

func findKind(doc ast.Node, k ast.NodeKind) ast.Node {
	var found ast.Node
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && found == nil && n.Kind() == k {
			found = n
		}
		return ast.WalkContinue, nil
	})
	return found
}

// exact returns a copy whose cap equals its len, as handler_http.go builds it.
func exact(s string) []byte { b := make([]byte, len(s)); copy(b, s); return b }

func TestProbeTabMatrix(t *testing.T) {
	conts := []string{"", ">", "> ", ">\t", ">  ", "> \t", "-", "- ", "-\t", "1.\t", "1. ", "* \t"}
	inds := []string{"", " ", "  ", "   ", "\t", " \t", "  \t", "   \t"}
	var srcs []string
	for _, a := range conts {
		for _, b := range conts {
			for _, i := range inds {
				srcs = append(srcs, a+b+i+"OPEN")
			}
		}
	}
	for _, i := range inds {
		srcs = append(srcs, "- item\n"+i+"OPEN", "> - item\n>"+i+"OPEN", "1. item\n"+i+"OPEN", "- item\n\n"+i+"OPEN")
	}
	panics, mangled, diverge := 0, 0, 0
	for _, s := range srcs {
		for _, end := range []string{"", "\n"} {
			ds := strings.ReplaceAll(s, "OPEN", ":::details Summary") + end
			fs := strings.ReplaceAll(s, "OPEN", "```go") + end
			doc, _, depth, perr := probeParse(exact(ds))
			if perr != nil {
				panics++
				t.Logf("PANIC %q: %v", ds, perr)
				continue
			}
			if depth != 0 {
				t.Logf("DEPTH %d %q", depth, ds)
			}
			db := findKind(doc, KindDetailsBlock)
			if db != nil {
				if sum := findKind(db, KindDetailsSummary); sum != nil {
					got := string(sum.Lines().Value(exact(ds)))
					if got != "Summary" {
						mangled++
						t.Logf("MANGLED %q -> %q", ds, got)
					}
				} else {
					mangled++
					t.Logf("NOSUMMARY %q", ds)
				}
			}
			fdoc, _, _, _ := probeParse(exact(fs))
			fc := findKind(fdoc, ast.KindFencedCodeBlock)
			if (fc != nil) != (db != nil) {
				diverge++
				t.Logf("DIVERGE fenced=%v details=%v %q", fc != nil, db != nil, ds)
			}
		}
	}
	t.Logf("cases=%d panics=%d mangled=%d diverge=%d", len(srcs)*2, panics, mangled, diverge)
}


func TestProbeTabCloser(t *testing.T) {
	for _, src := range []string{
		"> :::details A\n> body\n> \t:::\n> after\n",
		"> :::details A\n> body\n>\t:::\n> after\n",
		"- :::details A\n  body\n  \t:::\n  after\n",
	} {
		_, out, _, _ := probeParse([]byte(src))
		fmt.Printf("SRC %q\nOUT %s\n", src, out)
	}
}
