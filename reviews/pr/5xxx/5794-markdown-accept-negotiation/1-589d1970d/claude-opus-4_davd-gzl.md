# PR #5794: feat(gnoweb): serve realm pages as markdown via Accept negotiation

URL: https://github.com/gnolang/gno/pull/5794
Author: gfanton | Base: master | Files: 6 | +305 -13
Reviewed by: davd-gzl | Model: claude-opus-4 | Commit: `589d1970d` (stale — +31 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5794 589d1970d`

**Verdict: APPROVE** — clean, well-tested, vet-clean, behavior gated correctly on an explicit `text/markdown` Accept. Open items are non-blocking: no ADR (repo policy for AI-assisted feature PRs), a deliberate q-preference shortcut worth confirming, and a defensive `nosniff` worth adding. Merge still needs gnoweb codeowner approval (the only real CI gate).

## Summary

gnoweb learns to return a realm's raw `Render()` markdown (about 4.7 KB for the home realm) instead of the full HTML page (about 58 KB) when the client's `Accept` header names `text/markdown`. The path is `Accept`-only and never matches `*/*` or `text/*`, so browsers keep getting HTML; agent fetchers (Claude Code's `WebFetch` sends `Accept: text/markdown, text/html, */*`) get markdown with zero configuration. `Vary: Accept` is set on every GET so shared caches key the two representations separately. Scope is realm pages and static-markdown aliases only; source/help/directory/user views still fall back to HTML.

```
GET /r/gnoland/home
  Accept: */*                     -> text/html   (goldmark + IndexLayout, ~58 KB)
  Accept: text/markdown, ...      -> text/markdown (raw Render() bytes, ~4.7 KB)
  Accept: text/markdown;q=0       -> text/html   (explicit refusal honored)
```

## Glossary
- `negotiatesMarkdown` — `Accept`-header test; true only when `text/markdown`/`text/x-markdown` is named with non-zero q.
- `MarkdownView` — view type carrying raw bytes; the handler serves it verbatim, bypassing `IndexLayout`.
- `fetchRealm` — shared helper returning either raw `Render()` bytes (ok) or an HTML fallback view + status (no-Render / not-found / error).

## Fix

`ServeHTTP` now sets `Vary: Accept` on all GETs and a default `text/html` Content-Type ([`handler_http.go:103-106`](https://github.com/gnolang/gno/blob/589d1970d/gno.land/pkg/gnoweb/handler_http.go#L103-L106) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L103-L106)). `Get` computes `wantMarkdown` once and, when the resulting body view is a `MarkdownViewType`, writes the bytes verbatim with `text/markdown; charset=utf-8`, skipping the layout and the goldmark step ([`handler_http.go:181-194`](https://github.com/gnolang/gno/blob/589d1970d/gno.land/pkg/gnoweb/handler_http.go#L181-L194) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L181-L194)). The realm leaf was split into `GetRealmView` (HTML) and `GetMarkdownRealmView` (raw), sharing `fetchRealm` so both honor the same no-Render / not-found / error fallbacks ([`handler_http.go:348-407`](https://github.com/gnolang/gno/blob/589d1970d/gno.land/pkg/gnoweb/handler_http.go#L348-L407) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L348-L407)). The markdown short-circuit sits after the fetch error-switch, so a realm with no `Render()` correctly falls back to the HTML directory view even under `Accept: text/markdown` (locked by a dedicated test).

## Critical (must fix)
None.

## Warnings (should fix)

- **[repo policy: AI-assisted feature PR has no ADR]** [`gno/AGENTS.md:85`](https://github.com/gnolang/gno/blob/589d1970d/AGENTS.md#L85) · [↗](../../../../../.worktrees/gno-review-5794/AGENTS.md#L85) — feature is AI-assisted (commits carry `Co-Authored-By: Claude`) and non-trivial; no ADR in the diff.
  <details><summary>details</summary>

  `AGENTS.md` states "Every non-trivial AI-assisted PR must include an ADR" and lists only trivial bug fixes / formatting / simple tests / docs as exempt. This PR adds a content-negotiation feature with new public surface (`MarkdownView`, `MarkdownViewType`, `negotiatesMarkdown`) and a handler refactor, so it falls under the rule. Not a code defect; flag for the codeowner who has to approve anyway. Fix: add `gno.land/adr/pr5794_markdown_negotiation.md` (context: agents want raw markdown; decision: Accept-only negotiation, realm + static-md scope; alternatives: query param, separate endpoint, strict q-ordering; consequences: `Vary: Accept`, follow-up for source/help/user views).
  </details>

## Nits

- [`negotiate.go:15-32`](https://github.com/gnolang/gno/blob/589d1970d/gno.land/pkg/gnoweb/negotiate.go#L15-L32) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate.go#L15-L32) — markdown wins whenever it appears with q>0, even when another type has a strictly higher q. `text/html;q=0.9, text/markdown;q=0.8` returns markdown ([`negotiate_test.go:31`](https://github.com/gnolang/gno/blob/589d1970d/gno.land/pkg/gnoweb/negotiate_test.go#L31) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate_test.go#L31) locks this in). This is a deliberate shortcut, not strict RFC 9110 §12.5.1 preference ordering. Real-world impact is near-zero (browsers never send `text/markdown`; Claude WebFetch lists it first), so this is fine to keep — just confirm it's intended rather than an oversight. See the Questions section.
- [`negotiate.go:16`](https://github.com/gnolang/gno/blob/589d1970d/gno.land/pkg/gnoweb/negotiate.go#L16) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate.go#L16) — `strings.Split(accept, ",")` splits inside quoted parameter values too, so `text/markdown;x="a,b"` would mis-parse the fragments. Such a value is not realistic in an `Accept` header and `mime.ParseMediaType` would just reject the broken halves (`continue`), so behavior degrades safely. Not worth code, noting for completeness.

## Suggestions

- [`handler_http.go:187-189`](https://github.com/gnolang/gno/blob/589d1970d/gno.land/pkg/gnoweb/handler_http.go#L187-L189) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L187-L189) — set `X-Content-Type-Options: nosniff` on the raw markdown response.
  <details><summary>details</summary>

  This path serves attacker-controllable realm output (`Render()` bytes) verbatim, unlike the HTML path where goldmark safe-mode sanitizes. Practical XSS risk is low: the path is reached only via an explicit `Accept: text/markdown`, which browsers don't send on top-level navigation, and a declared `text/markdown` type is not in the browser HTML-sniffing set. gnoweb sets no `nosniff`/CSP anywhere today (grep: zero hits across the package; the `mux` in `app.go` adds no security middleware), so this is pre-existing and not introduced here. Adding `nosniff` on this one write is a cheap belt-and-suspenders for the new raw-bytes surface. Fix: `w.Header().Set("X-Content-Type-Options", "nosniff")` next to the markdown Content-Type set.
  </details>

## Missing Tests
None blocking. Coverage is thorough: `negotiate_test.go` tables the header rules (aliases, q-values, `*/*`/`text/*` wildcards, `q=0` refusal, malformed q, the real WebFetch string); `handler_http_test.go` exercises Content-Type, `Vary`, markdown-vs-HTML bodies, the static-alias path, and the no-Render HTML fallback; `view_markdown_test.go` asserts verbatim render.

## Questions for Author
- The q-preference shortcut (nit above): is serving markdown even when the client ranks HTML strictly higher (`text/html;q=0.9, text/markdown;q=0.8`) intended, or should the highest-q acceptable type win? Current behavior is fine for the agent use case; just confirming it's a choice, not an oversight.
- Source/help/directory/user views fall back to HTML "for now" — is the follow-up to extend markdown there tracked anywhere?
