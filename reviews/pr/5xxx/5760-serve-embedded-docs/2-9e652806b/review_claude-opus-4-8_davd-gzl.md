# PR [#5760](https://github.com/gnolang/gno/pull/5760): feat(gnoweb): serve embedded repository docs under /docs

URL: https://github.com/gnolang/gno/pull/5760
Author: moul | Base: master | Files: 14 | +1280 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `9e652806b` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5760 9e652806b`

Round 2. Head advanced `36a0f0b19` → `9e652806b`: one content commit (`f9114ba1a`: HEAD support, admonition kind validation, resolve tests) plus a master merge (`9e652806b`). The merge is only half-reconciled and flips the round-1 verdict.

**TL;DR:** Adds a `/docs` route to gnoweb that renders the repo's `docs/` folder through the existing goldmark pipeline, so the docs ship inside the binary and can't drift from the code. The latest master merge left the branch in a broken state: gnoweb no longer compiles, so the entire `gno.land` build tree and every gnoweb-dependent CI job are red.

**Verdict: REQUEST CHANGES** — the master merge on `9e652806b` didn't carry two upstream changes into the PR's files: gnoweb fails to build (stale `FooterData`/`AnalyticsData` shape), and `TestSidebarFromReadme` fails on a renamed README heading. Round 1's directory-listing Warning and stale-comment Nit still stand. Fix the merge and the PR returns to its round-1 posture.

## Summary

Round 1 (APPROVE) reviewed the feature at `36a0f0b19`: embed scope, path traversal, content-type, and routing collision all checked out. Since then `f9114ba1a` cleanly addressed the round-1 `flushClose` nit (removed) and the untested `resolve`/method/render-error paths (now partly covered by `TestResolve` and the HEAD/POST cases). The problem is the master merge that followed. Master's [#5554](https://github.com/gnolang/gno/pull/5554) reshaped `components.FooterData` (its `Analytics` field is now the `AnalyticsData` struct, and `AssetsPath`/`BuildTime` moved into `AnalyticsData`), and another master change renamed the README's third top-level heading from `Resources` to `References`. Neither landed in the PR's own files, so the package doesn't build and the sidebar invariant test now fails. Both are mechanical to fix. The `f9114ba1a` logic itself is correct: HEAD returns headers with no body, and admonition kind-validation converts every known kind while passing unknown kinds through verbatim.

## Glossary

- `FooterData` / `AnalyticsData` — gnoweb view-model structs in `components/`; master #5554 moved analytics fields between them.
- `resolve` — `docs.go` method mapping a clean URL path to embedded markdown bytes plus canonical file path.
- `transformAdmonitions` — pure text pre-processor converting Docusaurus `:::kind` blocks into GitHub `> [!KIND]` blockquotes.
- `http.FileServer` — stdlib handler serving the `images/` and `_assets/` raw asset subtrees.
- `RenderRealm` — gnoweb's shared goldmark render path, reused so docs match realm styling.

## Critical (must fix)

- **[gnoweb no longer compiles]** [`gno.land/pkg/gnoweb/docs.go:108-112`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs.go#L108-L112) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L108-L112) — `FooterData` is built with the pre-#5554 shape, so the whole gnoweb package fails to build.
  <details><summary>details</summary>

  Master [#5554](https://github.com/gnolang/gno/pull/5554) changed `components.FooterData`: its `Analytics` field is now the `AnalyticsData` struct (not a `bool`), and `AssetsPath`/`BuildTime` were removed from `FooterData` (they live on `AnalyticsData` now, and `IndexLayout` fills them from `HeadData` at [`layout_index.go:187-188`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/components/layout_index.go#L187-L188) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/components/layout_index.go#L187-L188)). `docs.go` still constructs `FooterData` the old way in both `ServeHTTP` (108-112) and `renderError` ([200-203](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs.go#L200-L203)), so `go build ./gno.land/pkg/gnoweb/` fails with six type errors. This blocks every gnoweb-dependent build. All the red CI jobs (`main / test`, `main / lint`, `gnodev / *`, `gnobro / *`, `e2e-test`, `build docker images`) fail on this one compile error: each log shows the same `cannot use h.Static.Analytics ... unknown field AssetsPath` lines, and `main / test` lists `gnokey`, `gnoland`, `gnoweb`, `gnoclient`, and `feature/state` as `[build failed]`. Verified on `9e652806b`: the build fails, and applying the fix below makes it pass. Fix: mirror [`handler_http.go:216-220`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/handler_http.go#L216-L220) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/handler_http.go#L216-L220) — `Analytics: components.AnalyticsData{Enabled: h.Static.Analytics}` — and drop the `AssetsPath`/`BuildTime` lines from both literals.
  </details>

- **[sidebar test asserts a renamed README heading]** [`docs/sidebar_test.go:60`](https://github.com/gnolang/gno/blob/9e652806b/docs/sidebar_test.go#L60) · [↗](../../../../../.worktrees/gno-review-5760/docs/sidebar_test.go#L60) — the test hardcodes the README section title `Resources`, which master renamed to `References`, so `go test ./docs/` fails.
  <details><summary>details</summary>

  `TestSidebarFromReadme` asserts the live `README.md` yields sections `["Use Gno.land", "Build on Gno.land", "Resources"]`. Master renamed the third heading to `## References` ([`docs/README.md:36`](https://github.com/gnolang/gno/blob/9e652806b/docs/README.md?plain=1#L36) · [↗](../../../../../.worktrees/gno-review-5760/docs/README.md#L36)), so `Sidebar()` returns `References` and the assertion fails: `section[2] = "References", want "Resources"`. This surfaces under `main / test` (the Go `docs` package test), which is currently masked behind the compile break; the `docs` CI job runs `make generate`/`make lint`, not `go test ./docs/`, so it stays green. The exact-label check is brittle by design: it couples a test to human-facing README headings that evolve independently of the parser. Fix: update the expected label to `References`. Consider dropping the exact-title assertion entirely, since the second check (every internal item resolves to an embedded `.md`) already covers the invariant that matters.
  </details>

