// Judge probes for gnolang/gno#6081 round 1 at bbeb58f8b.
//
// From a plain clone at bbeb58f8b83d775a6a1d492d7931827e49ec54c2:
//
//	cp <this file> gnovm/pkg/transpiler/zz_judge_probes_test.go
//	cd gnovm/pkg/transpiler && go test -count=1 -run 'TestJudge' -v .
//
// TestJudgeBlockLineDocOutput prints the doc-6 body both ways (candidate 1).
// TestJudgeMixedDocVsUnneutralized compares the mixed doc groups of
// TestMixedDocGroupMatchesInertEquivalent against the unneutralized print,
// the property that test's header states (candidate 2).
// TestJudgeNosecSurvives transpiles a "// #nosec" comment (candidate 5).
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

func judgeUnneutralized(t *testing.T, src string) string {
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

func judgeBody(t *testing.T, src string) string {
	t.Helper()
	res, err := Transpile(src, "gno", "tr.gno")
	if err != nil {
		t.Fatal(err)
	}
	_, body, _ := strings.Cut(res.Translated, "//line tr.gno:1:1\n")
	return body
}

func TestJudgeBlockLineDocOutput(t *testing.T) {
	src := "package tr\n\nimport (\n\t\"errors\"\n)\n\n/*line a\nb\nc\nd\nf.gno:9:1*/\nfunc F() error { return errors.New(\"x\") }\n"
	t.Logf("source:\n%s", src)
	t.Logf("neutralized:\n%s", judgeBody(t, src))
	t.Logf("unneutralized:\n%s", judgeUnneutralized(t, src))
}

func TestJudgeMixedDocVsUnneutralized(t *testing.T) {
	for name, doc := range map[string]string{
		"directive then prose":    "//go:noinline\n// prose doc",
		"prose then directive":    "// prose doc\n//go:noinline",
		"prose, directive, prose": "// one\n//go:noinline\n// two",
		"two directives":          "//go:noinline\n//go:nosplit\n// prose",
		"nolint then prose":       "//nolint:gosec\n// prose doc",
	} {
		src := "package tr\n\nimport (\n\t\"errors\"\n)\n\n" + doc + "\nfunc F() error { return errors.New(\"x\") }\n"
		n := strings.Count(judgeBody(t, src), "\n")
		u := strings.Count(judgeUnneutralized(t, src), "\n")
		t.Logf("%s: src=%d neutralized=%d unneutralized=%d", name, strings.Count(src, "\n"), n, u)
		if n != u {
			t.Errorf("%s: line counts differ", name)
		}
	}
}

func TestJudgeNosecSurvives(t *testing.T) {
	src := "package tr\n\nfunc F() int {\n\tx := 1 // #nosec G404\n\treturn x\n}\n"
	body := judgeBody(t, src)
	t.Logf("contains #nosec: %v", strings.Contains(body, "#nosec G404"))
}
