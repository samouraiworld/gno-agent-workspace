# Claims: PR 5593 https://github.com/gnolang/gno/pull/5593 round 4, 8ab095f49, claude-opus-5-5, solo review

Rounds 2 to 5 ran numbered 1 to 4, before the round at 76ccef0 was found beside them; the JSON under `candidates/` and `verdicts/` and the comments under `tests/` keep that numbering.

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:216 | cp tests/solo-finder-foreign-escape_test.go gno.land/pkg/gnoweb/markdown/zz_foreign_escape_test.go && go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloForeignEscape -v at 8ab095f49 and via go test -overlay at 02606904e; then tests/judge-1-foreign-in-list_test.go beside it, and tests/judge-1-foreign-exempt.patch applied with the package suite | Head: '--- FAIL: TestSoloForeignEscape', 'details_foreign: foreign bytes rendered a gno-form', output '<details class="gno-details">…</details>' then '<form class="gno-form" method="post" action="/r/test"'; TestJudgeForeignInList FAIL 'list-nested foreign escaped'. Merge base via -overlay: '--- PASS: TestSoloForeignEscape', '<p>:::details Comments</p>' then '<div class="gno-foreign" role="group">'. With judge-1-foreign-exempt.patch: both tests PASS, 'ok github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown 5.426s'. | tests/solo-finder-foreign-escape_test.go | hot |
| 2 | CONFIRMED | Warning | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3332 | cp gno.land/pkg/gnoweb/public/main.css beside tests/solo-finder-icon-rotated.html; chromium --headless --disable-gpu --no-sandbox --dump-dom file://…/solo-finder-icon-rotated.html \| grep -A3 '<pre id="out">'; then tests/judge-2-not-gno-icon.sed on the copy and rerun | Head main.css: 'closed gno-icon transform=matrix(0, -1, 1, 0, 0, 0) width=16px color=rgb(88, 88, 88)', 'open gno-icon transform=matrix(1, 0, 0, 1, 0, 0) width=16px', 'alert gno-icon transform=none width=18.3906px color=rgb(9, 9, 9)'. With the :not(.gno-icon) selectors: 'closed gno-icon transform=none width=18.3906px', 'open gno-icon transform=none width=18.3906px'. The merge base has no gno-details rule, the feature being new. | tests/solo-finder-icon-rotated.html | hot |
| 3 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/ext_details.go:153 | cp tests/solo-finder-tab-indent_test.go gno.land/pkg/gnoweb/markdown/zz_tab_indent_test.go && go test ./gno.land/pkg/gnoweb/markdown/ -run 'TestProbe(TabMatrix\|TabCloser)' -v | 'cases=2368 panics=0 mangled=0 diverge=92', e.g. 'DIVERGE fenced=true details=false "> \t:::details Summary"'; TestProbeTabCloser: '> \t:::' gives '<p>body\n:::\nafter</p>' inside the details, '>\t:::' gives '<p>body</p>' then '<p>after</p>' after it. | tests/solo-finder-tab-indent_test.go | hot |

Hit rate per tier, from the rows above: hot 3/3 confirmed over 3 files, warm 0/0 confirmed over 25 files, cold 0/0 confirmed over 4 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | gno.land/pkg/gnoweb/markdown/ext_details.go:193 | the segment.Padding subtraction in the summary offsets could mis-slice or panic on another container and indent shape than the three TestDetailsTabPadding covers | go test -run TestProbeTabMatrix over 2368 shapes (12x12 container prefixes x 8 indents plus list and quote continuations, with and without a trailing newline, source cap == len): 'cases=2368 panics=0 mangled=0 diverge=92' |
| catalog | gno.land/pkg/gnoweb/markdown/ext_details.go:32 | package-level detailsKeyword and detailsOpenFlag byte slices shared by concurrent renders (global mutable state class) | read of diff.md 1926-2013: the only uses are bytes.HasPrefix(trimmed[n:], detailsKeyword), bytes.HasPrefix(rest, detailsOpenFlag) and len(...), none writes; the parser keeps its fence on the DetailsBlock node and its depth on the per-parse context |

## Completeness

- Round-3 fix pinned: dropping ' - segment.Padding' from ext_details.go:193-194 turns TestDetailsTabPadding/list_tab red ('mmary text' where '</svg>Summary text</summary>' is expected), so the test guards the fix.
- Sibling sweep by shape for indentation measured by counting spaces: grep 'pos < 4|IndentWidth|segment.Padding|segment.Start +' over gno.land/pkg/gnoweb/markdown/*.go finds the two space-only loops at ext_details.go:153 and :173, both candidate 3, while ext_alert.go:104, 136 and 337 measure with util.IndentWidth(line, reader.LineOffset()); ext_icons.go:197 and ext_emphasis.go:79 are inline and not the shape.
- Trust-boundary sweep: <gno-foreign> is the only block whose body another party writes; candidate 1 covers it, and the judge extended it to a foreign block nested in a list inside the details (tests/judge-1-foreign-in-list_test.go), which escapes the same way, so the exemption has to walk the last-descendant chain rather than the direct child.
- Catalog walked: Error & panic handling cleared for the opener, the finder's 2368-shape matrix reporting panics=0 at the head; Global mutable state cleared as in round 3 (ExtDetails is a pointer to an empty struct); Determinism, Gas, Realm state, Caller, Coin, Storage deposit, VM fault, VM semantics and Type-check do not apply to a gnoweb renderer.
- Not read this round: the ADR and the quarantined docs realm (cold).

## Applied

| Commit | Closes | What changed |
| --- | --- | --- |
| 87aa65612 | 1 | `Continue` keeps the block open while an unclosed `ForeignNode` is its last descendant; the `valid_foreign_colons` goldens, top level and in a list item, fail with it reverted |
| 7d570a0c7 | 2 | the chevron selectors take `:not(.gno-icon)`, `main.css` rebuilt; no Go test reaches CSS, so the repro stays `tests/solo-finder-icon-rotated.html` |
| c05644643 | 3 | `Open` and `Continue` measure indentation with `util.IndentWidth`, as the alert parser does; the `valid_tab_close` golden fails with it reverted |

The probe matrix in `tests/solo-finder-tab-indent_test.go` reads `diverge=102` after c05644643, every case now the details opener accepting an indent goldmark's fence opener refuses, against 92 the other way before: goldmark measures its fence opener with `BlockOffset` and its closer with `IndentWidth`, and this block now measures both like its closer and like the alert parser.
