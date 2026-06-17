# PR #5604: feat: gnodev native loader

URL: https://github.com/gnolang/gno/pull/5604
Author: gfanton | Base: master | Files: 56 | +2280 -1789
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `7eb33db9e` (stale — +89 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5604 7eb33db9e`

**Verdict: REQUEST CHANGES** — CI is red on a self-included unit test (`TestGuessPath_NoGnoModProducesValidPath`) showing `sanitizePathSegment` produces import-path segments that `gnolang.IsUserlib` rejects, which blocks any user running `gnodev local` in a dir whose basename starts with `_` or doubled separators. `TestSanitizePathSegment` also hard-codes the same invalid outputs in its golden table, so a naïve unit-test pass is no protection.

## Summary

Replaces gnodev's bespoke loader/resolver/middleware subsystem with a thin wrapper around `gnovm/pkg/packages/Load`. The new `Loader` struct delegates eager workspace/root loading to gnovm and adds a local per-path `Resolve` for the lazy proxy, with a per-root FS index cache plus RPC fallback. User-facing flags collapse: `-resolver/-lazy-loader/-load` removed; `-extra-root`, `-no-examples`, `-remote-override` added. The diff drops ~1800 lines of duplicated machinery. Architecture is sound; the regression is concentrated in the basename-to-pkgpath sanitizer.

## Glossary

- `Loader` — gnodev's per-instance package resolver in `contribs/gnodev/pkg/packages/`. Owns `index` (path→Package cache), `tracked` (proxy-hit set re-resolved on Reload), and `rootIdx` (per-root scan cache).
- `Re_name` — gno's package-name regex: first char `[a-z]`, body `[a-z0-9]`, separators (`_`/`-`) only between alphanumerics, never consecutive, never trailing.
- `IsUserlib` — gates whether a path matches `Re_gnoUserPkgPath` (domain + `/r|p|e/` + `Re_name` segments). Used by the genesis-time and runtime path validators.
- `guessPath` — fallback that derives an import path from a directory basename when no `gnomod.toml` is present; sanitizes with `sanitizePathSegment`.
- `KindUnknown / KindFS / KindRemote` — Package.Kind classifier; the watcher only attaches to `KindFS`.
- `MPUserProd` — mempackage type that excludes `_test.gno` / `_filetest.gno` files.

## Fix

The PR removes the legacy `Resolver`-chain + middleware stack ([`contribs/gnodev/pkg/packages/resolver.go`](https://github.com/gnolang/gno/blob/origin/master/contribs/gnodev/pkg/packages/resolver.go) and siblings) and replaces it with a single `Loader` (eight files in [`contribs/gnodev/pkg/packages/`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/)). The before-state had Glob/Base/Resolver layers each running their own pattern expansion, stdlib filtering and remote fetch; the after-state delegates bulk work to `gnovm.Load` and keeps only per-path lookup + RPC fallback locally ([loader.go:69-107](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L69-L107) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L69-L107)). Mode flags are gone; lazy vs eager is now derived from the subcommand (`local`=lazy, `staging`=eager via `LoadAll`) and from CWD/workspace detection ([app.go:223-230](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/app.go#L223-L230) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L223-L230)). A new `InMemoryFetcher` lands in `gnovm/pkg/packages/pkgdownload/` for tests.

## Critical (must fix)

- **[CI red — sanitizer emits invalid pkgpath segments]** [`contribs/gnodev/paths.go:36-58`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/paths.go#L36-L58) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths.go#L36-L58) — `sanitizePathSegment` returns segments that fail `gnolang.IsUserlib`; failing test runs on every CI build.
  <details><summary>details</summary>

  **Shape:** `sanitizePathSegment("_test") → "_test"` and `sanitizePathSegment("--leading-dash") → "d__leading_dash"`. Both produce paths (`gno.land/r/dev/_test`, `gno.land/r/dev/d__leading_dash`) that `gnolang.IsUserlib` rejects.

  **Mechanism:** [`Re_name`](https://github.com/gnolang/gno/blob/7eb33db9e/gnovm/pkg/gnolang/mempackage.go#L69) · [↗](../../../../../.worktrees/gno-review-5604/gnovm/pkg/gnolang/mempackage.go#L69) is `r.G(r.C('a-z'), r.S(r.C('a-z0-9')), r.S(r.C('_-'), r.P(r.C('a-z0-9'))))` — leading char must be `[a-z]`, separators only between alphanumerics, never consecutive. The function's docstring claims "must start with a letter or `_<letter>`", but Re_name disallows leading `_`. The same docstring assertion is enshrined in [`paths.go:49-55`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/paths.go#L49-L55) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths.go#L49-L55) and produces `_test` (kept as-is when first letter is at index 1 and char 0 is `_`).

  For `--leading-dash`: dashes are mapped to `_`, yielding `__leading_dash`. First letter is at index 2, so `i==1 && out[0]=='_'` fails and the function prepends `d` → `d__leading_dash`. The `__` is two consecutive separators, which `Re_name` disallows (`r.P(r.C('a-z0-9'))` after each separator requires at least one alphanumeric).

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5604 -R gnolang/gno
  go test -v -run 'TestGuessPath_NoGnoModProducesValidPath' ./contribs/gnodev/
  ```
  Output:
  ```
  --- FAIL: TestGuessPath_NoGnoModProducesValidPath/_test
      Messages: guessed path "gno.land/r/dev/_test" must be a valid userlib path
  --- FAIL: TestGuessPath_NoGnoModProducesValidPath/--leading-dash
      Messages: guessed path "gno.land/r/dev/d__leading_dash" must be a valid userlib path
  ```

  CI: [gnodev / test (run 26469837157)](https://github.com/gnolang/gno/actions/runs/26469837157/job/77940022680) is failing for this exact reason.

  **Fix:** rewrite the sanitizer to enforce three invariants after lower-casing and `[^a-z0-9]→_` replacement: (a) drop all leading underscores; (b) collapse runs of `_` to a single `_`; (c) strip trailing `_`. Then if the first remaining char is non-letter, prepend `d`. Drop the `_<letter>` special case — gno's `Re_name` does not allow it. Also fix `TestSanitizePathSegment`'s golden table accordingly — `{"_test", "_test"}`, `{"_1ab", "d_1ab"}`, `{"__abc", "d__abc"}` all encode invalid outputs.
  </details>

- **[unit test enshrines wrong invariant]** [`contribs/gnodev/paths_test.go:11-33`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/paths_test.go#L11-L33) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths_test.go#L11-L33) — `TestSanitizePathSegment` table has rows whose `want` value is itself an invalid `Re_name` segment.
  <details><summary>details</summary>

  Three rows in the table assert outputs that `gnolang.IsUserlib` rejects when concatenated under `gno.land/r/dev/`:

  - `{"_test", "_test"}` — leading `_` violates `Re_name`'s "starts with `[a-z]`".
  - `{"__abc", "d__abc"}` — `__` is two consecutive separators, disallowed.
  - `{"_1ab", "d_1ab"}` — `d_1ab` is actually valid (`d` then `_` between alphanumerics `d` and `1`), so this row alone is fine. But the surrounding rows being wrong show the table was written from the docstring claim, not from `Re_name`.

  Without a sibling `IsUserlib` assertion in `TestSanitizePathSegment`, this test will continue to pass after fixing the sanitizer is partial — only `TestGuessPath_NoGnoModProducesValidPath` actually exercises the round-trip. Add `assert.True(t, gnolang.IsUserlib(path.Join("gno.land/r/dev", tc.want)), "want %q must be a valid IsUserlib tail", tc.want)` to the table loop so both tests fail together when the invariant breaks. Today the contract is asserted in only one place.

  **Fix:** rewrite the `want` column to match the corrected sanitizer (`_test → dtest`, `__abc → dabc`, etc.), and add the `IsUserlib` assertion inside the table loop so future regressions surface at the unit level.
  </details>

## Warnings (should fix)

- **[deletion of extra-root dir mid-session never detected]** [`contribs/gnodev/pkg/packages/loader.go:268-322`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L268-L322) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L268-L322) — `Reload` preserves `rootIdx` so a deleted dir under an `-extra-root` keeps returning a stale `Dir` until `gnodev` restarts.
  <details><summary>details</summary>

  The author documents this at [`loader.go:268-270`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L268-L270) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L268-L270) ("deletion of an extra-root directory mid-session is not detected"). The cost is bounded — next `Resolve` returns a `*Package` whose `Dir` no longer exists, and `ToMemPackage` then fails at use time with a read error. For a dev tool this is acceptable; flag it so the next maintainer doesn't trip over a "phantom package" report.

  **Fix:** add a stat check on the cached `dir` inside `fsLookupLocked` (or in `ensureRootIndexLocked` against `idx` entries) and drop entries whose directories no longer exist. Cheap (one `os.Stat` per hit), bounded by the rootIdx size, and removes the only mid-session staleness window the loader has.
  </details>

- **[unsafe-API exposes Reset/Reload without auth]** [`contribs/gnodev/app.go:422-435`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/app.go#L422-L435) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L422-L435) — `defaultLocalAppConfig.unsafeAPI = true` ([command_local.go:38](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/command_local.go#L38) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/command_local.go#L38)) means `/reset` and `/reload` are always reachable in `gnodev local` mode.
  <details><summary>details</summary>

  Not introduced by this PR but worth re-flagging in the migration window: the `gnodev local` default binds `webListenerAddr = 127.0.0.1:8888`, so by default `/reset` is only loopback. A user who overrides `-web-listener 0.0.0.0:...` and forgets `-unsafe-api=false` is one curl away from a state wipe. No auth, no rate-limiting, no CSRF. Out of scope for this PR but should land in the same migration window since the surrounding semantics are being reworked.

  **Fix:** at minimum, refuse to start the unsafe API endpoints when the web listener is not on a loopback interface unless `-unsafe-api` is explicitly set on the command line (distinguish default-true from explicit-true via `flag.Visit`).
  </details>

- **[GnoRoot from env, not flag, in loader config]** [`contribs/gnodev/app.go:200-207`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/app.go#L200-L207) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L200-L207) — the loader gets `GnoRoot: gnoenv.RootDir()` but the node gets `cfg.root` from the `-root` flag.
  <details><summary>details</summary>

  `setup_node.go:119` calls `DefaultNodeConfig(cfg.root, …)`, but `app.go:204` ignores `cfg.root` and goes straight to `gnoenv.RootDir()`. If a user passes `-root /alt/gnoroot`, the node sees `/alt/gnoroot` (its stdlibs come from there) while the loader looks for `examples/` under the env-derived `$GNOROOT`. With mismatched roots, `LookupFS("gno.land/p/<example>")` and the `-no-examples` import-graph check operate against the wrong tree.

  **Fix:** thread `cfg.root` into `loaderCfg.GnoRoot` (falling back to `gnoenv.RootDir()` when empty, as the loader already does inside `New`). One line, removes a foot-gun.
  </details>

- **[positional dirs added as extra-roots without existence check]** [`contribs/gnodev/app.go:181-194`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/app.go#L181-L194) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L181-L194) — `-extra-root` flag entries are stat-validated; positional `dirs` aren't.
  <details><summary>details</summary>

  Lines 183-189 validate every `cfg.extraRoots` entry and `Warn` + skip on stat failure. Lines 190-194 then take positional args unconditionally: `extraRoots = append(extraRoots, dir)` and `localPaths = append(localPaths, guessPath(...))`. If the user passes a typo'd path positionally, gnodev starts up fine but later the watcher's `filepath.Abs(pkg.Dir)` succeeds against a non-existent dir and `fsnotify.Add` returns silently — the package then appears in `localPaths` (`ds.paths`) but is never actually loadable.

  **Fix:** factor the validation into a small helper and apply it to both `cfg.extraRoots` and positional `dirs`, warning + skipping on stat failure. Also drop the corresponding `localPaths` entry when the directory is skipped so `ds.paths` doesn't reference an unresolvable path.
  </details>

- **[`stripStdlibs` aliasing — fragile, undocumented]** [`contribs/gnodev/pkg/packages/loader.go:475-493`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L475-L493) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L475-L493) — also raised by claude-sonnet-4-6 in [round 1](../1-ed10e81f3/claude-sonnet-4-6_davd-gzl.md#L21). `out := pkgs[:0]` and `kept := imps[:0]` mutate the caller's slices in place; safe today because `vmpackages.Load` returns fresh slices on every call.
  <details><summary>details</summary>

  Confirmed safe — [`gnovm/pkg/packages/load.go:47`](https://github.com/gnolang/gno/blob/7eb33db9e/gnovm/pkg/packages/load.go#L47) · [↗](../../../../../.worktrees/gno-review-5604/gnovm/pkg/packages/load.go#L47) does not cache its return value. But the contract is invisible: a future Load implementation that memoizes its output (perfectly reasonable optimization) would silently corrupt the upstream cache, since `stripStdlibs` writes back into the same backing arrays. Either copy explicitly (`out := make([]*Package, 0, len(pkgs))` and `kept := append([]string(nil), …)`), or document the no-cache requirement next to `loadWithPatterns` so it gets reviewed if/when Load grows a cache.

  **Fix:** add an explicit copy in `stripStdlibs`; the cost is one allocation per Reload on a list that's bounded by workspace size. Worth it to remove the at-a-distance invariant.
  </details>

- **[Setup writes banner to `os.Stderr`, not `ds.io.Err()`]** [`contribs/gnodev/app.go:177`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/app.go#L177) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L177) — also raised by claude-sonnet-4-6 in [round 1](../1-ed10e81f3/claude-sonnet-4-6_davd-gzl.md#L27). The banner bypasses the `commands.IO` abstraction.
  <details><summary>details</summary>

  Every other gnodev output flows through `cio` or `ds.logger`. `printDiscoveryBanner(os.Stderr)` is the only direct stderr write. Tests using `commands.NewTestIO()` cannot capture the banner. Fix: `printDiscoveryBanner(ds.io.Err())`. One line.
  </details>

- **[lock-order invariant undocumented]** [`contribs/gnodev/app.go:337-401`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/app.go#L337-L401) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app.go#L337-L401) — also raised by claude-sonnet-4-6 in [round 1](../1-ed10e81f3/claude-sonnet-4-6_davd-gzl.md#L24). The proxy callback's lock order (`l.mu → muClients → muNode → l.mu` via Reload) is consistent but encoded only in this single call site.
  <details><summary>details</summary>

  Three locks compose here: `loader.mu`, `emitter.muClients`, `node.muNode`. The first `l.mu` is released before `LockEmit`, so the held-locks state is `{muClients, muNode}` when `loader.Reload` is invoked inside `devNode.Reload`, which then takes `l.mu`. As long as nothing else holds `l.mu` while waiting for `muClients` or `muNode`, there's no cycle — but that's a property of the entire codebase, not a local one.

  **Fix:** add a comment block at the top of `loader.go` listing the lock-order invariant (`muNode → l.mu` and `muClients → muNode → l.mu`, never the reverse). Future contributors editing either lock site will see it.
  </details>

- **[scanRoot skip list misses common dirs]** [`contribs/gnodev/pkg/packages/loader.go:229-231`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L229-L231) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L229-L231) — skips `.foo`, `node_modules`, `_build` but no other `_*` dirs, no `vendor`, no `testdata`.
  <details><summary>details</summary>

  The gno tree itself contains `_assets`, `_tags`, `_all` (under `tm2/pkg/db/`), `docs/_assets`, etc. None contain `gnomod.toml`, so the perf cost is bounded — but a user pointing `-extra-root` at a Go-style monorepo will scan into `vendor/` and `testdata/` for nothing. The walker doesn't follow symlinks; that's fine.

  **Fix:** add `vendor` and a generic `_<x>` prefix (similar to `.<x>`) to the skip set. Three lines; tracks Go's own walk conventions (`go build` skips `_*` and `.*` dirs).
  </details>

## Nits

- [`contribs/gnodev/pkg/packages/loader.go:264-321`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L264-L321) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L264-L321) — `Reload` re-resolves every tracked path even when the workspace load already returned it. Self-acknowledged ("double-resolve, wasteful but not incorrect" in round 1); the cost is bounded by `len(tracked)`. Fix: after `appendUnique(pkgs...)`, mark each returned ImportPath in `seen` before iterating `trackedPaths`, then `continue` on hit before calling `Resolve` — saves one FS walk per overlap.

- [`contribs/gnodev/pkg/packages/loader.go:160-177`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L160-L177) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L160-L177) — `extractPackageName` silently drops parse failures. Acceptable for RPC fallback (we already have the bytes; if we can't parse one we still want the package), but a debug log on the swallowed error helps when an RPC fetch returns a malformed file.

- [`contribs/gnodev/pkg/packages/package.go:62-64`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/package.go#L62-L64) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/package.go#L62-L64) — also raised by claude-sonnet-4-6 in round 1. `packageFromMemPackage` sets `Kind: KindFS` with a comment saying "irrelevant... classification happens at resolve time". The caller (`rpcLookup`) immediately overrides with `KindRemote`; if a new caller appears without an override, the package wrongly registers as FS and the watcher attaches to a non-existent dir. Fix: default to `KindUnknown`.

- [`contribs/gnodev/pkg/dev/node_test.go:670-677`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/dev/node_test.go#L670-L677) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/dev/node_test.go#L670-L677) — also raised by claude-sonnet-4-6 in round 1. `holder.node.paths` reads unexported state directly to dodge a `muNode` re-entry; works but is fragile. The lock-recursion path is what `n.paths` exists for. Fix: split `Paths()` into `Paths()` (locked) and `pathsLocked()` (callers hold lock).

- [`contribs/gnodev/paths.go:11-21`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/paths.go#L11-L21) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths.go#L11-L21) — docstring claims `Re_name` allows "letters/digits/underscore only, must start with a letter or `_<letter>`". `Re_name` also permits `-` between alphanumerics, and does NOT permit a leading `_`. Two errors in one comment. Fix when fixing the Critical above.

- [`contribs/gnodev/pkg/packages/loader.go:144-158`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L144-L158) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L144-L158) — comment says "cfg.Fetcher is set once in New and never mutated, so no lock is required". But the field is `l.fetcher`, not `l.cfg.Fetcher`. The reasoning is correct, the field name in the comment is wrong. Trivial.

## Missing Tests

- **[no test asserts `Re_name` round-trip]** [`contribs/gnodev/paths_test.go`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/paths_test.go) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/paths_test.go) — `TestSanitizePathSegment` and `TestGuessPath_NoGnoModProducesValidPath` are siblings but don't share a check. Fix: fold the IsUserlib invariant into `TestSanitizePathSegment` itself (see Critical #2). Both tests should fail together when the contract breaks.

- **[no test for `cfg.GnoRoot != gnoenv.RootDir()`]** [`contribs/gnodev/pkg/packages/loader.go:195-202`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L195-L202) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L195-L202) — `lookupRoots` builds `filepath.Join(l.cfg.GnoRoot, "examples")` but no test exercises a non-default GnoRoot. The Warning above about `cfg.root` vs `gnoenv.RootDir()` lives here.

- **[no test for `Reload` after `loadRootStandalone`]** [`contribs/gnodev/pkg/packages/loader.go:378-397`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L378-L397) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L378-L397) — `LoadAll`'s root-standalone path inserts into `tracked` via Resolve. The next `Reload` then re-walks those paths. Not exercised by any test.

- **[no concurrent Resolve test]** [`contribs/gnodev/pkg/packages/loader.go:69-107`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L69-L107) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L69-L107) — the lock dance (RLock fast path / Write FS / lock-free RPC / Write merge) is core to the design but has no `t.Parallel` + concurrent-Resolve test. A 100-goroutine fanout against an `InMemoryFetcher` under `-race` would prove the no-deadlock + at-most-one-insert claim.

- **[no end-to-end `-remote-override` test]** [`contribs/gnodev/app_config_remote_override.go`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/app_config_remote_override.go) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/app_config_remote_override.go) — also raised by claude-sonnet-4-6 in round 1. Flag parsing is covered; the wiring through `rpcpkgfetcher.New(cfg.RemoteOverrides)` is not. A unit-level test using a stub PackageFetcher could assert the override map flows through to fetch attempts for paths under the overridden domain.

## Suggestions

- [`contribs/gnodev/pkg/packages/loader.go:78-89`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/loader.go#L78-L89) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/loader.go#L78-L89) — Resolve holds the write lock for the entire FS walk. For a large `-extra-root` (think the gnolang/gno monorepo as an extra root), the first miss serializes every other Resolve until the walk completes. Consider per-root mutex (`map[root]*sync.Mutex`) so concurrent first-misses against *different* roots don't serialize. Likely YAGNI for typical dev use; flag for future.

- [`contribs/gnodev/pkg/packages/examples_check.go:54-97`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/pkg/packages/examples_check.go#L54-L97) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/pkg/packages/examples_check.go#L54-L97) — `importsInDir` uses `go/parser.ImportsOnly`. Works because Gno's package + import declarations are valid Go syntax — but if a gno file ever has a leading directive Go's parser doesn't recognize (e.g. a hypothetical `//gno:build`), this silently drops all imports. Consider switching to `gnolang.PackageFromDir` or whatever gno's import-only extractor is (if one exists upstream), keeping the comment explicit about why `go/parser` is acceptable.

- [`contribs/gnodev/adr/pr4957_gnodev_native_loader.md`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/adr/pr4957_gnodev_native_loader.md) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/adr/pr4957_gnodev_native_loader.md) — the ADR is thorough and well-argued. Worth keeping as long-form context; suggest cross-linking it from [`contribs/gnodev/README.md`](https://github.com/gnolang/gno/blob/7eb33db9e/contribs/gnodev/README.md) · [↗](../../../../../.worktrees/gno-review-5604/contribs/gnodev/README.md) so future contributors find it.

## Questions for Author

- The default `unsafe-api=true` in `gnodev local` predates this PR but is touched by the broader UX rework. Is locking this down behind a loopback-only check in-scope for #5604 follow-ups, or a separate issue?
- `LookupFS` walks the example-tree every time `-no-examples` checks an import — fine for one-shot at startup. Is the helper expected to be called more than once per process? If yes, the rootIdx cache already memoizes the walk; if no, the entry point could narrate that.
- The Reload doc-comment says deletion of an extra-root dir is not detected. Was a per-Resolve `os.Stat` on the cached `Dir` evaluated and rejected as too expensive, or just not considered?
