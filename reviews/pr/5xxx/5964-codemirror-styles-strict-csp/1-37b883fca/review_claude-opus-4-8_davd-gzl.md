# PR [#5964](https://github.com/gnolang/gno/pull/5964): fix(gnoweb): allow CodeMirror styles under strict CSP

URL: https://github.com/gnolang/gno/pull/5964
Author: jefft0 | Base: playground2 | Files: 12 | +240 -13
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 37b883fca (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5964 37b883fca`

**TL;DR:** The code editor on gnoweb's run and playground pages renders unstyled when gnoweb runs with its strict security headers, because the browser blocks the stylesheet CodeMirror creates at runtime. This PR gives every response a fresh random token, lets the policy accept a stylesheet carrying that token, and hands the token to CodeMirror through a `<meta>` tag.

**Verdict: REQUEST CHANGES** — the fix works end to end, but the rewritten header is now built by formatting the same string twice, which corrupts `connect-src` for any `-remote` containing `%`; the handler wire that makes the whole feature work has no test (1 Warning, 1 Missing test).

## Summary

Under `style-src 'self'` the browser refuses the `<style>` element `style-mod` appends for CodeMirror, so the editor loses its gutter, cursor, and selection styling. The PR keeps the policy strict and adds a per-response nonce: [`SecureHeadersMiddleware`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main.go#L379-L386) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/cmd/gnoweb/main.go#L379-L386) generates 16 random bytes per request, puts `'nonce-<value>'` in `style-src`, and stores the value in the request context; [`HTTPHandler.Get`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/handler_http.go#L234) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/handler_http.go#L234) reads it back into `HeadData`, [`head.html`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/components/layouts/head.html#L10) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/components/layouts/head.html#L10) emits it as `<meta name="csp-nonce">`, and [`code-editor.ts`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/frontend/js/code-editor.ts#L72-L74) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/frontend/js/code-editor.ts#L72-L74) feeds it to CodeMirror's `EditorView.cspNonce` facet. Non-strict mode emits neither the header nor the meta tag, so nothing changes there.

## Fix

The CSP string used to be a single constant built once at middleware construction. It is now a template built once with `%%[1]s` escaped in place of the nonce, then formatted again per request with the nonce as its only argument, at [`main.go:358-362`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main.go#L358-L362) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/cmd/gnoweb/main.go#L358-L362) and [`main.go:386`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main.go#L386) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/cmd/gnoweb/main.go#L386). The load-bearing constraint is that the same value must appear in the header and in the HTML of one response, which is why the middleware rewrites the request context rather than letting the handler mint its own nonce.

## Critical (must fix)

None.

## Warnings (should fix)

- **[operator config silently corrupts a security header]** [`gno.land/cmd/gnoweb/main.go:358-362`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main.go#L358-L362) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/cmd/gnoweb/main.go#L358-L362) — a `-remote` value containing `%` reaches `connect-src` corrupted, because the header string now runs through `fmt.Sprintf` twice.
  <details><summary>details</summary>

  `remote` is interpolated into the template by the first `fmt.Sprintf`, then the result is passed to a second `fmt.Sprintf` at [`main.go:386`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main.go#L386) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/cmd/gnoweb/main.go#L386), which re-reads any `%` the operator supplied as a format verb. Because `%[1]s` sits earlier in the string and consumes argument 1, every later verb looks for a second argument that is not there. Booting gnoweb at 37b883fca with `-remote 'http://[fe80::1%25eth0]:26657'` emits `connect-src 'self' http://[fe80::1%!e(MISSING)th0]:26657/abci_query`; the same binary built at c355059a1 emits `http://[fe80::1%25eth0]:26657/abci_query`. The browser then blocks the `abci_query` requests the run and playground pages issue. Percent-encoded path segments and IPv6 zone identifiers both produce this. [repro](comment_claude-opus-4-8.md)

  Fix: assemble the header by concatenating a fixed prefix, the nonce, and a fixed suffix so the operator-supplied string is never re-parsed as a format string.
  </details>

## Nits

None.

## Missing Tests

- **[the feature can be deleted without a red test]** [`gno.land/pkg/gnoweb/handler_http.go:234`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/handler_http.go#L234) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/handler_http.go#L234) — no test asserts the nonce actually reaches the rendered page.
  <details><summary>details</summary>

  The two new tests cover the ends but not the seam. [`TestIndexLayout_CSPNonce`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/components/layout_test.go#L422) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/components/layout_test.go#L422) sets `HeadData.CSPNonce` by hand, and [`TestSecureHeadersMiddlewareNonceMatchesContext`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main_test.go#L131) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/cmd/gnoweb/main_test.go#L131) stops at the request context. Deleting the single assignment that joins them leaves `go test ./gno.land/pkg/gnoweb/... ./gno.land/cmd/gnoweb/...` fully green while the editor loses its styles again, which is exactly the failure this PR exists to fix.

  Fix: add a handler-level test that puts a nonce in the request context and asserts the `<meta name="csp-nonce">` tag in the response body, dropping it when the context carries none. Ready to add: [`tests/handler_csp_nonce_test.go`](tests/handler_csp_nonce_test.go); it passes at 37b883fca and fails once the assignment is removed.
  </details>

## Suggestions

None.

## Verified

- The editor renders under the strict policy, and the fix is what makes it do so. Booted gnodev as an RPC node plus `gno.land/cmd/gnoweb` in strict mode from this worktree, then loaded `/r/demo/counter$run` and `/_/play` in headless Chromium: 0 CSP violations on both pages at 37b883fca, 1 on both at c355059a1 (`"Applying inline style violates the following Content Security Policy directive 'style-src 'self''… The action has been blocked."`, sourced to `controller-run.js`). Same number of `cm-*` elements in the DOM on both sides, so only the stylesheet differed.
- Base64 standard encoding survives the round trip. [`NewCSPNonce`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/csp.go#L16-L21) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/csp.go#L16-L21) can emit `+` and `/`, and `html/template` writes `+` into the meta attribute as `&#43;`. Across six loads of `/_/play`, three nonces containing `+` and one containing `/` all produced 0 violations, so the browser's attribute decoding hands CodeMirror the value the header carries.
- The nonce is unreachable without the handler assignment. Removing [`handler_http.go:234`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/handler_http.go#L234) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/handler_http.go#L234) and re-running `./gno.land/pkg/gnoweb/... ./gno.land/cmd/gnoweb/...` returned `ok` for all eight packages.
- The header regression is authored by this PR. `-remote 'http://[fe80::1%25eth0]:26657'` yields `connect-src 'self' http://[fe80::1%!e(MISSING)th0]:26657/abci_query` at 37b883fca and `http://[fe80::1%25eth0]:26657/abci_query` at c355059a1.
- Non-strict mode is untouched: `-no-strict` emits no `Content-Security-Policy` header and no `csp-nonce` meta tag.
- Green at 37b883fca: `gno.land/cmd/gnoweb`, `gno.land/pkg/gnoweb`, `components`, `feature/playground`, `feature/run`, `feature/state`, `markdown`, `weburl`.

## Open questions

- The meta tag ships on every page, not just the two that host an editor. Nothing exploitable follows, since reading it already requires script execution that `script-src` forbids, so it is not worth a comment.
- State pages set [`Cache-Control: public, max-age=1`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/feature/state/helpers.go#L263) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/feature/state/helpers.go#L263) and now carry a nonce in both header and body. A shared cache stores the pair together, so they stay consistent; the only effect is a one-second reuse window on a `style-src`-only nonce. Not posted, no change needed.
- Dark mode was not exercised live, but the bundled `mountStyles()` re-reads `EditorView.cspNonce` on every style-module change, and [the facet is set outside the theme compartment](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/frontend/js/code-editor.ts#L72-L74) · [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/frontend/js/code-editor.ts#L72-L74), so swapping in `one-dark` re-mounts with the same nonce. Nothing to ask the author for.
- The invariant catalog walk is skipped: the PR changes Go, TypeScript, and HTML in gnoweb and touches no `.gno` code.
