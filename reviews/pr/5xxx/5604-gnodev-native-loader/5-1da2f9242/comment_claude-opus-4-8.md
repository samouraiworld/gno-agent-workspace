# Review: PR #5604
Event: APPROVE

## Body
The loader migration delegates cleanly to gnovm's native loader and the four follow-up commits are correct. Two CI-invisible checks on 1da2f9242. First: a workspace realm importing an unresolvable path boots network-free, failing locally with `remote fetching is disabled`, and only `-remote gno.land=<rpc>` turns the same import into a `qfile` query against the chain (repro below). Second: running gnodev from a non-package subdirectory of a bare-`gnomod.toml` realm boots in discovery mode, where restoring the previous `FindWorkspace` marker walk reproduces the `gnomod.toml doesn't exists in current directory` node-init crash.

<details><summary>repro: opt-in remote gate</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5604 -R gnolang/gno
export GNOROOT=$PWD
go build -o /tmp/gnodev ./contribs/gnodev
mkdir -p /tmp/ws/app && : > /tmp/ws/gnowork.toml
printf 'module = "gno.land/r/davdtest/app"\ngno = "0.9"\n' > /tmp/ws/app/gnomod.toml
printf 'package app\nimport "gno.land/r/davdtest/remoteonly"\nfunc Render(p string) string { return remoteonly.Hello() }\n' > /tmp/ws/app/app.gno
( cd /tmp/ws && timeout 25 /tmp/gnodev local -v -log-format json -node-rpc-listener 127.0.0.1:36661 -web-listener 127.0.0.1:38891 ) 2>&1 | grep -iE "remote fetching is disabled|qfile"
echo "--- now with -remote ---"
( cd /tmp/ws && timeout 25 /tmp/gnodev local -v -log-format json -remote gno.land=https://rpc.gno.land -node-rpc-listener 127.0.0.1:36662 -web-listener 127.0.0.1:38892 ) 2>&1 | grep -iE "remote fetching is disabled|qfile"
rm -rf /tmp/ws /tmp/gnodev
```

```
"err":"...: remote fetching is disabled, pass -remote <domain>=<rpc> to fetch \"gno.land/r/davdtest/remoteonly\" from a chain"
--- now with -remote ---
"err":"...: query files list for pkg \"gno.land/r/davdtest/remoteonly\": qfile failed: invalid package ... not available"
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5604-gnodev-native-loader/5-1da2f9242/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnodev/pkg/dev/node.go:316 [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L316)
Ctrl+R rebuilds genesis from `loader.Reload()`, which redeploys the loader's `tracked` set; `tracked` accumulates every resolved path and is never cleared, so realms browsed during a session survive the reset instead of being dropped. That also makes the handler's `pathManager.Reset()` + `SetPackagePaths` (app.go:614-615) dead, since `n.paths` now only feeds the webHome default. Either clear and re-seed `tracked` on Reset, or drop the dead calls and document that Reset keeps the loaded package set.

## contribs/gnodev/pkg/packages/examples_check.go:46 [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/examples_check.go#L46)
The `-no-examples` pre-flight diagnostic resolves imports only through the FS roots, so a `gno.land/*` import already present in the modcache is reported as unresolvable even though it would load. Skip modcache-resolvable paths, or note in the warning that they may still load.

## contribs/gnodev/pkg/packages/package_test.go:14 [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/package_test.go#L14)
No test calls `ToMemPackage()` twice across an on-disk edit to lock the documented "re-read on every call" invariant that hot reload depends on; a future memoization of the FS path would silently break reload and still pass every test. Add a write → read → rewrite → read assertion.
