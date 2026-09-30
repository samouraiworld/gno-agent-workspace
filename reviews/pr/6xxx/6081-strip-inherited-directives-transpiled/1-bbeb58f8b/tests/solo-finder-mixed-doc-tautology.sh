#!/usr/bin/env bash
# Repro: TestMixedDocGroupMatchesInertEquivalent compares Transpile(directive)
# with Transpile(marker already in source). The marker is itself a directive by
# gno.IsDirectiveComment, so neutralizeDirective maps it to itself, and the import
# block in build() forces format.Node to re-parse, which erases the only other
# difference (the stale comment End()). Both sides are the same AST: the test
# stays green whatever directiveMarker is, including the empty "//" whose line
# loss the test header says the directive shape exists to prevent.
#
# From a plain clone of gnolang/gno at bbeb58f8b83d775a6a1d492d7931827e49ec54c2, repo root:
#   bash <this file>
# Head bbeb58f8b, observed:
#   MUTANT marker=[//]:                   TestMixedDocGroupMatchesInertEquivalent PASS,
#                                         TestTranspilePreservesLineCount FAIL,
#                                         TestNeutralizingPreservesClassification FAIL
#   MUTANT marker=[// removed directive]: TestMixedDocGroupMatchesInertEquivalent PASS,
#                                         TestNeutralizingPreservesClassification FAIL
set -euo pipefail
cd gnovm/pkg/transpiler
for m in '//' '// removed directive'; do
  sed -i "s|directiveMarker = \"//gno:removed-directive\"|directiveMarker = \"$m\"|" transpiler.go
  echo "MUTANT marker=[$m]"
  go test -count=1 -run 'TestMixedDocGroupMatchesInertEquivalent|TestTranspilePreservesLineCount|TestNeutralizingPreservesClassification' -v . 2>&1 | grep -E '^--- (PASS|FAIL)' || true
  git checkout -q -- transpiler.go
done
