# PR #5604: feat: gnodev native loader

URL: https://github.com/gnolang/gno/pull/5604
Author: gfanton | Base: master | Files: 70 | +3965 -1913
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `1da2f9242` (stale — +18 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5604 1da2f9242`

Round 5 re-review. Prior rounds: [round 1](../1-ed10e81f3/claude-sonnet-4-6_davd-gzl.md) (`ed10e81f3`, APPROVE), [round 2](../2-7eb33db9e/claude-opus-4-7_davd-gzl.md) (`7eb33db9e`, REQUEST CHANGES), [round 3](../3-43528ef2e/claude-opus-4-8_davd-gzl.md) (`43528ef2e`, APPROVE), [round 4](../4-5038db249/review_claude-opus-4-8_davd-gzl.md) (`5038db249`, APPROVE). Since round 4 the PR added four commits, all in the package-discovery layer: re-scan extra roots on reload (`adba09b50`), warn when single-package mode drops nested packages (`28e3de121`), export `FindLoaderRoot` from gnovm (`219b804f0`), and route `FindWorkspace` through it so a subdir of a bare-`gnomod.toml` realm boots instead of crashing (`1da2f9242`). The last commit resolves the round-4 `FindWorkspace` Warning. This round verifies the four commits and re-checks the still-open round-4 findings.

**TL;DR:** gnodev used to carry its own package loader/resolver; this rips out that ~1,850-line subsystem and delegates to gnovm's native loader (`gnovm/pkg/packages`), keeping the same `local` (lazy) and `staging` (eager) modes. It also makes network fetching opt-in (`-remote`), restores booting from a bare `gnomod.toml` dir, and signs the genesis `users/init` bootstrap tx.

**Verdict: APPROVE** — the four new commits are correct and the round-4 `FindWorkspace` crash is fixed and verified live; three minor issues carry over, none blocking the core migration: Ctrl+R no longer drops lazily-loaded packages ([reset](#reset)), an in-place slice reuse that is safe today but decay-prone ([slice](#slice)), and the loader resolving `examples/` from the env root rather than `-root` ([gnoroot](#gnoroot)).

## Summary
The replacement is a single `Loader` (`pkg/packages/loader.go`) wrapping `gnovm.Load` for bulk eager loads plus a per-path `Resolve` for the lazy proxy. The old resolver/glob/utils files are deleted. Network fetching is opt-in per chain domain via `-remote`: with no flag the loader is filesystem-only, closing the path where default boots silently pulled packages off `rpc.gno.land`. Genesis gets two correctness fixes: the `r/sys/users/init.Bootstrap` tx is injected only when that realm is in the package set and carries one empty signature slot per signer, and `-paths`/`-txs-file` deps now reach genesis through a `Loader.Track` set. This round's commits tighten the workspace/root resolution: `FindWorkspace` now delegates to a new exported `gnovm.FindLoaderRoot` so gnodev and gnovm can't drift on what defines a root, single-package recursive loads warn about nested packages they drop, and extra roots are re-walked on every eager load so packages added mid-session surface without a restart.

## Glossary
- **lazy / `local` mode** — workspace eager-loaded; cross-workspace imports resolved on demand by the proxy as queries arrive.
- **eager / `staging` mode** — workspace, every `-extra-root`, and `$GNOROOT/examples` materialized up front.
- **tracked set** — `Loader.tracked`: paths the loader re-resolves on every reload (seeded by `-paths`/txs deps via `Track`, grown by every proxy `Resolve`).
- **single-package mode** — gnovm's loader context when the start dir holds a bare `gnomod.toml` with no `gnowork.toml` ancestor; a recursive pattern matches only the root package.
- **discovery mode** — gnodev's fallback when no workspace is found: packages resolve on demand from examples and, for `-remote` domains, a chain RPC.

## What changed since round 4
- **`adba09b50` re-scan extra roots.** `loadExtraRootVm` now calls `scanRoot` fresh on every eager load and overwrites `rootIdx[root]`, instead of reusing the cached index via `ensureRootIndexLocked`. A package added to (or removed from) an `-extra-root` mid-session now surfaces on the next reload, matching the workspace (which `gnovm.Load` re-walks each reload). The walk runs outside the lock; only the index swap is locked.
- **`28e3de121` warn on dropped nested packages.** In single-package mode a recursive pattern (`./...`, `...`) loads only the root and silently ignored nested `gnomod.toml` packages. `expandPatterns` now names them in a warning pointing to `gnowork.toml`, via a new `nestedPackageDirs` helper reusing `expandRecursive`.
- **`219b804f0` export `FindLoaderRoot`.** `findLoaderContext` is split into `findLoaderContextFor(dir)` (the no-arg form becomes a thin wrapper) and exported as `FindLoaderRoot`, so callers building a recursive pattern from a directory resolve the root by the same rule `Load` applies to its cwd. Behavior-preserving refactor.
- **`1da2f9242` boot from a subdir of a single-package realm.** `FindWorkspace` delegates to `FindLoaderRoot` instead of its own marker walk. A subdir of a bare-`gnomod.toml` realm with no `gnowork.toml` ancestor now resolves to `""` and boots in discovery mode, rather than reporting `workspace detected` and then crashing at node init.

## What I verified

Beyond green CI (only the unrelated `docs` remote-link linter fails, on YouTube URLs in `docs/MANIFESTO.md` and `docs/resources/effective-gno.md`, files this PR does not touch) and the full gnovm + gnodev `pkg/packages` suites (pass, including `-race` on the gnodev loader), the following are CI-invisible and verified live on 1da2f9242:

- **A subdir of a bare-`gnomod.toml` realm boots in discovery mode, not a crash.** Running gnodev from a non-package subdir of a single-package realm now logs `no workspace ... running in discovery mode` and reaches `node is ready`; on round-4's head the same invocation logged `workspace detected` then aborted node init with `gnomod.toml doesn't exists in current directory`. Restoring the old `FindWorkspace` marker walk reproduces the crash.
- **`gno test ./...` in a bare-`gnomod.toml` dir with a nested package warns.** Emits `gno: warning: "./..." matched only the root package in single-package mode; 1 nested package(s) ignored: nested (create a gnowork.toml to include them)`; the root package still loads.
- **Network fetching is off without `-remote`, on with it.** A workspace realm importing an unresolvable path boots network-free, failing locally with `remote fetching is disabled, pass -remote <domain>=<rpc>`; adding `-remote gno.land=https://rpc.gno.land` is what turns the same import into a `qfile` query against `rpc.gno.land`. Unchanged this round; re-confirmed on 1da2f9242.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5604 -R gnolang/gno
export GNOROOT=$PWD
go build -o /tmp/gnodev ./contribs/gnodev
mkdir -p /tmp/single/sub
printf 'module = "gno.land/r/davdtest/single"\ngno = "0.9"\n' > /tmp/single/gnomod.toml
printf 'package single\nfunc Render(p string) string { return "hi" }\n' > /tmp/single/single.gno
( cd /tmp/single/sub && timeout 20 /tmp/gnodev local -v -log-format json \
  -node-rpc-listener 127.0.0.1:36671 -web-listener 127.0.0.1:38991 ) 2>&1 \
  | grep -iE "workspace detected|no workspace|node is ready|gnomod.toml doesn"
rm -rf /tmp/single /tmp/gnodev
```

```
... "msg":"no workspace (gnomod.toml / gnowork.toml) found in ./ or any parent.\nrunning in discovery mode: ..."
... "msg":"node is ready","took":3.44
```

## Status of prior findings

Round-2 Criticals and round-3 fixes stay green. Round-4 findings on this head:

- **`FindWorkspace` boots from a non-package subdir then crashes** (Warning) — **RESOLVED** by `1da2f9242`. `FindWorkspace` ([workspace.go:12-16](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/packages/workspace.go#L12-L16) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/workspace.go#L12)) now returns whatever `gnovm.FindLoaderRoot` resolves, so it can no longer promote an ancestor `gnomod.toml` to a workspace that `gnovm.Load` then refuses. Verified live (boots in discovery mode) and pinned by `TestFindWorkspace_ModFileInAncestorIsNotWorkspace`.
- **Ctrl+R no longer drops lazily-loaded packages** (Warning) — still open; the Reset/`tracked` path is untouched this round. Carried below.
- **in-place slice aliasing** (Warning) — still open; `filterSourceImports` moved by +3 lines but the `imps[:0]` reuse is unchanged. Carried below.
- **GnoRoot from env, not `-root`** (Warning) — still open ([app.go:239](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/app.go#L239) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L239)). Carried below.
- Nits (examples_check modcache, bootstrapTxs gate), Missing Tests (ToMemPackage freshness, Chmod watch), and the txs-deps Suggestion are all in untouched files; carried below verbatim.

## Critical (must fix)
None.

## Warnings (should fix)
<a id="reset"></a>
- **[Ctrl+R no longer drops lazily-loaded packages]** `contribs/gnodev/pkg/dev/node.go:316` — Reset rebuilds genesis from `loader.Reload()`, whose package set is the loader's `tracked` map; `tracked` accumulates every path the proxy resolves and has no clear path, so realms browsed during a session survive Ctrl+R.
  <details><summary>details</summary>

  The Ctrl+R handler calls `pathManager.Reset()` + `SetPackagePaths(ds.paths)` ([app.go:614-615](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/app.go#L614-L615) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L614)) then `devNode.Reset`, but genesis is produced solely by `n.config.Reload()` = `loader.Reload()` ([node.go:316](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/dev/node.go#L316) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L316)), which reads `tracked` ([loader.go:336-359](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/packages/loader.go#L336-L359) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L336)). `tracked` only grows — via `Resolve` ([loader.go:94](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/packages/loader.go#L94) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L94),[114](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/packages/loader.go#L114) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L114)) and `Track` — with no reset method (grep: none). `n.paths` now feeds only `Paths()`→webHome default ([node.go:166-170](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/dev/node.go#L166-L170) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L166)), so the two `*PackagePaths` calls here (and the same pair in the proxy handler, [app.go:422-423](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/app.go#L422-L423) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L422)) no longer affect what deploys. On master the Ctrl+R reset of `n.paths` did shrink the set. Fix: either clear/re-seed `tracked` to the initial `-paths` on Reset, or drop the now-dead `pathManager`/`*PackagePaths` calls and document that Reset keeps the loaded package set.
  </details>

<a id="slice"></a>
- **[in-place slice aliasing]** `contribs/gnodev/pkg/packages/loader.go:657-675` — carried from rounds 2-4. `filterSourceImports`, the single helper behind `stripStdlibs` and `dropMissingDepImports`, reuses the package's own backing array via `kept := imps[:0]` ([loader.go:659](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/packages/loader.go#L659) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L659),[668](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/packages/loader.go#L668) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L668)), mutating `Imports`/`ImportsSpecs` in place. Safe today (inputs are fresh per `vmpackages.Load` / `loadExtraRootVm` and discarded after `Sort`), but an at-a-distance invariant: a future loader that memoizes its result would have its cached import lists silently truncated. Fix: copy explicitly (`make([]string, 0, len(imps))`), or note the no-cache requirement at each reuse.

<a id="gnoroot"></a>
- **[GnoRoot from env, not -root flag]** `contribs/gnodev/app.go:239` — carried from rounds 2-4. `gnoRoot := gnoenv.RootDir()` ([app.go:239](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/app.go#L239) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L239)) feeds the loader's `examples/` root and the quarantine exclude, while the node config uses `cfg.root` from `-root`; passing `-root /alt` then yields a node on `/alt` but a loader resolving `examples/` under the env root. Fix: thread `cfg.root` into `gnoRoot`, falling back to `gnoenv.RootDir()` when empty.

## Nits
- `contribs/gnodev/pkg/packages/examples_check.go:46` — the `-no-examples` pre-flight diagnostic resolves imports only through `LookupFS` (extra roots + examples), so a `gno.land/*` import already in the modcache is reported as unresolvable even though it loads fine. Warning-only output; the listed remedies still apply. Confirmed behaviorally: `LookupFS` never consults the modcache.
- `contribs/gnodev/pkg/dev/node.go:431` — `bootstrapTxs` gates on `users/init` being in the reload set, but `generateTxs` skips any package whose `ToMemPackage()` fails ([node.go:463-467](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/dev/node.go#L463-L467) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L463)); if `users/init` were ever in the set yet unreadable, the `Bootstrap` call would target an undeployed realm (warn-skipped at genesis). Near-unreachable — `users/init` is a static `$GNOROOT/examples` realm — so defensive only.

## Missing Tests
- **[hot-reload freshness invariant unpinned]** `contribs/gnodev/pkg/packages/package_test.go` — no test calls `ToMemPackage()` twice across an on-disk edit and asserts the second call returns the new content.
  <details><summary>details</summary>

  `package.go:39-42` documents the load-bearing invariant "re-read from disk on EVERY call — never memoize this: hot reload depends on it." `TestPackage_ToMemPackage_FS` reads once; node-level reload tests mutate in-memory `MemPackage.Files`, exercising the in-memory path, not the FS re-read. A future memoization of `ToMemPackage` for FS packages would silently break hot reload and pass every existing test. Add: write a `.gno` file, `ToMemPackage`, rewrite it, `ToMemPackage` again, assert the edit is observed.
  </details>
- `contribs/gnodev/pkg/watcher/watch_test.go` — the positive watch events (Write/Create/Rename/Remove) are each covered, but no negative test asserts a Chmod-only event does NOT trigger a package update ([watch.go:95](https://github.com/gnolang/gno/blob/1da2f9242/contribs/gnodev/pkg/watcher/watch.go#L95) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/watcher/watch.go#L95)). A regression dropping the `.Has` filter would go uncaught. Minor.

## Suggestions
- `contribs/gnodev/setup_node.go:29-30` — `extractDependenciesFromTxs` tracks deps from `vm.MsgCall` only (explicit `TODO: Support MsgRun`); a `-txs-file` whose txs carry `MsgRun`/`MsgAddPackage` imports won't have those deps reach genesis, so replay can fail on an undeployed dep. Low impact today; worth widening or documenting the limitation in the flag help.

## Open questions
- Is keeping lazily-loaded packages across Ctrl+R intended (lazy model: "loaded stays loaded"), or should Reset return to the initial `-paths` set? The answer decides whether the Reset Warning's fix is "clear tracked" or "remove the dead path-reset calls." Not posted as its own comment — folded into the Reset finding.
- The re-scan (`adba09b50`) reverses round-3's explicit "don't re-walk roots on every reload" decision. In `local` mode `examples/` stays lazy so only small `-extra-root` dirs are re-walked; in `staging` mode `LoadAll` re-walks `examples/` (480 dirs) every reload, but the stat-walk is sub-10ms against a multi-second node reset, so the cost is immaterial. Noted, not a finding.