## Warnings (should fix)

- **[bare directory listings break the consistent-layout promise]** [`gno.land/pkg/gnoweb/docs.go:59`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs.go#L59) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L59) — `http.FileServer` emits an unstyled HTML index for asset directories.
  <details><summary>details</summary>

  The asset server is a plain `http.FileServer(http.FS(fsys))`. Requesting an asset directory (`/docs/_assets/`, `/docs/_assets/minisocial/`, `/docs/images/`) returns status 200 with `Content-Type: text/html` and the stdlib auto-generated `<pre><a>…</a></pre>` listing: no site header, footer, theme, or CSS. Every other gnoweb surface is wrapped in `IndexLayout`; these three URLs are not, which contradicts the PR's stated goal that docs "look consistent with the rest of the site". Not a security issue (everything embedded is already-public source), but it surfaces a raw filesystem UI on a production route. Verified live: booting gnoweb from the worktree and requesting `/docs/_assets/minisocial/` returns 200, `text/html`, and the bare listing (`posts-1.gno`, `render-0.gno`, etc.). Fix: 404 asset-directory requests before delegating to `http.FileServer` (an `fs.Stat` + `IsDir()` guard), or serve assets through a handler that mirrors the docs 404 page.
  </details>

## Nits

- [`gno.land/pkg/gnoweb/docs.go:39-41`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs.go#L39-L41) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L39-L41) — the `DocsHandler` doc comment still reads "Minimal slice: no dedicated sidebar yet … Sidebar and admonition syntax extension are tracked as follow-ups." This PR ships both the sidebar (`buildSidebar`) and the admonition transform, so the comment is now wrong. Update it.
- [`gno.land/pkg/gnoweb/docs.go:351`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs.go#L351) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L351) — admonition nesting is unsupported: an inner `:::note` inside an open admonition emits a literal `> :::note` body line, and the first `:::` closes the outer block, dropping the rest out of the blockquote. Docusaurus nests via `::::`. No docs page nests today; worth a one-line note in the `transformAdmonitions` doc comment so it's a known limitation, not a silent surprise.

## Missing Tests

- **[the 500 render-error path is unexercised]** [`gno.land/pkg/gnoweb/docs.go:143`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs.go#L143) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L143) — no test covers the `RenderRealm` failure to 500 branch.
  <details><summary>details</summary>

  The new `TestResolve` and the HEAD/POST cases close the traversal-guard and non-GET gaps from the prior round. The `RenderRealm` error to `renderError(…, 500, "documentation page failed to render")` path stays untested. A `DocsHandler` with a stub renderer that returns an error would pin the status code and the error copy against a future refactor. Low priority; the branch is small.
  </details>

- **[HEAD body-suppression is asserted only as a status code]** [`gno.land/pkg/gnoweb/docs_test.go:75-78`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/docs_test.go#L75-L78) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs_test.go#L75-L78) — the HEAD case checks status 200 but not that the body is empty.
  <details><summary>details</summary>

  The handler writes the full page body for HEAD and relies on `net/http` to discard it; the test only asserts `wantStatus`. An assertion that the HEAD response body is empty while GET on the same route is non-empty would pin the spec-correct behavior. Verified live: HEAD `/docs` downloads 0 body bytes at status 200 while GET downloads 68526 bytes, so the current behavior is correct; the test just doesn't guard it.
  </details>

- **[no regression test for the removed alias]** [`gno.land/pkg/gnoweb/app.go:30`](https://github.com/gnolang/gno/blob/9e652806b/gno.land/pkg/gnoweb/app.go#L30) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/app.go#L30) — nothing asserts `/docs` no longer 302s to `/u/docs`.
  <details><summary>details</summary>

  The PR removes `"/docs": {"/u/docs", GnowebPath}` from `DefaultAliases`. The handler behavior is well tested, but no negative test pins that `/docs` now renders the README instead of redirecting. Low priority: the alias map is checked inside the `/` handler, which `/docs` no longer reaches, so a re-added alias wouldn't actually conflict, but a one-line assertion documents the intent.
  </details>

## Suggestions

- [`docs/docs_test.go:104-107`](https://github.com/gnolang/gno/blob/9e652806b/docs/docs_test.go#L104-L107) · [↗](../../../../../.worktrees/gno-review-5760/docs/docs_test.go#L104-L107) — `TestInternalLinksResolve` only checks `.md` link targets; image/asset links are explicitly skipped. Since `images/` and `_assets/` are embedded wholesale, a broken `![](images/foo.png)` reference ships silently. Extending the walk to stat image targets too would close the remaining silent-breakage gap the test set out to eliminate.

## Delta verification (round 2)

Commit `f9114ba1a` is the only PR-authored change since the round-1 head `36a0f0b19`; the rest of `36a0f0b19..9e652806b` is a master merge. Verified against a local build with the trivial `FooterData` merge-fix applied (reverted after, worktree left clean):

- HEAD `/docs`: status 200, `Content-Type: text/html`, 0 body bytes; GET same route 68526 body bytes. HEAD returns headers and status with no body.
- Known admonition kinds all convert: `NOTE`, `TIP`, `WARNING`, `CAUTION`, `IMPORTANT`, `INFO` each produce `> [!KIND]`. `getting-started` renders `gno-alert-tip` and `gno-alert-warning` live.
- Unknown kind `:::foo\nhi\n:::` passes through verbatim as `:::foo\nhi\n:::`, no `[!FOO]` markup.
- `go test ./gno.land/pkg/gnoweb/ -run 'Docs|Resolve|Admonition'` passes (with the build-fix applied); `go test ./docs/ -run TestSidebarFromReadme` fails on the renamed heading.

Invariant catalog: this PR is Go HTTP tooling, no GnoVM/stdlib/`.gno` change, so gas, realm-state, coin, storage-deposit, VM-semantics, and type-check classes do not apply. Determinism: the output path reads the embedded FS and does no `time.Now()`, real randomness, or map iteration (`admonitionKinds` is membership-only). Global mutable state: `admonitionKinds` and the compiled regexes are set once at init and read-only after, the safe shape, and the `t.Parallel()` tests read them concurrently without a race. Error handling: `readFile` and `resolve` return clean `ok=false` on failure with no swallowed errors.

## Open questions

- The `_assets/minisocial/*.gno` sample files are reachable raw at `/docs/_assets/minisocial/...` served as `text/plain`. Intended as a public browsable code sample, or inline-only? Not posted: no risk, purely a product-intent call for the author.
