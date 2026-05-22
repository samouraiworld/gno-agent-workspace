# PR #5434: chore(boards2): escape thread titles

**URL:** https://github.com/gnolang/gno/pull/5434
**Author:** jeronimoalbi | **Base:** master | **Files:** 5 | **+101 -1**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR addresses two rendering/security concerns in the boards2 realm:

1. **Thread title escaping** (`render_post.gno:31`): Wraps thread titles with `md.EscapeText()` before rendering them as Markdown H2 headings in the thread detail view. This prevents Markdown injection via titles (e.g., a title like `[Foo](https://evil.com)` would render as a clickable link without this fix). The escaping covers `*`, `_`, `[`, `]`, `(`, `)`, `~`, `>`, `|`, `-`, `+`, `.`, `!`, and backticks.

2. **GFM form blocking in replies** (`public.gno:784`): Adds a check in `assertReplyBodyIsValid` to reject reply bodies containing `gno-form`, preventing users from embedding Gno-Flavored Markdown forms in comments/replies.

Three new filetests cover the changes:
- `z_ui_thread_07_filetest.gno` — verifies title escaping in the thread detail view
- `z_create_reply_16_filetest.gno` — verifies `gno-form` rejection in `CreateReply`
- `z_edit_reply_08_filetest.gno` — verifies `gno-form` rejection in `EditReply`

### Changes since last review (commit `6357cdbc3`)

Only one change to the PR's own code: `strings.Index(body, "gno-form") != -1` was replaced with `strings.Contains(body, "gno-form")`, addressing the nit from both the previous review and notJoon's inline review comment.

### Correction from previous review

The previous review incorrectly stated that `md.EscapeText` does not escape `>`. It does — `>` is escaped to `\>` (`md.gno:256`). This means a title like `<gno-form>` would be rendered as `<gno-form\>`, which most Markdown parsers will NOT interpret as an HTML/GFM tag since the closing angle bracket is escaped. The previous concern about GFM form tags surviving title escaping was overstated. The remaining gap is that `<` is not escaped, but since `>` is, tag injection via titles is effectively neutralized for well-formed tags.

## Test Results
- **Existing tests:** PASS (all 143 filetests in boards2/v1 and all 30 tests in boards2/v1/hub pass)
- **CI:** All checks pass
- **Edge-case tests:** skipped

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `public.gno:784` — The `gno-form` check uses a naive substring match (`strings.Contains(body, "gno-form")`). This blocks any reply containing the literal text "gno-form" anywhere, even in innocuous contexts like `"I was reading about gno-form elements"` or inside code blocks. A more targeted check (e.g., matching `<gno-form` with the angle bracket) would reduce false positives while still preventing injection.

- [ ] `public.gno:784` — The `gno-form` check is only applied to replies via `assertReplyBodyIsValid`, but thread titles and thread/repost bodies have no such protection. A user can embed `<gno-form>` tags in thread bodies (via `CreateThread`/`EditThread`) or repost bodies (via `CreateRepost`) without restriction. While thread titles get `md.EscapeText()` on render (which escapes `>`), thread bodies and repost bodies are rendered raw. This means the `gno-form` blocking is incomplete for non-title content in threads.

- [ ] `render_thread.gno:61` — Thread titles in the **board listing view** (`renderThreadSummary`) are rendered via `md.H6(md.Link(summary, postURI))` without any `md.EscapeText()` call. The title is passed through `summaryOf()` (truncation) then used as the text of a Markdown link. While Markdown link text parsing may handle some special characters differently, Markdown-injected titles could still render unexpectedly in the board listing. The PR only escapes the title in the thread detail view (`renderPost`), not in the board listing, edit form (`render_thread.gno:147,161`), repost form (`render_thread.gno:209,229`), flag form (`render_post.gno:327`), or reply form (`render_post.gno:508`) views where the title is also displayed.

## Nits

- [ ] `public.gno:784` — The `strings.Contains` improvement was applied (fixing the previous nit). No new nits.

## Missing Tests

- [ ] No test for board listing rendering with Markdown in thread titles — the `z_ui_thread_07` test only verifies the thread detail view, not the board listing view where titles are unescaped.
- [ ] No test for `CreateRepost` with `gno-form` in the body — repost bodies are completely unvalidated.
- [ ] No test for `CreateThread` with `gno-form` in the thread body — thread bodies are unvalidated for `gno-form`.

## Suggestions

- Consider applying `md.EscapeText()` consistently to thread titles in all rendering contexts (board listing in `render_thread.gno:50-61`, edit form `render_thread.gno:147,161`, repost form `render_thread.gno:209,229`), not just in `renderPost`. This would prevent Markdown injection across all views.
- Consider blocking `gno-form` in thread bodies and repost bodies as well, not just replies. The current asymmetry (replies blocked, threads not) is inconsistent and could be exploited.
- Consider using a more targeted substring check like `strings.Contains(body, "<gno-form")` to reduce false positives from legitimate text mentioning "gno-form".

## Questions for Author

- Is the asymmetry between reply validation (blocks `gno-form`) and thread/repost body validation (no `gno-form` check) intentional? If forms are acceptable in thread bodies but not replies, what is the reasoning?
- Should title escaping be applied in all rendering contexts (board listing, edit form, repost form) or is the thread detail view sufficient for now?

## Verdict

**APPROVE** — The `strings.Contains` fix addresses the only code-level nit from the last review. The core changes are correct: title escaping in the thread detail view works as intended, and `gno-form` blocking in replies is functional. The remaining warnings about incomplete coverage (title escaping in other views, `gno-form` in thread/repost bodies) are valid but are scope expansions that can be addressed in follow-up PRs. The PR does what it says on the tin.
