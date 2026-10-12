# Claims: PR 5593 https://github.com/gnolang/gno/pull/5593 round 2, fbd2bc968, claude-opus-5-5, solo review

Rounds 2 to 5 ran numbered 1 to 4, before the round at 76ccef0 was found beside them; the JSON under `candidates/` and `verdicts/` and the comments under `tests/` keep that numbering.

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Critical | gno.land/pkg/gnoweb/markdown/ext_details.go:181 | cp projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-nest-oom_test.go gno.land/pkg/gnoweb/markdown/solo_nest_oom_test.go && go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloDetailsNestBounded -v | go1.25.9, judge worktree. Head: TestSoloDetailsNestBounded FAIL 'n=4000 bytes=52000 details=4000 alloc_MiB=989'; TestSoloProbeQuadratic n=1000/2000/4000 alloc_MiB=67/259/989, 17ms/130ms/458ms. Base 02606904e via -overlay: details_open=0 alloc_MiB=0/1/2. Head + judge-1-nest-pushpop.patch: 'details=4 alloc_MiB=4' PASS; n=10000 alloc_MiB=11 7ms; go test ./gno.land/pkg/gnoweb/markdown/ ok | projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-nest-oom_test.go | hot |
| 2 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:212 | cp projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-fence-close_test.go gno.land/pkg/gnoweb/markdown/solo_fence_close_test.go && go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloDetails -v | Head: TestSoloDetailsFenceKeepsColons FAIL, output '<pre><code></code></pre>...</details>\n<pre><code>after\n:::\n## Footer'. Head + judge-2-fence-exempt.patch: both TestSoloDetails* PASS, TestGnoExtension PASS, package ok. Fenced+HTML variant: TestSoloDetailsHTMLLineThenClose FAIL. Base: no parser triggers on ':::' (details_open=0). | projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-fence-close_test.go | hot |
| 3 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:92 | cp projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-sanitize-details-{fold,fold-rich,breakout}.txtar into gno.land/pkg/gnoweb/markdown/golden/sanitize/ and go test ./gno.land/pkg/gnoweb/markdown/ -run 'TestSanitizeIntegration$' | TestSanitizeIntegration with the three zz-solo-details-*.txtar fixtures: PASS at head, so head renders the recorded output.html: '<details class="gno-details">...nothing to see</summary><div><h2>Footer</h2><p><a href="/r/test$help&amp;func=Withdraw">'. At base ':::details' has no parser. | projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-sanitize-details-fold.txtar | hot |
| 4 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/markdown/golden/ext_details/valid_codefence_colons.md.txtar:1 | ls gno.land/pkg/gnoweb/markdown/golden/ext_details/; grep -l ':::' gno.land/pkg/gnoweb/markdown/golden/sanitize/*.txtar; ls gno.land/pkg/gnoweb/markdown/ext_details_test.go | ls golden/ext_details \| wc -l: 17; grep -l ':::' golden/sanitize/*.txtar \| wc -l: 0; ls ext_details_test.go: No such file or directory |  | warm |
| 5 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/markdown/ext_details.go:212 | go test -run TestSoloProbeShapes -v with projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-details-shapes_test.go | TestSoloProbeShapes equal_nest and longer_inner: both end '</details>\n</div>\n</details>\n<p>y\n:::</p>' | projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-details-shapes_test.go | hot |
| 6 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/markdown/ext_foreign.go:450 | sed -n 438,459p gno.land/pkg/gnoweb/markdown/ext_foreign.go; TestSoloProbeShapes shape foreign_details | TestSoloProbeShapes foreign_details: '<div class="gno-foreign__body">\n<p>:::details X\nbody\n:::</p>'; grep 'ExtAlerts.Extend\|ExtDetails.Extend' over *.go: ext_foreign.go:450 ExtAlerts only | projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/solo-finder-details-shapes_test.go |  |
| 7 | CONFIRMED | Nit | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3319 | render ':::details Closed **bold** summary' and measure the gap around <strong> | headless chromium on a page with the same rules and 'Closed <strong>bold</strong> summary': flex 'text_right=89.6 strong_left=97.6 gap_px=8.0 textContent_width=57.6' (trailing space gone); display:block 'gap_px=0.0 textContent_width=67.2' (space kept) |  | hot |
| 8 | CONFIRMED | Nit | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3325 | grep -n 'user-select' gno.land/pkg/gnoweb/frontend/css/06-blocks.css | grep -n user-select 06-blocks.css: 3325 and 3387 only |  | hot |
| 9 | REFUTED | Warning | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3316 | sed -n 470,510p gno.land/pkg/gnoweb/frontend/css/05-composition.css | 05-composition.css:498-500 display: block; width: 100%; max-width: 100%; and 05-composition.css:476 overflow-x: auto; |  | hot |
| 10 | REFUTED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:218 | go test -run TestJudgeFenceShapes -v with projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/judge-2-fence-shapes_test.go | cols_fence_unclosed output '<pre><code>left\n&lt;/gno-columns&gt;\n## Footer' with no details in the input | projects/gno-agent-workspace/checkout/reviews/pr/5xxx/5593-gnoweb-details-block/2-fbd2bc968/tests/judge-2-fence-shapes_test.go | hot |

Hit rate per tier, from the rows above: hot 6/8 confirmed over 2 files, warm 1/1 confirmed over 18 files, cold 0/0 confirmed over 3 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | gno.land/adr/pr5593_gnoweb_details_block.md:55 | ADR names the docs realm at examples/gno.land/r/docs/markdown/markdown.gno | ls examples/gno.land/r/docs/markdown -> 'No such file or directory'; the diff edits examples/quarantined/...; the only fix is prose, which is never a candidate |
| claims | gno.land/adr/pr5593_gnoweb_details_block.md:53 | ADR: details uses the same block-parser priority (799) as alerts | grep Prioritized ext_alert.go -> 448: util.Prioritized(NewAlertParser(), 799); ext_details.go:300 Prioritized(NewDetailsParser(), 799): holds |
| lines | gno.land/pkg/gnoweb/markdown/ext_details.go:99 | CRLF input could leave \r in the fence or summary | TestSoloProbeShapes crlf: ':::details A\r\nbody\r\n:::\r\nafter\r\n' renders <details>..<p>body</p></div></details><p>after</p> |
| reach | gno.land/pkg/gnoweb/public/main.css:1 | generated main.css stale after the master merge | grep -o counts: head gno-frame=20 gno-details=20, base gno-frame=20 gno-details=0 |
| reach | misc/gnoe2e/testdata/tour.txtar:1 | red main / test (TestScenarios/tour) caused by this diff | grep -n -i 'details\|:::\|docs/markdown\|collaps\|summary' misc/gnoe2e/testdata/tour.txtar returns nothing; the diff touches no misc/gnoe2e file; not attributed (no merge-base run found via gh run list --commit 02606904e) |

## Completeness

- Catalog: its classes are GnoVM, gas, realm state and banker invariants; the diff is gnoweb Go, CSS and one quarantined docs realm returning a Render string. Only global mutable state applies: defaultDetailsParser and detailsKeyword/detailsOpenFlag are package-level but never written after init, ext_details.go:31-34,86. No candidate needed.
- Sibling sweep by shape: every goldmark instance that registers ExtAlerts, grep 'ExtAlerts.Extend' over *.go minus tests: ext.go:85 and ext_foreign.go:450. ext.go:88 registers ExtDetails; ext_foreign.go has no ExtDetails, which is candidate 6. No third instance.
- Sibling sweep of the nest cap: the block parsers calling Push/Pop are alerts, columns, foreign and frames per nestdepth.go:1-24; details is the one block container added without it, candidate 1.
- Hot CSS file, overflow signal: .gno-details sets overflow: hidden at 06-blocks.css:3316; candidate 9 checks it clips nothing.
- Containment: an unclosed :::details inside <gno-columns> swallows </gno-columns> and the footer (finder shape columns_details_unclosed); candidate 10 checks whether the diff causes it.
- Candidate 2's fenced-only fix trades one shape: an unclosed fence inside a details runs to the end of the page instead of being cut by the closer, judge shape fence_unclosed_in_details.

## Applied

| Commit | Closes | What changed |
| --- | --- | --- |
| 0361bd557 | 1, 4 | `Open` calls `Push` and `Close` calls `Pop`; `TestDetailsNestBounded` and the `valid_nest_cap` golden fail with it reverted |
| ae18afe6f | 6 | `ExtDetails` registered in `buildInnerForeignMarkdown`; the `cross_foreign_details` golden fails with it reverted |
| 783fbe235 | 8 | `user-select: none` dropped from the summary, `public/main.css` rebuilt |
| 3f8ad27b9 | 4 | goldens for details in an alert, a blockquote, a list item, stray fences and `:::DETAILS` |
| fa1ce3108 | | the ADR names the quarantined docs realm path |

Rows 2, 3, 5 and 7 wait on the author: the code-fence exemption, the delimiter the sanitizer covers, which block an equal-length `:::` closes, and the summary's flex layout. Row 4's code-fence, HTML-block and sanitize fixtures follow those choices.
