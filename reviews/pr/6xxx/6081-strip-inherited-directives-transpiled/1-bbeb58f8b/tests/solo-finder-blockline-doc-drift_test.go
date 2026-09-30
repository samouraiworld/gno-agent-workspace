// Repro: a multi-line "/*line ...*/" block in doc position, in a file with a
// parenthesized import block, loses lines once neutralized: the diff writes
// "/*" + "\n"*(n-1) + "*/", formatDocComment collapses the blank interior, and
// every position after the comment moves up. Measured against the same file
// transpiled with the neutralization disabled, the ADR's own standard
// ("neutralizing a directive does not make it worse").
//
// From a plain clone of gnolang/gno at bbeb58f8b83d775a6a1d492d7931827e49ec54c2:
//
//	cp <this file> gnovm/pkg/transpiler/zz_blockline_drift_test.go
//	cd gnovm/pkg/transpiler && go test -count=1 -run TestBlockLineDocDrift -v .
//
// Head bbeb58f8b: FAIL, "doc-6: src=12 neutralized=9 unneutralized=14" (3 lines
// lost, against 2 gained without the neutralization); "doc-4: src=10
// neutralized=9 unneutralized=12"; floating and no-import-block shapes keep parity.
// Merge base c563c6ba5 (no neutralization, Transpile equals the unneutralized
// path): PASS. Run there without writing the tree:
//
//	echo '{"Replace":{"'$PWD'/gnovm/pkg/transpiler/zz_blockline_drift_test.go":"<this file>"}}' > /tmp/ov.json
//	cd gnovm/pkg/transpiler && go test -overlay /tmp/ov.json -count=1 -run TestBlockLineDocDrift -v .
package transpiler

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
)

func transpileUnneutralized(t *testing.T, src string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "tr.gno", src, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &transpileCtx{rootDir: gnoenv.RootDir()}
	ctx.importResolver = DefaultResolver(ctx.rootDir)
	tr, err := ctx.transformFile(fset, f)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, tr); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestBlockLineDocDrift(t *testing.T) {
	imp := "import (\n\t\"errors\"\n)\n\n"
	fn := "func F() error { return errors.New(\"x\") }\n"
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}
	for name, src := range map[string]string{
		"doc-4":       "package tr\n\n" + imp + "/*line a\nb\nf.gno:9:1*/\n" + fn,
		"doc-6":       "package tr\n\n" + imp + "/*line a\nb\nc\nd\nf.gno:9:1*/\n" + fn,
		"float-6":     "package tr\n\n" + imp + "/*line a\nb\nc\nd\nf.gno:9:1*/\n\n" + fn,
		"doc-6-noimp": "package tr\n\nimport \"errors\"\n\n/*line a\nb\nc\nd\nf.gno:9:1*/\n" + fn,
	} {
		res, err := Transpile(src, "gno", "tr.gno")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		_, body, _ := strings.Cut(res.Translated, "//line tr.gno:1:1\n")
		s, o, b := strings.Count(src, "\n"), strings.Count(body, "\n"), strings.Count(transpileUnneutralized(t, src), "\n")
		t.Logf("%s: src=%d neutralized=%d unneutralized=%d", name, s, o, b)
		if abs(o-s) > abs(b-s) {
			t.Errorf("%s: neutralizing moved positions further than the directive did (%+d vs %+d lines)", name, o-s, b-s)
		}
	}
}
