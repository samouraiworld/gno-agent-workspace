# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 0 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_details.go:193 [gh](https://github.com/gnolang/gno/blob/fa1ce31087d2ff53ae775bb71a1f52362543e45e/gno.land/pkg/gnoweb/markdown/ext_details.go#L193) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: a :::details opener on a tab-entered line drops the summary's first bytes, and at a page's end RenderRealm panics
Check: cp tests/solo-finder-tab-padding-render_test.go gno.land/pkg/gnoweb/solo_tab_padding_render_test.go && go test ./gno.land/pkg/gnoweb/ -run TestSoloRenderRealmTabDetails -v; same file via go test -overlay at the merge base; then tests/judge-1-padding-fix.patch and rerun with tests/solo-finder-tab-padding_test.go
Details: PeekLine returns the line with segment.Padding spaces prepended for a tab the container consumed, and Open counts them in pos and sStart, then adds segment.Start without subtracting Padding (ext_details.go:193-194), so the summary segment sits Padding bytes right of the text. '- item\n\t:::details Summary text\n\tbody' renders 'mmary text'; the same line ending a 64-byte page (cap == len, as handler_http.go:471 builds it with []byte(mdContent)) slices to [:66] and panics, and gnoweb has no recover outside tests, so that page fails for every viewer. goldmark's atx heading subtracts segment.Padding (parser/atx_heading.go:105-106). The merge base renders the same input as plain text. Fix is mechanical: subtract segment.Padding on both lines.
Evidence: Head: mid_page 'len=46 cap=48', '<summary><svg><use href="#ico-arrow"></use></svg>mmary text</summary>', 'summary text mangled'; page_end_64 'len=64 cap=64', 'RenderRealm panicked: runtime error: slice bounds out of range [:66] with capacity 64'. TestSoloDetailsTabPadding: quote_tab, alert_tab, list_tab_continuation FAIL 'mmary text', both controls PASS. Merge base 02606904e via -overlay: both subtests PASS, '<li>item'. With '- segment.Padding' on 193-194: TestSoloRenderRealmTabDetails PASS both, go test ./gno.land/pkg/gnoweb/markdown/ ok 5.925s (finder's tab test included).
Artifact: tests/solo-finder-tab-padding-render_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:226 [gh](https://github.com/gnolang/gno/blob/fa1ce31087d2ff53ae775bb71a1f52362543e45e/gno.land/pkg/gnoweb/markdown/ext_details.go#L226) · Missing test
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: nothing in the package fails when Close stops calling Pop
Check: Make Close a no-op at ext_details.go:226 and run go test ./gno.land/pkg/gnoweb/markdown/ without the finder's test, then with tests/solo-finder-tab-padding_test.go -run TestSoloDetailsDepthBalanced
Details: With Close reduced to {} the whole markdown package stays green, so the depth leak that would refuse every gno-* block after the fourth sequential details on a page is unpinned. The finder's TestSoloDetailsDepthBalanced turns all seven shapes red under the same mutation and passes at head, so it is the paste-ready test.
Evidence: Mutated: 'go test ./gno.land/pkg/gnoweb/markdown/' -> 'ok ... 6.103s'. Mutated with finder test: '--- FAIL: TestSoloDetailsDepthBalanced', subtests sequential_6, then_alerts, closed, unclosed, in_quote_lazy, in_list_ends, over_cap FAIL. Head: all 7 PASS.
Artifact: tests/solo-finder-tab-padding_test.go

## gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3316 [gh](https://github.com/gnolang/gno/blob/fa1ce31087d2ff53ae775bb71a1f52362543e45e/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3316) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: overflow: hidden on .gno-details cuts the external-link tooltip of a closed block's summary in half
Check: Render the valid_inline_summary shape inside .c-realm-view with public/main.css in chromium, force the .tooltip::after hover state (opacity, visibility, scale) and screenshot with and without .gno-details{overflow:visible}
Details: .tooltip is position: relative inside the summary and its ::after is position: absolute; top: 100%, so its containing block is inside the details box and overflow: hidden clips it at the card's bottom border when the block is closed. The hover state only toggles opacity, visibility and scale, so forcing those in a stylesheet stands in for the hover; the clip is layout. Removing overflow trades the rounded-corner clip it serves (round 2 row 9), so the fix is the author's decision.
Evidence: tests/judge-3-tooltip-clip.png: the 'External link' box is cut at the card's bottom edge, text half visible; tests/judge-3-tooltip-overflow-visible.png: same page with overflow: visible shows the whole box below the card.
Artifact: tests/judge-3-tooltip-clip.html
