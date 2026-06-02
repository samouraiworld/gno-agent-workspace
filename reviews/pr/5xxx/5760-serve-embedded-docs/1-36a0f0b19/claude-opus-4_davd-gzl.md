# PR #5760: feat(gnoweb): serve embedded repository docs under /docs

URL: https://github.com/gnolang/gno/pull/5760
Author: moul | Base: master | Files: 14 | +1202 -3
Reviewed by: davd-gzl | Model: claude-opus-4 | Commit: 36a0f0b19 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5760 36a0f0b19`

**Verdict: APPROVE** — security posture is sound (embed scope clean, no path traversal, assets served as `text/plain`/image not `text/html`, no routing collision); the only real gap is unstyled `http.FileServer` directory listings leaking under `/docs/_assets/` and `/docs/images/`, plus a few untested error paths. None block merge; PR is draft and feature-complete for parity.

## Summary

Adds a `/docs` route to gnoweb that renders the repo's `docs/` folder through the existing realm goldmark pipeline, so the documentation ships with the binary and stays in sync with the code. `docs/` becomes a Go package embedding 60 files (every `.md`, the `images/` and `_assets/` trees) via `//go:embed`; a handler resolves clean URLs (`/docs/builders/getting-started` → `builders/getting-started.md`), rewrites relative `.md` links to `/docs/...` (cross-repo escapes fall back to GitHub blob URLs), translates Docusaurus `:::kind` admonitions into the GitHub `> [!KIND]` blockquote already handled by `markdown/ext_alert.go`, and builds a section sidebar parsed from `README.md`. The `/docs → /u/docs` realm alias is removed.

Reviewed the four requested risk areas in depth:

- **Embed scope** — clean. Whitepaper `.tex/.pdf/.aux/.toc`, `Makefile`, and `*.go`/`*_test.go` are all excluded; only prose `.md`, images, and the `_assets/minisocial/*.gno` code samples are embedded. No leak.
- **Path traversal** — no hole. `resolve` runs `path.Clean` then rejects a leading `..`; the asset branch uses `http.FileServer(http.FS(...))`, which rejects traversal. All `../`, `%2e%2e`, and absolute-path probes return 404 with no source leak (adversarial test below).
- **Content-type** — safe. `.gno` assets serve as `text/plain; charset=utf-8` (sniffed), PNG as `image/png`; nothing reaches the client as `text/html`. Markdown renders with `UnsafeHTML` off by default, so raw HTML in docs is escaped.
- **Routing collision** — none. `mux.Handle("/docs", ...)` / `("/docs/", ...)` only capture the `/docs` subtree; `/r/docs/*` realms still route through `/` (confirmed by `app_test.go` passing). The removed alias was exact-match (`/docs` only), so no previously-working subpath regresses.

## Glossary

- `resolve` — `docs.go` method mapping a clean URL path to embedded markdown bytes + canonical file path.
- `rewriteDocsLinks` / `transformAdmonitions` — pure text pre-processors run before `RenderRealm`.
- `http.FileServer` — stdlib handler serving the `images/` and `_assets/` raw asset subtrees.
- `RenderRealm` — gnoweb's shared goldmark render path, reused so docs match realm styling.

## Critical (must fix)

None.

## Warnings (should fix)

