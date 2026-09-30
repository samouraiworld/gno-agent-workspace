#!/usr/bin/env bash
# Repro: go generate accepts "//go:generate\t" as well as "//go:generate "
# (cmd/go/internal/generate.isGoGenerate), and neutralizeDirective has a branch
# for the tab form inside block comments. No test in directives_test.go carries a
# tab-separated body, so deleting that branch leaves every added test green while
# a tab-separated directive in a doc-position block reaches column 1 of the output.
#
# From a plain clone of gnolang/gno at bbeb58f8b83d775a6a1d492d7931827e49ec54c2, repo root:
#   bash <this file>
# Expected: the added suite prints "ok" under the mutant; TestTabGenerateReachesColumnOne FAILs.
set -euo pipefail
cd gnovm/pkg/transpiler
sed -i 's|!strings.HasPrefix(trimmed, "//go:generate\\t")|true|' transpiler.go
grep -c 'true {' transpiler.go >/dev/null
cat > zz_tab_test.go <<'GO'
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
GO
echo "added suite under mutant:"
go test -count=1 -run 'TestTranspileStripsInheritedDirectives|TestIsNolintComment|TestTranspilePreservesLineCount|TestTranspileNeutralizes|TestTranspileLeavesNoEmptyCommentGroup|TestTranspileKeepsBlockCommentTerminated|TestNoDirectiveReachesColumnOne|TestLineParityAcrossShapes|TestMixedDocGroupMatchesInertEquivalent|TestOutputCarriesNoLiveDirective|TestNeutralizingPreservesClassification|TestDocGroupLineCountUnchanged' . 2>&1 | tail -1
echo "tab probe under mutant:"
go test -count=1 -run TestTabGenerateReachesColumnOne -v . 2>&1 | grep -E 'live|^--- ' || true
rm -f zz_tab_test.go
git checkout -q -- transpiler.go
