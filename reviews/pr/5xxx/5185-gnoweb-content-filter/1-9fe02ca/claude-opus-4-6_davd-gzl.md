# PR #5185: feat(gnoweb): content filter extension

**URL:** https://github.com/gnolang/gno/pull/5185
**Author:** alexiscolin | **Base:** master | **Files:** 97 | **+1030 -4**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR adds a goldmark markdown extension that filters spam/scam content at render time in gnoweb. It operates at the block level (paragraph, heading, text block): if any compiled regex pattern matches the full extracted text of a block, the entire block is replaced with a configurable label (e.g., `[blocked URL]`, `[filtered]`).

**Key components:**
- `ext_contentfilter.go` (170 lines) — Core `Filter` struct with `NewFilter()` (parses pattern definitions from text), `Match()` (first-match-wins), and custom goldmark `NodeRenderer` implementations for `ast.Paragraph`, `ast.Heading`, and `ast.TextBlock`.
- `contentfilter_patterns.txt` (192 lines) — Embedded pattern file with 65 categories covering credential phishing, social redirects, free crypto scams, financial promises, adult content, gore, known scam addresses/domains, unicode abuse, and content abuse (caps, excessive punctuation, link spam).
- `ext.go` — Adds `WithContentFilter()` option and wires the extension into `GnoExtension.Extend()`.
- `render_config.go` — Enables `DefaultContentFilter` by default for all gnoweb rendering.
- `ext_test.go` — Adds test filter patterns and conditional filter selection: mechanism tests use a minimal stable pattern set; pattern tests use the full production patterns.
- 94 golden test files — 29 mechanism tests (`ext_contentfilter/`) + 65 pattern category tests (`ext_contentfilter_patterns/`).

**Design:** The filter is applied at the renderer level (not parser). `nodeText()` recursively extracts all text content from a node's children (including inline code spans), then runs sequential regex matching. `html.EscapeString` is used on replacement text to prevent XSS. Patterns are compiled once at init time via `DefaultContentFilter = NewFilter(DefaultContentFilterPatterns)`.

**Context:** This is a WIP/PoC companion to PR #5178 (antispam scoring package). The branch also contains unrelated commits from jaekwon (tictac.md, whitepaper content).

## Test Results
- **Existing tests:** PASS (`ok github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown 0.104s`)
- **CI status:** All checks pass except "Merge Requirements" (awaits codeowner approval from gfanton)
- **Codecov:** 92.39% patch coverage, 7 lines uncovered in `ext_contentfilter.go`
- **Edge-case tests:** skipped

## Critical (must fix)

- [ ] `ext_contentfilter.go:110` / `golden/ext_contentfilter/20-preserves-inline-code.md.txtar` — **Inline code is NOT preserved despite claims.** The `nodeText()` function at `utils.go:70-88` recursively extracts text from ALL child nodes including `ast.CodeSpan` children. When a paragraph contains `` `evil-site.com` ``, the pattern matches against the extracted text `evil-site.com` and the entire paragraph is replaced. Test 20 confirms: input `` Use `evil-site.com` as example. `` renders as `<p>[blocked URL]</p>`. The PR description states "Code blocks and inline code are untouched" — this is false for inline code. Either `nodeText` should skip `ast.CodeSpan` nodes, or the documentation/claims should be corrected.

- [ ] `ext_contentfilter.go:108-121` / `golden/ext_contentfilter/22-in-table.md.txtar` — **Table structure is destroyed by the filter.** When a table cell contains filtered content, the entire table renders as `<p>[blocked URL]</p>` instead of a proper `<table>` with a filtered cell. This happens because goldmark's table extension creates paragraph-like nodes inside cells that the content filter's paragraph renderer intercepts. The filter should either (a) not register for nodes inside tables, or (b) implement a table cell renderer that replaces only the cell content while preserving table structure.

## Warnings (should fix)

- [ ] `ext_contentfilter.go:129,135,139` — **Heading level index without bounds check.** `"0123456"[n.Level]` will panic if `n.Level > 6` or `n.Level == 0`. While goldmark's standard parser constrains headings to levels 1–6, a custom AST transformer or extension could produce out-of-range values. A bounds check (`if n.Level < 1 || n.Level > 6`) would prevent potential panics.

- [ ] `contentfilter_patterns.txt:188` — **`[A-Z\s]{20,}` caps pattern has high false-positive risk.** This pattern matches 20+ characters consisting of uppercase letters AND whitespace. A normal paragraph like `"THE QUICK BROWN FOX JUMPS"` (25 chars) would be filtered as `[caps]`. More importantly, `\s` includes spaces, so even mixed-case text with enough spaces and some uppercase could match. Consider requiring a minimum ratio of uppercase to total characters instead of a simple length threshold, or at least raising the threshold significantly.

- [ ] `contentfilter_patterns.txt:192` — **`^.{0,20}https?://[^\s]+$` "link only" pattern may not work as intended for multi-line text.** The `^` and `$` anchors match start/end of the entire string by default in Go's RE2, not per-line. Since `nodeText()` concatenates all text children without line breaks, this pattern matches paragraphs that are essentially just a URL (up to 20 chars of prefix). However, if the paragraph has more text beyond the URL, this won't match — which may be the desired behavior. Clarify the intent in a comment.

