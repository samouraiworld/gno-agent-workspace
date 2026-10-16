# Claims: gnolang/gno#6377 perf(gnoweb): parse a run of math comments in linear time round 1, 9a2b61f5f, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:655 | In a scratch tree at 9a2b61f5f: copy tests/judge-1-fence-comment.go to gno.land/pkg/gnoweb/markdown/mathml/zz_fence_comment_test.go and run go test -count=1 -run TestZZFenceComment there; then sed -i '655s/toks\[j\].Kind&(tokWhitespace\|tokComment) > 0/toks[j].Kind\&tokWhitespace > 0/' tokenize.go, rerun it, remove it and run go test -count=1 ./gno.land/pkg/gnoweb/markdown/... | head: `ok .../markdown/mathml 0.003s` for TestZZFenceComment; mutated line 655 `for j < len(toks) && toks[j].Kind&tokWhitespace > 0 {`: `--- FAIL: TestZZFenceComment`, messages `\left%c`, `\left(x\middle%c`, `\left(x\right%c`, `\big%c`, `\left %c`; mutated suites without the new test: `ok .../gnoweb/markdown 10.718s`, `ok .../markdown/mathml 0.180s` | projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6377-math-comment-runs/1-9a2b61f5f/tests/judge-1-fence-comment.go | hot |  |
| 2 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/markdown/mathml/commands.go:325 | Copy tests/solo-finder-color-run.go to gno.land/pkg/gnoweb/markdown/mathml/zz_color_test.go and run go test -count=1 -run TestZZColorRun -v at 9a2b61f5f and with tokenize.go at 6c49f5420 (the only non-test file the diff changes) | head: `color x8192 (49154 bytes): 62ms`, `color x16384 (98306 bytes): 221ms`, `color x32768 (196610 bytes): 905ms`; base tokenize.go: `60ms`, `224ms`, `850ms`; `git diff --stat 6c49f5420 -- mathml/ ':!*_test.go'` empty with base tokenize.go in place | projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6377-math-comment-runs/1-9a2b61f5f/tests/solo-finder-color-run.go |  |  |
| 2001 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/mathml/mmlnode_test.go:214 | grep -n 'convertWithin(' and 'Repeat' in mathml/mmlnode_test.go: list every input the helper receives and its size | mmlnode_test.go:51 `deep := strings.Repeat(`\sqrt{`, MaxParseDepth+1) + ...` is the largest input convertWithin receives |  |  |  |
| 2002 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/mathml/edgecases_test.go:132 | go test -race -count=1 -run TestCommentRunIsLinear -v ./gno.land/pkg/gnoweb/markdown/mathml/ at 9a2b61f5f; grep the go test lines of .github/workflows/ci-race.yml | ci-race.yml:62 `go test -race -covermode=atomic -timeout 10m -run 'TestMathConcurrentRender' ./gno.land/pkg/gnoweb/markdown/` |  | cold |  |

Hit rate per tier, from the rows above: hot 1/1 confirmed over 1 files, warm 0/0 confirmed over 0 files, cold 0/1 confirmed over 2 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| user topic: output unchanged | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:319 | The cached blank run in Empty, or the moved guard in fixFences, changes MathML output or an error for some input | 150,000 seeded random expressions (fences, \big, comments, spaces, groups, \newcommand, \color, \kern, \sqrt[], \sideset, environments, \\) through ConvertInline and ConvertDisplay with head tokenize.go, then with `git show 6c49f5420:.../tokenize.go`: `cmp head.tsv base.tsv && echo IDENTICAL` printed IDENTICAL; 42,986 inline conversions succeeded, 0 panics |
| claims | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:278 | The cache is stale if Expr or a token's Kind is written after NewTokenBuffer | grep -nE '\]\.Kind *[\|&^]?=\|\.Kind *[\|&^]?= ' *.go returned only tokenize.go:466,666,668,695,702, all on local copies (tok, temp) before NewTokenBuffer; grep for `Expr =`/`Expr:` found only NewTokenBuffer at tokenize.go:310 |
| tests | gno.land/pkg/gnoweb/markdown/mathml/edgecases_test.go:130 | TestCommentRunIsLinear stays green with either fix reverted alone | Empty reverted to the base body: `conversion of "x %\n %\n..." did not terminate` / `--- FAIL: TestCommentRunIsLinear (2.02s)`; fixFences guard reverted: `conversion of "x%\n%\n..." did not terminate` / `--- FAIL: TestCommentRunIsLinear (2.00s)`; at head `--- PASS: TestCommentRunIsLinear (0.03s)`, a 60x margin under the 2 s bound |
| user topic: linear blank runs | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:369 | Another blank-skipping loop (GetNextToken, GetNextExpr, GetOptions, GetNextN, package GetNextExpr) still walks a comment run once per token | 18 shapes with a ` %\n` run of n = 32768/65536/131072 after \frac, \hat, \kern, \left/\right, \begin/\end, \sqrt[], \sideset, \newcommand, \\[], \bf, \color, ^, matrix cells, \text, \mathop: every doubling ratio between 1.5 and 2.5 except one noisy 3.6 step followed by 2.0 (e.g. `indent%: ... n=131072 32.445ms x1.9`, `cells: ... n=131072 128.524ms x1.8`) |
| lines | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:319 | The `b.idx < b.blankFrom` half of the cache test is unreachable | Reachable through the exported API by reading tokenize.go:433-454: GetNextN(2) sets idx=2 and jump=2, Empty caches blankFrom=2, Unget sets idx=0, and the next Empty must rescan |
| catalog | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:278 | New mutable fields on TokenBuffer race under concurrent renders | grep NewMathMLConverter in gnoweb/markdown: only ext_math.go:922 builds one converter per expression, and mathml.go:70 builds the TokenBuffer per conversion; no package-level TokenBuffer |

