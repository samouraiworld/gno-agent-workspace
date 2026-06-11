# PR #5794: feat(gnoweb): serve realm pages as markdown via Accept negotiation

URL: https://github.com/gnolang/gno/pull/5794
Author: gfanton | Base: master | Files: 6 | +305 -13
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 4ab275316 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5794 4ab275316`
Overview: [visual overview](https://samouraiworld.github.io/gno-agent-workspace/reviews/pr/5xxx/5794-markdown-accept-negotiation/overview.html) · [↗](../overview.html)

Round 2. The head advanced from `589d1970d` (round 1) to a clean `master` merge (`4ab275316`); the PR's own six files are byte-identical, only line numbers shifted, so all anchors are re-cut against the new head. CI is green except the gnoweb codeowner gate. Verdict and findings are unchanged from round 1.

**TL;DR:** gnoweb can now hand back a realm's raw `Render()` markdown instead of the full HTML page when the caller's `Accept` header asks for `text/markdown`. Browsers still get HTML; agent fetchers (Claude Code's `WebFetch`) get the ~4.7 KB markdown instead of ~58 KB of HTML, with no config.

**Verdict: APPROVE** — clean, well-tested, vet-clean, gated on an explicit `text/markdown` Accept so browsers are unaffected. Open items are all non-blocking: a deliberate q-preference shortcut worth a one-word confirm, and a defensive `nosniff` worth adding. The only real merge gate is gnoweb codeowner approval (CI bot requirement).

## Summary

gnoweb learns to return a realm's raw `Render()` markdown (~4.7 KB for the home realm) instead of the full HTML page (~58 KB) when the client's `Accept` header names `text/markdown`. The path is `Accept`-only and never matches `*/*` or `text/*`, so browsers keep getting HTML; agent fetchers get markdown with zero configuration. `Vary: Accept` is set on every GET so shared caches key the two representations separately. Scope is realm pages and static-markdown aliases only; source/help/directory/user views still fall back to HTML "for now".

```
GET /r/gnoland/home
  Accept: */*                     -> text/html     (goldmark + IndexLayout, ~58 KB)
  Accept: text/markdown, ...      -> text/markdown (raw Render() bytes, ~4.7 KB)
  Accept: text/markdown;q=0       -> text/html     (explicit refusal honored)
```

## Glossary
- `negotiatesMarkdown` — `Accept`-header test; true only when `text/markdown`/`text/x-markdown` is named with non-zero q.
- `MarkdownView` — view type carrying raw bytes; the handler serves it verbatim, bypassing `IndexLayout`.
- `fetchRealm` — shared helper returning either raw `Render()` bytes (ok) or an HTML fallback view + status (no-Render / not-found / error).

## Fix

`ServeHTTP` now sets `Vary: Accept` on all GETs plus a default `text/html` Content-Type ([`handler_http.go:121-124`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/handler_http.go#L121-L124) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L121-L124)). `Get` computes `wantMarkdown` once and, when the resulting body view is a `MarkdownViewType`, writes the bytes verbatim with `text/markdown; charset=utf-8`, skipping the layout and the goldmark step ([`handler_http.go:200-213`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/handler_http.go#L200-L213) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L200-L213)). The realm leaf was split into `GetRealmView` (HTML) and `GetMarkdownRealmView` (raw), sharing `fetchRealm` so both honor the same no-Render / not-found / error fallbacks ([`handler_http.go:368-427`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/handler_http.go#L368-L427) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L368-L427)). The markdown short-circuit sits after the fetch error-switch, so a realm with no `Render()` correctly falls back to the HTML directory view even under `Accept: text/markdown` (locked by a dedicated test).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits

- [`negotiate.go:15-32`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/negotiate.go#L15-L32) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate.go#L15-L32) — markdown wins whenever it appears with q>0, even when another type has a strictly higher q. `text/html;q=0.9, text/markdown;q=0.8` returns markdown ([`negotiate_test.go:31`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/negotiate_test.go#L31) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate_test.go#L31) locks this in). This is a deliberate shortcut, not strict RFC 9110 §12.5.1 preference ordering. Real-world impact is near-zero (browsers never send `text/markdown`; Claude WebFetch lists it first), so fine to keep — just confirm it's a choice, not an oversight.
- [`negotiate.go:16`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/negotiate.go#L16) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate.go#L16) — `strings.Split(accept, ",")` splits inside quoted parameter values too, so `text/markdown;x="a,b"` would mis-parse the fragments. Such a value is not realistic in an `Accept` header and `mime.ParseMediaType` rejects the broken halves (`continue`), so behavior degrades safely. Not worth code; noting for completeness.

## Suggestions

- [`handler_http.go:207`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/handler_http.go#L207) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http.go#L207) — set `X-Content-Type-Options: nosniff` on the raw markdown response.
  <details><summary>details</summary>

  This path serves attacker-controllable realm output (`Render()` bytes) verbatim, unlike the HTML path where goldmark safe-mode sanitizes. Practical XSS risk is low: the path is reached only via an explicit `Accept: text/markdown`, which browsers don't send on top-level navigation, and a declared `text/markdown` type is not in the browser HTML-sniffing set. gnoweb sets no `nosniff`/CSP anywhere today (grep: zero hits across the package at this head), so this is pre-existing and not introduced here. Adding `nosniff` on this one write is a cheap belt-and-suspenders for the new raw-bytes surface. Fix: `w.Header().Set("X-Content-Type-Options", "nosniff")` next to the markdown Content-Type set.

  Confirmed behaviorally: an httptest request with `Accept: text/markdown` returns the markdown body with no `X-Content-Type-Options` header at this head.
  </details>

## Open questions
- Markdown negotiation stops at realm pages and static aliases ("for now"). Of the remaining views, only `$help` (function docs) would plausibly benefit agents; source is `text/plain` territory, directory/user views are navigation chrome. Not posted — low gain, scoping is the author's call; revisit if agents start fetching `$help`.

## Missing Tests
None blocking. Coverage is thorough: [`negotiate_test.go`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/negotiate_test.go) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/negotiate_test.go) tables the header rules (aliases, q-values, `*/*`/`text/*` wildcards, `q=0` refusal, malformed q, the real WebFetch string); [`handler_http_test.go`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/handler_http_test.go) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/handler_http_test.go) exercises Content-Type, `Vary`, markdown-vs-HTML bodies, the static-alias path, and the no-Render HTML fallback; [`view_markdown_test.go`](https://github.com/gnolang/gno/blob/4ab275316/gno.land/pkg/gnoweb/components/view_markdown_test.go) · [↗](../../../../../.worktrees/gno-review-5794/gno.land/pkg/gnoweb/components/view_markdown_test.go) asserts verbatim render. All pass at this head; `go vet ./gno.land/pkg/gnoweb/...` is clean.