- **[bare directory listings break the consistent-layout promise]** [`gno.land/pkg/gnoweb/docs.go:59`](https://github.com/gnolang/gno/blob/36a0f0b19/gno.land/pkg/gnoweb/docs.go#L59) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L59) — `http.FileServer` emits an unstyled HTML index for asset directories.
  <details><summary>details</summary>

  The asset server is a plain `http.FileServer(http.FS(fsys))`. Requesting an asset *directory* (`/docs/_assets/`, `/docs/_assets/minisocial/`, `/docs/images/`) returns status 200 with `Content-Type: text/html` and the stdlib auto-generated `<pre><a>…</a></pre>` listing — no site header, footer, theme, or CSS. This contradicts the PR's stated goal that docs "look consistent with the rest of the site": every other gnoweb surface is wrapped in `IndexLayout`, these three URLs are not. Not a security issue (everything embedded is already-public source), but it surfaces a raw filesystem UI on a production route.

  Observed body for `/docs/_assets/minisocial/`:
  ```
  <!doctype html>
  <meta name="viewport" content="width=device-width">
  <pre>
  <a href="posts-0.gno">posts-0.gno</a>
  ... (9 entries)
  </pre>
  ```

  Fix: disable directory listing for the asset subtree, e.g. wrap the `http.FileServer` in a handler that 404s when the resolved target is a directory (a small `fs.Stat` + `IsDir()` guard before delegating), or serve assets through a custom handler that mirrors the 404 page used for missing markdown.
  </details>

## Nits

- [`gno.land/pkg/gnoweb/docs.go:371-374`](https://github.com/gnolang/gno/blob/36a0f0b19/gno.land/pkg/gnoweb/docs.go#L371-L374) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L371-L374) — `flushClose` is a no-op closure (empty body, comment says "nothing structurally to write"). It is called once at the closer. Dead indirection; inline-delete it and drop the call site for clarity.
- [`gno.land/pkg/gnoweb/docs.go:88-89`](https://github.com/gnolang/gno/blob/36a0f0b19/gno.land/pkg/gnoweb/docs.go#L88-L89) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L88-L89) — the "Minimal slice: no dedicated sidebar yet … Sidebar and admonition syntax extension are tracked as follow-ups" comment on `DocsHandler` (lines 39-41) is now stale: this PR adds both the sidebar and admonition transform. Update the doc comment so the next reader isn't misled.
- [`gno.land/pkg/gnoweb/docs.go:341`](https://github.com/gnolang/gno/blob/36a0f0b19/gno.land/pkg/gnoweb/docs.go#L341) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L341) — admonition nesting is not supported: an inner `:::note` inside an open admonition is emitted as a literal body line (`> :::note`) and the first `:::` closes the outer block, dropping the rest out of the blockquote. Docusaurus supports nesting via `::::`. Low impact (no docs page nests today), but worth a one-line note in the function comment so it's a known limitation, not a silent surprise.

## Missing Tests

- **[error/edge paths uncovered]** [`gno.land/pkg/gnoweb/docs.go:64-66`](https://github.com/gnolang/gno/blob/36a0f0b19/gno.land/pkg/gnoweb/docs.go#L64-L66) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/docs.go#L64-L66) — no test for non-GET (405), the `..` escape rejection in `resolve`, or the render-error path. These are the lines codecov flags (patch 88%).
  <details><summary>details</summary>

  `docs_test.go` covers the happy paths and 404-on-unknown well, but three branches are untested: the `r.Method != GET → 405` guard, the `strings.HasPrefix(rel, "..") → false` traversal guard in `resolve`, and the `RenderRealm` failure → 500 path. The traversal guard is security-relevant; an adversarial test asserting it holds is worth landing in-tree so a future refactor of `resolve` can't silently open the hole. See the path-traversal test written for this review: [`docs_traversal_test.go`](./tests/docs_traversal_test.go) (all escape attempts return 404, no source leak). Fix: lift a trimmed version of that test into `docs_test.go`, plus one `httptest` POST case.
  </details>

- **[no regression test for the removed alias]** [`gno.land/pkg/gnoweb/app.go:30`](https://github.com/gnolang/gno/blob/36a0f0b19/gno.land/pkg/gnoweb/app.go#L30) · [↗](../../../../../.worktrees/gno-review-5760/gno.land/pkg/gnoweb/app.go#L30) — nothing asserts `/docs` no longer 302s to `/u/docs`.
  <details><summary>details</summary>

  The PR removes `"/docs": {"/u/docs", GnowebPath}` from `DefaultAliases`. The new handler-based behavior is well tested, but there's no negative test pinning that `/docs` now renders the README instead of redirecting — if someone re-adds the alias, the alias map is checked inside the `/` handler, which `/docs` no longer reaches, so the behaviors wouldn't actually conflict, but a one-line assertion documents the intent. Low priority.
  </details>

## Suggestions

- [`docs/docs_test.go:104-107`](https://github.com/gnolang/gno/blob/36a0f0b19/docs/docs_test.go#L104-L107) · [↗](../../../../../.worktrees/gno-review-5760/docs/docs_test.go#L104-L107) — `TestInternalLinksResolve` only checks `.md` link targets; image/asset links are explicitly skipped ("can be added in a follow-up"). Since `images/` and `_assets/` are embedded wholesale, a broken `![](images/foo.png)` reference ships silently. Extending the walk to also stat image targets would close the remaining silent-breakage gap the test set out to eliminate.

## Questions for Author

- Should the asset subtree disable directory listing (see Warning)? Serving a bare `http.FileServer` index on `/docs/_assets/` and `/docs/images/` is the one surface that doesn't match the "consistent with the rest of the site" goal.
- The `_assets/minisocial/*.gno` sample files are reachable raw at `/docs/_assets/minisocial/...` and served as `text/plain` — intended as a public, browsable code sample, or should they be inlined into the prose only?

## Adversarial tests

[`tests/docs_traversal_test.go`](./tests/docs_traversal_test.go) — path-traversal against `resolve` and the raw-asset `http.FileServer`, calling `ServeHTTP` directly to bypass `http.ServeMux` path cleaning (the attacker's real lever behind a non-normalising proxy). Probes `../`, percent-encoded `%2e%2e`, `....//`, and absolute paths. All return 404 with no embedded Go source / `go.mod` leak — the PR's defenses hold. Flipping any `wantLeak` assertion to true would catch a future regression in the resolver or StripPrefix wiring.
