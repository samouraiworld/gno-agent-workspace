# PR #5618: feat(gnoweb): expose render link on realm directory views

URL: https://github.com/gnolang/gno/pull/5618
Author: AmozPay | Base: master | Files: 4 | +94 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7 (1M context) | Commit: `1d734600` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5618 1d734600`

**Verdict: APPROVE** — small, focused, well-tested; only open concerns are a duplicated realm-prefix check and two parallel realm/package conventions inside the same template.

*Edit: dropped a warning claiming the in-view `TrimSuffix` was redundant — it is load-bearing for `GetPathsListView`, which does not pre-trim.*

## Summary

Closes #5580. Realm directory views now show a `Render` button in the header that deep-links to the rendered realm output. The gate is `IsRealm && !Mode.IsExplorer`, so pure packages (`/p/...`) and explorer mode are excluded. Change is purely additive in the view layer: a new `DirData` field, one template branch, and component + handler tests.

## Fix

Before, `DirectoryView` only emitted the file listing; the header had no link to the rendered realm. After, [`view_directory.go:57-71`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/view_directory.go#L57-L71) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/view_directory.go#L57-L71) computes `isRealm = strings.HasPrefix(pkgPath, "/r/")` and `renderURL = strings.TrimSuffix(pkgPath, "/")`, threading both into `DirData`. The template at [`directory.html:8-10`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/views/directory.html#L8-L10) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/views/directory.html#L8-L10) renders the link under `{{ if and .IsRealm (not .Mode.IsExplorer) }}`. The load-bearing constraint is the `/r/` prefix as the realm marker, which is also the source of truth used elsewhere via `weburl.GnoURL.IsRealm()`.

## Critical (must fix)

None.

## Warnings (should fix)

- **[duplicated realm classification]** [`view_directory.go:58`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/view_directory.go#L58) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/view_directory.go#L58) — re-derives `IsRealm` from the path prefix instead of reusing the canonical check.
  <details><summary>details</summary>

  `weburl.GnoURL.IsRealm()` already encodes the `/r/` convention for the rest of gnoweb. Re-implementing it inline in the components layer means the realm-path convention now lives in two places; a future change to what counts as a realm (e.g. a new prefix, namespacing) only updates one. Components are deliberately decoupled from `weburl` today, so duplication is defensible — but it should be deliberate, not accidental. Fix: pass `isRealm bool` and `renderURL string` into `DirectoryView` from the handler where `gnourl.IsRealm()` is already authoritative, or leave the inline check with a one-line comment pointing at the canonical definition so the next person doesn't update only one site.
  </details>

- **[two realm/package conventions in one template]** [`directory.html:28-32`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/views/directory.html#L28-L32) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/views/directory.html#L28-L32) — existing branches classify packages with `hasPrefix $pkgpath "/p/"`; the new header branch uses `.IsRealm`.
  <details><summary>details</summary>

  Two different ways to ask "is this a realm?" now coexist in the same file. They happen to agree today (realm = `/r/`, non-realm = `/p/` or `/u/`), but the moment a new path class is introduced — or someone flips one without the other — the template will silently classify the same path two different ways. Fix: migrate the existing `hasPrefix $pkgpath "/p/"` checks to `(not .IsRealm)` (or vice versa) so the template has one convention. Cheap to do now, expensive to detangle later.
  </details>

## Nits

- [`view_directory.go:14`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/view_directory.go#L14) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/view_directory.go#L14) — `RenderURL` is empty for non-realms; could be a `ShowRender()` method on `DirData` to keep the data struct narrower.
- [`view_test.go:316`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/view_test.go#L316) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/view_test.go#L316) — `TestDirectoryView_PackageHasNoRenderURL` reads fine, but `TestDirectoryView_NonRealmHasNoRenderURL` would also cover future `/u/` and `/e/` cases.
- [`directory.html:9`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/views/directory.html#L9) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/views/directory.html#L9) — no `aria-label` on the Render link. Visible "Render" text is descriptive enough; worth confirming against the gnoweb a11y conventions used elsewhere in the file.

## Missing Tests

- **[user-path coverage]** [`handler_http_test.go`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/handler_http_test.go) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/handler_http_test.go) — `TestHTTPHandler_DirectoryViewRenderLink` exercises `/r/` and `/p/` rows but not `/u/`.
  <details><summary>details</summary>

  Today `/u/alice` falls through `HasPrefix(pkgPath, "/r/")` → false and is correctly excluded. If the realm-detection logic ever migrates to `GnoURL.IsRealm()` (see the warning above) and that helper classifies user paths differently, the regression has no failing test to catch it. Fix: add a `/u/` row to the existing table.
  </details>

- **[trailing-slash realm path through the HTTP handler]** [`handler_http_test.go`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/handler_http_test.go) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/handler_http_test.go) — covered at component level, not at handler level.
  <details><summary>details</summary>

  The component-level `TestDirectoryView_RealmTrailingSlash` confirms the trim works; the HTTP-level path goes through the handler's own trim first, so the component never sees a trailing slash in production. A handler-level fixture would catch a future refactor that drops either trim. Fix: add a request fixture with `/r/demo/blog/` and assert the rendered link is `/r/demo/blog`.
  </details>

- **[explorer-mode realm at HTTP level]** [`handler_http_test.go`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/handler_http_test.go) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/handler_http_test.go) — component test `TestDirectoryView_RealmExplorerHidesRender` confirms the gate at the view layer only.
  <details><summary>details</summary>

  The `!Mode.IsExplorer` gate lives in the template, so the explorer wiring (does the handler actually set `Mode.IsExplorer` for explorer routes?) is untested end-to-end for the Render-link case specifically. Fix: add an explorer-mode realm row to the handler table asserting the link is absent.
  </details>

## Suggestions

- [`directory.html:8-10`](https://github.com/gnolang/gno/blob/1d734600/gno.land/pkg/gnoweb/components/views/directory.html#L8-L10) · [↗](../../../../../.worktrees/gno-review-5618/gno.land/pkg/gnoweb/components/views/directory.html#L8-L10) — Render link only appears in the header, not on per-realm rows in explorer mode.
  <details><summary>details</summary>

  Explorer rows already expose `Open` / `Source` / `Action`; `Render` would round out the set for realm rows. Out of scope here but a natural follow-up; the gate is the same `.IsRealm && !Mode.IsExplorer` inverted to per-row.
  </details>

- PR description — change is purely visual; a before/after screenshot lets reviewers confirm intent without checking out the branch.

## Questions for Author

- Why introduce `.IsRealm` when the template already classifies via `hasPrefix $pkgpath "/p/"`? Picking one convention up-front avoids the cleanup pass flagged in the warnings.
- Is hiding the Render button in explorer mode the right call? An explorer view of a realm could still benefit from a header "render this realm" link — it isn't strictly equivalent to per-row `Open`. A sentence in the PR description would lock the intent in.
