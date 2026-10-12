# Claims: PR 5593 https://github.com/gnolang/gno/pull/5593 round 5, c05644643, claude-opus-5-5, solo review

Rounds 2 to 5 ran numbered 1 to 4, before the round at 76ccef0 was found beside them; the JSON under `candidates/` and `verdicts/` and the comments under `tests/` keep that numbering.

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:207 | go test ./gno.land/pkg/gnoweb/markdown -run 'TestZZWalkCost\|TestZZJudgeWalkGrowth' -v, at c05644643 and with ext_details.go from 8ab095f49 | c05644643: 'bytes=30001 D=5000 N=25000 plain=12.06ms details=262.17ms ratio=21.7', 'bytes=60001 D=10000 plain=40.06ms details=1.467s ratio=36.6', 'bytes=120001 D=20000 plain=140.9ms details=6.245s ratio=44.3'; finder's TestZZWalkCost 'plain=317.5ms details=6.34s' FAIL. ext_details.go at 8ab095f49: ratios 1.1, 1.1, 1.0, PASS. With tests/judge-1-lastopened.patch: ratios 1.0, 1.1, 1.1, package goldens PASS. | tests/judge-1-walk-growth_test.go | hot |
| 2 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:208 | go test ./gno.land/pkg/gnoweb/markdown -run 'TestZZStaleForeign\|TestZZJudgeStaleForeign' -v, at c05644643 and with ext_details.go from 8ab095f49 | c05644643: quote_blank, quote_noblank, list_blank all end '<p>footer</p>\n</details>'; closed_control PASS. ext_details.go at 8ab095f49: all four PASS. LastOpenedBlock patch: quote_blank PASS, quote_noblank FAIL, list_blank FAIL. | tests/judge-2-stale-foreign_test.go | hot |
| 3 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/markdown/ext_details_test.go:1 | replace 'w <= 3 && isCloseFence' with 'true && w >= 0 && isCloseFence' at ext_details.go:212 in a scratch tree, go test ./gno.land/pkg/gnoweb/markdown | mutant: 'ok github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown 5.332s'. Head probe output: '<p>body</p>\n<pre><code>:::\n</code></pre>\n<p>after</p>'. |  | cold |
| 4 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/ext_details.go:165 | sed -n 960,992p goldmark@v1.8.2/parser/parser.go; plant panic("reached") in the w > 3 branch, go test ./gno.land/pkg/gnoweb/markdown plus seven indented-opener probes | planted panic: 'ok github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown 5.950s' with probes ' :::details A', tab, list-nested, blockquote-tab, space-tab, nested-indented and after-paragraph openers. |  | hot |
| 5 | REFUTED | Nit | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:1876 | git log -L1875,1876 on 06-blocks.css; grep -rn b-accordion gno.land/pkg/gnoweb | components/ui/toc.html:73 <details class="b-accordion"> |  | hot |
| 6 | REFUTED | Missing test | gno.land/pkg/gnoweb/markdown/ext_details.go:208 | replace 'ok && !fn.Closed' with 'ok && fn != nil' at ext_details.go:208, go test ./gno.land/pkg/gnoweb/markdown | golden/ext_details/valid_foreign_colons.md.txtar |  | hot |
| 7 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/ext_details.go:205 | read every Continue in gno.land/pkg/gnoweb/markdown/*.go | ext_columns.go:229 return parser.Close |  | hot |

Hit rate per tier, from the rows above: hot 3/6 confirmed over 3 files, warm 0/0 confirmed over 28 files, cold 1/1 confirmed over 4 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| reach | gno.land/pkg/gnoweb/markdown/ext.go:88 | another goldmark instance listing gno-* extensions one by one might miss ExtDetails | grep -rn 'ExtAlerts.Extend\|ExtColumns.Extend\|NewGnoExtension(' --include='*.go' (non-test) returns only ext.go:85, ext_foreign.go:450 and render_config.go:42, and both lists now call ExtDetails.Extend |
| lines | gno.land/pkg/gnoweb/public/main.css:1 | main.css might not carry the :not(.gno-icon) selectors from 7d570a0c7 | grep -o in public/main.css returns '.c-realm-view .gno-details>summary>svg:not(.gno-icon){...transform:rotate(-90deg)...}' and '.gno-details[open]>summary>svg:not(.gno-icon){transform:rotate(0)}' |
| reach | gno.land/pkg/gnoweb/markdown/ext_details.go:182 | summary offset 'segment.Start + pos + sStart - segment.Padding' could go below segment.Start if pos were smaller than the padding | util.IndentWidth consumes every leading space or tab, padding spaces included, and parseOpenFence matched only when line[pos] is ':', so pos >= Padding; TestDetailsTabPadding quote_tab, alert_tab and list_tab pass at c05644643 ('go test -run ^Test[^Z]' ok) |

## Completeness

- Catalog: `skills/invariant-catalog.md` covers GnoVM, stdlib and realm classes; fbd2bc968..c05644643 touches gnoweb markdown and CSS only, so no class applies.
- Siblings of the three round-4 shapes, swept by shape: the `.b-accordion` chevron selectors lacking `:not(.gno-icon)` belong to a template (row 5, refuted); the `!fn.Closed` half of the exemption is pinned by two goldens (row 6, refuted); no other container parser in the package closes on a line while holding children (row 7, refuted).
- Thin angle: candidates 1 and 2 share one loop, and the measured LastOpenedBlock fix (`tests/judge-1-lastopened.patch`) settles 1 and only the `quote_blank` shape of 2, so the two fixes are one change for the author.
- Not run: whether gnoweb bounds the markdown parse with a request timeout; the 1 MiB projection of candidate 1 is read off the three-point curve.

## Applied

| Commit | Closes | What changed |
| --- | --- | --- |
| 7335da7a6 | 3 | golden `valid_indented_fence_literal`: a `    :::` inside a details stays code; it fails with the `w <= 3` guard removed |
| 48f900abf | 4 | the opener drops its unreachable `w > 3` check |

## One mechanism, measured

Rows 1 and 2 sit on 87aa65612, round 4's fix, and with round 2's row 2 they are one mechanism: `Continue` on the details runs before its children, so the details decides who owns a `:::` line before the block that holds it. `tests/parent-design-harness_test.go` runs 18 shapes through five designs, `go test -run TestZZDesign -v .` with `tests/parent-design-harness.patch` applied at c05644643:

| Shape | Z, none | A, head | B, foreign last opened | C, fence, HTML or foreign last opened | E, closer parser |
| --- | --- | --- | --- | --- | --- |
| `:::` in a code fence | breaks | breaks | breaks | ok | ok |
| `:::` in an HTML block | breaks | breaks | breaks | ok | ok |
| `:::` in a `<gno-foreign>` | breaks | ok | ok | ok | ok |
| the same, in a list item | breaks | ok | ok | ok | ok |
| unclosed foreign in a quote, blank line | ok | breaks | ok | ok | ok |
| the same, no blank line | ok | breaks | breaks | breaks | ok |
| unclosed foreign in a list | ok | breaks | breaks | breaks | ok |
| equal-length nesting closes the innermost | no | no | no | no | yes |
| unclosed fence ends at the closer | yes | yes | yes | no | no |
| `  :::` after a blank line in a list item | closes | closes | closes | closes | list text, as CommonMark reads it |
| 8 controls: longer outer fence, paragraph, list, quote, alert, list item, tab closer, 4-column `:::` | ok | ok | ok | ok | ok |
| parse 120 KB, against the plain parse | 1.0x | 1.2x | 1.5x | 1.3x | 1.0x |

E is a second block parser at priority 798 that opens only when its direct parent is a `DetailsBlock` and the line is that block's closer: it flags the details and the next `Continue` returns `Close`, so the line reaches the innermost container first, the way every other line does. Every existing golden and test passes under it. It needs the author's word: it changes which `:::` closes a nested block, round 2's row 5, and inherits the unclosed-fence trade of round 2's row 2.

## Retro

Four solo rounds, two agents each, measured by `./scripts/review-retro.py`: $6.01, $4.86, $5.34 and $4.85, $21.06 against a $31 projection; no agent died, moved model or hit its cap.

- What failed: the loop took four rounds to name one mechanism. Round 2's row 2, a `:::` inside a code fence, already showed `Continue` deciding before the children; it waited on the author, so round 4's foreign Warning was patched on its own shape and round 5 found that patch quadratic and stale in two containers.
- What worked: every Critical and Warning came with a run and a patch the parent applied unchanged or measured; the reflector's candidates were all refuted, each with its proving line.
- Hit rate per tier: every confirmed row sat in the hot tier, `ext_details.go` and `06-blocks.css`; the warm goldens carried none.
- Upgrade: when `round stop` prints `outside` for a Warning whose mechanism matches a row left waiting on the author, read it as `on the fix` and run the one-mechanism measurement then. Estimate: one round of two agents saved here, about $5 and 15 minutes, finding rate unchanged since the measured design still goes to the author. Not applied: `round stop` cannot match a mechanism today, since rows carry none; recorded in the workspace `TODO.md`.
