# PR #5666: feat(gnoweb): change built-in playground output to support copy buttons

**URL:** https://github.com/gnolang/gno/pull/5666
**Author:** jeronimoalbi | **Base:** playground2 | **Files:** 10 | **+156 -43**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Restructures the gnoweb built-in playground (`/_/play`) output panel so individual lines (typically shell commands embedded in mixed text/links/notes) can be copied to the clipboard via per-line copy buttons. The output area used to be a single `<pre>` whose `textContent` was replaced wholesale; it now becomes a flex column of `b-playground-output-item` rows, where each row is a `<pre>` plus an optional copy button.

Changes by file:
- `components/views/playground.html`: replaces the output `<pre>` with a `<div>` (no initial placeholder span — `clearOutput()` now fills it from JS).
- `frontend/css/06-blocks.css`: turns `.b-playground-output-content` into a flex column; introduces `.b-playground-output-item`, `.b-playground-output-item-text` (inherits the prior `white-space: pre-wrap; word-break: break-word;`), and `.b-playground-output-copy-btn` with hover styling.
- `frontend/js/controller-playground.ts`:
  - Renames the old `_setOutput` semantics into two methods: `_resetOutput(text, copyable?, isError?)` (clears then appends one row) and `_setOutput(text, copyable?, isError?)` (appends a row).
  - Adds `_setErrorOutput(text)` helper.
  - When `copyable=true`, appends a button with `data-controller="copy" data-action="click->copy#copy" data-copy-text-value="<text>"`, reusing the existing `CopyController` and `makeCopyIcon()`.
  - Splits the previous monolithic output blocks in `runCode`, `runTests`, `formatCode` into a non-copyable preamble + one or more copyable command rows.
  - Adds `this.clearOutput()` in `connect()` so the placeholder is rendered consistently from JS rather than from the template.
- `frontend/js/utils.ts`: extracts `makeIcon(name)` and `makeCopyIcon()` helpers (previously inlined in CopyController callsites for the commands panel).
- `frontend/js/controller.ts`: re-exports `makeIcon`/`makeCopyIcon` from the base controller barrel.
- `public/js/*` and `public/main.css`: regenerated/minified artefacts from `make generate`.

Mechanics for newly-injected copy buttons: `frontend/js/index.ts` runs a `MutationObserver` on `document.documentElement` filtered by `data-controller` attribute, so each dynamically appended `<button data-controller="copy">` is auto-initialised by `CopyController`. Each button is its own controller instance — `getTargets("icon")` is scoped to `this.element` (the button), so feedback animations don't leak across rows.

## Test Results
- **Existing tests:** PASS — `go test -run "Playground|playground" ./gno.land/pkg/gnoweb/...` (all `TestHTTPHandler_PlaygroundPage`, `TestHandlerPlaygroundEval`, `TestHandlerPlaygroundFuncs`, `TestStaticHeaderDevLinks_WithPlaygroundMode` pass).
- **CI:** all green per `gh pr checks 5666`. Codecov reports modified lines covered.
- **Edge-case tests:** skipped — frontend-only change, no Go unit test infra for the JS controller in this PR's scope.

## Critical (must fix)
- None.

## Warnings (should fix)
- None.

