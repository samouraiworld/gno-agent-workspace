# Review: PR [#5889](https://github.com/gnolang/gno/pull/5889)
Event: REQUEST_CHANGES

## Body
The loader half of the reset fix works. The `/reset` HTTP endpoint calls `devNode.Reset` alone, which drops the browsed package from the loader but never clears the app's `pathManager`. A re-browse then resolves in the loader but hits the `pathManager.Save` dedup and returns without redeploying, so a package loaded before the reset stays 404 for the session. Ctrl+R avoids this because it resets `pathManager` and re-seeds the node paths before reloading. Have `/reset` and Ctrl+R run the same reset sequence, clearing `pathManager` and re-seeding node paths in both, so the two paths can't drift.

Verified live on 197b335e1: after `/reset` a re-browse of a package loaded before the reset returns 404 for the session, while a package first browsed after the reset loads normally. With the shared-reset fix applied, the same browse, `/reset`, re-browse sequence returns 200 at every step and redeploys the package. Reverting only the `pathManager.Reset()` call reproduces the 404.

Missing test: a reset followed by a re-browse of a previously-loaded package. Loader-level tests pass while this App-level reset path is uncovered.

<details><summary>repro</summary>

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
  404
browse a fresh r/gnoland/pages:
  200
```
On master the re-browse returns 200, so the regression is on the `/reset` path added here.
</details>

<details><summary>test cases</summary>

App-level regression guard for the reset re-browse path. Lives in `package main` alongside `contribs/gnodev/app_test.go`. Red→green: dropping `pathManager.Reset()` from `resetState` fails the final assertion; the fix makes it pass. Run: `GNOROOT="$PWD" go test -run TestGnodev_Reset_RebrowseRedeploys ./contribs/gnodev/`.

```go
// browseLazyPath mirrors the proxy path callback: resolve, dedup against
// pathManager, re-seed node paths, reload. exist == false means the path was
// re-registered and the node reloaded.
func browseLazyPath(t *testing.T, ctx context.Context, app *App, path string) (registered bool) {
	t.Helper()
	_, err := app.loader.Resolve(path)
	require.NoError(t, err, "path must resolve for a lazy browse")
	if exist := app.pathManager.Save(path); exist {
		return false // stale entry: proxy short-circuits, package never redeploys
	}
	app.devNode.SetPackagePaths(app.paths...)
	app.devNode.AddPackagePaths(app.pathManager.List()...)
	require.NoError(t, app.devNode.Reload(ctx))
	return true
}

func TestGnodev_Reset_RebrowseRedeploys(t *testing.T) {
	if os.Getenv("GNOROOT") == "" {
		t.Skip("needs GNOROOT for the examples root")
	}
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "gnowork.toml"), []byte(""), 0o644))
	writeWorkspacePkg(t, filepath.Join(workspace, "w"), "gno.land/p/ws/only", "package only\n")
	t.Chdir(workspace)

	// Examples package: browsable lazily, absent from the initial genesis set.
	const browsed = "gno.land/p/demo/nestedpkg"

	cfg := defaultLocalAppConfig
	cfg.root = os.Getenv("GNOROOT")
	cfg.home = filepath.Join(t.TempDir(), "nokeybase")
	cfg.nodeRPCListenerAddr = "127.0.0.1:0"
	cfg.noWatch = true

	ctx := context.Background()
	app := NewApp(discardLogger(), &cfg, commands.NewTestIO())
	require.NoError(t, app.Setup(ctx))
	t.Cleanup(app.Close)

	require.NotContains(t, importPaths(app.devNode.ListPkgs()), browsed)

	require.True(t, browseLazyPath(t, ctx, app, browsed))
	require.True(t, app.devNode.HasPackageLoaded(browsed))

	require.NoError(t, app.resetState(ctx))
	assert.False(t, app.devNode.HasPackageLoaded(browsed), "reset drops the browsed package")

	// The fix: pathManager was cleared, so the re-browse re-registers and redeploys.
	require.True(t, browseLazyPath(t, ctx, app, browsed),
		"re-browse after reset must re-register the path (pathManager was left dirty)")
	assert.True(t, app.devNode.HasPackageLoaded(browsed),
		"re-browse after reset must redeploy the package")
}
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5889-gnodev-reset-loader-fixes/1-197b335e1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnodev/pkg/dev/node.go:44-46 [↗](../../../../../.worktrees/gno-review-5889/contribs/gnodev/pkg/dev/node.go#L44)
This says every reset entry point returns the package set to its initial state. That holds for the loader, but on `/reset` a package loaded before the reset can no longer be browsed back, so the claim overstates `/reset` today. Reword it to cover only what the loader restores until both entry points share one reset sequence.
