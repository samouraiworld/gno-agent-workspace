# Claims: gnolang/gno#6081 round 1, bbeb58f8b, claude-opus-5-5, solo review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gnovm/pkg/transpiler/transpiler.go:194 | cp tests/solo-finder-blockline-doc-drift_test.go gnovm/pkg/transpiler/ && go test -count=1 -run TestBlockLineDocDrift -v ./gnovm/pkg/transpiler: head FAIL on doc-6, merge base PASS via -overlay | head bbeb58f8b: "doc-6: src=12 neutralized=9 unneutralized=14", "doc-4: src=10 neutralized=9 unneutralized=12", FAIL "(-3 vs +2 lines)"; float-6 13/13/13, doc-6-noimp 10/10/12. Merge base c563c6ba5 via -overlay: "doc-6: src=12 neutralized=14 unneutralized=14", PASS. tests/judge-probes_test.go TestJudgeBlockLineDocOutput: neutralized body prints "/*\n */" in place of the five-line block | tests/solo-finder-blockline-doc-drift_test.go | hot |
| 2 | CONFIRMED | Nit | gnovm/pkg/transpiler/directives_test.go:332 | bash tests/solo-finder-mixed-doc-tautology.sh from the repo root: TestMixedDocGroupMatchesInertEquivalent PASS under both mutants | rerun at head: marker=[//] "--- FAIL: TestNeutralizingPreservesClassification", "--- FAIL: TestTranspilePreservesLineCount", "--- PASS: TestMixedDocGroupMatchesInertEquivalent"; marker=[// removed directive] "--- FAIL: TestNeutralizingPreservesClassification", "--- PASS: TestMixedDocGroupMatchesInertEquivalent". TestJudgeMixedDocVsUnneutralized: "two directives: src=10 neutralized=11 unneutralized=11", PASS | tests/solo-finder-mixed-doc-tautology.sh | cold |
| 3 | CONFIRMED | Missing test | gnovm/pkg/transpiler/transpiler.go:213 | bash tests/solo-finder-generate-tab-unpinned.sh: added suite ok under the mutant, TestTabGenerateReachesColumnOne FAIL | rerun at head under the mutant: added suite "ok github.com/gnolang/gno/gnovm/pkg/transpiler 0.028s"; probe "live go:generate at column 1: \"//go:generate\techo PWNED\"", "--- FAIL: TestTabGenerateReachesColumnOne" | tests/solo-finder-generate-tab-unpinned.sh | hot |
| 4 | CONFIRMED | Suggestion | gnovm/pkg/gnolang/directives.go:30 | gh pr diff 6078 -R gnolang/gno \| grep -n 'func isDirectiveText' (hit inside the mempackage.go hunk); apply that hunk on head and go vet ./gnovm/pkg/gnolang: expect 'isDirectiveText redeclared' \| gh pr diff 6078 -R gnolang/gno \| grep -n 'func isDirectiveText' (hit inside the mempackage.go hunk); apply that hunk on head and go vet ./gnovm/pkg/gnolang: expect 'isDirectiveText redeclared' | gh pr view 6078: OPEN omarsy; gh pr diff 6078: 609 "+++ b/gnovm/pkg/gnolang/mempackage.go", 873 "+func isDirectiveText(c string) bool {" |  | warm |
| 5 | CONFIRMED | Nit | gnovm/pkg/transpiler/transpiler.go:179 | read gosec v2 cmd/gosec/main.go:125 (exclude-generated default false) and analyzer.go ignore(); transpile a file with `// #nosec` and grep the output for #nosec | tests/judge-probes_test.go TestJudgeNosecSurvives: "contains #nosec: true"; gosec v2.24.8 main.go:125 flag.Bool("exclude-generated", false, ...) | tests/judge-probes_test.go | hot |
| 6 | REFUTED | Warning | gnovm/pkg/transpiler/directives_test.go:20 | go test -race -count=1 -run 'Directive\|Transpile\|Nolint\|LineParity\|Mixed\|Neutraliz\|DocGroup\|OutputCarries\|NoDirective' ./gnovm/pkg/transpiler at head | go test -race over the added transpiler tests: ok, no race report |  | cold |
| 7 | REFUTED | Suggestion | gnovm/pkg/transpiler/transpiler.go:155 | grep -rnE '//gno:[a-z]' over gnovm, gno.land and examples, *.go and *.gno, excluding the marker itself: expect no reader | grep for //gno:[a-z] outside the marker: no hit |  | hot |

Hit rate per tier, from the rows above: hot 3/4 confirmed over 2 files, warm 1/1 confirmed over 1 files, cold 1/2 confirmed over 2 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | gnovm/adr/pr6081_transpile_strip_directives.md:156 | ADR says a //go:noinline inside a block comment ships in strings/builder.gno and reaches the output unchanged | grep -n 'nocheckptr\\|nosplit' gnovm/stdlibs/strings/builder.gno -> "26://go:nosplit", "27://go:nocheckptr": line comments the diff does neutralize; only the ADR wording is wrong |
| reach | gnovm/pkg/transpiler/transpiler.go:138 | Result.File carries comments whose Text changed but whose positions did not, so End() disagrees with the fset | grep of transpile-result consumers: gnovm/cmd/gno/tool_transpile.go:264-296 reads only transpileRes.Translated and transpileRes.Imports; .File is read only by directives_test.go |
| lines | gnovm/pkg/transpiler/transpiler.go:251 | Doc-comment reformatting could move a //go:generate to column 1 from a line the prefix trim did not match | GOROOT go/printer: commonPrefix strips only `a[i] <= ' ' \|\| a[i] == '*'`, which trimPrintedCommentPrefix strips in full; formatDocComment emits pr.Comment(d), which only adds indentation or list markers, never removes leading non-space bytes |
| lines | gnovm/pkg/transpiler/transpiler.go:267 | isNolintComment might miss a spelling golangci-lint honours | golangci-lint v2.11.4 pkg/result/processors/nolint_filter.go:83 `regexp.MustCompile(`^nolint( \|:\|$)`)` after TrimLeft "/ "; isNolintComment accepts that set plus a tab |
| catalog | gnovm/pkg/transpiler/transpiler.go:191 | //go:build left inside a block comment trips vet's buildtag check | gnovm/cmd/gno/tool_transpile.go:408 `args := append(make([]string, 0, 5), "build")`: -gobuild runs go build, whose parseFileHeader skips /* */ content; no gno path runs go vet on transpiled output |

## Parent check

- Row 1 rerun by the parent through `go test -overlay` at head bbeb58f8b and at the merge base c563c6ba5: head FAIL, "doc-6: src=12 neutralized=9 unneutralized=14"; base PASS, "doc-6: src=12 neutralized=14 unneutralized=14".
- Rows 2, 3 and 5 rerun by the parent with mutated copies passed through `-overlay`, the worktree untouched: every observed line in the draft reproduced, and all twelve test names in row 3's `-run` pattern exist at head.
- Row 4 ships `SKIP`: the conflict is with another open pull request and the redeclaration was read, never built.
- Row 5 moves from Suggestion to Nit: the proposed fix, matching `#nosec` beside `isNolintComment`, was never applied and run, so the section states the gap alone. The judge cited gosec at tag v2.24.8, which answers 404; the draft links master at 8c77519419e9, where `-exclude-generated` defaults to false on line 133 and `-nosec` to false on line 91.

## Completeness

- Thin angle: the block `/*line ...*/` form had no shape test. `TestLineParityAcrossShapes` (directives_test.go:275) repeats the block shape as `"/*\n" + ind + body` and never opens on `/*line `, which is the sibling that let row 1 through; folded into row 1 rather than filed apart.
- Catalog: Global mutable state & concurrency applies, since every added test runs `t.Parallel()`; `go test -race` over them is clean (row 6). Determinism: `stripInheritedDirectives` walks `f.Comments` slices, no map. Gas, realm state, caller control, coin, storage deposit, VM-fault, VM semantics and type-check are not touched: the diff changes only the transpiler's Go output and a string predicate.
- Sibling sweep by shape, `format.Node`/`printer.Fprint` over parsed Gno: `gnovm/cmd/gno/fmt.go:336`, `fix.go:335`, `gnovm/pkg/gnofmt/processor.go:192`, `gnovm/pkg/doc/json_doc.go:442`, `doc/print.go:257`. By path they rewrite Gno source or render docs, not files the Go toolchain reads; traced by path only, so unverified.
- Marker name: no `//gno:` pragma reader exists (row 7).
- PR body and threads: no review threads, one bot comment. The body says directive comments are dropped from the AST and shows output without the marker lines, while the code replaces them with `//gno:removed-directive`; the ADR's Alternatives item still says blanking leaves a bare `//`, against its own Decision; and the finder's settled row on ADR line 156 names `//go:noinline` where `strings/builder.gno` carries `//go:nosplit` and `//go:nocheckptr`. These are ADR and description Nits, unwritten in the draft this round.
- The judge's gate hook named `skills/writing-style.md` and the review rule set as unread; the round prompt limited reads to `rules-solo.md`, so the draft follows that file's excerpts only.
