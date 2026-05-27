# PR #5384: fix(gnovm): recover from preprocessing panics on node restart

URL: https://github.com/gnolang/gno/pull/5384
Author: davd-gzl | Base: master | Files: 6 | +297 -24
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `5de8d0a` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5384 5de8d0a`

**Verdict: NEEDS DISCUSSION** — recovery logic is sound and well-tested, but two design choices need clarification: (1) the gnoweb broken-package detection swallows every non-specific `Realm()`/`Doc()` error into "Package Unavailable" — masking ordinary `Render()` panics and transient RPC failures; (2) `keeper.Initialize` treats stdlib preprocess panics as non-fatal despite the ADR claiming they remain fatal. Self-review by author — flagging openly.

## Summary

`PreprocessAllFilesAndSaveBlockNodes` previously ran all `MemPackage`s in one stack: a single bad package crashed `gnoland` on restart (cf. [#5238](https://github.com/gnolang/gno/issues/5238)). This PR moves each package into its own function with `defer recover()`, returns the list of failed paths, the keeper logs a warning per failure and continues. Gnoweb learns a "Package Unavailable" status (HTTP 503 + link to source) and shows it whenever `Realm()`/`Doc()` errors *and* `ListFiles()` reports files for that path.

```
restart flow:
  ms.IterMemPackage()
    └─ for each mpkg:
         preprocessMemPackage(mpkg)              <-- new isolation wrapper
           defer recover()  -> rerr = "preprocess <path>: <panic>"
           ParseMemPackage / SetBlockNode / PredefineFileSet / Preprocess / Save
         if err: failed = append(failed, path)
  logger.Warn("package preprocessing failed", ...) per entry
  stdlib type-check loop (unchanged) ───────────  <-- ADR says fatal, code does not enforce
  logger.Debug("preprocessed", failures=len(failed))
