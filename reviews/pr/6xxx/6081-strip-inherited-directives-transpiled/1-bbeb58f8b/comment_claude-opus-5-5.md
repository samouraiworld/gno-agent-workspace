# Review: [#6081](https://github.com/gnolang/gno/pull/6081)
Posted: https://github.com/gnolang/gno/pull/6081#pullrequestreview-5366156970
Verdict: REQUEST CHANGES. The Warning at `transpiler.go:194` blocks: a multi-line `/*line ...*/` block used as a doc comment under an `import ( ... )` block loses lines, the shape the ADR says keeps its line count; the rest is one missing test and two Nits.
Event: COMMENT
Model: claude-opus-5-5, deep review; finder xhigh, judge and writer high
Commit: bbeb58f8b
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/omarsy/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2) · [vscode.dev](https://vscode.dev/github/omarsy/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6081 bbeb58f8b`
Round: 1. Solo round: one finder, then one judge and writer answering the completeness question in place of a reflector; 7 candidates, 5 from the finder and 2 from the completeness pass, each rerun by the judge. The first judge run went off-task and was re-run with its task restated; the finder's candidates were reused. No text pass ran; the parent's writing-style pass stands in for it. The #6078 overlap was read, not built.

## Body

> AI review, claude-opus-5-5, deep review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6081-strip-inherited-directives-transpiled/overview.md) · Status: REQUEST CHANGES

## gnovm/pkg/transpiler/transpiler.go:194 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/transpiler.go#L194) · [↗](../../../../../.worktrees/gno-review-6081/gnovm/pkg/transpiler/transpiler.go#L194) · Warning [posted](https://github.com/gnolang/gno/pull/6081#discussion_r4144553282)

The printer drops the blank lines of this emptied block when it is a doc comment below an `import ( ... )` block, so a five-line `/*line ...*/` block prints as two and every later line of the generated file sits three lines above its `.gno` source.

<details><summary>Repro</summary>

```go
// gnovm/pkg/transpiler/zz_blockline_drift_test.go
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
```

```sh
# from a local clone of gnolang/gno:
gh pr checkout 6081 -R gnolang/gno
# write the test above to gnovm/pkg/transpiler/zz_blockline_drift_test.go, then:
cd gnovm/pkg/transpiler && go test -count=1 -run TestBlockLineDocDrift -v .
```

The doc-position block with an import block prints three lines short, further off than without the neutralization:

```text
zz_blockline_drift_test.go:75: doc-6-noimp: src=10 neutralized=10 unneutralized=12
zz_blockline_drift_test.go:75: doc-4: src=10 neutralized=9 unneutralized=12
zz_blockline_drift_test.go:75: doc-6: src=12 neutralized=9 unneutralized=14
zz_blockline_drift_test.go:77: doc-6: neutralizing moved positions further than the directive did (-3 vs +2 lines)
zz_blockline_drift_test.go:75: float-6: src=13 neutralized=13 unneutralized=13
--- FAIL: TestBlockLineDocDrift (0.01s)
```

