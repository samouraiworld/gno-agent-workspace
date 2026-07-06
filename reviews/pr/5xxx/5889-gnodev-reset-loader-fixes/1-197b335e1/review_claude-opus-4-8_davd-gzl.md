# PR [#5889](https://github.com/gnolang/gno/pull/5889): fix(gnodev): reset to initial package set; loader root + slice fixes

URL: https://github.com/gnolang/gno/pull/5889
Author: davd-gzl | Base: master | Files: 7 | +179 -7
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 197b335e1 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5889 197b335e1`

**TL;DR:** gnodev follow-up to [#5604](https://github.com/gnolang/gno/pull/5604) with three fixes: a reset (Ctrl+R or the `/reset` endpoint) now drops packages you browsed during the session and returns to the packages gnodev started with, the `-root` flag now feeds both the node and the package loader, and an import filter stops writing over a package's own import slice.

**Verdict: REQUEST CHANGES** — the loader half of the reset fix is correct and verified live, but the `/reset` HTTP endpoint (`-unsafe-api`) is left half-reset: it drops the browsed package from the loader without clearing the app's `pathManager`, so re-browsing a package that was loaded before the reset returns 404 and never recovers. Ctrl+R is unaffected. The `-root` and slice fixes are fine.

## Summary
Before this PR the loader's `tracked` set accumulated every lazily-browsed package and never reset, so a reset kept redeploying everything browsed since boot instead of returning to the initial set. The fix adds `Loader.seeded` (the setup-time package set) and `Loader.ResetTracked`, which restores `tracked` to `seeded` and evicts browsed packages from the index, wired through a new `NodeConfig.ResetState` hook that `Node.Reset` calls before rebuilding genesis. The eviction is what forces a re-browse to re-fetch. This half works. But the App layer keeps a second path-tracking set, `pathManager`, that the proxy consults to dedup browsed paths, and only the Ctrl+R handler clears it. The `/reset` endpoint calls `devNode.Reset` alone, so after a reset the proxy still thinks a previously-browsed path is known, short-circuits the re-registration, and the node never redeploys it.

## Fix
The `/reset` handler at [`app.go:475-480`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/app.go#L475-L480) · [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/app.go#L475) calls only `ds.devNode.Reset`, while the Ctrl+R handler at [`app.go:614-623`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/app.go#L614-L623) · [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/app.go#L614) first runs `ds.pathManager.Reset()` and `ds.devNode.SetPackagePaths(ds.paths...)`. The proxy callback at [`app.go:389-426`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/app.go#L389-L426) · [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/app.go#L389) skips any path where `ds.pathManager.Save(path)` reports the path already known, so a stale `pathManager` entry blocks re-registration and the node is never told to redeploy.

The fix factors the three-step reset the Ctrl+R handler already runs into a shared `App.resetState` method (`pathManager.Reset()`, `SetPackagePaths(ds.paths...)`, `devNode.Reset`) and points both `/reset` and Ctrl+R at it, so the two entry points cannot drift again:

```diff
 	if ds.cfg.unsafeAPI {
 		mux.HandleFunc("/reset", func(res http.ResponseWriter, req *http.Request) {
-			if err := ds.devNode.Reset(req.Context()); err != nil {
+			if err := ds.resetState(req.Context()); err != nil {
 				ds.logger.Error("failed to reset", slog.Any("err", err))
 				res.WriteHeader(http.StatusInternalServerError)
 			}
 		})

+// resetState returns the node to the package set gnodev started with. It clears
+// the app-level pathManager and re-seeds the node's package paths before calling
+// Reset, so a package browsed during the session is dropped from both the loader
+// and the path layer and can be browsed back afterward. Shared by every reset
+// entry point (Ctrl+R and the /reset endpoint) so they cannot drift.
+func (ds *App) resetState(ctx context.Context) error {
+	ds.pathManager.Reset()
+	ds.devNode.SetPackagePaths(ds.paths...)
+	return ds.devNode.Reset(ctx)
+}

 	case rawterm.KeyCtrlR: // Reset
 		ds.logger.WithGroup(NodeLogName).Info("resetting node state...")
-		ds.pathManager.Reset()
-		ds.devNode.SetPackagePaths(ds.paths...)
-		if err = ds.devNode.Reset(ctx); err != nil {
+		if err = ds.resetState(ctx); err != nil {
 			ds.logger.WithGroup(NodeLogName).Error("unable to reset node state", "err", err)
 		}
```

Verified live on 197b335e1 by A/B against master: on master a reset kept the browsed package (re-browse → 200), on the unpatched branch the same reset+re-browse returns 404 permanently, while a package first browsed *after* the reset loads normally. This isolates the cause to the `pathManager` dedup, not the loader. With the patch above applied and gnodev rebuilt, the identical sequence (browse `r/sys/users` → `/reset` → re-browse) returns 200 at every step, and the gnodev log shows the re-browse producing a fresh `new monitored path` + `reload done` for the package instead of silently deduping it.

## Glossary
- MemPackage: in-memory set of a package's source files, the unit loaded, type-checked, and run.

## Warnings (should fix)
- **[/reset leaves a browsed package unreachable]** `contribs/gnodev/app.go:475-480` — the `/reset` endpoint drops the browsed package from the loader but not from `pathManager`, so re-browsing a package loaded before the reset returns 404 and never recovers.
  <details><summary>details</summary>

  The `/reset` HTTP handler (enabled by `-unsafe-api`) calls `ds.devNode.Reset` alone. `Node.Reset` runs the new `ResetState` hook, which evicts browsed packages from the loader's index and `tracked` set, so the rebuilt genesis omits them. That much is correct. But the proxy's path callback dedups against `ds.pathManager`, and `/reset` never clears it. When the browser re-requests the package, the callback resolves it in the loader (the eviction forces a re-fetch, so `Resolve` succeeds) but then hits `if exist := ds.pathManager.Save(path); exist { continue }` and returns without calling `SetPackagePaths`/`AddPackagePaths`, so the node never redeploys it. The package stays 404 for the rest of the session. The Ctrl+R handler avoids this because it calls `ds.pathManager.Reset()` and `ds.devNode.SetPackagePaths(ds.paths...)` before `Reset`. The node.go comment at [`node.go:44-46`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/pkg/dev/node.go#L44-L46) · [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/pkg/dev/node.go#L44) states the hook makes "every reset entry point (Ctrl+R, /reset)" return to the initial set; the package set is restored, but on `/reset` the app's path layer is left inconsistent so the restored package can't be browsed back. Fix: factor the Ctrl+R reset sequence (`pathManager.Reset()`, `SetPackagePaths(ds.paths...)`, `devNode.Reset`) into a shared `App.resetState` and call it from both `/reset` and Ctrl+R, so they can't drift. See the Fix section for the diff.

  Live-verified with the fix applied: the repro sequence below (browse → `/reset` → re-browse) returns 200 at every step instead of 404 on the re-browse, and the gnodev log shows the re-browse emitting a fresh `new monitored path` + `reload done` for the package. Reverting only `pathManager.Reset()` from `resetState` reproduces the 404 (and fails the shipped test).

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5889 -R gnolang/gno
  ( cd contribs/gnodev && go build -o /tmp/gnodev-5889 . )
  GNOROOT="$PWD" /tmp/gnodev-5889 local -root "$PWD" \
    -web-listener 127.0.0.1:18888 -node-rpc-listener 127.0.0.1:18657 \
    -unsafe-api=true -log-format json >/tmp/gnodev-5889.log 2>&1 &
  GNODEV_PID=$!
  until curl -s -o /dev/null http://127.0.0.1:18888/; do sleep 0.5; done
  B=http://127.0.0.1:18888
  echo "browse r/sys/users (lazy load):"; curl -s -o /dev/null -w "  %{http_code}\n" "$B/r/sys/users"
  echo "hit /reset:";                     curl -s -o /dev/null -w "  %{http_code}\n" "$B/reset"; sleep 2
  echo "re-browse r/sys/users:";          curl -s -o /dev/null -w "  %{http_code}\n" "$B/r/sys/users"
  echo "browse a fresh r/gnoland/pages:"; curl -s -o /dev/null -w "  %{http_code}\n" "$B/r/gnoland/pages"
  kill "$GNODEV_PID"
  ```
  ```
  browse r/sys/users (lazy load):
    200
  hit /reset:
    200
  re-browse r/sys/users:
    404          # bug: browsed-before-reset package is gone for good
  browse a fresh r/gnoland/pages:
    200          # a package first seen after reset still loads
  ```
  On master the same sequence returns 200 for the re-browse (the reset kept the package), confirming the regression is on the `/reset` path introduced here.
  </details>

## Critical (must fix)
None.

## Nits
- `contribs/gnodev/pkg/dev/node.go:45` — the comment claims "every reset entry point (Ctrl+R, /reset) returns the package set to its initial state"; true for the loader, but on `/reset` the package can't be browsed back (see the Warning). The shared-`resetState` fix makes both entry points restore the browseable set, so the comment holds as written with no source edit needed.

## Missing Tests
- **[reset re-browse path is untested]** `contribs/gnodev/app.go:475` — no test exercises a reset followed by a re-browse of a previously-loaded package; the unit tests cover `ResetTracked` and the `ResetState` wiring in isolation, so the App-layer `pathManager` gap slips through green.
  <details><summary>details</summary>

  `TestLoader_ResetTracked_*` and `TestNode_Reset_InvokesResetState` verify the loader and the hook, but nothing covers the App-level reset path where the `pathManager` dedup lives. The shipped `TestGnodev_Reset_RebrowseRedeploys` ([reset_rebrowse_test.go](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5889-gnodev-reset-loader-fixes/1-197b335e1/tests/reset_rebrowse_test.go?plain=1#L1) · [↗](../tests/reset_rebrowse_test.go)) stands up a real App, browses an examples package that is absent from the initial genesis set, calls the shared `resetState`, and asserts the re-browse re-registers and redeploys the package. It is red→green: dropping `pathManager.Reset()` from `resetState` fails it (the re-browse hits the dedup), and it passes with the fix. Lives in `package main` alongside `app_test.go`; run with `GNOROOT="$PWD" go test -run TestGnodev_Reset_RebrowseRedeploys ./contribs/gnodev/`.
  </details>

## Suggestions
None.

## Open questions
- The slice-aliasing fix in `filterSourceImports` ([`loader.go:678-697`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/pkg/packages/loader.go#L678-L697) · [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/pkg/packages/loader.go#L678)) is a safe hardening: `stripStdlibs` and `dropMissingDepImports` both filter the same `vmpackages.Package`, and copying into a fresh slice removes any in-place backing-array reuse. Not posting: no reproduced mis-behavior on the current call graph, and the change can't regress. Note `stripStdlibs` itself still uses `out := pkgs[:0]` at [`loader.go:721`](https://github.com/gnolang/gno/blob/197b335e1/contribs/gnodev/pkg/packages/loader.go#L721) · [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/pkg/packages/loader.go#L721) to filter the top-level `PkgList` in place, which is fine since the input is a freshly-built list, but worth knowing the pattern the PR removes for imports is still present one level up.
