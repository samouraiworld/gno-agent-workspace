# Findings in posting order, from round assemble: 6 to post, 0 SKIP, 0 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_details.go:97 [gh](https://github.com/gnolang/gno/blob/5ad81a10214a0379b6099159e98431176732d1f7/gno.land/pkg/gnoweb/markdown/ext_details.go#L97) · Warning
State: CONFIRMED, band: Warning, angle: removed
TL;DR: an inner opener with a misspelled or valued attribute never goes on the stack, so its close ends the outer block and the outer body renders unfolded
Check: tests/solo-finder-malformed-inner_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestDetailsMalformedInner
Details: parseDetailsLineTag returns detailsTagInvalid for a malformed opener and a malformed close alike (97), and Open's switch (143) has no case for it, so nothing is pushed. The following </gno-details> pairs with the outer opener; the outer's real close becomes a stray comment. The cap-refused opener stays on the stack for exactly this reason (148-150), and ext_frame counts attribute-refused openers in frameRefusedKey (ext_frame.go:262).
Evidence: rerun at 5ad81a102: /valid PASS, /frame_sibling PASS; /misspelled FAIL with `</div>\n</details>\n<p>outer body</p>\n<!-- unexpected/invalid details tag omitted -->`, /open_valued FAIL the same.
Artifact: tests/solo-finder-malformed-inner_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:136 [gh](https://github.com/gnolang/gno/blob/5ad81a10214a0379b6099159e98431176732d1f7/gno.land/pkg/gnoweb/markdown/ext_details.go#L136) · Warning
State: CONFIRMED, band: Warning, angle: siblings
TL;DR: a details block left unclosed in a column ends at the column marker but keeps its depth level to the end of the page, refusing a later alert
Check: tests/judge-reflector-column-depth_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestJudgeColumnDepth
Details: Three columns each opening an unclosed <gno-details> leave depth 3 after </gno-columns>; a later grid takes level 4 and the > [!NOTE] inside it is refused. With the three blocks closed the alert renders. detailsParser.Open returns nil on a non-details line (136-138) and never sees the grid marker, while frameParser.Open pops on one (endFrame, ext_frame.go:149-152). Realm-authored only: BlockRich escapes top-level gno-* tags.
Evidence: run at 5ad81a102: /closed PASS, /unclosed FAIL `later alert refused; depth at EOF 0`.
Artifact: tests/judge-reflector-column-depth_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:145 [gh](https://github.com/gnolang/gno/blob/5ad81a10214a0379b6099159e98431176732d1f7/gno.land/pkg/gnoweb/markdown/ext_details.go#L145) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: four `> <gno-details>` lines a BlockRich comment passes through refuse every gno-* block the realm renders after them
Check: tests/solo-finder-depth-leak_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestDetailsDepthLeak
Details: Open pushes a level per opener (ext_details.go:145) and only a close tag of the same parent pops it (152-160); when the blockquote or list item ends without one, the level stays until Transform's popDetails at EOF (201). Four such openers hold the cap at 4, so the realm's later <gno-columns> falls to raw HTML, its > [!WARNING] renders as a plain quote, and its own <gno-details summary="Admin"> is a comment with its body unfolded. TestDetailsDepthBalanced reads the depth only after EOF and stays green. Frame accepts openers at document level only, so it has no such path.
Evidence: rerun at 5ad81a102: TestDetailsDepthLeakRaw/quote_x4 and /list_items_x4 FAIL with `realm grid refused`, `realm alert refused`, `realm details refused`; /control and /quote_x1 PASS. TestDetailsDepthLeakBlockRich/quote_x4 and /list_x4 FAIL the same, /control PASS.
Artifact: tests/solo-finder-depth-leak_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:153 [gh](https://github.com/gnolang/gno/blob/5ad81a10214a0379b6099159e98431176732d1f7/gno.land/pkg/gnoweb/markdown/ext_details.go#L153) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: close tags whose parent matches no opener scan the whole stack, refused openers included, so a 960 KB page costs about 1 s per render
Check: tests/solo-finder-close-scan_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestDetailsCloseScan
Details: The stack keeps every refused opener (ext_details.go:150) and each close walks it from the top comparing Parent() (152-153). Openers in a blockquote and closes at document level never match, so n of each cost n*n compares. Growth is about x4 per doubling, so the curve, not the size, is the finding; 960 KB stays under maxMarkdownRenderBytes (1 MiB, render.go:29).
Evidence: rerun at 5ad81a102: cross_parent 11.6 ms, 58.6 ms, 216 ms, 960 ms at n=4000, 8000, 16000, 32000 (120 KB to 960 KB); same_parent 2.9 ms to 19.3 ms. `cross_parent 960.473481ms is 50x same_parent 19.335209ms at n=32000`, FAIL.
Artifact: tests/solo-finder-close-scan_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:135 [gh](https://github.com/gnolang/gno/blob/5ad81a10214a0379b6099159e98431176732d1f7/gno.land/pkg/gnoweb/markdown/ext_details.go#L135) · Missing test
State: CONFIRMED, band: Missing test, angle: lines
TL;DR: no golden feeds CRLF tag lines, though they render correctly today
Check: grep -rlP '\r' golden/ext_details/ golden/sanitize/*details*; tests/judge-5-crlf_test.go via go test -overlay <overlay mapping zz_*_test.go to tests/*> -count=1 -v ./gno.land/pkg/gnoweb/markdown -run TestJudgeCRLF
Details: The grep returns 0 files and ext_details_test.go has no CRLF case. The judge probe shows a CRLF page renders byte-identical to the LF one once \r is removed, so the behaviour holds and only the guard is missing.
Evidence: grep count 0; TestJudgeCRLF PASS at 5ad81a102.
Artifact: tests/judge-5-crlf_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:230 [gh](https://github.com/gnolang/gno/blob/5ad81a10214a0379b6099159e98431176732d1f7/gno.land/pkg/gnoweb/markdown/ext_details.go#L230) · Missing test
State: CONFIRMED, band: Missing test, angle: lines
TL;DR: deleting the stop at a gno-columns or gno-frame marker keeps the package suite green
Check: mutation: ext_details.go:230 condition prefixed with `false &&`, overlaid, go test -count=1 ./gno.land/pkg/gnoweb/markdown; then tests/judge-4-grid-stop_test.go under the same mutation
Details: No golden or unit test puts a details block beside a grid separator or a frame. The judge probe (details in a column before <gno-columns-sep />, and details before <gno-frame>) passes at head and goes red under the mutation. The ordering half of the candidate is not shown: moving the transformer to priority 500, ahead of the frame transformer's 501, keeps both the suite and the probe green, so no input here distinguishes the order.
Evidence: mutation A: suite `ok ... 5.725s`; probe --- FAIL in_column and before_frame. At head the probe PASSes. Mutation B (priority 500): suite `ok ... 5.582s`, probe PASS.
Artifact: tests/judge-4-grid-stop_test.go
