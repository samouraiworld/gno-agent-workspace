# PR #5774: feat(gnoweb): change built-in playground and run views into features

URL: https://github.com/gnolang/gno/pull/5774
Author: jeronimoalbi | Base: playground2 | Files: 33 | +1165 -807
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: ca89afce6 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5774 ca89afce6`

**Verdict: APPROVE** — clean structural refactor that builds and passes the full `gnoweb` suite; the one genuine behavior change (`GetForkView` no longer errors on `ListFiles` failure) is intentional but mislabels the page, and the security items @alexiscolin raised are all pre-existing code carried over verbatim, not regressions. None block merge.

## Summary
Extracts the built-in playground and run views from the monolithic `gnoweb` package into self-contained `feature/playground` and `feature/run` sub-packages, each owning its own handler, view, template, CSS, and TS controller. The playground feature declares a local `ClientAdapter` interface so it does not import `gnoweb`, with `playgroundClientAdapter` in the parent bridging the type gap (`File` drops the `FileMeta` return). Eval/funcs JSON handlers and the per-IP token-bucket rate limiter move over essentially unchanged from the deleted `handler_playground.go`. Per-feature CSS is hoisted out of `06-blocks.css` (-313 lines) into co-located files re-imported in the same cascade slot; the Makefile gains `vpath`-based discovery of `feature/*/frontend/*.ts` plus a duplicate-basename guard.

## Glossary
- `GetForkView` — renders a package's source into the playground editor (`?fork` query).
- `ClientAdapter` — locally-declared subset of the gnoweb chain client the playground consumes; keeps `feature/playground` free of a `gnoweb` import.
- `playgroundClientAdapter` — parent-package bridge implementing `playground.ClientAdapter` over the real `gnoweb.ClientAdapter`.
- `ForkFrom` — `PlaygroundData` field that makes the page read "forked from <path>".

## Fix
Before: playground/run rendering, the eval/funcs API, and the rate limiter all lived in `gnoweb` (`handler_http.go` + `handler_playground.go`). After: two `feature/<name>` packages own their slices; `handler_http.go` holds only routing (`h.Playground.GetForkView`, `h.Run.GetRunView`) and the adapter, and `app.go` wires `/_/api/eval` and `/_/api/funcs` to `Playground.EvalHandler()`/`FuncsHandler()`. The load-bearing constraint is the local `ClientAdapter` interface ([`feature/playground/feature.go:14-28`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/feature.go#L14-L28) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/feature.go#L14)), which breaks the dependency cycle and is enforced by a compile-time assertion at [`handler_http.go:854`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/handler_http.go#L854) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/handler_http.go#L854) (added in `ca89afc`, addressing @alexiscolin's suggestion).

## Critical (must fix)
None.

## Warnings (should fix)
- **[fork page lies on fetch failure]** [@alexiscolin](https://github.com/gnolang/gno/pull/5774#discussion_r-fork) [`feature/playground/handler.go:60-72`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/handler.go#L60-L72) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/handler.go#L60) — when `ListFiles` fails, the page falls back to default code but still stamps `ForkFrom`, so it reads "forked from gno.land/r/foo" while showing the empty `Render` snippet.
  <details><summary>details</summary>

  This is a real behavior change introduced by the PR, not a carry-over. The old `GetForkView` returned `GetClientErrorStatusPage(gnourl, err)` on `ListFiles` failure (see base `handler_http.go:818-824`). The new version downgrades to a friendlier "write from scratch" fallback, which is a reasonable UX call, but it sets `ForkFrom: path.Join(h.deps.Domain, pkgPath)` in the same branch, so the rendered page claims a fork that never happened. Fix: either leave `ForkFrom` empty in the failure branch (silent degrade to a fresh playground) or add a `ForkFailed` flag so the template can show an honest "couldn't fetch source, starting fresh" hint.
  </details>

## Nits
- [`feature/playground/feature.go:18`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/feature.go#L18) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/feature.go#L18) — doc typo "fetche" → "fetches" (raised by @alexiscolin).
- [`feature/playground/template.go:16`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/template.go#L16) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/template.go#L16) and [`feature/run/template.go:16`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/run/template.go#L16) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/run/template.go#L16) — the `name` arg to `mustParse`/`template.New(name)` is dead: rendering goes through `ExecuteTemplate(w, "renderPage", …)`, which resolves the `{{ define "renderPage" }}` block, not the container name. Drop the param. The two `mustParse` helpers are also ~90% identical; at two features it's fine, but a shared `feature/internal/featuretmpl` helper would pay off if `feature/*` grows.
- [`feature/run/frontend/controller-run.ts:1`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L1) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L1) — `../../../frontend/js/code-editor.js` (three `../`) is brittle to any future move of `feature/`. A TS path alias (`"@gnoweb/js/*"` in `tsconfig.json`) would harden it (raised by @alexiscolin).

## Missing Tests
- **[new feature ships untested]** [`feature/run/handler.go`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/run/handler.go) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/run/handler.go) — `feature/run` has no `_test.go`; codecov reports `handler.go`/`component.go`/`view.go` at 0%.
  <details><summary>details</summary>

  `go test ./gno.land/pkg/gnoweb/...` confirms `feature/run [no test files]`. The package is small and purely client-side (no chain RPC), so one test asserting `GetRunView` returns 200 with the expected `RunData` (`PkgPath`, `PkgAlias`, `Domain`, `Remote`, `ChainId`) would close all three files. The playground feature, by contrast, carries `TestHandlerPlaygroundEval`, `TestHandlerPlaygroundFuncs`, and `TestRateLimiter`.
  </details>

## Suggestions
- [`feature/playground/handler.go:33`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/handler.go#L33) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/handler.go#L33), [`:157`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/handler.go#L157) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/handler.go#L157), [`ratelimit.go:88`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) — bundle the cheap security hardening @alexiscolin flagged while the code is open: `io.LimitReader` on the flate stream (decompression-bomb cap), `http.MaxBytesReader` on the eval body, a shared limiter on `/_/api/funcs`, and an opt-in/documented X-Forwarded-For trust model.
  <details><summary>details</summary>

  Verified against base `handler_playground.go`: every one of these is pre-existing code moved verbatim by this refactor, so none is a regression and none blocks a structural-refactor PR. They are still genuine and now have a natural home in `feature/playground`. The X-Forwarded-For one is the most consequential: [`clientIP`](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L87) trusts the first XFF hop unconditionally, so without a trusted XFF-terminating proxy in front an attacker rotates the spoofed IP per request and bypasses the eval rate limit entirely. Lowest-risk fix is a one-line deployment note; strictly safer is config-gated XFF trust defaulting to `RemoteAddr`.
  </details>
- [`feature/playground/handler.go:76-91`](https://github.com/gnolang/gno/blob/ca89afce6/gno.land/pkg/gnoweb/feature/playground/handler.go#L76-L91) · [↗](../../../../../.worktrees/gno-review-5774/gno.land/pkg/gnoweb/feature/playground/handler.go#L76) — `GetForkView` fetches files with one sequential `Client.File` RPC per file; a 20-file realm blocks the request on 20 round-trips on a cold cache. Pre-existing; parallelizing is optional (raised by @alexiscolin).

## Questions for Author
- `GetForkView` fallback: is stamping `ForkFrom` on the `ListFiles`-failure path intentional, or should the page degrade to a fresh playground with no fork label?
- `ratelimit.go` `pruneLoop` starts an unstoppable goroutine in `New` with no `Close()`/`context`; fine at one Handler per process, but the per-feature layout could spawn multiples later — worth an escape hatch now, or defer?
