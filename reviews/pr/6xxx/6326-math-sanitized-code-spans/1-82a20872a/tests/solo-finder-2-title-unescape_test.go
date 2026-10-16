// solo-finder-2: gnoweb rendering of the escaped dollar, link titles included.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6326/head && git checkout --detach 82a20872a7e4eb89cae377e8ef46c5c578b1b639
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_probe_test.go
//   (cd gno.land/pkg/gnoweb/markdown && go test -run 'TestProbe' -count=1 -v .)
//
// Observed at 82a20872a:
//   [a](/r/x "Price $5 v1\.0 \$6")   -> <a href="/r/x" title="Price $5 v1.0 $6">  (title backslashes unescaped)
//   [l]: /r/x "Price $x^{2}$ v1\.0"  -> title="Price $x^{2}$ v1.0"                 (same for a reference definition)
//   [a]( "Price $x^{2}$")            -> text with <math> for x^{2}                (title-shaped text that is not a link)
//   [a]( "Price \$x^{2}\$")          -> plain text, no <math>
//   \$ in paragraph, link label, image alt, strong, alert title, GFM table cell,
//   heading, TOC entry and Lead title -> literal $, no <math>
package markdown

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)


// Each source is what a realm emits after sanitize.InlineText / TableCell /
// LinkTitle at head; `\.` rows are the pre-existing escape, as a baseline.
func TestProbeEscapedDollar(t *testing.T) {
	srcs := []string{
		`Winner: \$x^{2} \+ ADMIN\_{verified}\$ with 12 votes`,
		`[\$x^{2}\$ v1\.0](/r/x)`,
		`![\$x^{2}\$ v1\.0](/img.png)`,
		`**\$x^{2}\$**`,
		"| a | b |\n|---|---|\n| \\$x^{2}\\$ | v1\\.0 |",
		"> [!NOTE] \\$x^{2}\\$ v1\\.0\n> body",
		`[a](/r/x "Price $5 v1\.0 \$6")`,
		`[a](/r/x "Price $x^{2}$")`,
		"<details><summary>$x^{2}$</summary>\n\nbody\n</details>",
		"<h5>$x^{2}$</h5>",
		`Price \\\$5 and \\$y$`,
	}
	gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
	for _, s := range srcs {
		var buf bytes.Buffer
		if err := gm.Convert([]byte(s), &buf); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("SRC  %q\nHTML %s\n", s, buf.String())
	}
	for _, s := range []string{"# Winner \\$x^{2}\\$ v1\\.0\n\n## Pay \\$5 v1\\.0\n\nFirst paragraph pays \\$5 for v1\\.0 and a lot more text so that it is a description of the page.\n"} {
		src := []byte(s)
		doc := gm.Parser().Parse(text.NewReader(src))
		toc, err := TocInspect(doc, src, TocOptions{MinDepth: 1, MaxDepth: 6})
		fmt.Printf("TOC %+v %v\n", toc, err)
		title, desc := Lead(doc, src)
		fmt.Printf("LEAD title=%q desc=%q\n", title, desc)
	}
}

func TestProbeEscapedDollarHeadings(t *testing.T) {
	gm := goldmark.New(goldmark.WithExtensions(extension.GFM, NewGnoExtension()), goldmark.WithParserOptions(parser.WithAutoHeadingID()))
	src := []byte("## Pay \\$x^{2}\\$ v1\\.0\n\n| a | b |\n|---|---|\n| \\$x^{2}\\$ | v1\\.0 \\| c |\n")
	pctx := parser.NewContext()
	doc := gm.Parser().Parse(text.NewReader(src), parser.WithContext(pctx))
	var buf bytes.Buffer
	if err := gm.Renderer().Render(&buf, src, doc); err != nil {
		t.Fatal(err)
	}
	fmt.Println(buf.String())
	toc, _ := TocInspect(doc, src, TocOptions{MinDepth: 2, MaxDepth: 6})
	for _, it := range toc.Items {
		fmt.Printf("TOC title=%q id=%q\n", it.Title, it.ID)
	}
}

func TestProbeTitleFallback(t *testing.T) {
	gm := goldmark.New(goldmark.WithExtensions(NewGnoExtension()))
	for _, s := range []string{
		`[a](/r/x "Price $5 v1\.0 \$6")`,
		`[a]( "Price $x^{2}$")`,
		`[a]( "Price \$x^{2}\$")`,
		"[l]\n\n[l]: /r/x \"Price $x^{2}$ v1\\.0\"",
	} {
		var buf bytes.Buffer
		_ = gm.Convert([]byte(s), &buf)
		fmt.Printf("SRC  %q\nHTML %s\n", s, buf.String())
	}
}
