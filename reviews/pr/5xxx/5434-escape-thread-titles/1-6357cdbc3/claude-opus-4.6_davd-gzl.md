# PR #5434: chore(boards2): escape thread titles

**URL:** https://github.com/gnolang/gno/pull/5434
**Author:** jeronimoalbi | **Base:** master | **Files:** 5 | **+101 -1**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR addresses two security/rendering concerns in the boards2 realm:

1. **Thread title escaping** (`render_post.gno:31`): Wraps thread titles with `md.EscapeText()` before rendering them as Markdown H2 headings, preventing Markdown injection via titles (e.g., a title like `[Foo](https://evil.com)` would render as a clickable link without this fix).

2. **GFM form blocking in replies** (`public.gno:762-765`): Adds a check in `assertReplyBodyIsValid` to reject reply bodies containing `gno-form`, preventing users from embedding Gno-Flavored Markdown forms in comments/replies. This is a basic substring check using `strings.Index`.

Three new filetests cover the changes: one for the title escaping render output, and two for the `gno-form` rejection in `CreateReply` and `EditReply`.

## Test Results
- **Existing tests:** PASS (all filetests in boards2/v1 and boards2/v1/hub pass, 0 failures)
- **CI:** All checks pass
- **Edge-case tests:** skipped

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `public.gno:763` — The `gno-form` check uses a naive substring match (`strings.Index(body, "gno-form") != -1`). This blocks any reply containing the literal text "gno-form" anywhere, even in innocuous contexts like `"I was reading about gno-form elements"` or code blocks. A more targeted check (e.g., matching `<gno-form` with the angle bracket) would reduce false positives while still preventing injection.

- [ ] `public.gno:763` — The `gno-form` check is only applied to replies via `assertReplyBodyIsValid`, but thread titles and thread/repost bodies have **no such protection**. A user can embed `<gno-form>` tags in thread titles (via `CreateThread`/`EditThread`) or repost bodies (via `CreateRepost`) without restriction. While thread titles get `md.EscapeText()` on render in one place, `md.EscapeText` does NOT escape `<` (angle brackets), so `<gno-form>` tags would survive escaping. This means the `gno-form` blocking is incomplete.

- [ ] `render_post.gno:31` — `md.EscapeText()` escapes Markdown special characters but **does not escape `<`** (HTML/GFM angle brackets). This means while Markdown-based injection (links, bold, etc.) is neutralized in thread titles, HTML-like GFM tags like `<gno-form>` in thread titles are NOT neutralized by this escaping. The title escaping and the `gno-form` blocking are solving related problems but leave a gap between them.

- [ ] `render_thread.gno:46-61` — Thread titles in the **board listing view** (`renderThreadSummary`) are rendered via `md.H6(md.Link(summary, postURI))` without any `md.EscapeText()` call. This means Markdown-injected titles (e.g., `[evil](url)`) would render unescaped in board listing summaries. The PR only escapes the title in the thread detail view (`renderPost`), not in board listing, edit form, repost form, flag form, or reply form views where the title is also rendered.

## Nits

- [ ] `public.gno:763` — `strings.Index(body, "gno-form") != -1` is idiomatic Go but `strings.Contains(body, "gno-form")` expresses intent more clearly. However, GnoVM may not support `strings.Contains` — if it does, prefer it.

## Missing Tests

- [ ] No test for a thread title containing `<gno-form>` tags — given that thread titles have no `gno-form` validation and `md.EscapeText` does not escape `<`, a test demonstrating that `<gno-form>` in a thread title renders unblocked would document the gap.
- [ ] No test for board listing rendering with Markdown in thread titles — the `z_ui_thread_07` test only verifies the thread detail view, not the board listing view where titles are unescaped.
- [ ] No test for `CreateRepost` with `gno-form` in the body — repost bodies are completely unvalidated.

## Suggestions

- Consider adding `<` (and `>`) to the `md.EscapeText` character set in the `gno.land/p/moul/md` package to fully neutralize GFM tag injection. This would be a separate PR affecting the `md` package (`examples/gno.land/p/moul/md/md.gno:247-265`).
- Consider applying `md.EscapeText()` consistently to thread titles in all rendering contexts (board listing in `render_thread.gno`, edit form, repost form, flag form, reply form), not just in `renderPost`. This would prevent Markdown injection in all views.
- Consider blocking `gno-form` in thread titles and thread bodies as well, not just replies. The current asymmetry (replies blocked, threads not) is inconsistent and could be exploited.
- Consider validating repost bodies similarly to reply bodies — currently `CreateRepost` performs no body validation at all (`public.gno:295-341`).

## Questions for Author

- Is the asymmetry between reply validation (blocks `gno-form`) and thread/repost validation (no `gno-form` check) intentional? If forms are acceptable in thread bodies but not replies, what is the reasoning?
- Should `md.EscapeText` be updated to also escape `<` and `>` to fully prevent GFM tag injection? Or is there a separate mechanism planned for that?
- Is blocking the literal substring `gno-form` (without requiring the `<` prefix) intentional to be extra cautious, or an oversight that could cause false positives?

## Verdict

**NEEDS DISCUSSION** — The changes are correct for what they do, but the `gno-form` blocking is incomplete (thread titles and bodies, repost bodies are unprotected) and `md.EscapeText` does not escape `<`, leaving a gap where GFM form tags can still be injected via thread titles. The PR should either expand protection consistently or document why the asymmetry is acceptable.
