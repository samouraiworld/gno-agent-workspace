# PR #5604: feat: gnodev native loader

URL: https://github.com/gnolang/gno/pull/5604
Author: gfanton | Base: master | Files: 56 | +2595 -1805
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `43528ef2e` (stale — +92 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5604 43528ef2e`

**TL;DR:** Reworks `gnodev`'s package loading. A new eager "native loader" discovers packages across the workspace plus any extra roots, merges them, and runs one cross-root topological sort so a realm that imports a sibling package in another root deploys in dependency order instead of failing to compile. Also adds package-path sanitization for dirs without a `gnomod.toml`, a `-without-quarantined-examples` flag, and a non-zero exit code on fatal config refusal.

Round 3 re-review. Prior rounds: [round 1](../1-ed10e81f3/claude-sonnet-4-6_davd-gzl.md) (`ed10e81f3`, APPROVE), [round 2](../2-7eb33db9e/claude-opus-4-7_davd-gzl.md) (`7eb33db9e`, REQUEST CHANGES). This round focuses on what changed since `7eb33db9e`: six gnodev commits (`9096ad7d9`, `dffc83111`, `19dc60b29`, `5a4043188`, `36f069443`, `43528ef2e`) plus a master merge.

**Verdict: APPROVE** — Both round-2 Criticals are fixed: `sanitizePathSegment` now emits valid `Re_name` segments and the test table + `TestGuessPath_NoGnoModProducesValidPath` assert the `IsUserlib` round-trip; the previously-red `gnodev / test` CI job is green. The new eager-extra-root Reload (`43528ef2e`) is a real correctness fix with a dedicated test. Remaining open items are the same round-2 Warnings (env-vs-flag GnoRoot, banner to `os.Stderr`, positional-dir stat gap, `stripStdlibs`/`dropMissingDepImports` slice aliasing), all non-blocking for a dev tool. The two red CI checks (`docs`, `scenario-08-five-validators`) are unrelated to this PR.

## Summary

Since round 2 the sanitizer regression is resolved exactly as recommended: drop leading `_`, collapse `_` runs, trim trailing `_`, prepend `d` only on a leading digit, fall back to `"app"` ([paths.go:36-58](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/paths.go#L36-L58) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths.go#L36-L58)). The bigger change is a Reload/LoadAll rework: both now route through a shared `loadEager` that merges the workspace load with a per-root VM-package walk, runs a single cross-root `PkgList.Sort`, and filters ignored packages. This fixes a deploy-order bug — an extra-root realm importing a sibling pure-package in the same root now deploys after its dep instead of failing to compile on first query. New flag `-without-quarantined-examples` (default on in staging) wires `Config.ExcludeDirs` into `scanRoot` to skip `$GNOROOT/examples/quarantined`. `main` now exits non-zero on a fatal config refusal.

## Glossary

- `loadEager` — shared eager-load path for `LoadAll`/`Reload`; merges workspace + per-root VM packages, then one unified topo-sort. [loader.go:352-395](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L352-L395) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L352-L395)
- `loadExtraRootVm` — walks one root, re-parses each `gnomod.toml` (for `Ignore`) and each package's imports, returns an unsorted `vmpackages.PkgList`.
- `dropMissingDepImports` — strips a package's source imports that aren't present in the merged list so cross-root/remote/stdlib deps don't break `PkgList.Sort`.
- `Re_name` / `IsUserlib` — gno's pkgpath-segment regex and userlib-path gate; the round-2 Criticals were `sanitizePathSegment` emitting segments these reject.
- `ExcludeDirs` — `Config` field; exact-match (after `filepath.Clean`) set of dirs `scanRoot` skips.

## Status of prior findings

Round-2 Criticals — both fixed:

- [CI red — sanitizer emits invalid pkgpath segments] FIXED in `9096ad7d9`. `sanitizePathSegment` rewritten to suppress leading separators, collapse `_` runs, trim trailing `_`, and prepend `d` only when the result starts with a digit ([paths.go:43-58](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/paths.go#L43-L58) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths.go#L43-L58)). `_test → test`, `__abc → abc`, `--leading-dash → leading_dash`, `_1ab → d1ab`. Verified green below.
- [unit test enshrines wrong invariant] FIXED. The golden table now encodes valid outputs ([paths_test.go:20-30](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/paths_test.go#L20-L30) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths_test.go#L20-L30)), and `TestGuessPath_NoGnoModProducesValidPath` asserts `gnolang.IsUserlib(path)` for every basename including `_test`, `--leading-dash`, `my.proj`, and `weird name with spaces` ([paths_test.go:42-64](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/paths_test.go#L42-L64) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths_test.go#L42-L64)). The two tests now fail together when the invariant breaks. The misleading docstring is corrected too.

Round-2 Warnings — partially addressed:

- [deletion of extra-root dir mid-session not detected] still documented at [loader.go:280-283](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L280-L283) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L280-L283). Reload now eager-loads extra roots, but `rootIdx` is still cached across reloads, so a deleted dir still returns its stale entry until restart. Same bounded cost as before. Not blocking.
- [GnoRoot from env, not flag] NOT addressed — `loaderCfg.GnoRoot = gnoRoot = gnoenv.RootDir()` still ignores `cfg.root` ([app.go:200-211](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/app.go#L200-L211) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L200-L211)). See Warnings.
- [positional dirs added without existence check] NOT addressed ([app.go:190-194](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/app.go#L190-L194) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L190-L194)). See Warnings.
- [stripStdlibs aliasing] NOT addressed and now joined by the same pattern in the new `dropMissingDepImports`. Still safe (inputs are fresh per call), still an at-a-distance invariant. See Warnings.
- [banner to os.Stderr] NOT addressed ([app.go:177](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/app.go#L177) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L177)).
- [lock-order invariant undocumented] NOT addressed; unchanged.
- [scanRoot skip list misses common dirs] partially superseded — `ExcludeDirs` now lets callers exclude specific dirs, but the built-in skip set is still only `.*` / `node_modules` / `_build` ([loader.go:239](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L239) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L239)).

## Fix (this round)

Reload and LoadAll were unified behind `loadEager(roots)` ([loader.go:352-395](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L352-L395) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L352-L395)). Before: `Reload` eager-loaded only the workspace and re-resolved tracked paths one at a time, so a sibling dep inside an extra root could be missing or out of deploy order. After: `Reload` passes `cfg.ExtraRoots` to `loadEager`, which walks each root via `loadExtraRootVm` (re-parsing `gnomod.toml` for the `Ignore` flag and each package's imports), merges with the workspace `vmpackages.Load`, strips stdlibs, drops missing-dep imports, runs one `PkgList.Sort`, and filters with `GetNonIgnoredPkgs`. `LoadAll` uses the same path with `lookupRoots()` (which adds `$GNOROOT/examples`). The load-bearing constraint: `generateTxs` deploys packages in slice order ([node.go:420-466](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/dev/node.go#L420-L466) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node.go#L420-L466)), so a topologically-sorted Reload output is required for cross-package extra-root realms to type-check at genesis.

## Critical (must fix)

None. Both round-2 Criticals are resolved.

**Repro (sanitizer fix + IsUserlib round-trip):**
```bash
# from a local clone of gnolang/gno:
gh pr checkout 5604 -R gnolang/gno
go test -C contribs/gnodev -v -run 'TestSanitizePathSegment|TestGuessPath_NoGnoModProducesValidPath' .
```
```
=== RUN   TestSanitizePathSegment/_test
=== RUN   TestGuessPath_NoGnoModProducesValidPath/_test
=== RUN   TestGuessPath_NoGnoModProducesValidPath/--leading-dash
--- PASS: TestSanitizePathSegment (0.00s)
--- PASS: TestGuessPath_NoGnoModProducesValidPath (0.00s)
ok  	github.com/gnolang/gno/contribs/gnodev	0.036s
```

## Warnings (should fix)

- **[in-place slice aliasing, now in two places]** [`contribs/gnodev/pkg/packages/loader.go:550-574`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L550-L574) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L550-L574) — also raised in [round 2](../2-7eb33db9e/claude-opus-4-7_davd-gzl.md) for `stripStdlibs`. `dropMissingDepImports` repeats the `kept := imps[:0]` pattern, mutating the package's own `Imports` and `ImportsSpecs` backing arrays.
  <details><summary>details</summary>

  Both `stripStdlibs` ([loader.go:580-598](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L580-L598) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L580-L598)) and `dropMissingDepImports` write back into the caller's slices. Safe today: the packages they mutate are either fresh from `vmpackages.Load` (no-cache, confirmed round 2) or freshly synthesized in `loadExtraRootVm` ([loader.go:480-487](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L480-L487) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L480-L487)), and they're discarded after `Sort`. But these are intermediate `vmpackages.Package` values, distinct from the gnodev `Package` entries written into `l.index` by `vmPkgListToPackages`, so the blast radius is contained to one call. The risk is the same at-a-distance invariant: a future Load that memoizes its result would have its cached import lists silently truncated. Fix: copy explicitly in both functions (`kept := make([]string, 0, len(imps))`), or add a one-line comment at each `[:0]` reuse noting the no-cache requirement. Cheap, removes the invisible contract.
  </details>

- **[GnoRoot from env, not -root flag]** [`contribs/gnodev/app.go:200-211`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/app.go#L200-L211) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L200-L211) — carried from [round 2](../2-7eb33db9e/claude-opus-4-7_davd-gzl.md); now also drives the `-without-quarantined-examples` exclude path.
  <details><summary>details</summary>

  `gnoRoot := gnoenv.RootDir()` feeds both `loaderCfg.GnoRoot` and the quarantine exclude dir, while the node config uses `cfg.root` from the `-root` flag ([setup_node.go](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/setup_node.go) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/setup_node.go)). A user passing `-root /alt/gnoroot` gets a node whose stdlibs come from `/alt/gnoroot` but a loader resolving `examples/` (and excluding `examples/quarantined`) under the env-derived root. The two `gnoRoot` uses are now internally consistent with each other, which is the silver lining, but both diverge from `-root`. Fix: thread `cfg.root` into `gnoRoot` (fall back to `gnoenv.RootDir()` when empty). One line, removes a foot-gun and keeps the exclude path aligned with the active root.
  </details>

- **[positional dirs added as extra-roots without existence check]** [`contribs/gnodev/app.go:190-194`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/app.go#L190-L194) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L190-L194) — carried from [round 2](../2-7eb33db9e/claude-opus-4-7_davd-gzl.md), unchanged.
  <details><summary>details</summary>

  `-extra-root` entries are stat-validated and skipped on failure ([app.go:183-189](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/app.go#L183-L189) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L183-L189)); positional `dirs` are appended unconditionally and also seed `localPaths` via `guessPath`. A typo'd positional path starts gnodev fine but flows into `cfg.ExtraRoots`. The new `loadEager` then walks it via `ensureRootIndexLocked`, which caches an empty map for a missing root (so no crash), but `localPaths` still references an unresolvable path. Fix: factor the stat-check helper and apply it to both lists, dropping the matching `localPaths` entry when a positional dir is skipped.
  </details>

- **[discovery banner bypasses commands.IO]** [`contribs/gnodev/app.go:177`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/app.go#L177) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L177) — carried from rounds 1 and 2, unchanged. `printDiscoveryBanner(os.Stderr)` is the only direct stderr write; tests using `commands.NewTestIO()` can't capture it. Fix: `printDiscoveryBanner(ds.io.Err())`.

## Nits

- [`contribs/gnodev/pkg/packages/loader.go:445-493`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L445-L493) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L445-L493) — `loadExtraRootVm` re-parses each `gnomod.toml` with `gnomod.ParseDir` solely to read `Ignore`, after `scanRoot` already parsed it once during the walk ([loader.go:251](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L251) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L251)). The comment acknowledges this ("scanRoot only captured the module path"). Bounded by root size and only at eager-load time; fine. Could be eliminated by having `rootIdx` carry `Ignore` alongside the dir.

- [`contribs/gnodev/pkg/packages/config.go:22-29`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/config.go#L22-L29) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/config.go#L22-L29) — `ExcludeDirs` matches by exact cleaned path, so a trailing-slash or relative entry silently no-ops. The doc comment is honest about this ("entries should match that form... entries that don't match any walked directory are no-ops"). For the one in-tree caller the path is always absolute and built from the same `gnoRoot`, so it matches; flag only because a future caller passing a relative dir gets a silent miss with no warning.

- [`contribs/gnodev/pkg/packages/package.go`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/package.go) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/package.go) — round-2 nit (`packageFromMemPackage` defaulting `Kind: KindFS` instead of `KindUnknown`) appears untouched; carry forward as a low-priority cleanup.

## Missing Tests

- **[no test for `cfg.GnoRoot != gnoenv.RootDir()`]** [`contribs/gnodev/pkg/packages/loader.go:196-203`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L196-L203) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L196-L203) — carried from round 2; the `-root` vs env divergence Warning still has no coverage. The new `ExcludeDirs` quarantine path is built from the same `gnoRoot`, so a wrong root mis-locates both `examples/` and the exclude in lockstep, which a test would catch.

- **[no negative test for unmatched ExcludeDirs entry]** [`contribs/gnodev/pkg/packages/loader.go:221-228`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/pkg/packages/loader.go#L221-L228) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L221-L228) — `TestLoader_ExcludeDirs_SkipsSubtree` covers the matching case well (with a control loop). No test asserts that a relative or trailing-slash entry is a silent no-op, which is the documented sharp edge. A one-line subtest would pin the contract so a future "make it forgiving" change is a conscious decision.

## Suggestions

- [`contribs/gnodev/main.go:64`](https://github.com/gnolang/gno/blob/43528ef2e/contribs/gnodev/main.go#L64) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/main.go#L64) — `os.Exit(1)` on `cmd.Run` error is correct and overdue (scripts can now detect a config refusal via `$?`). Note `os.Exit` skips deferred cleanup; here `main` has no defers after `cmd.Run`, so it's fine. Worth a one-line test asserting non-zero exit on the "nothing to load" refusal if gnodev grows a CLI-level test harness.

## Questions for Author

- `Reload` now returns topologically-sorted packages where it previously preserved workspace-then-tracked order. `Reset`/`ReloadAll` feed this straight into `generateTxs` deploy order, so the new order is the correct one — but is any external consumer (events, web UI listing) relying on the old ordering? If not, no action.
- The quarantine exclude is hard-wired to `$GNOROOT/examples/quarantined` and only applied when `Examples` is on. If a user also passes an `-extra-root` containing a `quarantined/` subtree they want excluded, they currently can't. Intentional scope limit, or worth exposing `ExcludeDirs` as a flag?
