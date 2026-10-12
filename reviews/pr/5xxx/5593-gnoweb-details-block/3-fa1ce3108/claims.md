# Claims: PR 5593 https://github.com/gnolang/gno/pull/5593 round 3, fa1ce3108, claude-opus-5-5, solo review

Rounds 2 to 5 ran numbered 1 to 4, before the round at 76ccef0 was found beside them; the JSON under `candidates/` and `verdicts/` and the comments under `tests/` keep that numbering.

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:193 | cp tests/solo-finder-tab-padding-render_test.go gno.land/pkg/gnoweb/solo_tab_padding_render_test.go && go test ./gno.land/pkg/gnoweb/ -run TestSoloRenderRealmTabDetails -v; same file via go test -overlay at the merge base; then tests/judge-1-padding-fix.patch and rerun with tests/solo-finder-tab-padding_test.go | Head: mid_page 'len=46 cap=48', '<summary><svg><use href="#ico-arrow"></use></svg>mmary text</summary>', 'summary text mangled'; page_end_64 'len=64 cap=64', 'RenderRealm panicked: runtime error: slice bounds out of range [:66] with capacity 64'. TestSoloDetailsTabPadding: quote_tab, alert_tab, list_tab_continuation FAIL 'mmary text', both controls PASS. Merge base 02606904e via -overlay: both subtests PASS, '<li>item'. With '- segment.Padding' on 193-194: TestSoloRenderRealmTabDetails PASS both, go test ./gno.land/pkg/gnoweb/markdown/ ok 5.925s (finder's tab test included). | tests/solo-finder-tab-padding-render_test.go | hot |
| 2 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/markdown/ext_details.go:226 | Make Close a no-op at ext_details.go:226 and run go test ./gno.land/pkg/gnoweb/markdown/ without the finder's test, then with tests/solo-finder-tab-padding_test.go -run TestSoloDetailsDepthBalanced | Mutated: 'go test ./gno.land/pkg/gnoweb/markdown/' -> 'ok ... 6.103s'. Mutated with finder test: '--- FAIL: TestSoloDetailsDepthBalanced', subtests sequential_6, then_alerts, closed, unclosed, in_quote_lazy, in_list_ends, over_cap FAIL. Head: all 7 PASS. | tests/solo-finder-tab-padding_test.go | hot |
| 3 | CONFIRMED | Nit | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3316 | Render the valid_inline_summary shape inside .c-realm-view with public/main.css in chromium, force the .tooltip::after hover state (opacity, visibility, scale) and screenshot with and without .gno-details{overflow:visible} | tests/judge-3-tooltip-clip.png: the 'External link' box is cut at the card's bottom edge, text half visible; tests/judge-3-tooltip-overflow-visible.png: same page with overflow: visible shows the whole box below the card. | tests/judge-3-tooltip-clip.html | hot |

Hit rate per tier, from the rows above: hot 3/3 confirmed over 3 files, warm 0/0 confirmed over 25 files, cold 0/0 confirmed over 4 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| reach | gno.land/pkg/gnoweb/markdown/ext_details.go:186 | Push in Open could be left unmatched when a details block ends by a lazy blockquote end, a list end, EOF, or a refused opener at the cap | go test ./gno.land/pkg/gnoweb/markdown/ -run TestSoloDetailsDepthBalanced -v at head: closed, unclosed, over_cap, in_quote_lazy, in_list_ends, sequential_6, then_alerts all PASS with Get(pc)==0 and the alert after five closed details rendered |
| reach | gno.land/pkg/gnoweb/public/main.css:1 | main.css not rebuilt after user-select was dropped from the summary rule | grep -o '\.gno-details>summary{[^}]*}' main.css: '.gno-details>summary{align-items:center;cursor:pointer;display:flex;font-weight:var(--g-font-bold);gap:var(--g-space-2);list-style:none;padding:var(--g-space-3) var(--g-space-4)}', no user-select |
| catalog | gno.land/pkg/gnoweb/markdown/ext_foreign.go:451 | ExtDetails registered per foreign render could share mutable state across concurrent renders | ext_details.go: defaultDetailsParser is an empty struct and detailsKeyword/detailsOpenFlag/KindDetailsBlock/ExtDetails are never written after init; NewDetailsHTMLRenderer builds a fresh renderer per Extend. Other catalog classes (determinism, gas, realm state, caller, banker, deposit, VM) are untouched by a gnoweb renderer and a quarantined docs string |

## Completeness

- Catalog walked: Error & panic handling is candidate 1 (RenderRealm panics on a tab-entered :::details at page end); Global mutable state cleared, ExtDetails and defaultDetailsParser are pointers to struct{} with no fields (ext_details.go:84-86, 295-298); Determinism, Gas, Realm state, Caller, Coin, Storage deposit, VM fault, VM semantics, Type-check do not apply to a gnoweb renderer.
- Sibling sweep by shape for a block segment built from PeekLine indices plus segment.Start: grep 'segment.Start|seg.Start|.Padding' over gno.land/pkg/gnoweb/markdown/*.go finds only ext_details.go:193-194 among block parsers (ext_icons.go:197 and ext_emphasis.go:79 are inline), so no sibling repeats the unpadded shape.
- Sibling sweep of the depth-balance test shape: TestFrameNestDepthBalanced (ext_frame_test.go:58) is the only test asserting Get(pc)==0 after a parse; alert, columns, foreign and details carry none. The details gap is candidate 2; the other three predate the diff and are not in scope.
- Not read this round: the ADR and the quarantined docs realm (cold), whose claims angle no finder ran.
- Own pull request: no `comment_claude-opus-5-5.md`, per *Own PR*; the three CONFIRMED rows go to the change's plan.

## Applied

| Commit | Closes | What changed |
| --- | --- | --- |
| 8ab095f49 | 1, 2 | the summary offsets subtract `segment.Padding`; `TestDetailsTabPadding` fails with it reverted, `TestDetailsDepthBalanced` fails with `Close` a no-op |

Row 3, the tooltip clipped by `overflow: hidden`, waits on the author beside the summary layout of round 2's row 7.