```

## Glossary

- `PreprocessAllFilesAndSaveBlockNodes` — restart-time pass that re-parses every persisted `MemPackage` and re-populates the in-memory cache of `BlockNode`s (types, decls). Called only from `VMKeeper.Initialize`.
- `doRecoverQuery` — existing keeper-side panic catcher for query (read-only) execution paths. Already wraps the qrender / qdoc handlers.
- `ListFiles` — gnoweb-side wrapper around the `vm/qfile` ABCI query that returns the file names persisted in the store. Independent of preprocessing.

## Fix

Before: [`PreprocessAllFilesAndSaveBlockNodes`](https://github.com/gnolang/gno/blob/5de8d0a/gnovm/pkg/gnolang/machine.go#L245-L254) · [↗](../../../../../.worktrees/gno-review-5384/gnovm/pkg/gnolang/machine.go#L245-L254) was a single inline loop with no recover; one panic killed the whole restart. After: each iteration delegates to [`preprocessMemPackage`](https://github.com/gnolang/gno/blob/5de8d0a/gnovm/pkg/gnolang/machine.go#L256-L287) · [↗](../../../../../.worktrees/gno-review-5384/gnovm/pkg/gnolang/machine.go#L256-L287) which holds a `defer recover()` and returns an `error`; `PreprocessAllFilesAndSaveBlockNodes` collects failures and returns `[]string`. [`VMKeeper.Initialize`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/sdk/vm/keeper.go#L166-L172) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/sdk/vm/keeper.go#L166-L172) logs each failure at `Warn`, includes a `failures` count in the summary `Debug` line, and proceeds. Gnoweb's [`GetRealmView`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/gnoweb/handler_http.go#L338-L348) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/gnoweb/handler_http.go#L338-L348) and [`GetHelpView`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/gnoweb/handler_http.go#L475-L485) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/gnoweb/handler_http.go#L475-L485) gain a default-case probe: on any other error, call `ListFiles()` and if files exist, return the new `StatusPackageBrokenComponent` (503).

## Critical (must fix)

None.

## Warnings (should fix)

- **[broken-package detection is too greedy]** [`gno.land/pkg/gnoweb/handler_http.go:339-345`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/gnoweb/handler_http.go#L339-L345) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/gnoweb/handler_http.go#L339-L345) — every non-specific `Realm()` error with files present is reported as a broken package, including ordinary user `Render()` panics and transient RPC errors.
  <details><summary>details</summary>

  The default branch in `GetRealmView` runs when `err` is *not* `ErrClientRenderNotDeclared` and not `ErrClientPackageNotFound`. That fallthrough catches a lot more than "package is broken because preprocessing failed":
  - `Render(path)` panicking in user code (nil deref, divide-by-zero, runtime panic in a library call) is caught by `doRecoverQuery` ([`keeper.go:1294`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/sdk/vm/keeper.go#L1294) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/sdk/vm/keeper.go#L1294)) and returned as a generic VM error — `query()` falls through to the `default` at [`client.go:234`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/gnoweb/client.go#L234) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/gnoweb/client.go#L234) and surfaces a wrapped error, not one of the typed errors.
  - `ErrClientBadRequest` / `ErrClientTimeout` (transient network/RPC failure) likewise don't match `ErrClientRenderNotDeclared` or `ErrClientPackageNotFound`.

  In both cases the package's files are still present (because the qrender failure doesn't mean qfile fails), so `ListFiles()` returns >0 and the user is shown "Package Unavailable. It may be outdated or contain incompatible code." That's misleading: the package is fine, the *Render() call* panicked on the given args, or the node is briefly unreachable.

  Fix: tighten the trigger. Either match the actual preprocessing-failure error shape from the keeper (introduce a typed error returned by qrender/qdoc when the package was in `failed`, or piggy-back on the existing `ErrInvalidPkgPath` variants), or — minimally — only run the `ListFiles` probe when `errors.Is(err, ErrClientResponse)` and the response message matches a "preprocess" prefix. The current "any error + files present = broken" is a UX regression that hides real per-call failures behind a generic page.
  </details>

- **[stdlib preprocess panic is silently downgraded]** [`gno.land/pkg/sdk/vm/keeper.go:166-194`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/sdk/vm/keeper.go#L166-L194) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/sdk/vm/keeper.go#L166-L194) — ADR says stdlib failures stay fatal; code does not enforce that.
  <details><summary>details</summary>

  The PR description / ADR ([`pr5384_graceful_preprocess_recovery.md:32-35`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/adr/pr5384_graceful_preprocess_recovery.md#L32-L35) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/adr/pr5384_graceful_preprocess_recovery.md#L32-L35)) says "Stdlib type checking remains unchanged — stdlib failures are still fatal, since they indicate a fundamental incompatibility." That's true for the *type-check* loop at L180-187, but the *preprocess* loop iterates **every** `MemPackage` including stdlibs ([`IterMemPackage`](https://github.com/gnolang/gno/blob/5de8d0a/gnovm/pkg/gnolang/store.go#L1073-L1099) · [↗](../../../../../.worktrees/gno-review-5384/gnovm/pkg/gnolang/store.go#L1073-L1099) walks all `backendPackageIndex` entries; nothing filters stdlibs). If a stdlib's `Preprocess` step panics — say a future GnoVM change broke `crypto/sha256` parsing — the new wrapper swallows it and the keeper just logs `Warn`. The subsequent stdlib type-check at L180-187 may still pass (it goes through a separate `TypeCheckMemPackage` codepath against fresh source), and the node boots with a half-initialized stdlib whose `BlockNode`s never made it into the cache. Any later realm transaction or query that imports that stdlib panics at runtime, caught by `doRecover*` — meaning every TX touching that stdlib silently fails.

  Fix: when iterating `failed`, classify by path. If the path is a stdlib (no leading `gno.land/`, or matches `stdlibs.InitOrder()`), `panic` instead of `logger.Warn`. Alternatively, surface `failed` to the type-check loop and bail there if any required stdlib is in the failed list.
  </details>

## Nits

- [`gno.land/pkg/sdk/vm/keeper.go:192-194`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/sdk/vm/keeper.go#L192-L194) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/sdk/vm/keeper.go#L192-L194) — the `failures` count is at `Debug` level. Operators won't see it in normal production logging (info+). The per-failure `Warn` lines do surface, but a one-line "X packages failed preprocessing" at `Info` (or `Warn` when `len(failed) > 0`) makes drift visible without scanning each line. Minor since per-package warns already exist.
- [`gnovm/pkg/gnolang/machine.go:261`](https://github.com/gnolang/gno/blob/5de8d0a/gnovm/pkg/gnolang/machine.go#L261) · [↗](../../../../../.worktrees/gno-review-5384/gnovm/pkg/gnolang/machine.go#L261) — error string `"preprocess %s: %v"` loses panic stack. Considered fine for restart logs (per-call stack would be huge), but a `runtime/debug.Stack()` at `Debug` level next to the `Warn` could help post-mortem. Optional.
- [`gnovm/pkg/gnolang/machine_test.go:138-156`](https://github.com/gnolang/gno/blob/5de8d0a/gnovm/pkg/gnolang/machine_test.go#L138-L156) · [↗](../../../../../.worktrees/gno-review-5384/gnovm/pkg/gnolang/machine_test.go#L138-L156) — `for range 1000 {}` reads as "intentional crash" but the actual mechanism is the `range 1000` int-range syntax not being supported in `MPStdlibAll` preprocessing context. Comment "// broken first so the panic path runs before good is preprocessed" already explains the ordering — adding one line on *why* this particular body panics would save the next reader a `git blame`.

## Missing Tests

- **[gnoweb broken-package paths have 0% coverage]** [`gno.land/pkg/gnoweb/handler_http.go:339-345`, `:476-482`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/gnoweb/handler_http.go#L339-L345) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/gnoweb/handler_http.go#L339-L345) — codecov flags 11 missing lines in `view_status.go` and 8 missing in `handler_http.go`. No test covers either of the two new branches.
  <details><summary>details</summary>

  The existing `stubClient` in [`handler_http_test.go`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/gnoweb/handler_http_test.go#L44-L80) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/gnoweb/handler_http_test.go#L44-L80) already supports configurable `Realm`/`Doc`/`ListFiles` returns — a table-driven test with cases `{realmErr: generic, listFiles: ["a.gno"]} -> 503 + "Package Unavailable"` and `{realmErr: generic, listFiles: nil} -> existing error page` adds two cases and pins the contract. Without these tests the broken-package branch can quietly stop firing under refactor and we won't notice.
  </details>

- **[keeper-level integration not tested]** [`gno.land/pkg/sdk/vm/keeper.go:166-172`](https://github.com/gnolang/gno/blob/5de8d0a/gno.land/pkg/sdk/vm/keeper.go#L166-L172) · [↗](../../../../../.worktrees/gno-review-5384/gno.land/pkg/sdk/vm/keeper.go#L166-L172) — the GnoVM-level test verifies `failed` is populated; nothing exercises the keeper consuming it.
  <details><summary>details</summary>

  The PR description includes a hand-crafted `test_preprocess_recovery.sh` that does an end-to-end node restart with an injected panic. That script is a good *manual* reproduction but won't run in CI. A `keeper_test.go` test that constructs a `VMKeeper`, seeds a `MemPackage` whose preprocessing panics, calls `Initialize`, and asserts (a) `Initialize` returns without panicking, (b) the failed pkgPath was logged, would catch regressions in the keeper-side log/skip wiring. Tests can use a captured `slog.Logger` to assert on the warn record.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/machine.go:256-287`](https://github.com/gnolang/gno/blob/5de8d0a/gnovm/pkg/gnolang/machine.go#L256-L287) · [↗](../../../../../.worktrees/gno-review-5384/gnovm/pkg/gnolang/machine.go#L256-L287) — consider returning `map[string]error` instead of `[]string`. The actual recovered panic value is discarded after the `fmt.Errorf` lossy format; the keeper can only log "package failed" with no detail beyond the path. Surfacing the error message all the way to the keeper would let operators see *why* a package failed without rummaging through `gnoland` logs.
  <details><summary>details</summary>

  Signature change is minor (`[]string` → `map[string]error` or `[]struct{Path string; Err error}`), keeper iteration trivial, no caller outside `Initialize`. Helps when a node operator sees `package preprocessing failed pkgPath=gno.land/r/foo/bar` and needs to file a bug — without the actual panic message they have to reproduce it locally.
  </details>

- [`gnovm/pkg/gnolang/machine.go:268-275`](https://github.com/gnolang/gno/blob/5de8d0a/gnovm/pkg/gnolang/machine.go#L268-L275) · [↗](../../../../../.worktrees/gno-review-5384/gnovm/pkg/gnolang/machine.go#L268-L275) — partial state from a panicked package (`SetBlockNode` + partial `PredefineFileSet` writes) is left in `m.Store.cacheNodes`. The ADR acknowledges this and the test asserts the partial `PackageNode` is present after recovery. The current behavior depends on every downstream consumer being defensive (caught by `doRecover*`). If the goal is true isolation, the recover could also evict the partial nodes via `store.DelBlockNode` on rollback. Not blocking; documented in the ADR as a trade-off.

## Questions for Author

- Should the broken-package gnoweb branch be gated on a specific error shape from the keeper, rather than "any non-typed error + files exist"? See first Warning.
- Is the stdlib-non-fatal behavior intentional, or a gap to close? See second Warning.
- Was the manual `test_preprocess_recovery.sh` script considered for porting into a `keeper_test.go` integration test, or is the cost (binary build, port plumbing) too high?
- Should the merge gate be relaxed? CI status shows `Merge Requirements: failure` because the gnoweb codeowners (`@alexiscolin` / `@gfanton`) haven't reviewed — the bot is correct that gnoweb component changes need their sign-off before merge.