## Nits
- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-playground.ts:480` — the placeholder `// Run code to see output here` lost its `u-color-muted` styling. The previous template had `<span class="u-color-muted">// Run code to see output here</span>`; the new path renders a plain `<pre>` row with no muted class. Add `u-color-muted` (or equivalent) when rendering the placeholder in `clearOutput()` so the empty-state still reads as a hint and not as real output.
- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-playground.ts:394` — `this._setOutput("\n\nTo test:");` produces a row whose `<pre>` starts with two blank lines. Inside a flex column this works visually, but it's a hidden coupling between text content and layout (the spacing comes from the literal `\n\n`, not from CSS gap). Prefer making each row a real flex item with a top-margin / gap utility, or split into a separate "heading" row, so spacing is driven by CSS instead of whitespace inside `<pre>`.
- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-playground.ts:262-285` — `_setOutput` now mixes "append a row" semantics with the historical "scroll into view" side-effect on every call. When `runCode()` appends multiple rows in sequence, `scrollIntoView` fires for each one. Likely fine in practice (smooth-scroll de-bounces visually), but consider moving the scroll call out of `_setOutput` to the public action methods so it runs once per user action.
- [ ] `gno.land/pkg/gnoweb/frontend/js/controller-playground.ts:267` — the `<pre>` tag is used as a flex child with `flex: 1`. The Markdown/HTML semantics of `<pre>` (line-preserving block) are preserved by CSS but the element is no longer the only output container, so the historical assumption "output area is a single pre" no longer holds. Worth a short comment explaining each row is a `<pre>` for whitespace fidelity.
- [ ] `gno.land/pkg/gnoweb/frontend/js/utils.ts:58-71` — `makeCopyIcon()` uses `svg.querySelector("use")?.setAttribute(...)` which silently no-ops if the SVG shape ever changes. Since this helper now has two callers, throwing (or asserting) if the `<use>` child isn't found would catch future regressions earlier.
- [ ] `gno.land/pkg/gnoweb/frontend/css/06-blocks.css:2922-2924` — `gap: 0;` on the output container is functionally a no-op (default is `0`); either drop it or set a real value if the intent is non-zero spacing between rows.

## Missing Tests
- [ ] No automated coverage that the copy button is rendered for commands and absent for preamble text (`gno.land/pkg/gnoweb/frontend/js/controller-playground.ts:271`). A small DOM-level test (jsdom or Playwright) asserting `runCode()` produces the expected count of `.b-playground-output-copy-btn` and that each button's `data-copy-text-value` equals the trimmed command would lock the behaviour described in the PR.
- [ ] No test that long/multiline text in `data-copy-text-value` round-trips correctly through `setAttribute`/`getAttribute` and `CopyController._copyTextToClipboard` (which calls `text.trim()`). Today's payloads are single-line trimmed commands, but the same code path will be reused for `/_/api/eval` results — worth pinning behaviour now.

## Suggestions
- The leading space convention (`" gnokey ..."`, `" gno run ..."`) is the only thing distinguishing "shell command" from "prose" visually. Since the row is now a real DOM element, consider adding a CSS-driven prefix (e.g. `::before { content: "$ "; }` on a `.b-playground-output-cmd` modifier) and dropping the in-text leading space. That avoids the implicit "trim removes my padding" coupling between display and clipboard value (`controller-playground.ts:382-410` ↔ `controller-copy.ts:82`).
- `_setOutput` and `_resetOutput` differ only by the clearing loop. A single method `appendOutput(...)` plus a separate `clearOutputDom()` (called from `_resetOutput`/`clearOutput`) would express intent more directly and remove the boolean-heavy signatures.
- `makeIcon`/`makeCopyIcon` now live in `utils.ts` but are re-exported from `controller.ts`. Pick one canonical import path and document it; the dual export will drift.

## Questions for Author
- Was the loss of the muted placeholder styling intentional, or an oversight from moving the empty state out of the template?
- Any plan to consolidate the per-row `<button data-controller="copy">` pattern into a small `makeCopyButton(text)` helper alongside `makeCopyIcon()`? It would shrink `_setOutput` and make the action-attribute wiring harder to typo.
- Should real eval responses (`result.result` from `/_/api/eval`) eventually be rendered with the same row-splitting heuristic, or is "always one row" the intended semantics for server results?

## Verdict
APPROVE — Frontend-only refactor that delivers per-line copy without expanding the XSS/clipboard surface (`textContent`/`setAttribute` only, no `innerHTML`; reuses the existing `CopyController` and `MutationObserver` wiring). All gnoweb Go tests still pass and CI is green; remaining items are presentation polish and small DX cleanups.
