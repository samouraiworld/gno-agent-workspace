# Claims: PR 5593 https://github.com/gnolang/gno/pull/5593 round 6, 5ad81a102, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:145 | tests/solo-finder-depth-leak_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestDetailsDepthLeak | rerun at 5ad81a102: TestDetailsDepthLeakRaw/quote_x4 and /list_items_x4 FAIL with `realm grid refused`, `realm alert refused`, `realm details refused`; /control and /quote_x1 PASS. TestDetailsDepthLeakBlockRich/quote_x4 and /list_x4 FAIL the same, /control PASS. | tests/solo-finder-depth-leak_test.go | hot |
| 2 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:153 | tests/solo-finder-close-scan_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestDetailsCloseScan | rerun at 5ad81a102: cross_parent 11.6 ms, 58.6 ms, 216 ms, 960 ms at n=4000, 8000, 16000, 32000 (120 KB to 960 KB); same_parent 2.9 ms to 19.3 ms. `cross_parent 960.473481ms is 50x same_parent 19.335209ms at n=32000`, FAIL. | tests/solo-finder-close-scan_test.go | hot |
| 3 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:97 | tests/solo-finder-malformed-inner_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestDetailsMalformedInner | rerun at 5ad81a102: /valid PASS, /frame_sibling PASS; /misspelled FAIL with `</div>\n</details>\n<p>outer body</p>\n<!-- unexpected/invalid details tag omitted -->`, /open_valued FAIL the same. | tests/solo-finder-malformed-inner_test.go | hot |
| 4 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/markdown/ext_details.go:230 | mutation: ext_details.go:230 condition prefixed with `false &&`, overlaid, go test -count=1 ./gno.land/pkg/gnoweb/markdown; then tests/judge-4-grid-stop_test.go under the same mutation | mutation A: suite `ok ... 5.725s`; probe --- FAIL in_column and before_frame. At head the probe PASSes. Mutation B (priority 500): suite `ok ... 5.582s`, probe PASS. | tests/judge-4-grid-stop_test.go | hot |
| 5 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/markdown/ext_details.go:135 | grep -rlP '\r' golden/ext_details/ golden/sanitize/*details*; tests/judge-5-crlf_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestJudgeCRLF | grep count 0; TestJudgeCRLF PASS at 5ad81a102. | tests/judge-5-crlf_test.go | hot |
| 6 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:136 | tests/judge-reflector-column-depth_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestJudgeColumnDepth | run at 5ad81a102: /closed PASS, /unclosed FAIL `later alert refused; depth at EOF 0`. | tests/judge-reflector-column-depth_test.go | hot |

Hit rate per tier, from the rows above: hot 6/6 confirmed over 2 files, warm 0/0 confirmed over 42 files, cold 0/0 confirmed over 5 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| removed | gno.land/pkg/gnoweb/markdown/ext_details.go:24 | Round 5 thread: a close line inside a fenced code block or HTML block closes the details | go test -count=1 ./gno.land/pkg/gnoweb/markdown at 5ad81a102: `ok ... 5.376s`, which runs goldens valid_codefence_close and valid_html_block_holds_close; markers are leaves, so a line inside a fence never reaches detailsParser.Open |
| reach | gno.land/pkg/gnoweb/markdown/ext_foreign.go:451 | Round 5 thread: extension missing from buildInnerForeignMarkdown, and any other extension enumeration missing it | grep -rn 'ExtAlerts\.Extend\\|ExtFrames\.Extend' --include='*.go' (non-test) returns only ext.go:85 and ext_foreign.go:450, and the diff adds ExtDetails.Extend(m) beside both |
| removed | gno.land/pkg/gnoweb/markdown/ext_details.go:274 | Round 5 threads: summary flex gap around inline children, and user-select: none on the summary | renderer writes `<summary><svg>…</svg><span>` and `</span></summary>` (ext_details.go:274,280), and the summary is escaped plain text; awk over the 06-blocks.css hunk shows no user-select among the added lines |
| removed | gno.land/pkg/gnoweb/markdown/ext_details.go:152 | Round 5 thread: equal-length fences close the outer block, unlike pandoc | close tags are unnamed and Open walks the stack from the top (ext_details.go:152), so the innermost same-parent opener pairs first; golden valid_nested passes in the green package run |
| reach | gno.land/pkg/gnoweb/markdown/ext_details.go:230 | User content inside a realm's details could end it early with a stray grid or frame marker | probe at 5ad81a102: BlockRich and Block both escape `</gno-columns>` and `</gno-frame>` to `\</gno-…>`, and a bare `\|\|\|` stays a paragraph; every case printed `tail-inside-details=true` |
| catalog | gno.land/pkg/gnoweb/markdown/ext_details.go:106 | Global mutable state: package-level vars in the new extension | ext_details.go read whole: detailsStackKey and the tag prefixes are set once at init, detailsParser, detailsASTTransformer, detailsRendererHTML and details are field-less structs, and the stack lives on the per-parse parser.Context |
| removed | gno.land/adr/pr5593_gnoweb_details_block.md:59 | Docs still naming the removed :::details syntax | grep -rln ':::details' over *.md, *.gno, *.go at head returns only the ADR, whose mentions sit in Alternatives Considered as the rejected design |

## Completeness

- Round 5 threads rechecked against the tag design: nest cap holds (Push at ext_details.go:145, TestDetailsNestBounded), its close-side quadratic is candidate 2; code fence and HTML block hold (markers are leaves, goldens valid_codefence_close and valid_html_block_holds_close); sanitizer holds for top-level tags (isExtDelimiter covers <gno-details>, five sanitize fixtures) and leaks depth through BlockRich quotes (candidate 1); foreign registration holds (ext_foreign.go:451); summary flex holds (<span> at ext_details.go:274,280); user-select is gone from 06-blocks.css 3311-3364; pandoc nesting holds (innermost same-parent opener pairs first, ext_details.go:152); missing goldens are candidates 4 and 5.
- Sibling of the edited guard: the refused opener kept on the stack (ext_details.go:148-150) is repeated for the cap-refused opener only; the malformed opener is candidate 3; ext_frame counts both (frameRefusedKey, ext_frame.go:251,262).
- Sibling by shape of the end-of-region Pop: frame pops at parse time on a grid tag (ext_frame.go:149-158); details pops only at EOF (ext_details.go:201), so two region ends leak: container end (candidate 1) and grid marker (candidate 6).
- Extension enumeration sweep: grep ExtColumns and ExtDetails over non-test .go returns ext.go:79/88 and ext_foreign.go:448/451 only, both carry details.
- Invariant catalog: the Go side holds no map iteration, no runtime-mutated package var (field-less structs, context key set at init), no swallowed error; the .gno change is docs text in a quarantined realm string. No class applies.
- Run note: the judge ran every artifact through `go test -overlay` against the head worktree rather than a scratch worktree, since the task's disk rule forbids a full tree copy (2528 MB free at start).

## Applied

| Commit | Closes | What changed |
| --- | --- | --- |
| a2b6e84c4 | 1, 2, 3, 5, 6 | `Open` runs on every `<` and `>` line and ends the openers whose container the line's parent no longer sits in; a document-level grid or frame tag ends the document's open blocks; a close pairs with the top of the stack alone; an opener with an invalid attribute stays on the stack. `TestDetailsReleasedWithContainer`, `TestDetailsReleasedBeforeAlert`, `TestDetailsEndsAtGrid`, `TestDetailsCRLF` and the `invalid_inner_attribute` golden each fail with their rule reverted |
| d207263b9 | 4 | golden `valid_stops_at_columns` |
| c9fd0ebc1 | | the simplify pass: the `refused` field reads off `tag`, and the grid check skips an empty stack |

Row 2's timing probe stays under `tests/`: the suite holds no timing assertion. Pairing on the top of the stack makes each close constant-time, and the probe's 960 KB cross-parent case read 68 ms against 98 ms for the same-parent one after the fix.
