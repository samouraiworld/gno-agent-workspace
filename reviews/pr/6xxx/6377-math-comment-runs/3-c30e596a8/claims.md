# Claims: gnolang/gno#6377 fix(gnoweb): stop malformed math crashing, and parse comment and \color runs in linear time round 3, c30e596a8, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:378 | TestZZSideset and TestZZJudge (tests/judge-1-sideset-line378.go) against master copied to zzbase; then line 378 alone reverted to master's guard | \sideset{\left}^a}}{}x base err=<nil> ...<mprescripts></mprescripts><none></none><none></none></mmultiscripts>; head err=<nil> ...<mprescripts></mprescripts><none></none><mi>a</mi></mmultiscripts>. Same for \bigl}; \sideset{^b\left}^a}}{}x adds <none></none><mi>a</mi>. Controls same=true: \sideset{\left(^a}{}x, \sideset{^a}{}x, x\left}^a (both: mismatched curly brace). Line 378 reverted: TestZZSideset ok, TestMalformedInputDoesNotCrash FAIL on \cr%c...\Bigl}}\middle. | tests/judge-1-sideset-line378.go | hot |  |
| 2 | REFUTED | Missing test | gno.land/pkg/gnoweb/markdown/mathml/edgecases_test.go:297 | python3 tests/judge-reflector-test-mutations.py <head worktree> gno.land/pkg/gnoweb/markdown/mathml | edgecases_test.go:15 out, err := convertErrWithin(t, tex) in mathBody; all 8 mutations RED | tests/judge-reflector-test-mutations.py | cold |  |
| 3 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/mathml/mmlnode_test.go:214 | read every convertWithin caller's input (mmlnode_test.go:84, 244-245) | mmlnode_test.go:84 and :244 pass only short literal inputs |  |  |  |

