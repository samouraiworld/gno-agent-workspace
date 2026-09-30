# Findings in posting order, from round assemble: 5 to post, 0 SKIP, 2 refuted kept out

## gnovm/pkg/transpiler/transpiler.go:194 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/transpiler.go#L194) · Warning
State: CONFIRMED, band: Warning, angle: claims
TL;DR: a multi-line /*line block in doc position, in a file with a parenthesized import block, is blanked to an empty interior that the printer collapses to two lines, so every later position moves up by n-2 lines where the unneutralized block moved them down by 2
Check: cp tests/solo-finder-blockline-doc-drift_test.go gnovm/pkg/transpiler/ && go test -count=1 -run TestBlockLineDocDrift -v ./gnovm/pkg/transpiler: head FAIL on doc-6, merge base PASS via -overlay
Details: transpiler.go:194 returns "/*" + strings.Repeat("\n", len(lines)-1) + "*/"; with an import block format.Node re-parses and formatDocComment renders the blank interior as "/*\n */" (judge probe, doc-6 printed body). Case space over position x import block: doc with import loses lines (3-line block -1, 5-line block -3, against +2 unneutralized, so worse than the merge base from five lines up); doc without import keeps parity (10 vs 12 unneutralized); floating keeps parity (13/13/13). The ADR's table names this shape as restored to parity. No added test reaches it: TestLineParityAcrossShapes builds every block as "/*\n" + ind + body (directives_test.go:275), so the block never starts "/*line ", and TestTranspilePreservesLineCount carries only the single-line /*line forged.gno:99:1*/. The /*generate path already keeps a marker on the line for the same reason.
Evidence: head bbeb58f8b: "doc-6: src=12 neutralized=9 unneutralized=14", "doc-4: src=10 neutralized=9 unneutralized=12", FAIL "(-3 vs +2 lines)"; float-6 13/13/13, doc-6-noimp 10/10/12. Merge base c563c6ba5 via -overlay: "doc-6: src=12 neutralized=14 unneutralized=14", PASS. tests/judge-probes_test.go TestJudgeBlockLineDocOutput: neutralized body prints "/*\n */" in place of the five-line block
Artifact: tests/solo-finder-blockline-doc-drift_test.go

## gnovm/pkg/transpiler/transpiler.go:213 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/transpiler.go#L213) · Missing test
State: CONFIRMED, band: Missing test, angle: tests
TL;DR: no test carries a tab-separated //go:generate inside a block comment, so deleting the tab branch keeps the added suite green while that line reaches column 1 of the output
Check: bash tests/solo-finder-generate-tab-unpinned.sh: added suite ok under the mutant, TestTabGenerateReachesColumnOne FAIL
Details: Every block body in directives_test.go is "//go:generate echo X" with a space; TestNoDirectiveReachesColumnOne varies the prefix bytes, never the separator. With transpiler.go:213 replaced by true, a doc-position block holding "//go:generate\techo PWNED" prints it at column 1.
Evidence: rerun at head under the mutant: added suite "ok  github.com/gnolang/gno/gnovm/pkg/transpiler 0.028s"; probe "live go:generate at column 1: \"//go:generate\techo PWNED\"", "--- FAIL: TestTabGenerateReachesColumnOne"
Artifact: tests/solo-finder-generate-tab-unpinned.sh

## gnovm/pkg/transpiler/directives_test.go:332 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/directives_test.go#L332) · Nit
State: CONFIRMED, band: Nit, angle: tests
TL;DR: TestMixedDocGroupMatchesInertEquivalent stays green whatever directiveMarker is, since the inert source carries the marker, which neutralizeDirective maps to itself, so both sides transpile the same comment text
Check: bash tests/solo-finder-mixed-doc-tautology.sh from the repo root: TestMixedDocGroupMatchesInertEquivalent PASS under both mutants
Details: directives_test.go:332 substitutes directiveMarker into the 'inert' source; neutralizeDirective returns directiveMarker for it (a directive) and returns "//" unchanged (not a directive), so both inputs reach the printer identical. The property the header states does hold: the judge probe compares each of the five shapes with the unneutralized print and the line counts match (10/10 on four, 11/11 on two directives), and both mutants are caught by TestNeutralizingPreservesClassification. The loss is a test that pins nothing and reads as if it did; banded Nit since nothing it claims is unguarded.
Evidence: rerun at head: marker=[//] "--- FAIL: TestNeutralizingPreservesClassification", "--- FAIL: TestTranspilePreservesLineCount", "--- PASS: TestMixedDocGroupMatchesInertEquivalent"; marker=[// removed directive] "--- FAIL: TestNeutralizingPreservesClassification", "--- PASS: TestMixedDocGroupMatchesInertEquivalent". TestJudgeMixedDocVsUnneutralized: "two directives: src=10 neutralized=11 unneutralized=11", PASS
Artifact: tests/solo-finder-mixed-doc-tautology.sh

## gnovm/pkg/gnolang/directives.go:30 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/gnolang/directives.go#L30) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: #6078, open by the same author, adds a second package-level isDirectiveText to package gnolang in mempackage.go, so whichever of the two merges second does not compile
Check: gh pr diff 6078 -R gnolang/gno | grep -n 'func isDirectiveText' (hit inside the mempackage.go hunk); apply that hunk on head and go vet ./gnovm/pkg/gnolang: expect 'isDirectiveText redeclared' | gh pr diff 6078 -R gnolang/gno | grep -n 'func isDirectiveText' (hit inside the mempackage.go hunk); apply that hunk on head and go vet ./gnovm/pkg/gnolang: expect 'isDirectiveText redeclared'
Details: directives.go:30 declares func isDirectiveText(c string) bool in package gnolang; #6078's diff adds the same declaration at its line 873, inside the gnovm/pkg/gnolang/mempackage.go hunk. Go rejects two package-level declarations of one name. Read-settled; the merged state was not built.
Evidence: gh pr view 6078: OPEN omarsy; gh pr diff 6078: 609 "+++ b/gnovm/pkg/gnolang/mempackage.go", 873 "+func isDirectiveText(c string) bool {"

## gnovm/pkg/transpiler/transpiler.go:179 [gh](https://github.com/gnolang/gno/blob/bbeb58f8b83d775a6a1d492d7931827e49ec54c2/gnovm/pkg/transpiler/transpiler.go#L179) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: a `// #nosec` comment reaches the generated file unchanged, while //nolint, the same kind of suppression, is neutralized
Check: read gosec v2 cmd/gosec/main.go:125 (exclude-generated default false) and analyzer.go ignore(); transpile a file with `// #nosec` and grep the output for #nosec
Details: neutralizeDirective matches IsDirectiveComment, isNolintComment and constraint.IsPlusBuild only (transpiler.go:170-182). gosec reads #nosec from comments (analyzer.go:800 findNoSecDirective) and does not skip generated files by default (cmd/gosec/main.go:125), unlike golangci-lint, whose generated-file skip the ADR relies on for //nolint. The ADR calls the //nolint handling belt-and-braces; this is the same belt with one strap missing.
Evidence: tests/judge-probes_test.go TestJudgeNosecSurvives: "contains #nosec: true"; gosec v2.24.8 main.go:125 flag.Bool("exclude-generated", false, ...)
Artifact: tests/judge-probes_test.go