- [ ] `contentfilter_patterns.txt:11` — **Contact redirect pattern matches bare email addresses.** The last alternative `[\w.+-]{3,}@(?:gmail|yahoo|hotmail|...)\.(?:com|...)` matches any email-like string, meaning legitimate content like "Contact us at support@gmail.com" would be filtered as `[contact redirect]`. Email-in-paragraph detection should require additional scam context rather than matching standalone email patterns.

- [ ] `ext_contentfilter.go:20` — **`DefaultContentFilter` compiled at package init time.** All 65+ complex regex patterns are compiled into DFAs at import time regardless of whether the content filter is enabled. For packages that import the markdown package but don't use filtering, this adds unnecessary startup cost. Consider using `sync.Once` for lazy initialization.

- [ ] PR branch — **Contains unrelated commits.** The branch includes commits from jaekwon about `tictac.md`, whitepaper content, and physics discoveries that are unrelated to the content filter feature. The branch should be rebased/cleaned before merge to keep git history clean.

## Nits

- [ ] `ext_contentfilter.go:67` — `log.Printf` for invalid patterns uses stdlib `log` rather than structured logging. In production code, this should use the project's logging approach or at minimum accept a logger parameter.
- [ ] `ext_contentfilter.go:166` — The `Extend` method signature `Extend(m goldmark.Markdown, filter *Filter)` differs from the standard `goldmark.Extender` interface (`Extend(goldmark.Markdown)`). This means `ExtContentFilter` cannot be used as a standalone goldmark extension — it must always be called through `GnoExtension`. This is fine for now but worth noting if the extension might be reused elsewhere.
- [ ] `contentfilter_patterns.txt:170,173` — Adult content and gore patterns are extremely long (>2000 chars each). These would benefit from being split into multiple focused patterns for maintainability and debuggability.

## Missing Tests

- [ ] No benchmark tests for filter performance with all 65+ patterns — sequential regex evaluation on every text block could be a bottleneck on pages with many paragraphs. Add `BenchmarkFilter` with realistic content (`ext_contentfilter.go`).
- [ ] No test for the `DEFAULT_REPLACEMENT=` parsing path being exercised when a pattern has no ` -> ` override. Test 05 (`default-replacement`) covers the render output but only with `test-spam-url\.xyz` which has no replacement — add a test that explicitly verifies `Match()` returns the configured default.
- [ ] No test for invalid regex patterns being gracefully skipped (the `log.Printf` path at `ext_contentfilter.go:67`). A unit test for `NewFilter` with a malformed pattern should verify the filter still works with remaining valid patterns.
- [ ] No test for `nil` filter behavior — `Match()` handles `nil` receiver at line 83, but no test exercises `WithContentFilter(nil)` or the `if e.cfg.contentFilter != nil` guard at `ext.go:93`.
- [ ] No test for a heading with filtered content at level > 3 (only level-1 headings are tested in test 15). Test levels 2–6 to ensure the heading renderer works across all valid levels.
- [ ] No test for `ast.TextBlock` filtering — the `renderTextBlock` function at line 145 is registered but no golden test explicitly covers a text block node (which differs from paragraphs in goldmark's AST).

## Suggestions

- **Consider operating at a finer granularity.** Currently, if any part of a paragraph matches, the entire paragraph is replaced. This means a long informative paragraph containing one suspicious URL loses all content. A per-link or per-segment approach would preserve non-matching content while redacting only the offending part. This is the most impactful UX improvement. (`ext_contentfilter.go:108-121`)

- **Add a pattern priority or weight system.** First-match-wins means pattern ordering in the text file determines which label appears. If patterns are reordered, the user-visible label changes. Making priority explicit (e.g., numbered priority in the pattern file) would make behavior more predictable. (`ext_contentfilter.go:82-96`)

- **Consider caching `nodeText` results.** The same node text is extracted every time a paragraph/heading/textblock is entered. If goldmark calls the renderer multiple times for the same node (entering + exiting), the text extraction work is duplicated. A `sync.Map` or node attribute cache could avoid this. (`ext_contentfilter.go:110,126,147`)

- **Use `regexp.MustCompile` for the embedded default patterns** or validate at test time that all patterns compile. Currently, invalid patterns are silently skipped in production. A test that calls `NewFilter(DefaultContentFilterPatterns)` and verifies the count matches expected would catch broken patterns before deployment. (`ext_contentfilter.go:65-68`)

- **Document the filter's scope explicitly.** Clarify in code comments which AST node types are filtered vs. preserved. Currently: paragraphs, headings, and text blocks are filtered; code blocks are preserved; inline code content is included in paragraph filtering (see Critical finding). Lists and blockquotes are filtered at the paragraph level within them. Tables are broken (see Critical finding). (`ext_contentfilter.go:102-106`)

## Questions for Author

- Is the inline code filtering behavior in test 20 intentional? If someone writes documentation with a scam URL in backticks as an example, should that paragraph be filtered?
- Is the table destruction behavior in test 22 acceptable for a PoC, or should it be fixed before merge? Tables with user-generated content in cells seem like a realistic scenario.
- What's the expected performance impact with 65+ regex patterns on a page with many paragraphs? Have you run any benchmarks?
- Is there a plan to make the pattern file configurable at runtime (e.g., admin can update patterns without rebuilding)?
- How does this interact with PR #5178 (antispam scoring)? Will both operate simultaneously, or will one replace the other?

## Verdict

REQUEST CHANGES — The inline code false-filtering (test 20) and table structure destruction (test 22) are correctness bugs that would affect legitimate content on gnoweb; both should be fixed before merge.
