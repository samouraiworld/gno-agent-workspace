# PR #5615: fix(gnoweb): copy origin

URL: https://github.com/gnolang/gno/pull/5615
Author: alexiscolin | Base: master | Files: 7 | +218 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — fix is correct, tests cover the cross-deployment matrix, and the client-side fallback makes the `Origin`-empty path harmless. Only concern is unconditional trust of `X-Forwarded-{Proto,Host}`, which is low-severity given the rendered URL is HTML-escaped and the realistic exposure path is direct-to-internet gnodev/gnoweb deployments.

## Summary

Action page "Link" button regressed in #4964: it copied a path-only URL (e.g. `/r/foo$help&func=Bar`) so the clipboard value was unusable outside the current site. Fix builds an absolute origin server-side from `r.TLS`, `r.Host`, and (when present) `X-Forwarded-{Proto,Host}`, stores it in `weburl.GnoURL.Origin`, threads it into `HelpData.Origin`, and prefixes it in `buildHelpURL`. The same template field also populates the `<form action="...">` of the Execute button, so both the copy and the form submission become absolute together. A defense-in-depth client-side fallback in [`controller-copy.ts:82-85`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/frontend/js/controller-copy.ts#L82-L85) prepends `window.location.origin` when the value still starts with `/`, so empty-`Origin` paths (e.g. unit tests, weird embeddings) keep working.

## Glossary

