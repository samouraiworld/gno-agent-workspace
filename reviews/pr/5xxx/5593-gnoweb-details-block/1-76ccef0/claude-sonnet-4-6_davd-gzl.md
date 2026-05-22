# PR #5593: feat(gnoweb): add :::details collapsible block

**URL:** https://github.com/gnolang/gno/pull/5593
**Author:** davd-gzl | **Base:** master | **Files:** 16 | **+577 -0**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR adds a new Goldmark extension `ExtDetails` to gnoweb that renders a `:::details` fenced-container block as a native HTML `<details>`/`<summary>` element — providing a neutral, plain collapsible block without the alert styling (coloured border, semantic icon, mandatory type) produced by the existing `> [!INFO]-` alert path.

**Syntax:**
```
:::details Summary text
arbitrary **markdown**
:::
```
Optional `[open]` flag (`:::details[open] Summary`) makes the block start expanded. A missing closing `:::` fence lets the block close at end-of-document (matching CommonMark fenced code behaviour). The summary supports full inline markdown (bold, links, etc.).

**What changed:**
- `gno.land/pkg/gnoweb/markdown/ext_details.go` (268 lines, new): block parser + renderer for `:::details`. Two AST node types (`DetailsBlock`, `DetailsSummary`) and a `DetailsHTMLRenderer`.
- `gno.land/pkg/gnoweb/markdown/ext.go` (3 lines): one-liner registration of `ExtDetails` in `GnoExtension.Extend`, between `ExtAlerts` and `ExtLinks`.
- `gno.land/pkg/gnoweb/frontend/css/06-blocks.css` (+54 lines): `.gno-details` styles scoped inside `.c-realm-view, .c-readme-view`. Uses CSS nesting, theme tokens, and the existing `#ico-arrow` SVG symbol.
- `gno.land/pkg/gnoweb/public/main.css`: rebuilt binary artifact (expected per repo convention).
- 10 golden `.txtar` fixtures in `gno.land/pkg/gnoweb/markdown/golden/ext_details/`: basic, open, no-summary, inline summary, rich content, unclosed, paragraph interruption, 1–3 space indent, 4-space rejected as code block, unrecognised name falls to paragraph.
- `examples/gno.land/r/docs/markdown/markdown.gno` (+38 lines): "Collapsible blocks" section in the live docs realm.
- `gno.land/adr/pr5593_gnoweb_details_block.md` (new): ADR covering context, decision, alternatives, consequences.

**Design decisions worth noting:**
- `DetailsSummary` extends `ast.BaseBlock` with `SetLines` set to the summary byte range. Goldmark treats it as a leaf block and runs its inline parser over the Lines, producing inline AST children (text, emphasis, link nodes). This is the same mechanism used by `ast.TextBlock` and produces correctly escaped, fully-featured inline rendering.
- The body `<div>` is opened in `detailsSummaryClose` (`</summary>\n<div>\n`) and closed in `detailsCloseTag` (`</div>\n</details>\n`). Goldmark's AST walk emits body blocks as siblings of `DetailsSummary` inside `DetailsBlock`, so they land between these two boundary writes. The structure is correct.
- Parser priority 799 matches `ExtAlerts`. No conflict because the two parsers trigger on different bytes (`:` vs `>`).

## Test Results

- **Existing tests:** PASS — `go test ./gno.land/pkg/gnoweb/markdown/...` passes all 10 new golden fixtures and all pre-existing extension fixtures.
- **Edge-case tests:** skipped (golden fixtures cover the critical paths; adversarial cases investigated analytically below)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gno.land/pkg/gnoweb/markdown/ext_details.go:137-140` — Indentation stripping only counts **space** characters (`line[pos] == ' '`), not tabs. Per CommonMark, a leading tab counts as a 4-column advance to the next tab stop. A line `\t:::details ...` therefore passes `pos==4` untripped (pos stays 0), falls to `parseOpenFence`, which then rejects it because `\t:::details` does not start with `:::details`. The net result is that `\t:::details` silently falls through to a paragraph rather than being treated as an indented code block. This is an edge case (tabs are rare in practice and the behaviour is safe), but it is inconsistent with CommonMark's tab handling and differs from how the rest of goldmark handles leading tabs.

- [ ] `gno.land/pkg/gnoweb/markdown/ext_details.go:105` — `:::details[open]Summary` (no space between `[open]` and the summary) is silently treated as a non-details paragraph because `rest[0]` (`'S'`) is neither a space nor a tab. This is undocumented. Users who type `:::details[open]Changelog` will see their text rendered verbatim as a paragraph without any error. The ADR and docs only show examples with a space, but the rejection of no-space syntax should be explicitly mentioned in the docs or the parser should at minimum fail visibly (e.g., render a comment `<!-- invalid details fence -->`).

## Nits

- [ ] `gno.land/pkg/gnoweb/markdown/ext_details.go:192-198` — The constant names `detailsOpenTag` / `detailsOpenTagExpand` are slightly confusing: "open" in `detailsOpenTag` means "opening HTML tag", while `open` in `Open bool` means the HTML `open` attribute. Renaming to `detailsStartTag` / `detailsStartTagOpen` (or similar) would remove the ambiguity.

- [ ] `gno.land/pkg/gnoweb/markdown/ext_details.go:250-268` — The renderer is registered at priority `0`, the same as goldmark's default renderer. This is consistent with `ExtAlerts` (also 0) and works because custom node kinds have no overlap with the default renderer. A short comment (`// Same priority as ExtAlerts; no conflict because KindDetailsBlock is a custom kind.`) would make the intent clear for future maintainers.

