# PR #5612: feat(gnoweb): accept gno.land URLs in search bar

**URL:** https://github.com/gnolang/gno/pull/5612
**Author:** davd-gzl | **Base:** master | **Files:** 3 | **+103 -21**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR modifies the gnoweb search bar to recognise `gno.land/...` URLs (with or without a scheme) and rewrite them to the current origin, so that realm/package paths copied from chat, documentation, or the public gno.land address bar work locally without hand-editing the host.

**Changed files:**
- `gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts` — primary logic: replaces the old inline `if/try-catch` block with a new static `resolveTarget(input)` method that applies a regex strip followed by `URL.parse`.
- `gno.land/pkg/gnoweb/public/js/controller-searchbar.js` — rebuilt minified bundle (committed binary-equivalent).
- `gno.land/adr/prxxxx_gnoweb_search_gnoland_url.md` — ADR documenting context, decision, alternatives, and consequences.

**How it works:**

The new `resolveTarget` method applies a single regex `^(?:https?:\/\/)?gno\.land(?=\/|$|\?|#)` to strip a leading `gno.land` host (with or without scheme). The remainder is then resolved via `URL.parse(stripped, window.location.origin)`. Non-`gno.land` absolute URLs are untouched because the regex does not match them; relative paths (those that never contained `gno.land`) are also unaffected.

The old code prepended the origin directly for any non-`http(s)://` input using string concatenation. The new code delegates resolution entirely to `URL.parse`, which introduces both improvements and regressions relative to the old approach.

## Test Results

- **Existing tests (Go):** PASS — `go test -short -run TestRender ./gno.land/pkg/gnoweb/` passes in 0.285 s. No Go code was changed; the pass is expected.
- **Edge-case tests:** 0 written (no JS test framework is wired up; the project has no TypeScript/Jest test suite).

## Critical (must fix)

- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:29-33` — **Open redirect regression via protocol-relative URLs.** The old code prepended the origin for any non-`https?://` input, so `//attacker.com/evil` became `https://<origin>//attacker.com/evil` (path stays on the same host). The new code passes `//attacker.com/evil` unchanged to `URL.parse`, which treats `//host` as a valid authority-relative reference and resolves it to `https://attacker.com/evil` — a full cross-origin redirect. An attacker who can influence the search-bar value (e.g. via a crafted share link that auto-populates the field) can redirect the user off-origin. Fix: after stripping, validate that the resolved URL's `origin` matches `window.location.origin`, or at minimum that the stripped string does not start with `//`.

- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:29-33` — **`javascript:` / `data:` protocol regression.** The old code always prepended the current origin for non-`https?://` input, so `javascript:alert(1)` became `https://<origin>/javascript:alert(1)` (harmless path). The new code passes `javascript:alert(1)` unchanged to `URL.parse(stripped, origin)`, which resolves it as `javascript:alert(1)` (protocol `javascript:`). Setting `window.location.href = "javascript:..."` executes script in many browsers that lack a sufficiently strict CSP. gnoweb has no `Content-Security-Policy` header. Fix: after resolution, reject (or at least log and abort) any URL whose protocol is not `https:` or `http:`.

## Warnings (should fix)

- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:33` — **Silent failure on invalid input is a UX regression.** The old code showed `console.error("Invalid URL…")` when `new URL()` threw. The new code falls back with `?? ""` and assigns `""` to `window.location.href`, which silently reloads the current page without any feedback. Users entering gibberish will see no error message. Fix: check the return value of `URL.parse` and show an error when it returns `null`.

- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:29-33` — **`URL.parse` browser compatibility.** `URL.parse()` (static, non-throwing) was introduced in Chrome 126 (May 2024), Firefox 126 (May 2024), and Safari 18 (September 2024). The project's `browserslist` is `"defaults"`, which includes browsers with ≥ 0.5 % global usage. Some users still on older Safari (e.g. iOS < 18) will hit `TypeError: URL.parse is not a function` and the search bar will break entirely. The old code used `new URL()` (supported everywhere for years). Fix: add a `typeof URL.parse === "function"` guard with a fallback, or keep `new URL()` for the resolution step.

- [ ] `gno.land/adr/prxxxx_gnoweb_search_gnoland_url.md` — **ADR filename should use the actual PR number.** Per project convention (AGENTS.md), ADR files are named `pr<number>_<description>.md`. The placeholder `prxxxx` must be replaced with `pr5612` before merge.

## Nits

- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:25-27` — The JSDoc comment references `searchUrl` delegating to `resolveTarget` but does not mention that the method also handles the empty-string/null fallback or the silent-reload edge case. Worth documenting.

- [ ] `gno.land/adr/prxxxx_gnoweb_search_gnoland_url.md:1` — The ADR decision table lists `https://staging.gno.land/r/foo` as passing through unchanged, but the text under "Decision" step 1 says "hostname is exactly `gno.land`" — the asymmetry is correct behaviour but a reader might wonder about `www.gno.land`. A note that `www.gno.land` is also left unchanged (passes through) would prevent confusion.

## Missing Tests

- [ ] No unit tests for `resolveTarget`. The project has no JS/TS test runner configured (no Jest, Vitest, etc.). All the cases documented in the ADR table (`gno.land/r/foo`, `https://gno.land/r/foo`, `//attacker.com`, `javascript:alert()`, `gno.land` alone, `gno.land?q=1`, `gno.land#hash`) are completely untested. The open-redirect and protocol-injection regressions identified above would have been caught by a small test matrix against `resolveTarget`.

- [ ] No test for the `URL.parse` null-return path (i.e. what happens when the user submits an input that cannot be resolved to any URL).

## Suggestions

- Consider adding a lightweight test setup (Vitest + jsdom) to the `frontend` package. `resolveTarget` is a pure function (except for the `window.location.origin` read); with a mock origin it is trivially testable. Other controllers like `controller-action-function.ts` also have complex logic with no tests. The ADR explicitly notes that "existing handlers, redirects, and tests are untouched" — that is accurate, but the lack of JS tests means the core user-visible behaviour of gnoweb has no regression coverage.

- The regex `/^(?:https?:\/\/)?gno\.land(?=\/|$|\?|#)/i` is case-insensitive (`/i`), which means `GNO.LAND/r/foo` is also rewritten. This is arguably correct, but the ADR does not mention it. Add a note if intentional.

## Questions for Author

- Is `URL.parse` intentionally chosen over `new URL()` for its non-throwing behaviour? If so, please document why the silent-reload fallback (`?? ""`) is preferable to an explicit error message. If not, was the browser-compatibility implication considered?
- The commit message includes `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>`. Per `AGENTS.md`: "Do NOT add Co-Authored-By lines or AI tool credits in commits — they are legally misleading and carry no useful information." Please remove this trailer before merge.
- Was the `//attacker.com` open-redirect vector considered? The old code was safe against it; the ADR does not mention it.

## Verdict

REQUEST CHANGES — The PR solves a real usability problem and the regex approach is clean, but it introduces two security regressions relative to the existing code (protocol-relative open redirect and `javascript:` protocol acceptance) that must be fixed before merge, along with a silent-failure UX regression and a browser-compatibility risk for Safari < 18.
