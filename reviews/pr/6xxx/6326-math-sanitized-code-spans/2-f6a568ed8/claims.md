# Claims: gnolang/gno#6326 round 2, f6a568ed8, claude-opus-5-5, standard review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`ext_math.go:362`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/pkg/gnoweb/markdown/ext_math.go#L362) | 1001 | Warning | CONFIRMED | differs | [posted](https://github.com/gnolang/gno/pull/6326#discussion_r4443978295) |
| [`ext_math.go:542`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/pkg/gnoweb/markdown/ext_math.go#L542) | 1 | Warning | CONFIRMED | differs | [posted](https://github.com/gnolang/gno/pull/6326#discussion_r4443978345) |
| [`ext_math.go:542`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/pkg/gnoweb/markdown/ext_math.go#L542) | 3 | Nit | CONFIRMED | same | with row 1 |
| [`markdown.go:86`](https://github.com/gnolang/gno/blob/f6a568ed8/gnovm/stdlibs/chain/markdown/markdown.go#L86) | 3001 | Suggestion | CONFIRMED | same | SKIP |
| [`markdown.go:86`](https://github.com/gnolang/gno/blob/f6a568ed8/gnovm/stdlibs/chain/markdown/markdown.go#L86) | 2 | Suggestion | REFUTED |  |  |

### 1001 · Warning · CONFIRMED · `gno.land/pkg/gnoweb/markdown/ext_math.go:362`

A cache hit returns sc.open, the backtick run an earlier opener's search left open, so a later $ opener whose own search sees no open run is told it starts in a code span, and its formula on the next line stays text

- Check: go test ./gno.land/pkg/gnoweb/markdown -run TestZZCachedOpenRun -v at head and base, then again at head with findCloseCached bypassed (return find(b, ticks, nil) after its len check)
- Observed: head: 'alone: 2 math, withA: 1 math', --- FAIL on withA and raw HTML. head with the cache bypassed: 'alone: 2 math, withA: 2 math', --- PASS. base: 'alone: 2 math, withA: 1 math', the math being `a \\(`\\) $b c⏎d`.
- Test: [`tests/solo-finder-2-cached-open-run_test.go`](tests/solo-finder-2-cached-open-run_test.go)

### 1 · Warning · CONFIRMED · `gno.land/pkg/gnoweb/markdown/ext_math.go:542`

parenEnd counts the parentheses inside a quoted link title or a <...> destination, so a $$, \\( or \\[ closer after such a link is never found and a formula the merge base rendered stays text, or pairs with a later closer

- Check: cp tests/solo-finder-1-dest-hides-closer_test.go gno.land/pkg/gnoweb/markdown/zz_dest_hides_closer_test.go && go test ./gno.land/pkg/gnoweb/markdown -run TestZZDestHidesCloser -v at f6a568ed8 and at 1cae57ed7
- Observed: head f6a568ed8: math=0 for `$$ [a](x "(") b $$`, `\\( [a](x "(") b \\)`, `\\[ [a](x "(") b \\]`, `\\( [a](<x(>) b \\)`, `\\( [a](<x b \\)`, `$$ [a](x 'it(s') b $$`; base 1cae57ed7: math=1 for each. PairsLater at head: annotation ` [a](x &#34;(&#34;) b $$ then\nsee `; at base ` [a](x &#34;(&#34;) b `. Third shape at head: <math display="block"> outside <p>, annotation ending `some text\n`; at base inline math then `some text $$` as paragraph text.
- Test: [`tests/solo-finder-1-dest-hides-closer_test.go`](tests/solo-finder-1-dest-hides-closer_test.go)

### 3 · Nit · CONFIRMED · `gno.land/pkg/gnoweb/markdown/ext_math.go:542`

A quoted title holding ) ends parenEnd inside the title, so a $$ in that title closes the expression and the link is cut, at base and head alike

- Check: TestZZDestHidesCloserPairsLater first case in tests/solo-finder-1-dest-hides-closer_test.go at head and base
- Observed: head and base: math=1 links=0, annotation ` [a](x &#34;t) `; plain goldmark: <a href="x" title="t) $$ q">a</a>
- Test: [`tests/solo-finder-1-dest-hides-closer_test.go`](tests/solo-finder-1-dest-hides-closer_test.go)

### 3001 · Suggestion · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:86`

A treasury reason is escaped once when the proposal is created and stored, and the stored text is rendered raw, so a reason stored before the upgrade keeps a bare `$` that still opens math; no proposal on gnoland-1 holds one today

- Check: on gnoland-1 at height 1262951: vm/qfile gno.land/r/gov/dao/impl/v0/prop_requests.gno and render.gno, vm/qfile gno.land/p/nt/markdown/sanitize/v0/sanitize.gno, then vm/qrender gno.land/r/gov/dao:<n> for every proposal 0 to 15, counting `$`
- Observed: qfile: prop_requests.gno:248 `safeReason := sanitize.InlineText(...)`; sanitize.gno:349 `return markdown.EscapeInline(s)`; render.gno:124 `out += p.Description()`. qrender gno.land/r/gov/dao:0..15: dollars=1 on each, `/r/gov/dao$help&func=MustVoteOnProposalSimple`.

### 2 · Suggestion · REFUTED · `gnovm/stdlibs/chain/markdown/markdown.go:86`

The PR flags the native change as breaking: its title carries the ! marker and the ADR's Consequences state realms now get \$ where they got $

- Check: grep the PR title and the ADR for the native's state effect; read prop_requests.gno:248
- Refuted by: PR title `fix(gnoweb - gnovm)!:`; ADR line 76 '`chain/markdown` is a VM native: realms that call `sanitize.InlineText` or `TableCell` now get `\$` where they got `$`'

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`ext_math.go:517`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/pkg/gnoweb/markdown/ext_math.go#L517) | lines | single $ regresses on the same title/pointy shapes | read of base findDollarClose (diff.md lines 545-559): master already skipped a destination by counting parentheses to depth 0 with no title or <> handling, the same count parenEnd does, so $ behaves as on master |
| [`ext_math.go:310`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/pkg/gnoweb/markdown/ext_math.go#L310) | lines | tickRuns index built over the longest suffix misreads a run that a shorter suffix starts in the middle of | every searched b starts right after an opener ($, $$, \\(, \\[) whose last byte is not a backtick, or at a line start, so no suffix begins mid-run; all searches of one cache slot share the line end (stop), which last[] distances are measured from |
| [`markdown.go:86`](https://github.com/gnolang/gno/blob/f6a568ed8/gnovm/stdlibs/chain/markdown/markdown.go#L86) | catalog | global escape tables mutated at runtime race under parallel tests | markdown.go init(): inlineEscapeSet and titleEscapeSet are written only in init and read after; titleEscapeSet = inlineEscapeSet copies the [128]bool array by value, so clearing '$' on it leaves the inline set intact |
| [`ext_math.go:362`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/pkg/gnoweb/markdown/ext_math.go#L362) | reach | Cached answers diverge from a fresh search for later openers in general (tickRuns reuse, ticks key, end) | A render differential (TestZZRenderDiff, 300k random lines, cache forced off vs on) printed 'diffs: 2'. Both are "$[\\(]($`$(("-shaped: a link destination skipped to the line end, which is the ADR's documented known limit. A raw-level differential showed only bases inside a span the earlier search skipped, or bases preceded by a backslash or '$$', and none of those is an opener goldmark asks. |
| [`markdown.go:86`](https://github.com/gnolang/gno/blob/f6a568ed8/gnovm/stdlibs/chain/markdown/markdown.go#L86) | reach | The native escape set now holds `$`, so heading ids or TOC anchors built from InlineText headings could change | TestZZIDs with parser.WithAutoHeadingID and newLinearIDs: '## Price $5 off' and '## Price \\$5 off' both give id="price-5-off"; '## Pay $x$ now' and '## Pay \\$x\\$ now' both give id="pay-x-now". |
| [`prop_requests.gno:248`](https://github.com/gnolang/gno/blob/f6a568ed8/examples/gno.land/r/gov/dao/impl/v0/prop_requests.gno#L248) | removed | Text stored already escaped (the gov/dao treasury Reason) keeps a raw `$` from before the upgrade, and replaying with the new native would change it | chain/markdown was added in 72aff5a95 (2026-05-28). grep -rln 'chain/markdown\|nt/markdown/sanitize' over misc/deployments returned nothing, so no listed deployment carries the native. The rest is speculative. |
| [`markdown.gno:33`](https://github.com/gnolang/gno/blob/f6a568ed8/gnovm/stdlibs/chain/markdown/markdown.gno#L33) | removed | The godoc 'Set: \\ * _ [ ] ( ) ~ > - + . ! ` # < &' and the EscapeTitle line 'inline set plus \" and '' still describe the old set | The only fix is comment wording, and the ADR keeps it unchanged on purpose: a change to a stdlib .gno moves the app hash pinned by TestAppHashCrossrealm38. sed -n 33p prints the old set. |
| [`pr4879_gnoweb_math_extension.md:309`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/adr/pr4879_gnoweb_math_extension.md#L309) | removed | The earlier math ADR still says sanitize.InlineText and EscapeInline do not escape `$` | The ADR is the governing document the no-critique rule covers, it records a past decision, and the fix would be doc wording only (sed -n 309,313p shows 'do not escape `$` ... left to a follow-up PR'). |
| [`SANITIZE.md:211`](https://github.com/gnolang/gno/blob/f6a568ed8/gno.land/pkg/gnoweb/markdown/SANITIZE.md#L211) | removed | The InlineText 17, TableCell 4 and '260 fixtures' counts | TestSanitizeCoverageTableMatchesCorpus checks the Cases column against the corpus in CI, per SANITIZE.md's own text. |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_math.go:542 | cp tests/solo-finder-1-dest-hides-closer_test.go gno.land/pkg/gnoweb/markdown/zz_dest_hides_closer_test.go && go test ./gno.land/pkg/gnoweb/markdown -run TestZZDestHidesCloser -v at f6a568ed8 and at 1cae57ed7 | hot | link-title-paren-count |
| 3 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/ext_math.go:542 | TestZZDestHidesCloserPairsLater first case in tests/solo-finder-1-dest-hides-closer_test.go at head and base | hot |  |
| 1001 | CONFIRMED | Warning | gno.land/pkg/gnoweb/markdown/ext_math.go:362 | go test ./gno.land/pkg/gnoweb/markdown -run TestZZCachedOpenRun -v at head and base, then again at head with findCloseCached bypassed (return find(b, ticks, nil) after its len check) | hot | cache-carries-foreign-tick-run |
| 3001 | CONFIRMED | Suggestion | gnovm/stdlibs/chain/markdown/markdown.go:86 | on gnoland-1 at height 1262951: vm/qfile gno.land/r/gov/dao/impl/v0/prop_requests.gno and render.gno, vm/qfile gno.land/p/nt/markdown/sanitize/v0/sanitize.gno, then vm/qrender gno.land/r/gov/dao:<n> for every proposal 0 to 15, counting `$` | hot | stored-escape-predates-upgrade |
| 2 | REFUTED | Suggestion | gnovm/stdlibs/chain/markdown/markdown.go:86 | grep the PR title and the ADR for the native's state effect; read prop_requests.gno:248 | hot |  |

</details>

Hit rate per tier, from the record: hot 4/5 confirmed over 3 files, warm 0/0 confirmed over 4 files, cold 0/0 confirmed over 3 files.

## Completeness

- Sibling sweep of `TestMathUnclosedInlineOpenersAreLinear`, the table the diff widened to `\\(` and `\\[`: `$$` has no row by design, since `findDelimClose(_dollarDisplay)` closes at every later `$$` on the line, so a line of unclosed `$$` openers cannot be built.
- Catalog: Determinism cleared, `tickRuns.last` is only looked up, never iterated, and the native's escape sets are filled once in `init`. Gas cleared, `native_gas.go:179` is keyed on input length. Global mutable state cleared, `inlineEscapeSet` and `titleEscapeSet` are written in `init` only and the search cache lives in `parser.Context`. Data an older binary wrote: row 3001, confirmed by the parent on gnoland-1, SKIP since no stored proposal holds a `$`. VM semantics, caller identity, coin and realm state safety are not touched.
- Candidate 2 refuted: the title carries the `!` breaking marker and the ADR's Consequences state that realms calling `sanitize.InlineText` or `TableCell` get `\$` where they got `$`.
- Row 3 (Nit) shares line 542 with row 1 and went out inside its `<details>`, then on its own as a [reply](https://github.com/gnolang/gno/pull/6326#discussion_r4444021857) in that thread, since a reader of the collapsed block can miss it. One title-reading fix in `linkEnd` reaching both is the judge's reasoning, not a run.
- The three AI findings a maintainer posted at f6a568ed8 (next-line cache past the label stop, the 2^18 nested-label timing bound, the `LinkTitle` doc) were not candidates and were not re-run here.
- Not reached: a render differential with the cache on and off over inputs that put a backtick inside `\\(…\\)` or an HTML attribute; row 1001 was shown by construction only.

## Retro

- What failed: the draft said the branch introduces the `sc.open` defect and its repro block left out that the merge base fails the same test on another path; the parent's re-run at the base caught it and the block now says so. The writer named the preset `solo review` and left the Body empty, both set by the parent for the AI post. No agent died, refused or moved model; 3 agents, 68 turns, 123k output, 8.4M cache read, about $7.70 against the $7 projected, 19 minutes against 21.
- What worked: both Warnings reproduced on the parent's own run at the head; the bypass run isolated the cache as the cause of row 1001.
- Hit rate per tier: hot 4 confirmed of 5 rows over 3 hot files, row 3001 confirmed by the parent; warm 0 of 0 over 4; cold 0 of 0 over 3.
- Upgrade, applied: a repro whose check also fails at the merge base says so in its block, in *Repros* of `skills/review-comment.md`; the solo round's claims title names the preset rather than `solo review`; and `claims.md` opens on a Findings table whose Draft column names a row the draft dropped. Estimate: no turn added to any stage, the parent's step 6 read reads the Draft column instead of matching sections by hand.
