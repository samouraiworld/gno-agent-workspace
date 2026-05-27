# PR #4776: feat: rpc override in gnowork.toml

URL: https://github.com/gnolang/gno/pull/4776
Author: n0izn0iz | Base: master | Files: 4 | +85 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-4776 a752ac3b` (then `gh -R gnolang/gno pr checkout 4776` inside it)

**Verdict: NEEDS DISCUSSION** — design direction is still unresolved (moul wants strict domain identity; leohhhn pushed back on per-domain config; author defended it on multi-chain grounds), and the implementation lacks tests, has a missing newline in the warning, swallows a malformed-toml error opaquely, and silently overrides a CLI flag (`--remote-overrides`) that already does the same job.

## Summary

Adds a `[domains."<domain>"] rpc = "<url>"` block to `gnowork.toml` so the loader can swap the default `https://rpc.<domain>:443` endpoint for a custom one per chain domain. Targets the workaround case for #4739 (official RPC unreachable / running a local devnet against published package paths). Implementation: parse the toml in `findLoaderContext`, type-assert the configured fetcher to a new `RPCPackageFetcher` interface, call `OverrideDomainsRPCs(map)` on it; non-RPC fetchers print a warning and continue.

```
gnowork.toml ─► Gnowork.Domains{rpc} ─► loaderCtx.Gnowork.rpcOverrides()
                                              │
                                              ▼
                  conf.Fetcher ── type-assert ─► RPCPackageFetcher
                                              │
                              .OverrideDomainsRPCs(map[domain]rpcURL)
                                              │
                                              ▼
                            rpcURLFromPkgPath picks override > default
```

## Glossary

- `gnowork.toml` — workspace manifest (analog to `go.work`), found by walking up from cwd in `findWorkspaceRootDir`.
- `RPCPackageFetcher` — new interface this PR introduces, embeds `PackageFetcher` plus `OverrideDomainsRPCs(map[string]string)`.
- `remoteOverrides` — pre-existing map inside `gnoPackageFetcher` already settable via the `--remote-overrides` CLI flag on `gno mod download`.
- `findLoaderContext` — entry point in `load.go` that locates the workspace; now also parses `gnowork.toml`.

## Fix