## Completeness

- Sibling of an edited test helper: `convertWithin` at `mmlnode_test.go:214` repeats the `%q` timeout message the diff truncates in `convertErrWithin`. It is filed as candidate 2001 and refuted, because no caller passes an input larger than `Repeat(`\sqrt{`, MaxParseDepth+1)`.
- Sibling of the edited blank-skip loop, swept by the shape `Kind&(tokComment|tokWhitespace)` across `gno.land/pkg/gnoweb/markdown/`. The hits are `tokenize.go:248,321,342,346,369,394,436,655` and `parse.go:181,275`. The finder's 18-shape growth run already covers the `TokenBuffer` loops, with every doubling ratio between 1.5 and 2.5. No new candidate.
- The cache in `Empty` stays valid when something moves `idx` directly, as `commands.go:438` and `:449` do. The cache is keyed on the `idx` range, and `Expr` is written only at `tokenize.go:310`. No candidate.
- The wall-clock bound of `TestCommentRunIsLinear` under `-race`: filed as candidate 2002 and refuted. The test passes in 0.26 s under `-race`, and `ci-race.yml:62` never runs the mathml package.
- Catalog: gnoweb rendering is off-chain, so determinism, gas, realm state, caller identity, coins, storage deposit, VM faults and type-check do not apply. Concurrency was cleared by the finder: one `TokenBuffer` per conversion. Twin input paths were cleared by the finder's 150,000-expression `ConvertInline`/`ConvertDisplay` equivalence run.
- Thin angle: the `tests` angle had no case with a comment between `\left`/`\big` and its delimiter. That is candidate 1, now CONFIRMED with five paste-ready cases.
- Not done: no issue draft for candidate 2's pre-existing `\color` quadratic. `MaxMathInputLen` caps it at about 1365 copies, so a reader loses nothing measurable. The parent decides whether it is worth an issue.

## Applied

| 3ab12d855 | 1 | the five fence-and-comment cases of `tests/judge-1-fence-comment.go` added to `TestCommentKeepsTheWindow`; they fail with comments dropped from the `fixFences` skip |

Row 2 is not applied: it predates this pull request and its fix is a design decision, held for the author.

## Retro

- What failed: nothing died, no cap was hit, every turn ran on claude-opus-5-5; the overview gate flagged a base sha in its before-and-after caption, fixed by the parent.
- What worked: the finder's two candidates both held, one Missing test with a paste-ready test that a mutation turns red, one pre-existing Suggestion measured on both trees; the reflector's two candidates were both refuted by a run.
- Hit rate per tier: hot 1 of 1 confirmed, cold 0 of 1, untiered 1 of 1 outside the diff.
- Cost: 2 agents, 40 turns, 74k output, 4.7M cache read, 0.34M cache write, about $5.17, 20 minutes.
- Upgrade: none applied; a solo round over a 33-line diff landed inside its projection.
