# PR [#5966](https://github.com/gnolang/gno/pull/5966): feat(gnoweb): open render on package-name click, add Browse link in explorer

URL: https://github.com/gnolang/gno/pull/5966
Author: moul | Base: master | Files: 3 | +99 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: 207eda421 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5966 207eda421`

**TL;DR:** In the gnoweb package-listing view for a namespace (like `/r/somebody/`), clicking a package's name used to open its raw file listing. Now it opens the realm's rendered page, and a new "Browse" button reaches the file listing instead.

**Verdict: APPROVE** — cleanly scoped explorer-link flip, verified end-to-end; one non-blocking gap: the pure-package (`/p/`) explorer branch has no test.

## Summary
The gnoweb explorer paths-list view ([`GetPathsListView`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/handler_http.go#L793-L821) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/handler_http.go#L793), `ViewModeExplorer`) flips the primary click target of each entry. The main entry link drops the explorer-only trailing slash, so the package name now opens the realm render (`{{ .Link }}`) rather than the directory listing (`{{ .Link }}/`); a new `Browse` button (`{{ .Link }}/`) keeps the file listing one click away. The right-side group in explorer mode becomes `Open`, `Browse`, `Source`, `Action`, with `Open`/`Action` still hidden for pure `/p/` packages and `Browse` always shown. One-line template edit plus one handler test.

## Examples
Explorer listing of a namespace, per entry:

| Click target | Before | After |
|---|---|---|
| Package name (`/r/x`) | `/r/x/` → file listing | `/r/x` → realm render |
| `Open` button (`/r/`) | `/r/x` → render | `/r/x` → render |
| `Browse` button | (did not exist) | `/r/x/` → file listing |
| Package name (`/p/x`) | `/p/x/` → file listing | `/p/x` → file listing |

## Glossary
- gnoweb: web frontend serving chain content; server-rendered Go `html/template` views.
- realm: stateful on-chain package under `/r/`; has a `Render()` the explorer render link targets.
- pure package: stateless importable package under `/p/`; no `Render()`.

## Fix
The main link at [`directory.html:15`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/components/views/directory.html#L15) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/components/views/directory.html#L15) changes from `{{ .Link }}{{ if $.Mode.IsExplorer }}/{{ end }}` to `{{ .Link }}`, and a `Browse` button is added at [`directory.html:28`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/components/views/directory.html#L28) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/components/views/directory.html#L28) inside the `IsExplorer` block. Directory mode (source-file listing) always renders with `IsExplorer` false, so the old form already collapsed to `{{ .Link }}` there and the new form is byte-identical output; the added button never renders outside explorer mode.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[label reads the same as its neighbor]** `gno.land/pkg/gnoweb/components/views/directory.html:26,28` — for a `/r/` entry, `Open` (render) and `Browse` (file listing) are both bare anchors with no `aria-label`/`title`; to a screen reader or a first-time user both read as "go to the package", and the distinction lives only in the trailing slash of the href. Pre-existing terseness (`Action` for `$help` shares it), not introduced here. Cited: [`directory.html:26`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/components/views/directory.html#L26) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/components/views/directory.html#L26).
- **[test gates only one direction]** `gno.land/pkg/gnoweb/handler_http_test.go:508` — the flip is caught by the positive `assert.Contains(body, ` + "`href=\"/r/mock/sub\">`" + `)`; a `assert.NotContains(body, ` + "`href=\"/r/mock/sub/\">`" + `)` would lock it from the other direction against a future template reshuffle. The positive assertion already fails against the old template (revert-proofed), so this is polish. Cited: [`handler_http_test.go:508`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/handler_http_test.go#L508) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/handler_http_test.go#L508).

## Missing Tests
- **[edited branch never runs in tests]** `gno.land/pkg/gnoweb/handler_http_test.go:483` — the pure-package (`/p/`) explorer branch is uncovered.
  <details><summary>details</summary>

  The new [`TestHTTPHandler_ExplorerPathsListBrowse`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/handler_http_test.go#L483) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/handler_http_test.go#L483) only lists an `/r/` namespace, so the `/p/` gate at [`directory.html:25-31`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/components/views/directory.html#L25-L31) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/components/views/directory.html#L25) that hides `Open`/`Action` while keeping the always-shown `Browse` for pure packages is never executed by any test. `TestHTTPHandler_DirectoryViewPurePackage` requests `/p/pkg/`, but that package has files, so it renders directory mode with `IsExplorer` false and the inline-button block never runs. A regression that dropped the `/p/` gate or moved `Browse` inside it would leave CI green. Ready-to-add sibling test, green as written, in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md). Fix: add a `/p/mock/sub` case requesting `/p/mock/`, asserting `Browse`+`Source` present and `Open`+`Action` absent.
  </details>

## Suggestions
- **[button duplicates the name link]** `gno.land/pkg/gnoweb/components/views/directory.html:26,28` — after the flip, the button row always duplicates the name link's destination for one package kind.
  <details><summary>details</summary>

  For `/r/` entries the name link and `Open` both resolve to the render (`{{ .Link }}`). For `/p/` entries the name link (`/p/x`) and `Browse` (`/p/x/`) both resolve to the same file listing, because [`IsPure()`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/weburl/url.go#L222-L223) · [↗](../../../../../.worktrees/gno-review-5966/gno.land/pkg/gnoweb/weburl/url.go#L222) routes `/p/x` and `/p/x/` identically to `GetDirectoryView`. Harmless, and it follows from the deliberate "always show Browse" choice. Not a change to make in this PR; noted so the duplication is a known property. Confirmed behaviorally: `/p/mock/sub` and `/p/mock/sub/` both return the directory listing (status 200, same files).
  </details>

## Verified
- Live end-to-end on a real node (not the mock): booted gnodev from this branch, deployed two realms under `/r/acme/`, loaded the explorer listing. The package name and `Open` link to `/r/acme/alpha` (the render, `<h1>alpha render</h1>`); `Browse` links to `/r/acme/alpha/` (Directory · 2 Files). The two intents diverge as claimed. CI unit tests exercise only the mock client, so this real `ListPaths`→template→routing chain is CI-invisible.
- Revert-proof: restoring the old `{{ .Link }}{{ if $.Mode.IsExplorer }}/{{ end }}` main link renders `href="/r/mock/sub/">` and fails `TestHTTPHandler_ExplorerPathsListBrowse` at [`handler_http_test.go:508`](https://github.com/gnolang/gno/blob/207eda421/gno.land/pkg/gnoweb/handler_http_test.go#L508). The test genuinely gates the flip.
- Directory mode unchanged: with `IsExplorer` false the old main-link form collapses to `{{ .Link }}`, byte-identical to the new form; `TestHTTPHandler_DirectoryView*` pass.
- `/p/` routing: `/p/mock/sub` (no slash) and `/p/mock/sub/` both return status 200 with the file listing, settling that the no-slash name link does not land on an empty render page for pure packages.
- Tests green at 207eda421: `go test ./gno.land/pkg/gnoweb/...`.

## Open questions
- A `/r/` entry whose `Render()` panics now surfaces the render error page on a name click instead of the old file listing; `Browse` is the escape hatch to the directory. Expected given the "open render on click" intent, not posted.
- Whether `Browse` should be gated to `/r/` entries only, since it duplicates the name link for `/p/`. The ADR weighed always-show-Browse deliberately, so no decision is forced here; not posted.
</content>
</invoke>
