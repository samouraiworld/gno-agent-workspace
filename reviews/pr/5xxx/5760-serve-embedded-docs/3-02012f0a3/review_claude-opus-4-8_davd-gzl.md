# PR [#5760](https://github.com/gnolang/gno/pull/5760): feat(gnoweb): serve embedded repository docs under /docs

URL: https://github.com/gnolang/gno/pull/5760
Author: moul | Base: master | Files: 14 | +1280 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `02012f0a3` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5760 02012f0a3`

Round 3. Head advanced `9e652806b` → `02012f0a3`: both round-2 blockers fixed (`32e5c4d42` FooterData/AnalyticsData API, `02012f0a3` sidebar label), plus a master merge (`abf1267dd`).

**TL;DR:** Adds a `/docs` route to gnoweb that renders the repo's `docs/` folder through the existing goldmark pipeline, so the docs ship inside the binary and can't drift from the code. The two regressions the previous master merge introduced are fixed, and the branch builds and tests clean again.

**Verdict: APPROVE** — both round-2 blockers are gone: gnoweb compiles and `go test ./docs/` passes. What remains is round 1's directory-listing Warning plus nits and test gaps, none of them blocking.

## Summary

Round 2 (REQUEST CHANGES) found the master merge on `9e652806b` half-reconciled: `docs.go` still built `components.FooterData` with the pre-[#5554](https://github.com/gnolang/gno/pull/5554) shape, so the whole gnoweb package failed to compile, and `TestSidebarFromReadme` still asserted a README heading master had renamed. Both are now fixed exactly as prescribed. `32e5c4d42` replaces the two `FooterData` literals with `Analytics: components.AnalyticsData{Enabled: h.Static.Analytics}` and drops the removed `AssetsPath`/`BuildTime` fields, matching [`handler_http.go:216-220`](https://github.com/gnolang/gno/blob/02012f0a3/gno.land/pkg/gnoweb/handler_http.go#L216-L220) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/handler_http.go#L216-L220). `02012f0a3` updates the expected sidebar label to `References`. CI is green apart from the codeowners bot gate. The feature itself was reviewed in depth at round 1 (embed scope, path traversal, content-type, routing collision all checked out) and has not changed since.

Verified on `02012f0a3`: `go build ./gno.land/pkg/gnoweb/` succeeds where it failed at `9e652806b`, and `go test ./docs/` passes. The round-1 directory-listing Warning still reproduces (httptest hit on the three asset directories returns 200 with the stdlib `<pre>` index).

## Glossary

- `FooterData` / `AnalyticsData` — gnoweb view-model structs in `components/`; master #5554 moved analytics fields between them.
- `resolve` — `docs.go` method mapping a clean URL path to embedded markdown bytes plus canonical file path.
- `transformAdmonitions` — pure text pre-processor converting Docusaurus `:::kind` blocks into GitHub `> [!KIND]` blockquotes.
- `http.FileServer` — stdlib handler serving the `images/` and `_assets/` raw asset subtrees.
- `RenderRealm` — gnoweb's shared goldmark render path, reused so docs match realm styling.

## Critical (must fix)

None. Both round-2 blockers are fixed.

## Warnings (should fix)

- **[bare directory listings break the consistent-layout promise]** [`gno.land/pkg/gnoweb/docs.go:59`](https://github.com/gnolang/gno/blob/02012f0a3/gno.land/pkg/gnoweb/docs.go#L59) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L59) — `http.FileServer` emits an unstyled HTML index for asset directories.
  <details><summary>details</summary>

  The asset server is a plain `http.FileServer(http.FS(fsys))`. Requesting an asset directory (`/docs/_assets/`, `/docs/_assets/minisocial/`, `/docs/images/`) returns status 200 with `Content-Type: text/html` and the stdlib auto-generated `<pre><a>…</a></pre>` listing: no site header, footer, theme, or CSS. Every other gnoweb surface is wrapped in `IndexLayout`; these three URLs are not, which contradicts the PR's stated goal that docs "look consistent with the rest of the site". Not a security issue (everything embedded is already-public source), but it surfaces a raw filesystem UI on a production route. Confirmed behaviorally on `02012f0a3`. Fix: 404 asset-directory requests before delegating to `http.FileServer` (an `fs.Stat` + `IsDir()` guard), or serve assets through a handler that mirrors the docs 404 page.
  </details>

## Nits

- [`gno.land/pkg/gnoweb/docs.go:39-41`](https://github.com/gnolang/gno/blob/02012f0a3/gno.land/pkg/gnoweb/docs.go#L39-L41) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L39-L41) — the `DocsHandler` doc comment still reads "Minimal slice: no dedicated sidebar yet … Sidebar and admonition syntax extension are tracked as follow-ups." This PR ships both the sidebar (`buildSidebar`) and the admonition transform, so the comment is wrong. Update it.
- [`gno.land/pkg/gnoweb/docs.go:351`](https://github.com/gnolang/gno/blob/02012f0a3/gno.land/pkg/gnoweb/docs.go#L351) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L351) — admonition nesting is unsupported: an inner `:::note` inside an open admonition emits a literal `> :::note` body line, and the first `:::` closes the outer block, dropping the rest out of the blockquote. Docusaurus nests via `::::`. No docs page nests today; worth a one-line note in the function comment so it's a known limitation, not a silent surprise.

## Missing Tests

- **[error path uncovered]** [`gno.land/pkg/gnoweb/docs.go:143`](https://github.com/gnolang/gno/blob/02012f0a3/gno.land/pkg/gnoweb/docs.go#L143) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L143) — no test covers the render-failure to 500 branch.
  <details><summary>details</summary>

  `TestResolve` and `TestDocsHandlerRoutes` cover the happy paths, 404, HEAD, and 405, but the `RenderRealm` error to `renderError(500)` path is never exercised. A handler built with a stub renderer that returns an error would pin both the status code and the error copy.
  </details>

- **[HEAD parity not actually asserted]** [`gno.land/pkg/gnoweb/docs_test.go:75-78`](https://github.com/gnolang/gno/blob/02012f0a3/gno.land/pkg/gnoweb/docs_test.go#L75-L78) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs_test.go#L75-L78) — the HEAD case checks status 200 but not that the body is empty.
  <details><summary>details</summary>

  `f9114ba1a` added HEAD support on the reasoning that the GET path writes no body for HEAD. The test only asserts `wantStatus: http.StatusOK`, so the property the change is about is unpinned. Asserting an empty HEAD body against a non-empty GET body on the same route would catch a regression that starts writing a body on HEAD.
  </details>

- **[no regression test for the removed alias]** [`gno.land/pkg/gnoweb/app.go:30`](https://github.com/gnolang/gno/blob/02012f0a3/gno.land/pkg/gnoweb/app.go#L30) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/app.go#L30) — nothing asserts `/docs` no longer 302s to `/u/docs`.
  <details><summary>details</summary>

  The PR removes `"/docs": {"/u/docs", GnowebPath}` from `DefaultAliases`. The handler behavior is well tested, but no negative test pins that `/docs` now renders the README instead of redirecting. Low priority: the alias map is checked inside the `/` handler, which `/docs` no longer reaches, so a re-added alias wouldn't actually conflict, but a one-line assertion documents the intent.
  </details>

## Suggestions

- [`docs/docs_test.go:104-107`](https://github.com/gnolang/gno/blob/02012f0a3/docs/docs_test.go#L104-L107) · [↗](../../../../../.worktrees/gno-review-5760/docs/docs_test.go#L104-L107) — `TestInternalLinksResolve` only checks `.md` link targets; image/asset links are explicitly skipped. Since `images/` and `_assets/` are embedded wholesale, a broken `![](images/foo.png)` reference ships silently. Extending the walk to stat image targets too would close the remaining silent-breakage gap the test set out to eliminate.

## Open questions

- The `Docs` button on the gnoweb home page still points at `docs.gno.land`, so nothing links to the new `/docs` surface. Raised as a PR thread comment at round 1, not re-posted: it's a follow-up wiring decision, not a defect in this diff.