The same test at the merge base passes, with `doc-6: src=12 neutralized=14 unneutralized=14`. The neutralized `doc-6` body prints the block as `/*` and ` */` on two lines. No added test reaches this shape: [`TestLineParityAcrossShapes`](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/directives_test.go#L275) builds every block as `"/*\n" + ind + body`, so none starts with `/*line `, and [`TestTranspilePreservesLineCount`](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/directives_test.go#L79) carries a single-line block only.

</details>

## gnovm/pkg/transpiler/transpiler.go:213 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/transpiler.go#L213) · [↗](../../../../../.worktrees/gno-review-6081/gnovm/pkg/transpiler/transpiler.go#L213) · Missing test [posted](https://github.com/gnolang/gno/pull/6081#discussion_r4144553293)

Missing test: a tab-separated `//go:generate` inside a block comment; with the `\t` check removed, every added test stays green and the directive prints at column 1.

<details><summary>Repro</summary>

```go
// gnovm/pkg/transpiler/zz_tab_test.go
package transpiler

import (
	"strings"
	"testing"
)

func TestTabGenerateReachesColumnOne(t *testing.T) {
	src := "package tr\n\nimport (\n\t\"errors\"\n)\n\n/*\n//go:generate\techo PWNED\n*/\nfunc F() error { return errors.New(\"x\") }\n"
	res, err := Transpile(src, "gno", "tr.gno")
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(res.Translated, "\n") {
		if strings.HasPrefix(line, "//go:generate\t") {
			t.Errorf("live go:generate at column 1: %q", line)
		}
	}
}
```

```sh
# from a local clone of gnolang/gno:
gh pr checkout 6081 -R gnolang/gno
# write the test above to gnovm/pkg/transpiler/zz_tab_test.go, then:
cd gnovm/pkg/transpiler
sed -i 's|!strings.HasPrefix(trimmed, "//go:generate\\t")|true|' transpiler.go
go test -count=1 -run 'TestTranspileStripsInheritedDirectives|TestIsNolintComment|TestTranspilePreservesLineCount|TestTranspileNeutralizes|TestTranspileLeavesNoEmptyCommentGroup|TestTranspileKeepsBlockCommentTerminated|TestNoDirectiveReachesColumnOne|TestLineParityAcrossShapes|TestMixedDocGroupMatchesInertEquivalent|TestOutputCarriesNoLiveDirective|TestNeutralizingPreservesClassification|TestDocGroupLineCountUnchanged' . | tail -1
go test -count=1 -run TestTabGenerateReachesColumnOne -v . | grep -E 'live|^--- '
git checkout -- transpiler.go
```

With the tab check gone, the added suite stays green and the probe finds the directive at column 1:

```text
ok  	github.com/gnolang/gno/gnovm/pkg/transpiler	0.028s
    zz_tab_test.go:16: live go:generate at column 1: "//go:generate\techo PWNED"
--- FAIL: TestTabGenerateReachesColumnOne (0.00s)
```

</details>

## gnovm/pkg/transpiler/directives_test.go:332 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/directives_test.go#L332) · [↗](../../../../../.worktrees/gno-review-6081/gnovm/pkg/transpiler/directives_test.go#L332) · Nit [posted](https://github.com/gnolang/gno/pull/6081#discussion_r4144553299)

Test: [`TestMixedDocGroupMatchesInertEquivalent`](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/directives_test.go#L309) passes whatever the marker is, because [`neutralizeDirective`](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/transpiler.go#L163) maps the `directiveMarker` substituted here to itself.

<details><summary>Mutations</summary>

```sh
# from a local clone of gnolang/gno:
gh pr checkout 6081 -R gnolang/gno
cd gnovm/pkg/transpiler
for m in '//' '// removed directive'; do
  sed -i "s|directiveMarker = \"//gno:removed-directive\"|directiveMarker = \"$m\"|" transpiler.go
  echo "marker=[$m]"
  go test -count=1 -run 'TestMixedDocGroupMatchesInertEquivalent|TestTranspilePreservesLineCount|TestNeutralizingPreservesClassification' -v . | grep -E '^--- (PASS|FAIL)'
  git checkout -- transpiler.go
done
```

Both markers leave this test green; the other two tests are what go red:

```text
marker=[//]
--- FAIL: TestNeutralizingPreservesClassification (0.00s)
--- FAIL: TestTranspilePreservesLineCount (0.00s)
--- PASS: TestMixedDocGroupMatchesInertEquivalent (0.00s)
marker=[// removed directive]
--- FAIL: TestNeutralizingPreservesClassification (0.00s)
--- PASS: TestTranspilePreservesLineCount (0.01s)
--- PASS: TestMixedDocGroupMatchesInertEquivalent (0.00s)
```

The property its header names does hold: compared with the same file printed without the neutralization, all five doc groups keep their line count (10 against 10, and 11 against 11 for the two three-line groups).

</details>

## gnovm/pkg/transpiler/transpiler.go:179 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/transpiler.go#L179) · [↗](../../../../../.worktrees/gno-review-6081/gnovm/pkg/transpiler/transpiler.go#L179) · Nit [posted](https://github.com/gnolang/gno/pull/6081#discussion_r4144553304)

Nit: `// #nosec` passes this check unchanged, and gosec [scans generated files](https://github.com/securego/gosec/blob/8c77519419e934a3e158dbf1a3fbbc0871b8c691/cmd/gosec/main.go#L133) and [honours `#nosec`](https://github.com/securego/gosec/blob/8c77519419e934a3e158dbf1a3fbbc0871b8c691/cmd/gosec/main.go#L91) by default, so the `.gno` source's suppression still applies to the output.

<details><summary>Repro</summary>

```go
// gnovm/pkg/transpiler/zz_nosec_test.go
package transpiler

import (
	"strings"
	"testing"
)

func TestNosecSurvives(t *testing.T) {
	src := "package tr\n\nfunc F() int {\n\tx := 1 // #nosec G404\n\treturn x\n}\n"
	res, err := Transpile(src, "gno", "tr.gno")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("contains #nosec: %v", strings.Contains(res.Translated, "#nosec G404"))
}
```

```sh
# from a local clone of gnolang/gno:
gh pr checkout 6081 -R gnolang/gno
# write the test above to gnovm/pkg/transpiler/zz_nosec_test.go, then:
cd gnovm/pkg/transpiler && go test -count=1 -run TestNosecSurvives -v .
```

The suppression survives into the `.gen.go`:

```text
contains #nosec: true
```

</details>

## SKIP gnovm/pkg/gnolang/directives.go:30 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/gnolang/directives.go#L30) · [↗](../../../../../.worktrees/gno-review-6081/gnovm/pkg/gnolang/directives.go#L30) · Suggestion

Suggestion: [#6078](https://github.com/gnolang/gno/pull/6078) declares its own `isDirectiveText` in package `gnolang`, so the second of the two to merge fails to compile; #6078 can call `IsDirectiveComment` instead.

Not posted: the conflict is with another open pull request, which its author meets at merge, and the redeclaration was read in the #6078 diff, never built.