Hit rate per tier, from the rows above: hot 1/1 confirmed over 2 files, warm 0/0 confirmed over 1 files, cold 0/1 confirmed over 2 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| user topic: no input crashes | gno.land/pkg/gnoweb/markdown/mathml/parse.go:229 | some other input still crashes the branch's converter | Ran the differential fuzzer tests/solo-finder-diff-fuzz.go (TestZZDiff and TestZZGram, seeds 1,2,3,5,6,7,11,12; 5.4M expressions) and FuzzZZ (go test -fuzz FuzzZZ -fuzztime 150s, 4,439,246 execs). Every run reported headCrash=0 and regress=0. Master crashed on 72,879 of 800,000 in seed 11 alone. |
| user topic: claimed defects reproduce through NewGnoExtension | gno.land/pkg/gnoweb/markdown/mathml/parse.go:229 | the crash does not show as an 'invalid math input' error in a realm page | TestZZPageCrash in tests/solo-finder-page-repro.go, run with master's mathml: FALLBACK "$\\bf\\sqrt[a&b]x$" -> <p>Text <span class="math-inline">\bf\sqrt[a&amp;b]x</span> end.</p>. All six crash inputs fall back to raw source, and all six give CONVERTED at head. The defect reproduces: the whole expression goes unrendered. Only the body's wording differs from what a page shows, because ext_math.go:919-960 discards the converter's error MathML and prints the escaped source. |
| user topic: claimed defects reproduce through NewGnoExtension | gno.land/pkg/gnoweb/markdown/mathml/commands.go:312 | the bare \color quadratic cannot be reached through a realm page, since MaxMathInputLen caps an expression at 8 KiB | TestZZPageColor (best of 5), bare \color k=170/341/682/1365 in $...$. Master: 0.30/0.75/2.21/6.54 ms, about 2.5x, 2.9x and 3.0x per doubling. Head: 0.25/0.58/0.47/0.98 ms. A 1 MiB page of 128 such expressions: master 176 ms, head 48 ms, with 19 converted before the page budget ran out. The defect reproduces up to the cap. The 35.8 s row uses 131,072 commands, past MaxMathInputLen (ext_math.go:21, 8<<10), so no page reaches it. The table labels that row ConvertInline, so this is wording only. |
| user topic: linear runs | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:350 | a comment run, or a bare \color run, is still walked once per element on some path | TestZZPageColor, indented % lines k=256/512/1024/2040 in $$...$$. Master: 0.35/1.04/3.45/12.3 ms. Head: 0.14/0.26/0.50/1.17 ms, linear. Bare \color at head: 0.25 to 0.98 ms up to 8 KiB, linear. The fuzzer's slowest head conversion was 61 ms, on a 300-token mixed input. |
| lines | gno.land/pkg/gnoweb/markdown/mathml/commands.go:336 | colorArgMissing answers differently from the old GetOptions/GetNextExpr on the sliced cell. The cases are a [ closing at or past cellEnd, and a { closing past the cell | By reading: checkGroupNesting unpairs any [ ] that straddles a group, and cellEnd steps over whole groups, so a { reached before cellEnd closes before it. A [ whose closer is at index n falls on the merror path both ways. The fuzzer agrees: TestZZGram over \color, \color[RGB], \color{red} and \color[RGB]{1,2,3} found diffs=0 outside the \left} shape. |
| catalog: global mutable state | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:321 | the cellEnds table or last goes stale when Expr changes, or is shared between goroutines | grep -n 'TokenBuffer{\\|\.Expr = \\|\.Expr\[...\]\. =' over the package matches only tokenize.go:316 &TokenBuffer{Expr: t, idx: 0, last: last}. Nothing writes Expr after construction, and every buffer is built per render (mathml.go:70). |
| reach | gno.land/pkg/gnoweb/markdown/mathml/edgecases_test.go:305 | under -race in CI, TestBareColorRunIsLinear's 2 s bound could time out | grep -n race .github/workflows/ci-race.yml. Its only gnoweb line is ci-race.yml:62, go test -race ... -run 'TestMathConcurrentRender' ./gno.land/pkg/gnoweb/markdown/, so the mathml package never runs under -race. |

## Completeness

- Tests angle, thin in the finder's return: the PR body says reverting any one fix alone turns an added test red. Row 2 ran it, eight reverts each alone, and every one went red; `mathBody` goes through `convertErrWithin`, so the two linear tests carry the 2 s bound.
- Sibling of an edited test helper, swept by shape (`Fatalf(` with `%q` over `tex`): `convertWithin` at `mmlnode_test.go:214` keeps `%q` where `convertErrWithin` moved to `%.80q`. Row 3: its callers pass only short literals, so nothing floods the log.
- Sibling of the edited guard, swept by shape (`MatchOffset > 0 && ... tokEscaped == 0` without the bound `matched` adds): the package-level `GetNextExpr` at `tokenize.go:254` lacks it, and its only callers, `tokenize.go:717` and `:724`, pass the whole token list, where every `MatchOffset` lies within it. Not a crash site; no candidate.
- Candidate 1's case space: the folded `}` inside `\sideset` differs (`\left}`, `\bigl}`, with and without a script before it); a folded `(` does not (`\sideset{\left(^a}{}x` same); a folded `}` outside any group fails with `mismatched curly brace` on both sides. Reverting `tokenize.go:378` alone restores master's output and reopens the `\Bigl}}` crash in `TestMalformedInputDoesNotCrash`, so the difference is the price of that crash fix, on input that is not valid TeX.
- Catalog: `invariant-catalog.md` holds realm audit patterns and caller identity predicates, which a gnoweb Go converter does not touch; the finder's global-mutable-state check is the one class that applies, settled above.

## Retro

- What failed: nothing died, no cap was hit, every turn ran on claude-opus-5-5.
- What worked: the finder's differential fuzzing, 5.4M grammar-built expressions and 4.4M fuzzer runs, found the one shape whose output moves, and the page probes settled the user topic that every claimed defect reproduces through `NewGnoExtension`.
- Hit rate per tier: hot 1 of 1 confirmed; cold 0 of 2.
- Cost: 2 agents, 53 turns, 77k output, 6.6M cache read, 0.34M cache write, about $5.57, 26 minutes.
- Applied: no code change; the body and overview now scope the identical-output claim to valid TeX and say a crash showed the expression as source in a page.
