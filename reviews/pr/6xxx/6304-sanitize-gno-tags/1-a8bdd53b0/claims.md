# Claims: gnolang/gno#6304 round 1, a8bdd53b0, claude-opus-5-5, standard review

Round shape: 6 finders, one reflector, 12 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted.

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`markdown.go:565`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L565) | 8 | Warning | CONFIRMED | same | posts |
| [`markdown.go:882`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L882) | 11 | Warning | CONFIRMED | differs | posts |
| [`markdown.go:917`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L917) | 9 | Warning | CONFIRMED | differs | posts |
| [`markdown.go:1057`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1057) | 6 | Warning | CONFIRMED | differs | posts |
| [`markdown.go:1123`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1123) | 1 | Warning | CONFIRMED | same | posts |
| [`markdown.go:1316`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1316) | 7 | Warning | CONFIRMED | same | posts |
| [`markdown.go:1712`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1712) | 10 | Warning | CONFIRMED | same | posts |
| [`sanitize.gno:173`](https://github.com/gnolang/gno/blob/a8bdd53b0/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L173) | 13 | Nit | UNVERIFIED |  | SKIP |
| [`sanitize.gno:578`](https://github.com/gnolang/gno/blob/a8bdd53b0/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L578) | 14 | Nit | UNVERIFIED |  | SKIP |
| [`pr6304_sanitize_gno_tags.md:41`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/adr/pr6304_sanitize_gno_tags.md#L41) | 5 | Nit | CONFIRMED | differs | posts |
| [`pr6304_sanitize_gno_tags.md:61`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/adr/pr6304_sanitize_gno_tags.md#L61) | 2 | Nit | CONFIRMED | differs | posts |
| [`markdown.go:476`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L476) | 4 | Nit | CONFIRMED | same | posts |
| [`native_gas.go:184`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/native_gas.go#L184) | 3 | Nit | CONFIRMED | differs | posts |
| [`markdown_bench_test.go:72`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/cmd/calibrate/markdown_bench_test.go#L72) | 12 | Suggestion | CONFIRMED | differs | posts |

### 8 · Warning · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:565`

An HTML block opened after a list marker hides a fence from goldmark but not from the tracker, so `[home]` in the user's item binds to the realm's definition; same at the merge base

- Check: Copy the repro file in and run go test ./gno.land/pkg/gnoweb/markdown -run TestB2FenceMismatch/list-html-hides-fence -v. It fails with `user text bound to the realm's [home] 1 time(s)`, while control-no-fence (`- a\n\n see [home]`) passes.
- Observed: head a8bdd53b0: `--- FAIL: TestB2FenceMismatch/list-html-hides-fence` sanitized: "\n\n- <div>\n ```\n\n see [home]\n ```\n\n", `user text bound to the realm's [home] 1 time(s)`; `--- PASS: .../control-no-fence` sanitized "\n\n- a\n\n see \\[home\\]\n\n". base d25b308c1: `--- FAIL: .../list-html-hides-fence`, sanitized "\n\n- <div>\n ```\n\n see [home]\n```\n\n", same bound-1-time error.
- Test: [`tests/b2-lines-removed-reach-catalog-fence-mismatch_test.go`](tests/b2-lines-removed-reach-catalog-fence-mismatch_test.go)

### 11 · Warning · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:882`

When a user's fence sits inside a list item, indented 4-6 spaces, the head leaves `[home]` in that item unescaped, and it links to the realm's own [home] definition; the merge base escaped it

- Check: Add both inputs to a TestB2FenceMismatch-style gnoweb integration case (realm chrome defining [home]: https://realm.example/home, BlockRich) and run go test ./gno.land/pkg/gnoweb/markdown -run <case> -v. The realm href inside the user's item confirms it. Measure it in one harness with candidates 8 and 9.
- Observed: Judge test at head a8bdd53b0: `bullet-fence-indent3-of-item: out="- a\n\n ```\n ```\n see [home]\n ```\n" realm-link-bound=true` and `ordered-fence-indent4-abs: out="1. a\n\n ```\n ```\n see [home]\n ```\n" realm-link-bound=true`, rendered `<p>see <a href="https://realm.example/home">home</a></p>`. The control `- a\n\n see \\[home\\]` is not bound. FAIL. Merge base d25b308c1, the same test: both cases `see \\[home\\]` with `realm-link-bound=false`, PASS.
- Test: [`tests/judge-11-list-item-fence-indent_test.go`](tests/judge-11-list-item-fence-indent_test.go)

### 9 · Warning · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:917`

A tab-indented fence line in a list item opens or closes a fence the tracker does not see, so `[home]` binds to the realm's definition; the opener form is new in this PR

- Check: Copy the repro file in and run go test ./gno.land/pkg/gnoweb/markdown -run 'TestB2FenceMismatch/tab' -v. tab-indented-closer, tab-indented-opener and tab-closer-full-reference fail on the realm href, and tab-closer-lrd-still-escaped passes.
- Observed: head a8bdd53b0: `--- FAIL: TestB2FenceMismatch/tab-indented-closer` sanitized "\n\n- a\n\n ```\n\t```\n see [home]\n ```\n\n"; `--- FAIL: .../tab-indented-opener` sanitized "\n\n- a\n\n\t```\n ```\n see [home]\n ```\n\n"; `--- FAIL: .../tab-closer-full-reference`; `--- PASS: .../tab-closer-lrd-still-escaped` (` \\[home]: https://evil.example`). base d25b308c1: tab-indented-closer and tab-closer-full-reference FAIL as well; `--- PASS: TestB2FenceMismatch/tab-indented-opener` with sanitized "\n\n- a\n\n\t```\n ```\n see \\[home\\]\n```\n\n".
- Test: [`tests/b2-lines-removed-reach-catalog-fence-mismatch_test.go`](tests/b2-lines-removed-reach-catalog-fence-mismatch_test.go)

### 6 · Warning · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:1057`

In BlockRich, everything a user writes after a fence opened 1 to 3 spaces in renders as code; at the merge base it rendered as prose

- Check: cp tests/b4-lines-removed-reach-catalog-dedent-closer-swallows-prose_test.go gno.land/pkg/gnoweb/markdown/zz_dedent_closer_test.go && go test ./gno.land/pkg/gnoweb/markdown -run TestRichDedentCloserSwallowsProse -count=1 -v: defect cases assert the prose renders inside <pre><code>; controls (content at the opener's column, strict Block) keep <p>prose after</p>
- Observed: head a8bdd53b0: `--- PASS: TestRichDedentCloserSwallowsProse/backtick-1-space`, sanitized="\n\n ```\n ```\ncode\n```\n\nprose after\n\n## heading after\n```\n\n" rendered="<pre><code></code></pre>\n<p>code</p>\n<pre><code>\nprose after\n\n## heading after\n</code></pre>\n"; tilde-2-spaces and quote-rich pass the same way; control-strict rendered="<pre><code>code\n</code></pre>\n<p>prose after</p>\n". base d25b308c1, same test: `--- FAIL: .../backtick-1-space` with rendered="<pre><code>code\n</code></pre>\n<p>prose after</p>\n<h2>heading after</h2>\n"; tilde-2-spaces and quote-rich also render <p>prose after</p> at base.
- Test: [`tests/b4-lines-removed-reach-catalog-dedent-closer-swallows-prose_test.go`](tests/b4-lines-removed-reach-catalog-dedent-closer-swallows-prose_test.go)

### 1 · Warning · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:1123`

Strict Block lets a pipe-less GFM table render as a live <table>; same at the merge base

- Check: cp tests/b3-lines-removed-reach-catalog-block-pipeless-table.txtar gno.land/pkg/gnoweb/markdown/golden/sanitize/ && go test ./gno.land/pkg/gnoweb/markdown -run 'TestSanitizeIntegration/b3-lines-removed-reach-catalog-block-pipeless-table' -v: passes with output.html holding <table>; run the same case at the merge base d25b308c1 to settle whether it predates the diff (judge) Issue draft: [issue.md](issue.md).
- Observed: head a8bdd53b0 and base d25b308c1, same command: `--- PASS: TestSanitizeIntegration/b3-lines-removed-reach-catalog-block-pipeless-table` on both, with the pinned output.html `<p>REALM BEFORE</p>\n<table>\n<thead>\n<tr>\n<th>a</th>\n<th>b</th>...<td>c</td>\n<td>d</td>...</table>\n<p>REALM AFTER</p>`. Base sanitize.gno:161 `Strict Block escapes the line-leading `\|` of any GFM table row`.
- Test: [`tests/b3-lines-removed-reach-catalog-block-pipeless-table.txtar`](tests/b3-lines-removed-reach-catalog-block-pipeless-table.txtar)

### 7 · Warning · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:1316`

BlockRich lets a gnoweb alert, or a shortcut reference bound to the realm's definition, through a multi-line link span; same at the merge base

- Check: Copy the repro file in and run go test ./gno.land/pkg/gnoweb/markdown -run TestB2AlertInLinkSpan -v. link-text and link-title fail on `gno-alert`, link-text-quote-ref and link-text-list-ref fail on the realm href, and control-bare-alert passes. Fix: reject a link span that crosses any line isBlockInterrupt accepts, or split the span at each such line the way appendFenceSpans does.
- Observed: head a8bdd53b0: `--- FAIL: TestB2AlertInLinkSpan/link-text` sanitized: "\n\n[a\n> [!WARNING] spoof](/u)\n\n" html holds `<details class="gno-alert gno-alert-warning" open>`; link-title the same; link-text-quote-ref and link-text-list-ref `user text bound to the realm's [home]`; `--- PASS: .../control-bare-alert` sanitized "\n\n> \\[!WARNING\\] spoof\n\n". base d25b308c1: identical PASS/FAIL set and sanitized strings. markdown.go head: `if ok && fenceLines.after(start) >= end && lrdLines.after(start) >= end {`.
- Test: [`tests/b2-lines-removed-reach-catalog-fence-mismatch_test.go`](tests/b2-lines-removed-reach-catalog-fence-mismatch_test.go)

### 10 · Warning · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:1712`

Once the scan budget runs out, a definition line with text after its URL is still deleted, so a fence inside that text escapes the trackers and the realm's footer renders inside a code block

- Check: cp tests/b5-lines-removed-reach-catalog-lrd-budget-trailing-text_test.go gnovm/stdlibs/chain/markdown/zz_lrd_budget_test.go && go test ./gnovm/stdlibs/chain/markdown -run TestLRDTrailingTextOutOfBudget -v: fails at a8bdd53b0; change line 1712 to `if !ok \|\| end < len(s) && s[end] != '\n' {` and it passes with TestEscapeBlockHazards and TestEscapeBlockHazardsRich still green
- Observed: Head a8bdd53b0, finder's file as zz_lrd_budget_test.go: `k=9 tail=269 pad=0: `[a]: u ```` deleted; footer in code block: true`, output tail "\n\n ```\n```\n- y\n ```\n~~~\n\n\n ```\n", FAIL. With 1712 changed to `if !ok \|\| ...`: `ok github.com/gnolang/gno/gnovm/stdlibs/chain/markdown 3.790s` (repro plus TestEscapeBlockHazards*). Pre-merge parent 1c377bf89 read: unbudgeted `for end < len(s) && (s[end] == ' ' \|\| ...)` then an unconditional `if end < len(s) && s[end] != '\n' { return 0, false }`. Merge base d25b308c1, the same test: `baseline changed: "```\n```\n- y\n ```\n~~~\n```\n"`, so the line is deleted within budget too.
- Test: [`tests/b5-lines-removed-reach-catalog-lrd-budget-trailing-text_test.go`](tests/b5-lines-removed-reach-catalog-lrd-budget-trailing-text_test.go)

### 13 · Nit · UNVERIFIED · `examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:173`

Package doc still says HTML block types 6 and 7 need no escape; the diff rewrote this claim at 362-365 and 453-456 but missed this sibling

- Check: sed -n 165,174p sanitize.gno against sed -n 362,365p and 453,456p; Block("<div>\n```\nx") output starts `\<div>`
- Observed: not run: on the finder's read

### 14 · Nit · UNVERIFIED · `examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:578`

BlockRich row promises every unclosed fence is autoclosed, but a fence opened on a list-marker or quote line is never tracked or closed

- Check: BlockRich("- ```\nx") and BlockRich("> ```\n> x") through the integration harness: output has no closing fence
- Observed: not run: on the finder's read

### 5 · Nit · CONFIRMED · `gno.land/adr/pr6304_sanitize_gno_tags.md:41`

Decision 7's calibration shape is written with raw line breaks inside a code span, so readers see `[a]: u (x ` instead of the shape. The unclosed backtick fragments that follow pair up with the wrong backticks.

- Check: sed -n 41,43p gno.land/adr/pr6304_sanitize_gno_tags.md; render the ADR and read Decision 7
- Observed: Read at head: sed -n 30,70p shows line 41 ending 'calibration shape `[a]: u', line 42 '(x', and line 43 beginning '`, both timed alternately …' with '(slowest: 16× `[a]: u\n(x\n` then `\n to the end, …'. grep -n puts 'calibration shape' on line 41.

### 2 · Nit · CONFIRMED · `gno.land/adr/pr6304_sanitize_gno_tags.md:61`

In Rich mode, a document-level fence split at a dedent deletes the user's `[label]: url` code line and renders `#` lines as headings. The ADR's fidelity cost says only that such a line shows a backslash.

- Check: cp tests/b3-lines-removed-reach-catalog-dedent-split-deletes-lrd.txtar gno.land/pkg/gnoweb/markdown/golden/sanitize/ && go test ./gno.land/pkg/gnoweb/markdown -run 'TestSanitizeIntegration/b3-lines-removed-reach-catalog-dedent-split-deletes-lrd' -v: passes with output.md missing the `[a]: http://x` line
- Observed: Read at head: markdown.go:404 `if !escapeLeader { s = closeFencesAtDedent(s) }` runs before :412 `walked := escapeBracketsOutsideLinks(s)`. The comment at 408-410 reads 'deletes LRD spans entirely'. The finder's golden pins output.md ` ```sh / ```/# install/make/ ```/ ``` ` with no `[a]` line, and output.html `<h1>install</h1><p>make</p>` between two empty `<pre><code>`. Not rerun: this judge writes nothing to the worktree.
- Test: [`tests/b3-lines-removed-reach-catalog-dedent-split-deletes-lrd.txtar`](tests/b3-lines-removed-reach-catalog-dedent-split-deletes-lrd.txtar)

### 4 · Nit · CONFIRMED · `gnovm/stdlibs/chain/markdown/markdown.go:476`

With `[home](` followed by `<gno-a b>)` on the next line, the user's `home` links to the realm's own `[home]` URL instead of the user's destination.

- Check: Copy tests/b1-removed-claims-pointy-dest-quote_test.go to gno.land/pkg/gnoweb/markdown/zz_pointy_dest_quote_test.go and run go test ./gno.land/pkg/gnoweb/markdown -run TestPointyDestOnQuoteContinuation -v at a8bdd53b0 and at d25b308c1; a fix moves escapeGnoExtDelimiters before escapeBracketsOutsideLinks as escapeGnoButtonTags was
- Observed: Read at head: :391 `s = escapeGnoButtonTags(s)` comes before :412 `escapeBracketsOutsideLinks(s)`, which comes before :476 `escapeGnoExtDelimiters(line)` inside `for idx, line := range lines`. scanLinkURL :1468 is `skipSpace(s, i, 1, budget)`, then `if s[i] == '<'`. markdown_test.go:258 pins only the one-line `[home](<gno-a b>)`. Base: :397 walker, :434 line-start escape in the loop. The finder's run shows BlockRich `[home](\n<gno-a b>)` bound-to-realm=true own-dest=false. Not rerun here.
- Test: [`tests/b1-removed-claims-pointy-dest-quote_test.go`](tests/b1-removed-claims-pointy-dest-quote_test.go)

### 3 · Nit · CONFIRMED · `gnovm/stdlibs/native_gas.go:184`

On this host the PR's own slowest Rich shape costs 3.0x-3.3x the old calibration per byte, above the charged 2.95x

- Check: From a clone at a8bdd53b0: copy d25b308c1's chain/markdown/markdown.go into gnovm/stdlibs/chain/markdown/zzold as package zzold, drop tests/b1-removed-claims-gas-factors_test.go in as gnovm/stdlibs/chain/markdown/zz_gasprobe_test.go, run go test ./gnovm/stdlibs/chain/markdown -run 'TestGasProbe$\|TestGasProbeSweep\|TestGasProbeLongFence' -v -count=1 and read new/oldcal for the rich rows against 2.95; repeat on the calibration reference machine, since the ratio is machine-dependent \| cp tests/b5-lines-removed-reach-catalog-rich-long-fence-gas_test.go into gnovm/stdlibs/chain/markdown with the merge-base markdown.go as package zzold (header has the commands); GOMAXPROCS=1 go test -run TestGasShapeFactors -v on a quiet host: the `rich, PR slow shape` median above 2.95 confirms
- Observed: head a8bdd53b0 with merge-base markdown.go as zzold, GOMAXPROCS=1. b1 TestGasProbe$ run 1: `rich author rich: 75% fence-dedent + 64x lrd-title + `\n new/oldcal= 3.33 (factor 2.95) ... <-- above factor`; run 2: `new/oldcal= 3.28`; strict 16x shape 2.11 and 2.20 vs 2.45. b5 TestGasShapeFactors: `rich, PR slow shape out=155919 factor median 3.01 [p25 2.92, p75 3.08] vs charged 2.95 UNDERCHARGED by 2%`, `strict, PR slow shape ... median 2.42 ... covered`, `rich, frac=0.93 run=3 ... 2.74 covered`, `rich, frac=0.90 run=30 ... 2.57 covered`.
- Test: [`tests/b5-lines-removed-reach-catalog-rich-long-fence-gas_test.go`](tests/b5-lines-removed-reach-catalog-rich-long-fence-gas_test.go)

### 12 · Suggestion · CONFIRMED · `gnovm/cmd/calibrate/markdown_bench_test.go:72`

No step of the calibration pipeline reads the new _Slow_ benchmarks, so the next re-fit pasted into native_gas.go silently drops the x2.45 and x2.95 hand factors

- Check: grep -n EscapeBlockHazards gnovm/cmd/calibrate/gen_native_table.py; grep -c Slow gnovm/cmd/calibrate/markdown_bench_output.txt; run go test -bench='BenchmarkNative_Markdown_EscapeBlockHazards' -count=3 . in gnovm/cmd/calibrate, feed the output to gen_native_table.py, and compare the emitted slope with native_gas.go's 67034
- Observed: Read at a8bdd53b0: `grep -n 'EscapeBlockHazards\\|Slow' gen_native_table.py` gives only lines 209-210 (`r"BenchmarkNative_Markdown_EscapeBlockHazards_(\d+)-\d+\s+..."`). `grep -c Slow markdown_bench_output.txt` gives 0. native_gas.go:184 `Slope: 67034 ... // 27361×2.45 (#6304 ... not re-fit ...)` and :185 `Slope: 81411 ... // 27597×2.95`. The bench run and re-fit were not run; the regex non-match is settled by the read.

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`markdown.go:543`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L543) | claims | A \f- or \v-only line is non-blank to prevParagraph but blank to goldmark, so strict would leave an empty list item unescaped | grep spaceTable in goldmark@v1.8.2/util/util.go: entries 11 (\v) and 12 (\f) are 0, so goldmark does not treat those lines as blank either |
| [`markdown.go:1268`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1268) | claims | ADR Decision 1: walker no longer quadratic in leading spaces | TestGasProbeLongFence on head |
| [`pr6304_sanitize_gno_tags.md:63`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/adr/pr6304_sanitize_gno_tags.md#L63) | claims | Not-addressed O(n^1.5) LRD-title-then-thematic-break shape costs the same before and after | TestGasProbe on head vs d25b308c1 copy |
| [`markdown.go:476`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L476) | removed | Quote continuation "> [home](\n> <gno-a b>)" newly escaped after > would bind [home] to the realm | TestPointyDestOnQuoteContinuation at head |
| [`native_gas.go:184`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/native_gas.go#L184) | claims | Slope arithmetic 27361x2.45 and 27597x2.95 against the merge-base rows | git show d25b308c1:gnovm/stdlibs/native_gas.go \| grep EscapeBlockHazards: Slope 27361 / 27597; head 67034 / 81411 = 27361x2.45, 27597x2.95 |
| [`pr6299_gnoweb_frame.md:172`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/adr/pr6299_gnoweb_frame.md#L172) | claims | Annotation says frame tags after > or list markers are now escaped, which the blockrich-gno-frame-* goldens show | cat blockrich-gno-frame-in-list.txtar / -in-blockquote.txtar: output.md '- \<gno-frame>' and '> \<gno-frame>', html '&lt;gno-frame&gt;' |
| [`SANITIZE.md:107`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/SANITIZE.md#L107) | removed | Removed gnolang/gno#6300 known-gap sentence leaves a sibling doc still naming the gap | grep -rn 6300 --include=*.md: pr5593_gnoweb_details_block.md:53 ends 'gnolang/gno#6304 closes the gap.'; pr6299 annotated '(Resolved in PR gnolang/gno#6304)' |
| [`SANITIZE.md:108`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/SANITIZE.md#L108) | claims | Blockquote row: button after a fence under an HTML tag line, the tag escaped | cat blockquote-gno-button-after-fence-in-html-block.txtar: output.md '> \<div>' and '> \<gno-button ...' inside the code block |
| [`pr6297_gnoweb_inline_icons.md:251`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/adr/pr6297_gnoweb_inline_icons.md#L251) | claims | Icons at the start of a list item or quote keep rendering | escapeGnoExtDelimiters markdown.go:713 '!isClose && hasContainer && hasGnoIconPrefix(rest)' returns line; sanitize_integration_test.go:660-675 asserts <svg class="gno-icon" for list-bullet, list-numbered, blockquote |
| [`markdown.go:1012`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1012) | lines | closeFencesAtDedent ends a document-level fence indented 1-3 spaces at a less-indented body line, so goldmark renders that code line as live markdown | grep -n 'dedent' gno.land/adr/pr6304_sanitize_gno_tags.md, line 49: 'For a document-level fence indented 1–3 spaces, goldmark keeps the dedented line as code ... The written closer makes both readings agree.' The trade-off is recorded in the ADR as a decision, not missed |
| [`markdown.go:713`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L713) | reach | the gno-icon exemption after a container prefix could open an HTML block that hides a fence line, like `- <div>` | go test -run TestB2FenceMismatch/list-gno-icon-hides-fence: '--- PASS: TestB2FenceMismatch/list-gno-icon-hides-fence', gnoweb's icon parser takes the tag, so no HTML block opens |
| [`markdown.go:880`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L880) | catalog | a gnoweb display-math block ($$) could hide a fence opener from goldmark while the tracker opens a fence | sed -n 640,780p gno.land/pkg/gnoweb/markdown/ext_math.go: hasClosingLine stops at p.starters.endsMath(pc, line), which is blank or startsBlock, so a fence line ends the math lookahead and the block never opens over it |
| [`markdown.go:1155`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1155) | lines | isCloseFence rejects a closer ending in \r, which goldmark accepts as a line ending | grep -n 'func BlockRich' -A3 examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno: 's = markdown.NormalizeBreaks(s)' runs first, so no \r reaches the native through sanitize; isCloseFence is also unchanged by the diff |
| [`markdown.go:1877`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1877) | catalog | fenceLineFinder, lrdLineFinder, htmlBlockHidesFence and closeFencesAtDedent could rescan lines and make the native super-linear against its linear gas slope | read: each finder memoises its next hit (f.next > p, scanEnd, nextAt) and every opensLinkLabel scan stops at the next '[', so the scans cover disjoint ranges; the gas factors are already in alexiscolin's thread and not re-raised |
| [`markdown.go:623`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L623) | reach | a form feed before `<gno-card>` stops parseContainerPrefix while goldmark's util.TrimLeftSpace strips it, so the tag might open live | probe BlockRich("\f<gno-card>\nx\n</gno-card>") through the gnoweb renderer: every gno block parser triggers on '<' (or '>'), goldmark picks parsers by the first non-space byte, and the render is `<p><!-- raw HTML omitted -->⏎x⏎&lt;/gno-card&gt;</p>`, no card |
| [`sanitize.gno:435`](https://github.com/gnolang/gno/blob/a8bdd53b0/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L435) | reach | the new HTML-hides-fence escape, empty-item escape and dedent pre-pass could break Block/BlockRich idempotency, drop the realm footer, leave a gno-* element live or bind `[home]` | 3000 random 1-6 line inputs over 42 line shapes (fences at 0-3 spaces and tab, `- ```, `> ```, <div>, <script>, <gno-card>, empty items, LRDs, `[home]`, alerts, tables) through Block and BlockRich twice and rendered between realm chrome with a realm `[home]` definition: zero idempotency, footer, gno-live or home-bound failures |
| [`sanitize.gno:389`](https://github.com/gnolang/gno/blob/a8bdd53b0/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L389) | lines | Block table row claims empty list items are escaped too | probe: Block("-") -> "\\-", Block("- ") -> "\\- ", Block("1.") -> "1\\.", Block("1)") -> "1\\)", Block("+\t") -> "\\+\t", Block("a\n\n*") -> "\\*"; each renders <p>, no list |
| [`sanitize.gno:409`](https://github.com/gnolang/gno/blob/a8bdd53b0/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L409) | removed | Block row says an unclosed fence is autoclosed at its opener's column, and the rewritten 362-365 claim that types 6/7 are escaped only when a fence line follows | probe: Block(" ```\nx") -> "\n\n ```\nx\n ```\n\n"; Block("<div>\n```\nx") escapes `\<div>`; Block("<div>\nx\n\n```\ny") leaves <div> and renders `<!-- raw HTML omitted -->` |
| [`blockrich-gno-details-in-blockquote.txtar:10`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/golden/sanitize/blockrich-gno-details-in-blockquote.txtar#L10) | removed | the merge regenerated master's fixture, dropping the live nested gno-details master asserted inside a blockquote; suspected a removed feature or a container shape the escape misses | 3e2f34c70's message: 'User content cannot open or close one'; probe TestZZProbe over 37 container shapes (list, ordered, nested quote, list-in-quote, alert, tab, form feed, NBSP, uppercase, task item, html-block-hidden fence) x Block/BlockRich/Blockquote/BlockquoteRich x plain/details/columns contexts: 0 BAD rows for the helpers, 87 BAD rows for the unsanitized control |
| [`block-ext-delim-inside-fence-escaped.txtar:15`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/golden/sanitize/block-ext-delim-inside-fence-escaped.txtar#L15) | removed | renamed from gnolang/gno#5714's block-ext-delim-inside-fence-preserved (Jae Kwon), flipping the preserved-in-fence contract: every user code sample naming a gno tag now shows a stray backslash | ADR pr6304_sanitize_gno_tags.md:61 records 'a block-start `<gno-…>` tag ... inside real code shows a backslash'; alexiscolin's CHANGES_REQUESTED at 43afbb384: 'apart from the costs the ADR records (a backslash in code, 1-3 space fences split in Rich), which I'm fine with' |
| [`block-ext-delim-four-spaces.txtar:12`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/golden/sanitize/block-ext-delim-four-spaces.txtar#L12) | lines | a 4-space indented `<gno-card>` after a blank line now shows `\<gno-card>` in an indented code block; suspected an escape goldmark never needed | a 4-space line is a block start inside a list item (`- a\n\n <gno-card>`), which the sanitizer cannot tell apart; ADR pr6304_sanitize_gno_tags.md:14 places the backslash after the indent to keep the code block; fence probe: Block(" <gno-columns>") renders <pre><code>\\&lt;gno-columns&gt;</code></pre>, no live tag |
| [`blockrich-gno-form-in-list-fence.txtar:25`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/golden/sanitize/blockrich-gno-form-in-list-fence.txtar#L25) | lines | the user's closer after the dedented tag becomes an empty doc-level code block; suspected the same swallow as the document-level case | in the list reading goldmark itself ends the item at the dedented line and reads the user's closer as a doc-level opener (ADR pr6304_sanitize_gno_tags.md:11), so the sanitized output agrees with goldmark on unsanitized input; only the document-level fence diverges, which the Warning covers |
| [`markdown.go:1049`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1049) | catalog | Determinism/consensus: the fixture rewrites show a native stdlib's output changing for existing inputs, which changes realm state on replay | ADR pr6304_sanitize_gno_tags.md:64 'This change requires a coordinated protocol upgrade on gno.land ... cannot be deployed unilaterally'; status line 5 'Proposed (coordinated chain upgrade required)' |
| [`native_gas.go:185`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/native_gas.go#L185) | reach | Rich shapes with a larger or longer-run fence region (budget is 8*len after closeFencesAtDedent, up to ~2x) beat the PR's slow shape | TestGasShapeFactors GOMAXPROCS=1, same rounds as the PR shape |
| [`markdown.go:1155`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1155) | catalog | isCloseFence rejects a closer followed by \f or \v that goldmark accepts, leaving the tracker in a fence goldmark closed | goldmark v1.8.2 util/util.go:806 spaceTable marks only bytes 9, 10, 13 and 32 as space, so goldmark rejects that closer too |
| [`native_gas.go:185`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/native_gas.go#L185) | reach | Rich output up to ~1.9x the input is returned without allocation accounting | generated.go wrapper returns via gno.Go2GnoValue(m.Alloc,...), gonative.go:92 `tv.V = alloc.NewString(rv.String())` |
| [`markdown_bench_test.go:321`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/cmd/calibrate/markdown_bench_test.go#L321) | removed | Rich calibration bench moved off the fit shape `[a]: u\n(x\n` to ` ```\nx\n`, and gen_native_table.py reads neither the Slow benches nor a Rich row, so a refit would drop the x2.95 | native_gas_table.go.txt carries no chain/markdown row (grep empty), so the markdown rows are hand-written; the bench shape is alexiscolin's 2026-10-26 native_gas.go thread (`the bench's ... comes out ~1.1x`) |
| [`markdown.go:1012`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1012) | reach | closeFencesAtDedent closes every 1-3 space document-level fence at the first less-indented line, moving code into markdown or the rest of the text into code | alexiscolin CHANGES_REQUESTED 2026-10-26: `apart from the costs the ADR records (a backslash in code, 1-3 space fences split in Rich), which I'm fine with` |
| [`markdown.go:1933`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown.go#L1933) | removed | 1c377bf89 alert guard misses an alert shape gnoweb renders | ext_alert.go:102-119 opens an alert only on `>` plus at most one space or tab then `^\[!`; parseContainerPrefix (markdown.go:623-646) skips exactly those and the guard requires `>` in s[ls:q] |
| [`markdown_test.go:525`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown_test.go#L525) | catalog | isBlockInterruptFn is a package var a test reassigns | `git show d25b308c1:.../markdown.go \| grep -c isBlockInterruptFn` = 3 (predates the PR); no t.Parallel in the package |
| [`markdown_test.go:1`](https://github.com/gnolang/gno/blob/a8bdd53b0/gnovm/stdlibs/chain/markdown/markdown_test.go#L1) | removed | merge a8bdd53b0 dropped master-side lines in bundle files | every master-side added line since merge-base bcfcfd525 is present verbatim in a8bdd53b0 except two scanLRDTail lines, both rewritten (one is the Warning above) |
| [`apphash_crossrealm38_test.go:254`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L254) | removed | The rewritten hash comment no longer has the bisection line from 1c377bf89 that pinned the move to markdown.gno alone, so the new value might hide another consensus-relevant change from the merge | In a scratch worktree at a8bdd53b0: `go test ./gno.land/pkg/sdk/vm -run 'TestAppHashCrossrealm38$'` printed `ok`. With `git show d25b308c1:gnovm/stdlibs/chain/markdown/markdown.gno` written over the file, the same run printed `actual : "91649bb8181ca02d61f2880ddb7b5c9f59359d14d8e5a4c2cb429ab7893937b4"`, the merge base's pinned value. markdown.gno alone moves the hash, and its diff against the base is comment lines only. |
| [`sanitize_integration_test.go:575`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/sanitize_integration_test.go#L575) | reach | The `[" ]` suffix in liveExtension might miss the class attribute some extensions actually render | grep of the gnoweb ext_*.go render calls: `<form class="gno-form"`, `<div class="gno-columns">`, `<div class="gno-column">`, `<section class="gno-frame">`, `<div class="gno-foreign" role=...`, `class="gno-button gno-button-outline`, `class="gno-alert gno-alert-%s`. Each is followed by `"` or a space, so the suffix matches every container the regex names. |
| [`sanitize_integration_test.go:568`](https://github.com/gnolang/gno/blob/a8bdd53b0/gno.land/pkg/gnoweb/markdown/sanitize_integration_test.go#L568) | catalog | The new test sends 167 subtests through the shared package-level sanitize driver Machine (global mutable state class) | grep 'Parallel()' gno.land/pkg/gnoweb/markdown/*_test.go finds hits only in ext_links_test.go. TestExtensionAndFooterIsolation and callSanitizeGno's callers run sequentially, so nothing touches the driver Machine concurrently. |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gnovm/stdlibs/chain/markdown/markdown.go:1123 | cp tests/b3-lines-removed-reach-catalog-block-pipeless-table.txtar gno.land/pkg/gnoweb/markdown/golden/sanitize/ && go test ./gno.land/pkg/gnoweb/markdown -run 'TestSanitizeIntegration/b3-lines-removed-reach-catalog-block-pipeless-table' -v: passes with output.html holding <table>; run the same case at the merge base d25b308c1 to settle whether it predates the diff (judge) | hot | pipeless-table-row |
| 2 | CONFIRMED | Nit | gno.land/adr/pr6304_sanitize_gno_tags.md:61 | cp tests/b3-lines-removed-reach-catalog-dedent-split-deletes-lrd.txtar gno.land/pkg/gnoweb/markdown/golden/sanitize/ && go test ./gno.land/pkg/gnoweb/markdown -run 'TestSanitizeIntegration/b3-lines-removed-reach-catalog-dedent-split-deletes-lrd' -v: passes with output.md missing the `[a]: http://x` line | hot |  |
| 3 | CONFIRMED | Nit | gnovm/stdlibs/native_gas.go:184 | From a clone at a8bdd53b0: copy d25b308c1's chain/markdown/markdown.go into gnovm/stdlibs/chain/markdown/zzold as package zzold, drop tests/b1-removed-claims-gas-factors_test.go in as gnovm/stdlibs/chain/markdown/zz_gasprobe_test.go, run go test ./gnovm/stdlibs/chain/markdown -run 'TestGasProbe$\|TestGasProbeSweep\|TestGasProbeLongFence' -v -count=1 and read new/oldcal for the rich rows against 2.95; repeat on the calibration reference machine, since the ratio is machine-dependent \| cp tests/b5-lines-removed-reach-catalog-rich-long-fence-gas_test.go into gnovm/stdlibs/chain/markdown with the merge-base markdown.go as package zzold (header has the commands); GOMAXPROCS=1 go test -run TestGasShapeFactors -v on a quiet host: the `rich, PR slow shape` median above 2.95 confirms | warm |  |
| 4 | CONFIRMED | Nit | gnovm/stdlibs/chain/markdown/markdown.go:476 | Copy tests/b1-removed-claims-pointy-dest-quote_test.go to gno.land/pkg/gnoweb/markdown/zz_pointy_dest_quote_test.go and run go test ./gno.land/pkg/gnoweb/markdown -run TestPointyDestOnQuoteContinuation -v at a8bdd53b0 and at d25b308c1; a fix moves escapeGnoExtDelimiters before escapeBracketsOutsideLinks as escapeGnoButtonTags was | hot |  |
| 5 | CONFIRMED | Nit | gno.land/adr/pr6304_sanitize_gno_tags.md:41 | sed -n 41,43p gno.land/adr/pr6304_sanitize_gno_tags.md; render the ADR and read Decision 7 | hot |  |
| 6 | CONFIRMED | Warning | gnovm/stdlibs/chain/markdown/markdown.go:1057 | cp tests/b4-lines-removed-reach-catalog-dedent-closer-swallows-prose_test.go gno.land/pkg/gnoweb/markdown/zz_dedent_closer_test.go && go test ./gno.land/pkg/gnoweb/markdown -run TestRichDedentCloserSwallowsProse -count=1 -v: defect cases assert the prose renders inside <pre><code>; controls (content at the opener's column, strict Block) keep <p>prose after</p> | hot | dedent-closer-swallow |
| 7 | CONFIRMED | Warning | gnovm/stdlibs/chain/markdown/markdown.go:1316 | Copy the repro file in and run go test ./gno.land/pkg/gnoweb/markdown -run TestB2AlertInLinkSpan -v. link-text and link-title fail on `gno-alert`, link-text-quote-ref and link-text-list-ref fail on the realm href, and control-bare-alert passes. Fix: reject a link span that crosses any line isBlockInterrupt accepts, or split the span at each such line the way appendFenceSpans does. | hot | link-span-block-interrupt |
| 8 | CONFIRMED | Warning | gnovm/stdlibs/chain/markdown/markdown.go:565 | Copy the repro file in and run go test ./gno.land/pkg/gnoweb/markdown -run TestB2FenceMismatch/list-html-hides-fence -v. It fails with `user text bound to the realm's [home] 1 time(s)`, while control-no-fence (`- a\n\n see [home]`) passes. | hot | fence-tracker-goldmark-mismatch |
| 9 | CONFIRMED | Warning | gnovm/stdlibs/chain/markdown/markdown.go:917 | Copy the repro file in and run go test ./gno.land/pkg/gnoweb/markdown -run 'TestB2FenceMismatch/tab' -v. tab-indented-closer, tab-indented-opener and tab-closer-full-reference fail on the realm href, and tab-closer-lrd-still-escaped passes. | hot | fence-tracker-goldmark-mismatch |
| 10 | CONFIRMED | Warning | gnovm/stdlibs/chain/markdown/markdown.go:1712 | cp tests/b5-lines-removed-reach-catalog-lrd-budget-trailing-text_test.go gnovm/stdlibs/chain/markdown/zz_lrd_budget_test.go && go test ./gnovm/stdlibs/chain/markdown -run TestLRDTrailingTextOutOfBudget -v: fails at a8bdd53b0; change line 1712 to `if !ok \|\| end < len(s) && s[end] != '\n' {` and it passes with TestEscapeBlockHazards and TestEscapeBlockHazardsRich still green | hot | lrd-trailing-text-fail-open |
| 11 | CONFIRMED | Warning | gnovm/stdlibs/chain/markdown/markdown.go:882 | Add both inputs to a TestB2FenceMismatch-style gnoweb integration case (realm chrome defining [home]: https://realm.example/home, BlockRich) and run go test ./gno.land/pkg/gnoweb/markdown -run <case> -v. The realm href inside the user's item confirms it. Measure it in one harness with candidates 8 and 9. | hot | container-relative-fence-indent |
| 12 | CONFIRMED | Suggestion | gnovm/cmd/calibrate/markdown_bench_test.go:72 | grep -n EscapeBlockHazards gnovm/cmd/calibrate/gen_native_table.py; grep -c Slow gnovm/cmd/calibrate/markdown_bench_output.txt; run go test -bench='BenchmarkNative_Markdown_EscapeBlockHazards' -count=3 . in gnovm/cmd/calibrate, feed the output to gen_native_table.py, and compare the emitted slope with native_gas.go's 67034 | cold |  |
| 13 | UNVERIFIED | Nit | examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:173 | sed -n 165,174p sanitize.gno against sed -n 362,365p and 453,456p; Block("<div>\n```\nx") output starts `\<div>` | warm |  |
| 14 | UNVERIFIED | Nit | examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:578 | BlockRich("- ```\nx") and BlockRich("> ```\n> x") through the integration harness: output has no closing fence | warm |  |

</details>

Hit rate per tier, from the record: hot 10/10 confirmed over 2 files, warm 1/3 confirmed over 15 files, cold 1/1 confirmed over 8 files.

## Completeness

- Anchor moved: row 8 sat on the `mayOpenHTMLBlock` func line, markdown.go:565; the draft anchors it on the call site at markdown.go:505, the line its fix edits.
- Repros: every Warning's repro block was extracted from the draft and run verbatim at a8bdd53b0 and at the merge base d25b308c1. Outputs: [tests/writer-repros-head.out](tests/writer-repros-head.out), [tests/writer-repros-base.out](tests/writer-repros-base.out). Each matches its verdict: rows 11, 6 and the tab opener of row 9 differ from the base; rows 8, 1, 7 and the tab closer of row 9 fail the same at the base; row 10 deletes the line at k=9 tail=269 at head and at k=8 tail=0 at the base.
- Threads: row 3 ships SKIP, since the gas factors sit in alexiscolin's CHANGES_REQUESTED thread on native_gas.go, answered in 1c377bf89, and the 3.01 median comes from a loaded host. Row 7 is not that review's alert thread, which used a tab-indented document-level fence; here a link span crosses a blockquote line. Row 6 is not the ADR thread on the list-item split into empty blocks; here the prose after the closer becomes code.
- ADR residual: rows 8, 9 and 11 sit in the class the ADR's Residual bullet accepts, user `[home]` binding to the realm's definition on lines the tracker counts as code. They are posted because rows 11 and 9's opener are regressions against the base, and row 8 is a missed member of the class the diff sweeps with `htmlBlockHidesFence`.
- Verdict: REQUEST CHANGES rests on rows 6, 11 and 9, which the branch introduces, and on row 10, the branch's own check failing open, which 1c377bf89 had closed and the merge reopened. Rows 8, 1 and 7 predate the branch and do not block; row 1 has its draft in issue.md.
- Pre-existing defects: row 1 has an issue draft at [issue.md](issue.md); an open-PR search found no fix. Rows 8 and 7 are in scope as missed class members and carry no separate draft.
- Row 12: markdown_bench_test.go:72 is a `StripBidiAndZeroWidth` benchmark; the Body bullet links gen_native_table.py:209-210 and the `_Slow_1000` benchmark at markdown_bench_test.go:379 instead.
- Not settled: `round check` still lists overview.md:138, a bare `#6300`, left as written since the overview is not touched this round. The writer read neither diff.md nor risk.md; each anchor was read at the head instead.

## Retro

- What failed: 42 min, inside the 46-49 projected, but $39.12 against $21: 647k output against 436k, the finders running long probes over 37 container shapes and 3000 random inputs.
- What worked: 12 candidates kept, 0 refuted; the merge-conflict check confirmed every master-side line survived but two rewritten `scanLRDTail` lines, one of them a Warning.
- Hit rate per tier: hot `markdown.go` carries all 7 Warnings and one Nit; the ADR and `native_gas.go` the other Nits, the calibration script the Suggestion.
- Upgrade, proposed and not applied: price a `complex` stdlib round from its own class's measured output rather than the pipeline mean, which ran 1.5x under here.
