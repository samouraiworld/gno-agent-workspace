# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 1 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_frame.go:226 [gh](https://github.com/gnolang/gno/blob/11a5d796a07a92ece6344770ca86e6baac84ebbb/gno.land/pkg/gnoweb/markdown/ext_frame.go#L226) · Missing test
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: no golden pins the refused-card flag reset at a columns tag: deleting it leaves every golden green and lets a refused card left open eat the outer frame's close
Check: delete line 226, go test -run 'TestGnoExtension|TestFrame', render c1002_refused_unclosed
Details: With `pc.Set(frameRefusedKey, false)` removed, TestGnoExtension stays green; on a refused card with no close followed by <gno-columns-sep>, the outer </gno-frame> is eaten as the refused close and the frame runs to EOF around 'after'.
Evidence: del226 run: no TestGnoExtension failure; c1002_refused_unclosed ends '<!-- unexpected/invalid frame tag omitted -->\n<p>after</p>\n</section>'; at head it ends '</section>\n<p>after</p>'.
Artifact: tests/judge-1001-frame-tag-probe_test.go

## gno.land/pkg/gnoweb/markdown/ext_frame.go:254 [gh](https://github.com/gnolang/gno/blob/11a5d796a07a92ece6344770ca86e6baac84ebbb/gno.land/pkg/gnoweb/markdown/ext_frame.go#L254) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: a frame refused inside an open card still lets the card's own close end the outer frame mid-grid, so the grid renders outside an empty frame
Check: render c1001_frame_in_card and c1001_two_refused at head and base
Details: The refused case at 254 requires !frameInner(pc), so a frame opened inside a card is refused without being remembered; its close ends the card and the card's close ends the outer frame. The two-refused-openers half is refuted: each refused opener's close is consumed before the next opener, so the frame keeps its grid. Same output at base; it is the member of the refused-card class the fix commits left out.
Evidence: head c1001_frame_in_card: '<section class="gno-frame">\n</section>\n<div class="gno-columns">' then a trailing invalid comment; identical at base. head c1001_two_refused: grid inside the section, '</section>\n<p>after</p>'.
Artifact: tests/judge-1001-frame-tag-probe_test.go
State: CONFIRMED, band: Nit, angle: removed
TL;DR: a nested frame opener's close still ends the outer frame, as golden invalid_nested_frame pins
Check: render '<gno-frame>\n<gno-frame>\ninner\n</gno-frame>\nouter rest\n</gno-frame>\nafter\n' at head and base; general fix trial
Details: Same output at base, and golden invalid_nested_frame asserts it. Extending the refused rule to every refused opener while a frame is open, tested ahead of the inner close, closes 2 and 1001 together but reddens invalid_frame_in_card, invalid_nested_frame and invalid_nested_keeps_depth, so it is the author's decision. Folded into the 1001 section.
Evidence: head c2_nested: '</section>\n<p>outer rest</p>\n<!-- unexpected/invalid frame tag omitted -->'; base identical. General patch: '--- FAIL: TestGnoExtension/ext_frame/invalid_nested_frame.md' and two more.
Artifact: tests/judge-1001-frame-tag-probe_test.go
State: CONFIRMED, band: Nit, angle: claims
TL;DR: a self-closing <gno-frame/> in a framed grid column counts as a refused card opener, so the next </gno-frame> is eaten and an unclosed outer frame runs to the end of the page
Check: render c3_selfclosing and c3_selfclosing_closed at head and base
Details: line[1] != '/' is true for <gno-frame/>. With the outer close present the head is right (c3_selfclosing_closed keeps the grid in the frame). Only the page whose one </gno-frame> sits in the column changes: base ended the frame there with the grid outside, head wraps the rest of the page. The ADR groups the self-closing form with attributes as one invalid tag, so the head's reading is consistent; two readings, the author's call.
Evidence: head c3_selfclosing ends '<p>after</p>\n</section>'; base ends the section before the grid. c3_selfclosing_closed identical at both.
Artifact: tests/judge-1001-frame-tag-probe_test.go

## gno.land/pkg/gnoweb/markdown/ext_frame.go:292 [gh](https://github.com/gnolang/gno/blob/11a5d796a07a92ece6344770ca86e6baac84ebbb/gno.land/pkg/gnoweb/markdown/ext_frame.go#L292) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: an `<!--` comment in a frame with no grid open now ends at a `<gno-frame>` or invalid frame tag line, so the rest of the comment renders as page text
Check: go test ./pkg/gnoweb/markdown -run TestFrameCommentHidesOpener at head and at 96fcf8a2b via -overlay; probe cases c1_no_grid_opener and c1_commented_card; narrow (type 6/7) and grid-only fix trials
Details: Continue cuts on parseFrameLineTag(tag) != frameTagNone for every HTML block type. With no grid open the opener is an invalid leaf and changes no structure, so the cut only exposes the hidden text. The finder's narrow fix (opener cut only for HTMLBlockType6/7) is wrong: it brings back the base behaviour for a commented-out card in a framed grid column, where the card's close inside the comment ends the outer frame and pushes the grid out (c1_commented_card under the narrow patch: empty section, grid after it, trailing invalid comment). Cutting on an opener or invalid line only while the frame's grid is open (frameGrid(pc) && gridOpen(pc)) keeps the package green, passes the finder's test and leaves c1_commented_card as at head. Band Nit: the ADR documents the comment cut as the price of structure, a preview shows the leak, and nothing outside the page is reached.
Evidence: head: '--- FAIL: TestFrameCommentHidesOpener', c1_no_grid_opener renders '<p>secret\n--&gt;</p>' in the frame; base -overlay: '--- PASS: TestFrameCommentHidesOpener'. Grid-only patch: 'ok github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown 5.239s', finder test included.
Artifact: tests/solo-finder-1-comment-opener-leak_test.go