- `GnoURL.Origin` — runtime-populated scheme+host, not parsed from the URL string ([`weburl/url.go:26`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/weburl/url.go#L26))
- `requestOrigin` — derives Origin from request, honoring X-Forwarded-* ([`handler_http.go:751-774`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/handler_http.go#L751-L774))
- `buildHelpURL` — template func emitting `Origin + pkgPath + "$help&..."` ([`view_action.go:70-86`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/components/view_action.go#L70-L86))

## Fix

Before: `buildHelpURL` produced `/r/foo$help&func=Bar` (path-only) because `pkgPath` came from `path.Join(Domain, gnourl.Path)` with no scheme. After: `buildHelpURL` prepends `data.Origin`, which is set once per request via `requestOrigin(r)` in [`handler_http.go:257`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/handler_http.go#L257). The load-bearing detail is that `requestOrigin` returns `""` when `r.Host == ""` (test environments), and the client-side fallback at [`controller-copy.ts:82-85`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/frontend/js/controller-copy.ts#L82-L85) catches the resulting path-relative string and absolutizes it from `window.location.origin`.

## Critical (must fix)

None.

## Warnings (should fix)

- **[X-Forwarded-* trusted without proxy allowlist]** [`handler_http.go:751-774`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/handler_http.go#L751-L774) — `X-Forwarded-Host` overrides `r.Host` unconditionally, and the same value drives both the copied clipboard URL **and** the `<form action="...">` for Execute.
  <details><summary>details</summary>

  When gnoweb sits behind a trusted reverse proxy this is correct: the proxy strips inbound `X-Forwarded-*` and reinjects them. But for direct-to-internet deployments (gnodev, standalone `gnoweb`, single-binary prod with no proxy in front), any client can send `X-Forwarded-Host: evil.com` and the rendered page will both:
  - render `data-copy-text-value="https://evil.com/r/foo$help&func=Bar"` — copy-to-clipboard now hands the victim an evil URL, and
  - render `<form action="https://evil.com/...">` ([`action.html:104`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/components/views/action.html#L104)) — clicking Execute submits to `evil.com`.

  Both values are HTML-escaped by `html/template` so this is not XSS, and a browser-driven attack would require the attacker to control the victim's headers (not generally possible). The realistic exposure is a same-origin-but-misconfigured deployment where the attacker is the one looking at their own crafted request — so severity is low. Still, this is the kind of footgun that bites the next operator who exposes `gnoweb` directly.

  Fix: add an explicit `TrustForwardedHeaders bool` (or `TrustedProxies []netip.Prefix`) to `StaticMetadata`/`HTTPHandlerConfig`, default to `false`, and gate the `X-Forwarded-*` reads on it. Cleaner alternative: a single `BaseURL` override that, when set, returns unconditionally from `requestOrigin` and bypasses header inspection entirely — this also collapses the gnodev/prod/proxy/customDomain test matrix into one config knob.
  </details>

- **[form action also absolutized, not just the copy URL]** [`action.html:104`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/components/views/action.html#L104) — `buildHelpURL` feeds both `data-copy-text-value` and `<form action>`; the PR description only mentions the copy fix.
  <details><summary>details</summary>

  Functionally fine — browsers handle absolute and relative form actions identically — but worth flagging since the surface area of the change is broader than "fix the Link button." If a future operator wants to switch back to path-relative form submissions (e.g. to keep them on the backend hostname behind a proxy), they'll need a separate template variable. Mention this in the PR body so reviewers don't focus only on the clipboard path.
  </details>

## Nits

- [`weburl/url.go:26`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/weburl/url.go#L26) — `Origin` is the only field on `GnoURL` populated *after* `ParseFromURL` returns; all others come from parsing. A one-line comment ("set by HTTP layer post-parse; empty for non-HTTP callers") prevents future callers from assuming `Parse(s).Origin` is meaningful.
- [`view_action.go:34`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/components/view_action.go#L34) — same: note that empty `Origin` is the expected path-relative fallback handled by the client-side controller, not a bug.
- [`handler_origin_test.go:88`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/handler_origin_test.go#L88) — `tc := tc` loop-capture shim is unnecessary on `go 1.25.9` ([`go.mod:3`](../../../../../.worktrees/gno-review-5615/go.mod#L3)); same applies to [`handler_http_test.go:232`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/handler_http_test.go#L232) in this PR's new subtest. Cosmetic.
- [`handler_http.go:753-759`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/handler_http.go#L753-L759) — comma-split + `TrimSpace` correctly takes the leftmost entry (closest to client), but a one-line comment ("RFC 7239: leftmost is the original client") would save the next reader a lookup.

## Missing Tests

- **[buildHelpURL with empty Origin not unit-tested]** [`view_action.go:70-86`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/components/view_action.go#L70-L86) — the path-relative fallback is the load-bearing recovery path that makes empty-`Origin` non-fatal, but no test asserts the output of `buildHelpURL` when `data.Origin == ""`.
  <details><summary>details</summary>

  `TestRequestOrigin` covers the empty-Host case at the helper level, and `TestHTTPHandler_HelpURLOrigin` covers the populated-Origin path end-to-end, but there's no test that an empty `Origin` actually yields a `/r/...`-prefixed string that the client-side fallback can then catch. Worth adding one subtest with `req.Host = ""` (or an `httptest.NewRequest` variant that triggers it) asserting the rendered `data-copy-text-value` starts with `/`.
  </details>

- **[client-side fallback: absolute URL passthrough not covered]** [`controller-copy.ts:82-85`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/frontend/js/controller-copy.ts#L82-L85) — JS unit test missing for the `startsWith("/")` branch (path-relative gets absolutized) and the non-branch (already-absolute passes through unchanged, no `window.location.origin` double-prefix). The TS file has no test sibling under `frontend/js/`.

## Suggestions

- Replace `X-Forwarded-*` trust with a `BaseURL` config knob (see Warning above). Single source of truth, no header surface, and matches how most Go web stacks handle this.
- Consider lazy-evaluating `requestOrigin(r)` — it runs on every request at [`handler_http.go:257`](../../../../../.worktrees/gno-review-5615/gno.land/pkg/gnoweb/handler_http.go#L257) but only the help view actually consumes `gnourl.Origin`. Trivial cost, but the value of always populating is unclear.

## Questions for Author

- Is `public/js/controller-copy.js` regenerated by an existing build target (e.g. `make web` / `esbuild`)? If so, is there CI that fails when the compiled JS drifts from the TS source? The binary diff is opaque to review.
- For `gnodev` local-dev UX, is `http://127.0.0.1:<port>/r/foo$help&...` the intended clipboard value, or should the local mode skip absolutization and rely entirely on the client-side fallback?

## Prior Review

Round 1 ([`1-091b1ff79/claude-sonnet-4-6_davd-gzl.md`](../1-091b1ff79/claude-sonnet-4-6_davd-gzl.md), commit `091b1ff79`) reached the same verdict. The code in this round (`13e5bf528`) is unchanged versus round 1 — only master merges in between (no further gnoweb edits in `091b1ff79..13e5bf528`).
