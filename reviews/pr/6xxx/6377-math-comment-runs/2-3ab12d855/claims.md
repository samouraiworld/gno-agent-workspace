# Claims: gnolang/gno#6377 perf(gnoweb): parse a run of math comments in linear time round 2, 3ab12d855, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:319 | Replace blankFrom/blankTo with a lazily set `last int` (plus a computed flag) as above, run `go test -count=1 ./gno.land/pkg/gnoweb/markdown/mathml/` and time ConvertDisplay on 65,536 indented `%` lines against the branch: expect green and the same time | rewrite: go test -count=1 ./gno.land/pkg/gnoweb/markdown/mathml/ ok 0.527s; TestJudgeTiming best of 3, head vs rewrite: `%` 16384/65536/262144 lines 1.81/9.96/30.0 ms vs 1.25/7.44/26.6 ms; ` %` 4.02/18.7/60.8 ms vs 2.45/16.9/50.1 ms, linear in both. Instrumented head with panic("JUDGE lower bound reached") in the idx < blankFrom case: suite ok 0.244s, no panic. | tests/judge-1-empty-last.patch | hot |  |
| 2 | REFUTED | Missing test | gno.land/pkg/gnoweb/markdown/mathml/edgecases_test.go:140 | Restore the base Empty alone, then the base fixFences branch alone, and run go test -count=1 -run TestCommentRunIsLinear -v ./gno.land/pkg/gnoweb/markdown/mathml/ under each: expect FAIL both times | edgecases_test.go:140 `mathBody(t, "x"+strings.Repeat(line, (1<<18)/len(line))+"y")` fails under convertErrWithin's 2 s bound with either half reverted |  | cold |  |
| 3 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/mathml/mmlnode_test.go:214 | List every convertWithin caller's input in mmlnode_test.go and its size against MaxParseDepth: expect one large enough to flood the log | mmlnode_test.go:359 builds the largest convertWithin input, 40 copies of \raisebox{1em}{ and their braces, about 640 bytes |  |  |  |

Hit rate per tier, from the rows above: hot 1/1 confirmed over 1 files, warm 0/0 confirmed over 0 files, cold 0/1 confirmed over 2 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| user topic: linear walks | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:342 | a sibling blank-skipping loop (GetNextToken, the GetNextExpr method, GetOptions, GetNextN, package GetNextExpr at 248) still walks a run once per step | grep -n 'GetNextExpr(\\|Empty()\\|Unget()' over the package: every call site (parse.go:97-100, 130-137, 163; commands.go:275-277, 342, 400-402, 427-449, 522-531, 551-574; commands_defs.go:158-189; tokenize.go:693, 700) either consumes past the run or restores and is followed by a consuming call; no loop calls one per position inside a run, and ParseTex's GetNextToken steps two tokens of an indented run per Empty |
| user topic: output unchanged | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:650 | the split of `if set == 0 \|\| j == len(toks)` in fixFences changes which delimiter a fence command takes | diff.md lines 270-284: the j loop has no side effect, set == 0 now appends before computing j and j == len(toks) appends after it, so every token takes the same branch as at the base |
| user topic: output unchanged | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:326 | the cached Empty answers or moves idx differently from the base scan | diff.md lines 189-211: base returned true with idx at the first non-blank at or after idx, which is len, and restored idx on false; head returns true with idx = blankTo = len and leaves idx on false; for idx in [blankFrom, blankTo) the tokens up to blankTo are blank and blankTo is non-blank or len, and grep '\.Expr = \\|\.Expr\[[^]]*\] *=' over the package returns no write |
| removed | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:319 | Empty's rescan when idx moves below blankFrom is pinned by no test | grep -n 'Unget()\\|\.idx = ' returns commands.go:348, 438, 449 and the methods' restores to their own start, none of which follows an Empty that cached past the new position; groups_test.go:108 `assert.True(t, b.Empty())` runs at idx == len, the early return. The branch is unreachable from every caller, so it is folded into candidate 1 rather than asked a test |
| reach | gno.land/pkg/gnoweb/markdown/mathml/edgecases_test.go:140 | TestCommentRunIsLinear's 2 s wall-clock bound could stay green on a fast host with one quadratic walk reverted | gh pr view 6377 body: '65,536 `%` lines, `ConvertDisplay` \| 1.57 s' and '65,536 indented `%` lines, `ConvertDisplay` \| 15.3 s'; the test runs 131,072 plain lines (4x the fixFences walk, about 6 s) and 65,536 indented lines, where Empty's walk is about 9 s of the 15.3 s, each 3x or more over the 2 s bound on the author's host, and CI runs `go test -covermode=set` (_ci-go.yml:131), slower still |
| catalog | gno.land/pkg/gnoweb/markdown/mathml/tokenize.go:278 | Empty now writes blankFrom/blankTo, mutable state that could race | grep -n 'NewTokenBuffer' shows mathml.go:70 per render and tokenize.go sub-buffers only, no package-level TokenBuffer; mathml.go:26-27 'it must not be shared across concurrent conversions' |

## Completeness

- Tests angle: the one added test, `TestCommentRunIsLinear`, turns red with the base `Empty` alone restored and again with the base `fixFences` branch alone restored, each through `convertErrWithin`'s 2 s bound (candidate 2, refuted).
- Sibling of the edited guard: `mmlnode_test.go:214` keeps `%q` in the timeout message that `groups_test.go:29` narrows to `%.80q`, but no `convertWithin` caller passes more than about 640 bytes (candidate 3, refuted).
- Sibling of the edited walk, swept by the shape `Kind&(tokComment|tokWhitespace)`: seven loops in `tokenize.go` (248, 321, 342 and 346, 369, 394, 436, 655); 321 and 655 are the two the diff edits, and the finder's row on 342 covers the rest. No new candidate.
- Output unchanged: the base `tokenize.go` under the head's tests fails `TestCommentRunIsLinear` alone; the five new fence-comment cases of `TestCommentKeepsTheWindow` pass at the base, so they pin existing behaviour.
- Catalog: its classes target the GnoVM, stdlibs and `.gno` code, and this diff is gnoweb Go. Of the two that could apply, global mutable state is the finder's catalog row (per-render buffers) and determinism gets nothing new: no map iteration, clock or randomness enters the diff.

## Applied

| d3aaa47b1 | 1 | `Empty` finds once per buffer the index past its last token that is not a space or a comment, replacing the cached blank run; base and branch agree on 200,000 random expressions, and restoring the base `Empty` turns `TestCommentRunIsLinear` red |
| a02f62326 | 1 | the simplify pass moves the `last` scan into `NewTokenBuffer`, the only place a buffer is built, and drops the lazy flag; output and the mutation result unchanged, and a fresh closing read found no Critical or Warning |

## Retro

- What failed: nothing died, no cap was hit, every turn ran on claude-opus-5-5.
- What worked: the finder's one candidate held and came with its patch; the reflector's two candidates were both refuted by a run. No Critical or Warning, so the loop closes on this round.
- Hit rate per tier: hot 1 of 1 confirmed, cold 0 of 2.
- Cost: 2 agents, 33 turns, 76k output, 3.5M cache read, 0.24M cache write, about $4.15, 21 minutes.
- Upgrade: none applied.
