# PR #5604: feat: gnodev native loader

URL: https://github.com/gnolang/gno/pull/5604
Author: gfanton | Base: master | Files: 69 | +3837 -1908
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 5038db249 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5604 5038db249`

Round 4 re-review. Prior rounds: [round 1](../1-ed10e81f3/claude-sonnet-4-6_davd-gzl.md) (`ed10e81f3`, APPROVE), [round 2](../2-7eb33db9e/claude-opus-4-7_davd-gzl.md) (`7eb33db9e`, REQUEST CHANGES), [round 3](../3-43528ef2e/claude-opus-4-8_davd-gzl.md) (`43528ef2e`, APPROVE). Since round 3 the PR merged master and gfanton landed the fixes for [@thehowl](https://github.com/thehowl)'s GitHub review pass: opt-in `-remote` fetching, eager deps resolved from the FS roots, the single-package recursive pattern (gnovm), eager-root logging, and `Loader.Track` for `-paths`/txs-file deps. This round focuses on those changes and re-checks the still-open round-3 Warnings.

**TL;DR:** gnodev used to carry its own package loader/resolver; this rips out that ~1,850-line subsystem and delegates to gnovm's native loader (`gnovm/pkg/packages`), keeping the same `local` (lazy) and `staging` (eager) modes. It also makes network fetching opt-in (`-remote`), restores booting from a bare `gnomod.toml` dir, and signs the genesis `users/init` bootstrap tx.

**Verdict: APPROVE** — the loader migration is correct, the prior review round is fully addressed, and the headline network-fetch concern is fixed; two minor issues remain, neither blocking the core: gnodev crashes when run from a non-package subdir of a single-package realm ([workspace.go](#findworkspace)), and Ctrl+R no longer drops lazily-loaded packages ([reset](#reset)).

## Summary
The replacement is a single `Loader` (`pkg/packages/loader.go`) wrapping `gnovm.Load` for bulk eager loads plus a per-path `Resolve` for the lazy proxy. The old resolver/glob/utils files (`resolver_*.go`, `glob*.go`, `loader_base.go`, `setup_loader.go`, `utils*.go`) are deleted. Network fetching is now opt-in per chain domain via `-remote` (renamed from `-remote-override`): with no flag the loader is filesystem-only, closing the path where default boots silently pulled packages off `rpc.gno.land`. Genesis gets two correctness fixes: the `r/sys/users/init.Bootstrap` tx is injected only when that realm is in the package set and carries one empty signature slot per signer, and `-paths`/`-txs-file` deps now reach genesis through a new `Loader.Track` set since genesis txs never pass through the proxy. A gnovm change (`patterns.go`) lets a recursive pattern rooted at a single-package dir expand to exactly that root, so `cd myrealm && gnodev` and `gno test ./...` work without a `gnowork.toml`.

## Glossary
- **lazy / `local` mode** — workspace eager-loaded; cross-workspace imports resolved on demand by the proxy as queries arrive.
- **eager / `staging` mode** — workspace, every `-extra-root`, and `$GNOROOT/examples` materialized up front.
- **tracked set** — `Loader.tracked`: paths the loader re-resolves on every reload (seeded by `-paths`/txs deps via `Track`, grown by every proxy `Resolve`).
- **modcache** — `$GNOHOME/pkg/mod`: on-disk copies of chain packages, classified `KindRemote`.

## Fix
Resolution funnels through one type. `Resolve` ([loader.go:78-116](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L78-L116) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L78)) tries the index, then FS roots, then the fetcher, memoizing hits into `index` and `tracked`. The fetcher is gated in `newRemoteFetcher` ([fetcher.go:16-21](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/fetcher.go#L16-L21) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/fetcher.go#L16)): no `-remote` → `disabledFetcher` (every fetch refused), else a `domainFetcher` that fetches only configured domains. `Reload` ([loader.go:334-362](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L334-L362) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L334)) eager-loads the roots then merges each tracked path's dependency-first closure, so a lazily-loaded realm deploys with everything it imports.

## What I verified

Beyond the green CI (all checks pass) and the full gnodev + gnovm `pkg/packages` test suites (pass locally), the following are CI-invisible and verified live on 5038db249:

- **Network fetching is off without `-remote`, on with it.** A workspace realm importing an unresolvable path boots network-free, failing locally with `remote fetching is disabled, pass -remote <domain>=<rpc>`; adding `-remote gno.land=https://rpc.gno.land` is what turns the same import into a `qfile` query against `rpc.gno.land`. The gate sits on both the lazy path (`rpcLookup`, [loader.go:155](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L155) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L155)) and the eager path (`gnovm.Load` is handed `Fetcher: l.fetcher`, [loader.go:514](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L514) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L514), so gnovm never builds its own nil-fallback rpc fetcher). This resolves the [prior network-fetch finding](https://github.com/gnolang/gno/pull/5604#issuecomment-4673867296).
- **The daodemo phantom was a disk load, not a network fetch.** `gno.land/r/samcrew/daodemo` lives in `examples/quarantined/`, so `-extra-root $GNOROOT/examples` eager-loads it wholesale from disk (307 packages, quarantined included, zero network). On this head the disk copy type-checks clean (`gno test` → `ok`), and the old on-chain copy that failed with `undefined: basedao.DAOWrapper` is unreachable without `-remote`.
- **Genesis `users/init.Bootstrap` signing/gating is correct.** Injected only when `usersInitPkgPath` is in the reload set ([node.go:430-436](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/dev/node.go#L430-L436) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L430)), with `len(signers)` empty signature slots ([node.go:451](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/dev/node.go#L451) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L451)) matching the chain's own `genesis.go`, and re-evaluated on every rebuild.
- **`cd <bare gnomod.toml dir> && gnodev` boots** (workspace detected, gnoweb up), and nested `gnomod.toml` dirs are not pulled in (gnovm `singlepkg-1` fixture asserts the `nested/` package is excluded).

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5604 -R gnolang/gno
go build -o /tmp/gnodev ./contribs/gnodev
mkdir -p /tmp/ws/app && : > /tmp/ws/gnowork.toml
printf 'module = "gno.land/r/davdtest/app"\ngno = "0.9"\n' > /tmp/ws/app/gnomod.toml
printf 'package app\nimport "gno.land/r/davdtest/remoteonly"\nfunc Render(p string) string { return remoteonly.Hello() }\n' > /tmp/ws/app/app.gno
cd /tmp/ws && GNOROOT=$PWD/../.. timeout 25 /tmp/gnodev local -v -log-format json \
  -node-rpc-listener 127.0.0.1:36661 -web-listener 127.0.0.1:38891 2>&1 | grep -i "remote fetching is disabled"
```

```
... "err":"...: remote fetching is disabled, pass -remote <domain>=<rpc> to fetch \"gno.land/r/davdtest/remoteonly\" from a chain"
```

## Status of prior findings

Round-2 Criticals (sanitizer emitting invalid pkgpath segments) were fixed at round 3 and stay green. Round-3 Warnings on this head:

- **slice aliasing** — still open. The `kept := imps[:0]` reuse moved into the shared `filterSourceImports` ([loader.go:654-673](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L654-L673) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L654)), but the in-place backing-array mutation remains. Carried below.
- **GnoRoot from env, not `-root`** — still open ([app.go:239](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L239) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L239)). Carried below.
- **positional dirs added without existence check** — RESOLVED: positional `dirs` now route through `detectLocalPackage` ([app.go:202](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L202) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L202)), which validates the dir and skips non-packages with a warning; `-extra-root` entries are `os.Stat`-checked ([app.go:179](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L179) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L179)).
- **discovery banner bypasses commands.IO** — RESOLVED: the no-workspace notice now goes through `logNoWorkspace` to the column logger ([app.go:152](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L152) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L152),[232](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L232) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L232)), no raw `os.Stderr` write.

[@thehowl](https://github.com/thehowl)'s GitHub review pass (between rounds 3 and 4) is fully addressed. Spot-checked every "Fixed in" claim against current code: `-paths`/`-txs-file` to genesis via `Track` (PASS), bare gnomod.toml boot + gnovm recursive-pattern change & tests (PASS), `-no-examples` workspace-internal skip (PASS), modcache prefix boundary + remote-cached-across-reloads + shared `filterSourceImports` (PASS), package.go re-read comment (PASS), watch.go newline nit (PASS). None reopened.

## Critical (must fix)
None.

## Warnings (should fix)
<a id="findworkspace"></a>
- **[boots from a non-package subdir, then crashes]** `contribs/gnodev/pkg/packages/workspace.go:16` — `FindWorkspace` walks up accepting either marker, but gnovm only enters a workspace for a `gnowork.toml` ancestor and honors a bare `gnomod.toml` only in cwd; an ancestor `gnomod.toml` it refuses.
  <details><summary>details</summary>

  Run gnodev from a non-package subdir of a single-package realm (a `gnomod.toml` dir with no `gnowork.toml`): `FindWorkspace` returns the ancestor and gnodev logs `workspace detected root=<ancestor>` ([app.go:236](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L236) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L236)), then `gnovm.Load` re-derives its context from `os.Getwd()` ([load.go:163-190](https://github.com/gnolang/gno/blob/5038db249/gnovm/pkg/packages/load.go#L163-L190) · [↗](../../../../../.worktrees/gno-review-5604/gnovm/pkg/packages/load.go#L163)), finds neither a `gnowork.toml` ancestor nor a cwd `gnomod.toml`, and aborts node init with `gnowork.toml file not found ... and gnomod.toml doesn't exists in current directory`. The reassuring log directly precedes the crash. The canonical `cd <realm> && gnodev` is unaffected (cwd holds the `gnomod.toml`). Fix: mirror gnovm — walk up only for `gnowork.toml`; accept `gnomod.toml` only when it sits in the start dir, never an ancestor.
  </details>

<a id="reset"></a>
- **[Ctrl+R no longer drops lazily-loaded packages]** `contribs/gnodev/app.go:611-617` — Reset rebuilds genesis from `loader.Reload()`, whose package set is the loader's `tracked` map; `tracked` accumulates every path the proxy resolves and has no clear path, so realms browsed during a session survive Ctrl+R.
  <details><summary>details</summary>

  The Ctrl+R handler calls `pathManager.Reset()` + `SetPackagePaths(ds.paths)` ([app.go:614-615](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L614-L615) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L614)) then `devNode.Reset`, but genesis is produced solely by `n.config.Reload()` = `loader.Reload()` ([node.go:316](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/dev/node.go#L316) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L316)), which reads `tracked` ([loader.go:336-359](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L336-L359) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L336)). `tracked` only grows — via `Resolve` ([loader.go:94](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L94) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L94),[114](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L114) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L114)) and `Track` — with no reset method (grep: none). `n.paths` now feeds only `Paths()`→webHome default ([node.go:166-170](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/dev/node.go#L166-L170) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L166)), so the two `*PackagePaths` calls here (and the same pair in the proxy handler, [app.go:422-423](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L422-L423) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L422)) no longer affect what deploys. On master the Ctrl+R reset of `n.paths` did shrink the set. Fix: either clear/re-seed `tracked` to the initial `-paths` on Reset, or drop the now-dead `pathManager`/`*PackagePaths` calls and document that Reset keeps the loaded package set.
  </details>

- **[in-place slice aliasing]** `contribs/gnodev/pkg/packages/loader.go:654-673` — carried from rounds 2-3. `filterSourceImports`, the single helper behind `stripStdlibs` and `dropMissingDepImports`, reuses the package's own backing array via `kept := imps[:0]` ([loader.go:656](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L656) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L656),[665](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/packages/loader.go#L665) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L665)), mutating `Imports`/`ImportsSpecs` in place. Safe today (inputs are fresh per `vmpackages.Load` / `loadExtraRootVm` and discarded after `Sort`), but an at-a-distance invariant: a future loader that memoizes its result would have its cached import lists silently truncated. Fix: copy explicitly (`make([]string, 0, len(imps))`), or note the no-cache requirement at each reuse.
- **[GnoRoot from env, not -root flag]** `contribs/gnodev/app.go:239` — carried from rounds 2-3. `gnoRoot := gnoenv.RootDir()` ([app.go:239](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/app.go#L239) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L239)) feeds the loader's `examples/` root and the quarantine exclude, while the node config uses `cfg.root` from `-root`; passing `-root /alt` then yields a node on `/alt` but a loader resolving `examples/` under the env root. Fix: thread `cfg.root` into `gnoRoot`, falling back to `gnoenv.RootDir()` when empty.

## Nits
- `contribs/gnodev/pkg/packages/examples_check.go:46` — the `-no-examples` pre-flight diagnostic resolves imports only through `LookupFS` (extra roots + examples), so a `gno.land/*` import already in the modcache is reported as unresolvable even though it loads fine. Warning-only output; the listed remedies still apply. Confirmed behaviorally: `LookupFS` never consults the modcache.
- `contribs/gnodev/pkg/dev/node.go:431` — `bootstrapTxs` gates on `users/init` being in the reload set, but `generateTxs` skips any package whose `ToMemPackage()` fails ([node.go:463-467](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/dev/node.go#L463-L467) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L463)); if `users/init` were ever in the set yet unreadable, the `Bootstrap` call would target an undeployed realm (warn-skipped at genesis). Near-unreachable — `users/init` is a static `$GNOROOT/examples` realm — so defensive only.

## Missing Tests
- **[hot-reload freshness invariant unpinned]** `contribs/gnodev/pkg/packages/package_test.go` — no test calls `ToMemPackage()` twice across an on-disk edit and asserts the second call returns the new content.
  <details><summary>details</summary>

  `package.go:39-42` documents the load-bearing invariant "re-read from disk on EVERY call — never memoize this: hot reload depends on it." `TestPackage_ToMemPackage_FS` reads once; node-level reload tests mutate in-memory `MemPackage.Files`, exercising the in-memory path, not the FS re-read. A future memoization of `ToMemPackage` for FS packages would silently break hot reload and pass every existing test. Add: write a `.gno` file, `ToMemPackage`, rewrite it, `ToMemPackage` again, assert the edit is observed.
  </details>
- `contribs/gnodev/pkg/watcher/watch_test.go` — the positive watch events (Write/Create/Rename/Remove) are each covered, but no negative test asserts a Chmod-only event does NOT trigger a package update ([watch.go:95](https://github.com/gnolang/gno/blob/5038db249/contribs/gnodev/pkg/watcher/watch.go#L95) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/watcher/watch.go#L95)). A regression dropping the `.Has` filter would go uncaught. Minor.

## Suggestions
- `contribs/gnodev/setup_node.go:29-30` — `extractDependenciesFromTxs` tracks deps from `vm.MsgCall` only (explicit `TODO: Support MsgRun`); a `-txs-file` whose txs carry `MsgRun`/`MsgAddPackage` imports won't have those deps reach genesis, so replay can fail on an undeployed dep. Low impact today; worth widening or documenting the limitation in the flag help.

## Open questions
- Is keeping lazily-loaded packages across Ctrl+R intended (lazy model: "loaded stays loaded"), or should Reset return to the initial `-paths` set? The answer decides whether the Reset Warning's fix is "clear tracked" or "remove the dead path-reset calls." Not posted as its own comment — folded into the Reset finding.