- [ ] `gno.land/adr/pr5593_gnoweb_details_block.md:35` — Rendered HTML in the ADR shows `<details class="gno-details" [open]>` literally with brackets, which is not valid HTML. The bracketed notation is a documentation shorthand, but since the ADR is a formal decision record it would be cleaner to use prose ("…with the `open` attribute when the `[open]` flag is present…") or show the two concrete forms.

## Missing Tests

- [ ] No golden fixture for **nested** `:::details` blocks (a `:::details` opening fence inside the body of another `:::details`). The implementation should handle this correctly (goldmark's block parser stack manages nesting), but it is the most likely real-world usage that could expose a latent bug and it should be tested explicitly.

- [ ] No golden fixture for the **empty-body** case: `:::details Summary\n:::` (opening fence immediately followed by closing fence). The code path produces valid HTML (`<details>…<summary>…</summary><div></div></details>`), but a fixture would lock the output in.

- [ ] No golden fixture for `:::details[open]Summary` (flag immediately followed by summary text, no space). The current behaviour (falls through to paragraph) is testable and should be documented as `invalid_no_space_flag.md.txtar` so the contract is explicit.

- [ ] No golden fixture for **consecutive** details blocks (two or more `:::details` blocks with no blank line between them). Tests that the `Continue → Close` → new `Open` transition is clean.

## Suggestions

- The `parseOpenFence` function could be simplified slightly: the `consumed` variable tracks the running offset but is only needed to compute `sStart`/`sEnd`. Consider making it a named return or a single expression — no correctness issue, purely readability.

- `gno.land/pkg/gnoweb/markdown/ext_details.go:234-236`: The default-summary guard (`if _, ok := n.FirstChild().(*DetailsSummary); !ok`) executes during the `entering=true` call of `renderDetailsBlock`. If a future refactor moves `DetailsSummary` creation to a post-parse transformer rather than Open(), this guard will silently break. A `// DetailsSummary is always created in Open() when a summary is present` comment would make the invariant explicit.

- The CSS `& > div` selector in `06-blocks.css:2582` targets the first `<div>` child of `.gno-details`, which is the body wrapper written by the renderer. If a future feature adds another `<div>` directly inside `.gno-details` (e.g., a toolbar), the padding selector will also apply to it. Using a dedicated class on the body wrapper (`<div class="gno-details-body">`) would be more future-proof, though this mirrors the alert extension's pattern.

## Questions for Author

- What is the intended behaviour when `:::details` appears inside a blockquote (`> :::details ...`)? The current parser only strips leading spaces, not `>` prefix, so the fence would not be recognised inside a blockquote. Is that by design?

- Should `:::details[OPEN]` (uppercase flag) be accepted? Currently it is not (only `[open]` matches). This is consistent with how HTML boolean attributes are case-sensitive in serialisation, but worth confirming.

- Is there a plan for additional `:::<name>` container types (the ADR mentions `:::columns`, `:::warning`)? If so, should `parseOpenFence` eventually be extracted into a shared `parseFenceHeader` helper to avoid duplicating the 1–3 space stripping, flag parsing, and boundary detection logic across extensions?

## Verdict

APPROVE — The implementation is correct, well-structured, and well-tested. The parser handles all standard CommonMark indentation rules, inline markdown in summaries, the `[open]` flag, unclosed fences, and the paragraph-interruption case. No new XSS vectors are introduced. The warnings are minor usability/documentation gaps that do not block merging.
