# Review: PR #5604
Event: APPROVE

## Body
Re-verified on 5038db249 (d2d0da30e + 448343c3b are in), following up on the network-fetch thread: the concern is resolved. With no `-remote`, `newRemoteFetcher` returns a `disabledFetcher` that refuses every fetch, wired into both the lazy `rpcLookup` and the eager `gnovm.Load` (now handed a non-nil `Fetcher`, so gnovm no longer builds its own rpc fallback). Live: a realm importing an unresolvable path boots network-free, failing locally with `remote fetching is disabled, pass -remote ...`, and only `-remote gno.land=<rpc>` makes it issue a `qfile` query to the chain (repro below). The `r/samcrew/daodemo` phantom is the `examples/quarantined` disk copy that `-extra-root $GNOROOT/examples` loads wholesale (type-checks clean here), not a network fetch. The genesis `users/init.Bootstrap` tx fires only when that realm is in the package set.

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

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5604-gnodev-native-loader/4-5038db249/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnodev/pkg/packages/workspace.go:16 [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/workspace.go#L16)
Running gnodev from a non-package subdir of a single-package realm logs `workspace detected` then crashes at node init, because `FindWorkspace` promotes an ancestor `gnomod.toml` to a workspace while gnovm honors `gnomod.toml` only in the current dir. Walk up only for `gnowork.toml`, and accept `gnomod.toml` solely when it sits in the start dir, matching gnovm's loader context.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5604 -R gnolang/gno
export GNOROOT=$PWD
go build -o /tmp/gnodev ./contribs/gnodev
mkdir -p /tmp/single/sub
printf 'module = "gno.land/r/davdtest/single"\ngno = "0.9"\n' > /tmp/single/gnomod.toml
printf 'package single\nfunc Render(p string) string { return "hi" }\n' > /tmp/single/single.gno
( cd /tmp/single/sub && /tmp/gnodev local ) 2>&1 | grep -iE "workspace detected|unable to initialize"
rm -rf /tmp/single /tmp/gnodev
```

```
INF workspace detected ... root=/tmp/single
unable to initialize the node: reload packages: load packages: gnowork.toml file not found in current or any parent directory and gnomod.toml doesn't exists in current directory
```
</details>

## contribs/gnodev/pkg/dev/node.go:316 [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L316)
Reset rebuilds genesis from `loader.Reload()`, which deploys the loader's `tracked` set; `tracked` accumulates every resolved path and is never cleared, so realms browsed during a session survive Ctrl+R. The handler's `pathManager.Reset()` + `SetPackagePaths(ds.paths)` (app.go:614-615) no longer affect the deployed set, since `n.paths` now only feeds the webHome default. Either clear and re-seed `tracked` on Reset, or drop the dead calls and document that Reset keeps the loaded package set.

## contribs/gnodev/pkg/packages/examples_check.go:46 [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/examples_check.go#L46)
The `-no-examples` pre-flight diagnostic resolves imports only through the FS roots, so a `gno.land/*` import already present in the modcache is reported as unresolvable even though it would load. Skip modcache-resolvable paths, or note in the warning that they may still load.

## contribs/gnodev/pkg/packages/package_test.go:14 [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/package_test.go#L14)
No test calls `ToMemPackage()` twice across an on-disk edit to lock the documented "re-read on every call" invariant that hot reload depends on; a future memoization of the FS path would silently break reload and still pass every test. Add a write → read → rewrite → read assertion.