Before, `findLoaderContext` only located the workspace dir and returned `Root + IsWorkspace`. After, it reads `gnowork.toml` and attaches the parsed `*Gnowork` struct to the loader context; `Load` extracts non-empty per-domain RPC overrides and pushes them into the fetcher via the new `RPCPackageFetcher` interface. The interface segregation is so the noop/examples fetchers can stay unchanged — but it silently no-ops with a warning when the user picked a non-RPC fetcher ([`gnovm/pkg/packages/load.go:60-67`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/load.go#L60-L67) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L60-L67)).

## Critical (must fix)

None.

## Warnings (should fix)

- **[unresolved design pushback from maintainer]** [`PR #4776 thread`](https://github.com/gnolang/gno/pull/4776#issuecomment-3309948437) — moul objected to the approach in principle ("a package should only come from two sources: Published [unique domain] / Dev [local file]"); leohhhn then asked the author to drop the `domains.<name>` keying and use a single top-level `rpc = ...`; the author pushed back saying multi-chain workflows need per-domain mapping. Thread is unresolved as of last activity (2025-09-24).
  <details><summary>details</summary>

  Three positions on the table, none reconciled:
  - moul: don't ship the override at all; force users to improve tooling around multi-domain instead. Asked for one safeguard if shipped — "keep the fully qualified domain in the download path". The current code does satisfy that (`PackageDir` keys cache by full pkgPath including domain in `gnovm/pkg/packages/modcache.go:15-17`), but the broader objection remains.
  - leohhhn: keep RPC config, drop `domains.<name>`, default to Staging. So `rpc = "..."` instead of `[domains."gno.land"] rpc = "..."`.
  - n0izn0iz: per-domain mapping is necessary because each chain has its own RPC namespace (`https://rpc.gno.land` vs hypothetical `https://rpc.osmosis.land`); a single global `rpc` field can't express this.

  Fix: don't merge until the three of them agree. The schema in `gnowork.toml` is a one-way door — once published, removing or renaming `[domains.<name>]` is a breaking change for every workspace that adopted it.
  </details>

- **[silently overrides CLI flag without precedence rule]** [`gnovm/pkg/packages/load.go:62-63`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/load.go#L62-L63) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L62-L63) — `gno mod download --remote-overrides=gno.land=...` already exists ([`gnovm/cmd/gno/mod.go:151-163`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/cmd/gno/mod.go#L151-L163) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/cmd/gno/mod.go#L151-L163)); `Load` always calls `OverrideDomainsRPCs` after the CLI map was injected at fetcher construction, so the workspace file silently wins.
  <details><summary>details</summary>

  Sequence today:
  1. `execModDownload` parses `--remote-overrides` and calls `rpcpkgfetcher.New(cliMap)` ([`gnovm/cmd/gno/mod.go:246`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/cmd/gno/mod.go#L246) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/cmd/gno/mod.go#L246)).
  2. `Load` reads `gnowork.toml`, calls `cpf.OverrideDomainsRPCs(workspaceMap)` ([`load.go:63`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/load.go#L63) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L63)).
  3. `OverrideDomainsRPCs` writes each `workspaceMap[domain]` over the existing entry ([`rpcpkgfetcher.go:68-74`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L68-L74) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L68-L74)).

  Result: workspace toml always beats the CLI flag. The opposite (CLI overrides file) is the conventional precedence for everything in this codebase. Either pick a direction explicitly, document it, and test it — or refuse to apply file overrides when a CLI flag is set. Fix: in `Load`, only apply file overrides for domains not already in the fetcher map, OR document the file > flag precedence in `--remote-overrides` help text and add a test that pins the behavior.
  </details>

- **[shared fetcher state leaks across `Load` calls]** [`gnovm/pkg/packages/load.go:60-67`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/load.go#L60-L67) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L60-L67) — `OverrideDomainsRPCs` mutates the fetcher in place and never clears prior entries; callers that reuse one `LoadConfig` across multiple `Load` invocations (which already happens — see [`gnovm/cmd/gno/tool_deplist.go:54-89`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/cmd/gno/tool_deplist.go#L54-L89) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/cmd/gno/tool_deplist.go#L54-L89)) accumulate overrides from every workspace they touch.
  <details><summary>details</summary>

  Inside `execDeplist`, the same `loadCfg` (same `Fetcher`) is passed to `packages.Load` once for the user args and then once per discovered dep in a loop. If `cwd` changes between calls (it can — `tool_deplist` is per-package), each call's `findLoaderContext` resolves to a different workspace, and `OverrideDomainsRPCs` keeps merging into the fetcher map. There's no clear/reset between calls. The current additive semantics (`map[domain] = rpc`, never delete unless the input has an empty string) mean override entries linger from a previous workspace even after switching workspaces.

  Practically: a single CLI invocation is unlikely to traverse multiple workspaces, so this is more of a future-pitfall than an active bug. But it's a footgun for any test harness or long-running process that calls `Load` repeatedly.

  Fix: replace the in-place mutation with a per-call wrapper — e.g. construct a fresh fetcher with the merged map for the duration of this `Load` call, or pass overrides explicitly to `FetchPackage` via `LoadConfig` rather than as fetcher state. The interface segregation (`RPCPackageFetcher`) is only there to support this mutation — if you remove the mutation, the interface goes too.
  </details>

- **[malformed gnowork.toml gives an opaque error]** [`gnovm/pkg/packages/gnowork.go:33`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/gnowork.go#L33) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L33) — `toml.Unmarshal` errors are returned bare from `ParseGnowork` and propagate up through `Load`; the error message looks like `(1, 6): was expecting token =, but got "is" instead` with no mention of `gnowork.toml` or the file path. Before this PR, the workspace dir lookup never read the file, so a malformed file never produced a parse error at this stage.
  <details><summary>details</summary>

  Reproducer (manual): drop `this is not valid toml ===` into a workspace's `gnowork.toml` and run any `gno` command from inside. The user sees a cryptic position-prefixed message and no file path.

  Fix: in `ReadGnowork`, wrap with `fmt.Errorf("parse %s: %w", file, err)`. Two lines, makes the error self-describing.
  </details>

- **[missing newline in warning]** [@n0izn0iz](https://github.com/gnolang/gno/pull/4776#discussion_r2362787000) [`gnovm/pkg/packages/load.go:65`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/load.go#L65) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L65) — `fmt.Fprintf(conf.Out, "gno: warning: ignored rpc overrides, fetcher has no support for it")` — no trailing `\n`. Author already self-flagged this in a review-suggestion thread but never pushed the change.
  <details><summary>details</summary>

  Fix: append `\n` per author's own suggestion in the PR thread.
  </details>

- **[no tests for the new feature]** [`gnovm/pkg/packages/gnowork.go`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/gnowork.go) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go) — leohhhn explicitly asked for a short test in [the review thread](https://github.com/gnolang/gno/pull/4776#issuecomment-3309448275); none was added. `ParseGnowork`, `ReadGnowork`, `rpcOverrides`, and the new `OverrideDomainsRPCs` path are entirely uncovered (codecov reports 35.9% patch coverage, 25 missing lines).
  <details><summary>details</summary>

  Minimum useful tests:
  1. `ParseGnowork` round-trip: well-formed `[domains."gno.land"] rpc = "..."` produces the expected map.
  2. `(*Gnowork)(nil).rpcOverrides()` returns nil (the nil-receiver guard added in commit 2 is otherwise untested).
  3. Empty `gnowork.toml` (existing workspace fixtures all look like this) parses cleanly and yields an empty override map.
  4. `Load` end-to-end with a fixture workspace that sets an override, asserting the fetcher receives it.

  Fix: add `gnovm/pkg/packages/gnowork_test.go` covering (1)-(3); cover (4) by extending `load_test.go` with a workspace fixture under `testdata/`.
  </details>

## Nits

- [`gnovm/pkg/packages/pkgdownload/pkgfetcher.go:13-16`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/pkgdownload/pkgfetcher.go#L13-L16) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/pkgfetcher.go#L13-L16) — interface has no godoc. Every public interface in this package has a comment; this is the only one without.
- [`gnovm/pkg/packages/gnowork.go:31`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/gnowork.go#L31) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L31) — `ParseGnowork(bz []byte)` and `ReadGnowork(file string)` are exported with no godoc. Sibling parsers (`gnomod.Parse...`) follow `// ParseX ... description` convention.
- [`gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go:28`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L28) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L28) — unrelated change: doc comment markdown link `[pkgdownload.PackageFetcher]` collapsed to plain `pkgdownload.PackageFetcher`. This loses the godoc cross-reference. Revert or apply consistently.
- [`gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go:68-74`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L68-L74) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L68-L74) — the `if rpc == "" { delete(...) }` branch is dead given the only caller (`Load`) already filters empty values out in `rpcOverrides`. If you want the delete semantics, drop the filter in `rpcOverrides`; otherwise drop the delete branch.

## Missing Tests

- **[parse round-trip, nil receiver, empty file]** [`gnovm/pkg/packages/gnowork.go`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/gnowork.go) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go) — see Warnings section. Self-flagged by leohhhn, never addressed.
- **[CLI flag + workspace file interaction]** [`gnovm/pkg/packages/load.go:60-67`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/load.go#L60-L67) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L60-L67) — no test pins which one wins when both are set. Current behavior (file wins) should be locked in by a test once the precedence question is settled.

## Suggestions

- [`gnovm/pkg/packages/gnowork.go:9-15`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/gnowork.go#L9-L15) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L9-L15) — consider adding `toml` struct tags so the binding is explicit (currently relies on case-insensitive field name matching by pelletier/go-toml v1). Makes the on-disk schema independent of Go field naming refactors.
  <details><summary>details</summary>

  E.g.:
  ```go
  type GnoworkDomain struct {
      RPC string `toml:"rpc"`
  }
  type Gnowork struct {
      Domains map[string]GnoworkDomain `toml:"domains"`
  }
  ```
  </details>

- [`gnovm/pkg/packages/gnowork.go`](https://github.com/gnolang/gno/blob/a752ac3b/gnovm/pkg/packages/gnowork.go) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go) — once the design lands, document the schema in `docs/` (the gnowork.toml docs added in #4666 per leohhhn's comment). Discoverability of new toml keys is otherwise zero.

## Questions for Author

- Where does this leave moul's objection? Has the principle ("a package == its blockchain by domain, period") been overruled, deferred, or accepted-with-mitigation? The PR sat 7+ months with no resolution on the thread.
- What is the intended precedence between `--remote-overrides` CLI flag and `gnowork.toml`? Document and test whichever you pick.
- Is `domains.<chain>` table form preferred over `[[domain]]` array-of-tables (which would also support the multi-fallback case leohhhn raised)? The current map keying makes one domain = exactly one override; any future "fallback RPC list" would need a schema break.
