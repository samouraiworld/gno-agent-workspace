# PR #5572: feat(gnoweb): add package overview source doc

URL: https://github.com/gnolang/gno/pull/5572
Author: alexiscolin | Base: master | Files: 33 | +2988 -123
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `714d2f8` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5572 714d2f8`

**TL;DR:** Today, opening `/r/<pkg>$source` jumps straight into the first source file. This PR replaces that with a pkg.go.dev-style landing page: package doc, README, the exported funcs/types/consts/vars (each deep-linking to its exact source line), imports, file list, subdirectories, and a metadata sidebar with code stats. The deep-link line numbers come from new `File`/`Line` fields added to the `vm/qdoc` JSON output.

**Verdict: APPROVE** — one should-fix (the sidebar "Code stats" count unexported symbols that the page never lists, so the numbers don't match what's shown), plus a couple of small cleanups. Routing dispatch, source deep-linking, and the concurrent fetch were all verified live and under `-race`; nothing here blocks merge.

## Summary

`/r/<pkg>$source` now dispatches on the `file` query: with a file it is the old source-code view, without one it is the new `OverviewView` ([handler_http.go:309-315](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/gnoweb/handler_http.go#L309-L315) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L309-L315)). The handler fans out `ListFiles`+meta-file fetch, `Doc`, README render, and `ListPaths` over an `errgroup`, then `BuildOverview` turns the results into a pure view payload. `gnovm/pkg/doc` gains `File`/`Line` on values/funcs/types (from `extractPosition`) and an `Imports` list (from `imports()`), so the overview deep-links symbols and lists dependencies without refetching or reparsing any `.gno` source. Test coverage is strong: per-helper unit tests, four handler tests (all-sections, qdoc-degraded, 404, routing matrix), a real-node `TestRoutes` case, and `File`/`Line` assertions in the qdoc handler test. CI is green except Merge Requirements (codeowner approval pending, not a code issue).

## Glossary

- **qdoc** — the `vm/qdoc` ABCI query: returns a package's documentation as JSON (`JSONDocumentation`). The overview's symbols, imports, and source line numbers all come from it.
- **degraded mode** — when the qdoc query fails the handler substitutes an empty `JSONDocumentation` and still renders the file list / README rather than erroring the whole page.

## Critical (must fix)

None

## Warnings (should fix)

- **[sidebar counts symbols the page never shows]** [`overview_build.go:41-48`](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/gnoweb/components/overview_build.go#L41-L48) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_build.go#L41-L48) — Code stats count unexported types/consts/vars, but only exported ones are rendered, so the numbers disagree with the page.
  <details><summary>details</summary>

  qdoc is queried with `unexported=true` ([keeper.go:1418](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/sdk/vm/keeper.go#L1418) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/sdk/vm/keeper.go#L1418)), so `jdoc.Values`/`jdoc.Types` include unexported declarations. `computeStats` counts all of them (`s.TypeCount = len(jdoc.Types)`; `VarCount`/`ConstCount` over every value group), while `buildSymbols`/`buildValues` drop unexported entries via `token.IsExported` before rendering. The result: the sidebar shows a count, but the matching section is shorter or absent. `Funcs` reconciles this with a "(N exported)" suffix; `Types`/`Consts`/`Vars` do not.

  Confirmed behaviorally on the current head (714d2f8): `/r/gnoland/blog$source` shows "Vars 3" in Code stats, but no Variables section renders, because all three var groups (`b`, the `errNot*` errors, the `adminAddr`/`moderatorList`/... block) are unexported. Repro in [comment](../3-714d2f8/comment_claude-opus-4-8.md).

  Fix: count only exported declarations in `computeStats` (mirror the `token.IsExported` filter the render path already uses), or label these stats as totals the way `Funcs` does.
  </details>

## Nits

- [`handler_http.go:585-601`](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/gnoweb/handler_http.go#L585-L601) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L585-L601) — `GetSourceView`'s "no file specified" fallback (prefer README, then `.gno`, then first file) is now unreachable: its only caller routes to it only when `IsFile()` or `file != ""` ([handler_http.go:311](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/gnoweb/handler_http.go#L311) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L311)), so the `else` branch never executes. The PR removed the tests that covered it (`TestHTTPHandler_GetSourceView_FilePreference`, `_NoFiles`) but left the code. Drop the dead branch, or keep `GetSourceView` reachable without a file.

- [`handler_http.go:855`](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/gnoweb/handler_http.go#L855) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L855) — Subdirectories are read from `ListPaths(..., 50)` and then filtered to direct children. A package with more than 50 descendant paths whose direct children sort late could have some children silently dropped from the Directories section. 50 is generous for today's packages; worth a comment noting the cap, or raise it to match `GetPathsListView`'s 1000.

## Missing Tests

- [`overview_build.go:14-50`](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/gnoweb/components/overview_build.go#L14-L50) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_build.go#L14-L50) — no test pins `computeStats` against a `jdoc` containing unexported symbols, which is why the count/render mismatch above slipped through. A case asserting the stat equals the rendered count would lock the intended behavior. See the adversarial test in [comment](../3-714d2f8/comment_claude-opus-4-8.md).

## Suggestions

- [`handler_http.go:849-851`](https://github.com/gnolang/gno/blob/714d2f8/gno.land/pkg/gnoweb/handler_http.go#L849-L851) · [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/handler_http.go#L849-L851) — the README goroutine discards `renderReadme`'s error (`readme, _ = ...`). That is the right degraded behavior (a missing README should not fail the page), but a one-line comment saying so would stop a future reader from "fixing" it into a hard error.

## Open questions

- jefft0 (PR thread): clicking a subdirectory like `/r/sys/users` lands on the file listing rather than the realm's Render page. The Directories links point at the bare package path (`{{ .Path }}`, no `$source`), so their routing is the existing `/r/<pkg>` behavior this PR does not touch; whether a subpackage shows Render or a file listing depends on whether that path has a `Render()` on the running node. Likely a devnet-data difference, but the author should confirm the intended target. Not posted — needs the author's product call, not a code change here.
- jefft0 (PR thread): the Copy button does nothing on `$source&file=README.md`. That is the unchanged source-code view, not the overview, so it is pre-existing behavior outside this PR's diff. Not posted for the same reason.
