# Findings in posting order, from round assemble: 7 to post, 0 SKIP, 2 refuted kept out

## Body

- Suggestion: gno.land/pkg/gnoweb/markdown/ext_foreign.go:450 [gh](https://github.com/gnolang/gno/blob/fbd2bc9687ddebeef9f09c16032068fcca159696/gno.land/pkg/gnoweb/markdown/ext_foreign.go#L450), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: reach
  TL;DR: buildInnerForeignMarkdown registers alerts and columns but not ExtDetails, so :::details inside <gno-foreign> renders as a literal paragraph
  Check: sed -n 438,459p gno.land/pkg/gnoweb/markdown/ext_foreign.go; TestSoloProbeShapes shape foreign_details
  Details: ext_foreign.go:447-457 extends the inner instance with Foreign, Columns, Frames, Alerts and others; ExtDetails appears only at ext.go:88. '<gno-foreign>\n:::details X\nbody\n:::\n</gno-foreign>' renders '<p>:::details X\nbody\n:::</p>' in the foreign body. Registering it is safe only once candidate 1's Push/Pop lands, since Seed carries the depth into the inner context. The line is outside the diff, so it goes in the Body. Mechanical after candidate 1.
  Evidence: TestSoloProbeShapes foreign_details: '<div class="gno-foreign__body">\n<p>:::details X\nbody\n:::</p>'; grep 'ExtAlerts.Extend|ExtDetails.Extend' over *.go: ext_foreign.go:450 ExtAlerts only
  Artifact: projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-details-shapes_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:181 [gh](https://github.com/gnolang/gno/blob/fbd2bc9687ddebeef9f09c16032068fcca159696/gno.land/pkg/gnoweb/markdown/ext_details.go#L181) · Critical
State: CONFIRMED, band: Critical, angle: reach
TL;DR: each :::details opener nests without the shared 4-level cap, so a ~130 KB page makes one render allocate gigabytes
Check: cp projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-nest-oom_test.go gno.land/pkg/gnoweb/markdown/solo_nest_oom_test.go && go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloDetailsNestBounded -v
Details: Open (ext_details.go:181-206) never calls Push and Close (222) never calls Pop, so strings.Repeat(":::details a\n", n) opens n nested blocks. Allocation grows about 4x per doubling of n: 67, 259, 989 MiB at n=1000, 2000, 4000, which reaches 6 GiB near n=10000 (130 KB), well under maxMarkdownRenderBytes = 1 << 20 (render.go:29), and repeats on every visit. At the merge base the same input is a paragraph. Adding `if !Push(pc) { return nil, parser.NoChildren }` after parseOpenFence and Pop(pc) in Close bounds it at 4 levels and keeps every golden green. Mechanical.
Evidence: go1.25.9, judge worktree. Head: TestSoloDetailsNestBounded FAIL 'n=4000 bytes=52000 details=4000 alloc_MiB=989'; TestSoloProbeQuadratic n=1000/2000/4000 alloc_MiB=67/259/989, 17ms/130ms/458ms. Base 02606904e via -overlay: details_open=0 alloc_MiB=0/1/2. Head + judge-1-nest-pushpop.patch: 'details=4 alloc_MiB=4' PASS; n=10000 alloc_MiB=11 7ms; go test ./gno.land/pkg/gnoweb/markdown/ ok
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-nest-oom_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:92 [gh](https://github.com/gnolang/gno/blob/fbd2bc9687ddebeef9f09c16032068fcca159696/gno.land/pkg/gnoweb/markdown/ext_details.go#L92) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: sanitize.Block and BlockRich pass :::details and ::: through, so a user's text folds the realm's own content or breaks out of its details
Check: cp projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-sanitize-details-{fold,fold-rich,breakout}.txtar into gno.land/pkg/gnoweb/markdown/golden/sanitize/ and go test ./gno.land/pkg/gnoweb/markdown/ -run 'TestSanitizeIntegration$'
Details: The parser triggers on ':' (line 92) while chain/markdown isExtDelimiter (gnovm/stdlibs/chain/markdown/markdown.go:548-550) returns false for any line not starting with '<'. A comment 'hi\n\n:::details nothing to see' run through sanitize.Block before a realm's '## Footer' and a Withdraw tx link renders both inside a closed <details> labelled 'nothing to see'; BlockRich does the same; 'hello\n:::\n...' inside the realm's own ':::details Comments\n%s\n:::' closes it early and leaves a stray ':::'. The visitor loses sight of the realm's content with no sign it was hidden. The fix is a choice: a <gno-details> tag the existing check covers, or a native-stdlib change to isExtDelimiter with sanitize fixtures. Decision.
Evidence: TestSanitizeIntegration with the three zz-solo-details-*.txtar fixtures: PASS at head, so head renders the recorded output.html: '<details class="gno-details">...nothing to see</summary><div><h2>Footer</h2><p><a href="/r/test$help&amp;func=Withdraw">'. At base ':::details' has no parser.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-sanitize-details-fold.txtar

## gno.land/pkg/gnoweb/markdown/ext_details.go:212 [gh](https://github.com/gnolang/gno/blob/fbd2bc9687ddebeef9f09c16032068fcca159696/gno.land/pkg/gnoweb/markdown/ext_details.go#L212) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: a ::: line inside a fenced code block closes the details, leaving the fence open so the rest of the page renders as code
Check: cp projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-fence-close_test.go gno.land/pkg/gnoweb/markdown/solo_fence_close_test.go && go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloDetails -v
Details: Continue runs on the details before its open children, and isCloseFence(line, Fence) at line 212 matches a code line made of colons. ':::details A\n```\n:::\n```\nafter\n:::\n\n## Footer\n' renders an empty <pre> in the details, then 'after', the closer and '## Footer' as one <pre><code> outside it. The same happens with the fence inside a list item in the details (judge shape fence_in_list). Returning Continue|HasChildren when pc.LastOpenedBlock() is a KindFencedCodeBlock fixes both and keeps every golden; adding KindHTMLBlock as well breaks ':::details A\n<div>x</div>\n:::' (the HTML block swallows the closer and the footer). The fenced-only form has one trade-off the author decides: an unclosed fence in the details now runs to the end of the page instead of being cut by the closer (judge shape fence_unclosed_in_details). Decision.
Evidence: Head: TestSoloDetailsFenceKeepsColons FAIL, output '<pre><code></code></pre>...</details>\n<pre><code>after\n:::\n## Footer'. Head + judge-2-fence-exempt.patch: both TestSoloDetails* PASS, TestGnoExtension PASS, package ok. Fenced+HTML variant: TestSoloDetailsHTMLLineThenClose FAIL. Base: no parser triggers on ':::' (details_open=0).
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-fence-close_test.go
State: CONFIRMED, band: Suggestion, angle: claims
TL;DR: the first fence long enough closes the outermost open details, so an inner block closes its parent unless colon counts strictly decrease inward
Check: go test -run TestSoloProbeShapes -v with projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-details-shapes_test.go
Details: Line 212 compares the line against the outer block's Fence before any inner block sees it. ':::details A\n:::details B\nx\n:::\ny\n:::' closes A and B on the first ':::' and renders 'y' and the last ':::' as a paragraph outside; ':::details O\n::::details I\nx\n::::\ny\n:::' does the same on '::::'. The ADR (gno.land/adr/pr5593_gnoweb_details_block.md:88-89) calls this the pandoc convention, which closes the innermost div. Letting the innermost open details own a fence line, at the same Continue site as candidate 2, gives pandoc nesting. Decision.
Evidence: TestSoloProbeShapes equal_nest and longer_inner: both end '</details>\n</div>\n</details>\n<p>y\n:::</p>'
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-details-shapes_test.go

## gno.land/pkg/gnoweb/markdown/golden/ext_details/valid_codefence_colons.md.txtar:1 [gh](https://github.com/gnolang/gno/blob/fbd2bc9687ddebeef9f09c16032068fcca159696/gno.land/pkg/gnoweb/markdown/golden/ext_details/valid_codefence_colons.md.txtar#L1) · Missing test
State: CONFIRMED, band: Missing test, angle: tests
TL;DR: no golden or test pins the nest cap, a fence holding :::, sanitize behaviour or details embedded in other blocks
Check: ls gno.land/pkg/gnoweb/markdown/golden/ext_details/; grep -l ':::' gno.land/pkg/gnoweb/markdown/golden/sanitize/*.txtar; ls gno.land/pkg/gnoweb/markdown/ext_details_test.go
Details: golden/ext_details holds 17 files, none with the shapes candidates 1, 2, 3 and 5 turn on; no golden/sanitize fixture contains ':::'; there is no ext_details_test.go. Cases to add: a fence holding ':::' under a 3-colon block, ':::details A\n<div>x</div>\n:::' (guards against exempting HTML blocks), equal-length nesting, a 5th level falling through, details inside an alert, a list item and <gno-foreign>, sanitize fixtures for the opener and the closer, an allocation guard like solo-finder-nest-oom_test.go.
Evidence: ls golden/ext_details | wc -l: 17; grep -l ':::' golden/sanitize/*.txtar | wc -l: 0; ls ext_details_test.go: No such file or directory

## gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3319 [gh](https://github.com/gnolang/gno/blob/fbd2bc9687ddebeef9f09c16032068fcca159696/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3319) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: display: flex with gap on the summary turns each inline run into a flex item, so the space around **bold** or a link becomes an 8px gap
Check: render ':::details Closed **bold** summary' and measure the gap around <strong>
Details: 06-blocks.css:3319-3321 sets display: flex and gap: var(--g-space-2) on .gno-details > summary. In a flex container the text 'Closed ' becomes an anonymous item whose trailing space collapses, and the gap replaces it. .gno-alert > summary (3211-3214) has the same shape and predates the diff. Wrapping the summary's inline content in a <span> (detailsSummaryOpen, detailsSummaryClose, detailsSummaryDefault at ext_details.go:236-238) keeps the chevron layout; either that or dropping flex is the author's choice. Decision.
Evidence: headless chromium on a page with the same rules and 'Closed <strong>bold</strong> summary': flex 'text_right=89.6 strong_left=97.6 gap_px=8.0 textContent_width=57.6' (trailing space gone); display:block 'gap_px=0.0 textContent_width=67.2' (space kept)

## gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3325 [gh](https://github.com/gnolang/gno/blob/fbd2bc9687ddebeef9f09c16032068fcca159696/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3325) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: user-select: none on the summary stops visitors from selecting or copying summary text
Check: grep -n 'user-select' gno.land/pkg/gnoweb/frontend/css/06-blocks.css
Details: 06-blocks.css:3325 sets user-select: none inside .gno-details > summary; a summary holding an address, a version or a log name cannot be copied. .gno-alert > summary does not set it; the only other use is 3387. Deleting the line is mechanical.
Evidence: grep -n user-select 06-blocks.css: 3325 and 3387 only
