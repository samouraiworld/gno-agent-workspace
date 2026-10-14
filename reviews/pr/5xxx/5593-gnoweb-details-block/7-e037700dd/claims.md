# Claims: PR 5593 https://github.com/gnolang/gno/pull/5593 round 7, e037700dd, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_details.go:139 | Copy tests/solo-finder-lazy-release_test.go to gno.land/pkg/gnoweb/markdown/zz_solo_finder_lazy_test.go and run go test -count=1 -v ./gno.land/pkg/gnoweb/markdown -run 'TestSoloFinderLazy$'; then tests/judge-1-lazy-curve_test.go and tests/judge-1-sanitize-lazy-nest.txtar under -update-golden-tests | bq_lazy (stray tags 1): `<!-- unexpected/invalid details tag omitted -->` then `<p>after</p>` then `</div>` `</details>`; bq_control (stray tags 0): `</details>` then `<p>after</p>`; list_lazy and list_lazy_autolink stray tags 1; TestJudgeLazyCurve: `n=10 lazy opened=10 control opened=4`, `n=100 lazy opened=100 control opened=4`, `n=1000 lazy opened=1000 control opened=4`; BlockRich golden zzjudge-lazy-nest: 9 `<details class="gno-details">`, `<h2>Footer</h2>` after the realm's `</details>` | tests/solo-finder-lazy-release_test.go | hot | lazy-continuation-release |
| 2 | CONFIRMED | Missing test | gno.land/pkg/gnoweb/markdown/ext_details_test.go:26 | go test -count=1 -v ./gno.land/pkg/gnoweb/markdown -run 'TestDetails\|TestJudgeLazyCurve' with tests/judge-1-lazy-curve_test.go copied in | --- PASS: TestDetailsNestBounded, --- PASS: TestDetailsReleasedWithContainer (list, quote); TestJudgeLazyCurve n=1000 lazy opened=1000 | tests/judge-1-lazy-curve_test.go | cold |  |
| 3 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/ext_details.go:228 | tests/judge-3-peers_test.go, cases math_list_interrupt and math_details_interrupt | ext_details.go:228 `func (*detailsParser) CanInterruptParagraph() bool { return true }`, the same answer goldmark's list parser gives | tests/judge-3-peers_test.go | hot |  |

Hit rate per tier, from the rows above: hot 1/2 confirmed over 2 files, warm 0/0 confirmed over 21 files, cold 1/1 confirmed over 5 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | examples/quarantined/gno.land/r/docs/markdown/markdown.gno:1041 | the sentence says a close tag inside a code block stays part of the code, but the example after it shows only nesting | sed -n 1041,1055p markdown.gno: the ±±±markdown fence right after the sentence holds two `</gno-details>` lines that render as code, so the sample itself shows the claim |
| removed | gno.land/pkg/gnoweb/frontend/css/06-blocks.css:3351 | b2fc6b550 drops font-weight and cursor: pointer from the details summary | grep -n summary frontend/css/*.css: 04-elements.css:101-102 `summary {` `cursor: pointer;` and 05-composition.css:401-403 `summary {` `font-weight: var(--g-font-bold);` `cursor: pointer;` cover both |
| reach | gno.land/pkg/gnoweb/markdown/ext.go:111 | the details parser is recorded into the math extension's block starters, and its Open could leave depth or stack state on the real context when math probes a line | ext_math.go:574-583 probeContext.Set is a no-op on a fresh parser.NewContext, so the probe's Push and pc.Set store nothing; TestSoloFinderLazyPeers prints depth 0 for math_holds_details_line, math_inside_details and math_close_after_details_close, and a `<gno-details>` line ending display math matches the documented block-start trade-off at ext_math.go:634-637 |
| reach | gno.land/pkg/gnoweb/markdown/ext_details.go:166 | the `<` and `>` Trigger at priority 498 runs before the button tag-line parser (899) and could change how `<gno-button />` parses | TestSoloFinderBtn: `<gno-button href="/r/x" label="L" />` renders `<p><a href="/r/x">L</a></p>` at document level, in a blockquote and inside a details block alike |
| catalog | gno.land/pkg/gnoweb/markdown/ext_details.go:74 | uppercase `</GNO-DETAILS>` in user content could pass the sanitizer while the parser accepts it case-insensitively | gnovm/stdlibs/chain/markdown/markdown.go:574 `return hasCaseInsensitivePrefix(rest, "gno-")` |

## Completeness

- Thin angle: the finder returned one candidate, on `lines`. The judge added the `tests` angle (row 2: no details test feeds a lazy continuation line, so `TestDetailsNestBounded` and `TestDetailsReleasedWithContainer` stay green over row 1) and the `reach` angle into the sanitizer: 8 BlockRich openers under the realm's own `<gno-details>` render 9 nested blocks past the cap of 4, while the realm's block and `## Footer` stay outside it (`tests/judge-1-sanitize-lazy-nest.txtar`).
- Upstream peers: `<gno-button />` renders `<p><a href="/r/x">L</a></p>` inside a details block and on its own (row 3), and a `$$` span split by a details opener splits as a list item splits it (row 4); `$$ … $$` inside a details block renders as display math.
- Merge hunks: `SANITIZE.md` counts match the corpus, 107 `block`, 57 `blockrich`, 263 fixtures by `ls golden/sanitize`; `git show e037700dd --cc` prints no combined hunk for `06-blocks.css`, and `markdown.gno` keeps the Buttons section before Collapsible blocks.
- Catalog: determinism, gas, realm state, caller, coin, storage, twin paths, stored data, VM classes do not touch a gnoweb markdown parser. Global mutable state: `detailsStackKey` is set once at package init and the stack lives in the per-parse `parser.Context`, so nothing shared mutates at runtime.
- Swept siblings by shape: the frame fixture `blockrich-gno-frame-ctx-lazy-quote.txtar` has no details twin, folded into row 2.

## Applied

| Commit | Closes | What changed |
| --- | --- | --- |
| fc76b5466 | 1, 2 | `Open` skips the release on a line that may still continue a paragraph lazily: after a paragraph, only a details tag line or a `>` line releases. `TestDetailsLazyLineKeepsBlock` fails with the paragraph check removed, and the `after_paragraph` case of `TestDetailsReleasedBeforeAlert` fails with the `>` exemption removed. The judge's curve now reads `n=10, 100, 1000 lazy opened=4 control opened=4` |

`round stop` read row 1 as on the fix of round 6, mechanism `lazy-continuation-release`, so no further round runs. The simplify pass moved the lazy check from a `releaseDetails` parameter to its one call site and kept one assertion per mutation. The altitude angle named a deeper form, a release hook inside `Push` called by every gno-* parser that takes a level, which touches alert, foreign, columns and frame, and is left out.

## Retro

- What failed: nothing died and no cap was hit; the finder's one candidate held.
- What worked: the solo shape found the one Warning in 15.5 min for about $5.70, against the $5 projected; the judge added the missing lazy-line test and checked the merge hunks.
- Hit rate per tier: hot 1/2 over 2 files, warm 0/0 over 21, cold 1/1 over 5. Cold beats hot on one row, too few to move a weight.
- Upgrade: none; one confirmed row measures no change to the run's shape.
