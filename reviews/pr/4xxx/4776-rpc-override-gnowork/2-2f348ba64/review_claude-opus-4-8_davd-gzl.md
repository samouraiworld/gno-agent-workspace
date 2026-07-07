# PR [#4776](https://github.com/gnolang/gno/pull/4776): feat: rpc override in gnowork.toml

URL: https://github.com/gnolang/gno/pull/4776
Author: n0izn0iz | Base: master | Files: 7 | +262 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `2f348ba64` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4776 2f348ba64`

Round 2. Head advanced `a752ac3b` → `2f348ba64` (branch rebased, +2 commits): tests added, `ReadGnowork` now wraps the file path, and the previous silent warning became a hard error. That last change is the source of this round's headline finding. Prior-round warnings resolved: opaque parse error (now wrapped), missing newline (warning removed entirely), no tests (three test files added).

**TL;DR:** Lets a developer point package downloads at a custom RPC per chain domain by adding `[domains."gno.land"] rpc = "http://localhost:26657"` to `gnowork.toml`, instead of the default `https://rpc.gno.land:443`. Aimed at the case where the public RPC is unreachable and you fetch published packages from a local node.

**Verdict: NEEDS DISCUSSION** — the design is still unresolved (moul opposes per-domain source overrides in principle and gated merge on a cache safeguard that is not implemented; leohhhn asked to drop the `domains.<name>` keying, which the code still ships), and the new hard-error path makes gnodev fail to boot in any workspace whose `gnowork.toml` carries an override, while a mistyped override key is silently ignored and falls back to the public endpoint.

## Summary

Adds a `[domains."<domain>"] rpc = "<url>"` block to `gnowork.toml`. `findLoaderContextFor` now parses the file on every workspace resolution and attaches a `*Gnowork`; `Load` calls `applyRPCOverrides`, which type-asserts the fetcher to a new `RPCPackageFetcher` interface and pushes the overrides via `OverrideDomainsRPCs`, or returns an error when the fetcher does not implement it. The override endpoint replaces the default in `rpcURLFromPkgPath`. The hard-error behavior is new this round (commit 2f348ba64, "error instead of silently dropping unsupported rpc overrides").

```
gnowork.toml ─► findLoaderContextFor ─► ReadGnowork (parse on EVERY workspace Load)
                                              │
                        loaderCtx.Gnowork.rpcOverrides()  (drops empty rpc values)
                                              │
                     applyRPCOverrides(conf.Fetcher, overrides)
                        │                                   │
             fetcher is RPCPackageFetcher            fetcher is NOT
                        │                                   │
             OverrideDomainsRPCs(map)            return error  ◄── gnodev lands here
```

## Glossary

- `gnowork.toml` — workspace manifest found by walking up from cwd in `findWorkspaceRootDir`. Historically a presence marker only; every one in the repo is an empty file.
- `RPCPackageFetcher` — new interface: `PackageFetcher` plus `OverrideDomainsRPCs(map[string]string)`.
- `disabledFetcher` / `domainFetcher` — gnodev's two fetchers. Filesystem-only by default; with `-remote <domain>=<rpc>` it fetches only the listed domains. Neither implements `OverrideDomainsRPCs`.
- `--remote-overrides` — pre-existing `gno mod download` flag that sets the same per-domain map at fetcher construction.
- modcache marker — `.markers/<bech32(pkgPath)>` file; its presence makes `DownloadPackageToCache` skip re-download.

## Examples

| `gnowork.toml` | Parses? | Override applied | Effect |
|---|---|---|---|
| `[domains."gno.land"]`<br>`rpc = "http://localhost:26657"` | yes | `gno.land → localhost` | works for `gno mod download`; **gnodev refuses to boot** |
| `[domain."gno.land"]` (missing `s`) | yes | none | silently fetches from `https://rpc.gno.land:443`, no error |
| `[domains."gno.land"]`<br>`rcp = "..."` (typo) | yes | none | silently fetches from public endpoint, no error |
| `[domains."gno.land"` (unclosed) | **no** | — | every workspace command aborts (`gno test`/`lint`/`list`/gnodev) |

## Critical (must fix)

None.

## Warnings (should fix)

- **[gnodev refuses to boot when the workspace file has an override]** [`gnovm/pkg/packages/load.go:167-177`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/load.go#L167-L177) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L167-L177) — the new hard-error path aborts `Load` for any fetcher that is not an `RPCPackageFetcher`; gnodev's default fetcher is not one, so a `gnowork.toml` override kills gnodev startup.
  <details><summary>details</summary>

  gnodev drives `packages.Load` with `l.fetcher` ([`contribs/gnodev/pkg/packages/loader.go:514`](https://github.com/gnolang/gno/blob/2f348ba64/contribs/gnodev/pkg/packages/loader.go#L514) · [↗](../../../../../.worktrees/gno-review-4776/contribs/gnodev/pkg/packages/loader.go#L514)), which is `disabledFetcher` by default or `domainFetcher` with `-remote` ([`contribs/gnodev/pkg/packages/fetcher.go:16-40`](https://github.com/gnolang/gno/blob/2f348ba64/contribs/gnodev/pkg/packages/fetcher.go#L16-L40) · [↗](../../../../../.worktrees/gno-review-4776/contribs/gnodev/pkg/packages/fetcher.go#L16-L40)). Neither implements `OverrideDomainsRPCs`, so `applyRPCOverrides` hits the error branch and gnodev never initializes. Live boot from the PR worktree in a workspace whose `gnowork.toml` has one override:

  ```
  unable to initialize the node: reload packages: load packages: gnowork.toml requests rpc
  overrides but the configured package fetcher (packages.disabledFetcher) does not support them
  ```

  Removing the override line boots gnodev normally. The file is workspace-global, so adding an override for `gno mod download` breaks gnodev in the same tree. This is a footgun for the exact dev workflow the feature targets. It also does not help to merely forward the type-assert: `domainFetcher.FetchPackage` refuses any domain not present in its `-remote` map ([`fetcher.go:36-38`](https://github.com/gnolang/gno/blob/2f348ba64/contribs/gnodev/pkg/packages/fetcher.go#L36-L38) · [↗](../../../../../.worktrees/gno-review-4776/contribs/gnodev/pkg/packages/fetcher.go#L36-L38)), and that map is never updated by overrides, so a gnowork override for a domain not also passed via `-remote` stays refused. Fix: decide whether a workspace override should apply to gnodev at all; if yes, have gnodev's fetchers implement `OverrideDomainsRPCs` (forward to `inner` and register the domain); if no, do not hard-fail fetchers that legitimately never fetch remotely. Either way, pin the chosen behavior with a test. Repro in [comment.md](comment_claude-opus-4-8.md).
  </details>

- **[a mistyped override is silently ignored and falls back to the public endpoint]** [`gnovm/pkg/packages/gnowork.go:9-11`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/gnowork.go#L9-L11) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L9-L11) — `ParseGnowork` does a non-strict `toml.Unmarshal` with no `toml:"rpc"` tag and no validation, so a misspelled table (`[domain."gno.land"]`) or key (`rcp =`) yields no override and no error, and dependencies silently download from `https://rpc.gno.land:443`.
  <details><summary>details</summary>

  go-toml v1.9.5 does not reject unknown keys or tables, and `GnoworkDomain.RPC` ([`gnowork.go:9-11`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/gnowork.go#L9-L11) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L9-L11)) matches only exact `rpc`/`RPC`. Verified at 2f348ba64: `[domain."gno.land"]` (missing `s`), `rcp = ...`, and an unknown top-level key all parse to an empty override map with `err == nil`. For a feature whose whole purpose is redirecting where source is fetched from, a silent fall back to the public endpoint is worse than a loud failure. Notably the parser fails loudly on the wrong thing (a syntax typo, next finding) and stays silent on the semantic typo that matters. Fix: add the `toml:"rpc"` tag and reject unknown keys at parse time, or validate that a declared domain produced a usable override. Repro in [comment.md](comment_claude-opus-4-8.md).
  </details>

- **[a malformed gnowork.toml now breaks every workspace command]** [`gnovm/pkg/packages/load.go:201-205`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/load.go#L201-L205) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L201-L205) — `findLoaderContextFor` now reads and parses the file on every workspace resolution; before this PR the content was never read here, so a syntactically broken file that was silently tolerated now aborts `gno test`/`lint`/`list`/`deplist`/`transpile` and gnodev.
  <details><summary>details</summary>

  On master, the workspace branch returned `{Root, IsWorkspace: true}` without opening the file; the PR calls `ReadGnowork(filepath.Join(root, "gnowork.toml"))` and propagates any parse error. This is the first time `gnowork.toml` must be valid TOML for unrelated commands to run. All 20 tracked `gnowork.toml` files are empty (empty parses cleanly), so nothing breaks today, but a hand-edited broken file now fails the whole workspace. Reproduced PR vs master with `gno list ./...` over an unclosed table header: master prints the package list, the PR aborts with `parse gnowork file "...": unexpected token unclosed table key`. This blast-radius widening is not called out anywhere. Repro in [comment.md](comment_claude-opus-4-8.md).
  </details>

- **[workspace file silently overrides the --remote-overrides CLI flag]** [`gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go:61-71`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L61-L71) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L61-L71) — `gno mod download --remote-overrides=gno.land=X` builds the fetcher with the CLI map, then `Load` merges the workspace map on top, so for a shared domain the file wins over the explicit flag; no test and no doc pins this.
  <details><summary>details</summary>

  `execModDownload` calls `rpcpkgfetcher.New(cliMap)` ([`gnovm/cmd/gno/mod.go:246`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/cmd/gno/mod.go#L246) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/cmd/gno/mod.go#L246)); `applyRPCOverrides` then calls `OverrideDomainsRPCs(workspaceMap)`, which is last-write-wins per domain. Verified: with the CLI setting `gno.land=CLI` and the workspace setting `gno.land=WORKSPACE`, the fetcher ends up with `WORKSPACE`. The conventional direction is the reverse (an explicit flag beats a config file). This also sits oddly next to the existing guard at [`mod.go:248`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/cmd/gno/mod.go#L248) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/cmd/gno/mod.go#L248) that forbids `--remote-overrides` with a custom fetcher. Fix: pick a precedence, document it, and add a test. Repro in [comment.md](comment_claude-opus-4-8.md).
  </details>

- **[the only gnowork.toml doc says the file must be empty]** [`docs/resources/configuring-gno-projects.md:139-141`](https://github.com/gnolang/gno/blob/2f348ba64/docs/resources/configuring-gno-projects.md?plain=1#L139-L141) · [↗](../../../../../.worktrees/gno-review-4776/docs/resources/configuring-gno-projects.md#L139) — the doc states `gnowork.toml` "does not have any configuration options and should be an empty file"; the PR adds a real schema and ships no docs change, so the sole reference now tells users the feature does not exist.
  <details><summary>details</summary>

  leohhhn's thread asked to document the schema in the `gnowork.toml` docs added by [#4666](https://github.com/gnolang/gno/pull/4666). The fix must update this section and the `<!-- TODO: allow configuration of dependency source priority/hierarchy -->` note at line 147, which this PR partially fulfills.
  </details>

- **[design still unresolved; moul's cache safeguard is not implemented]** [`PR #4776 thread`](https://github.com/gnolang/gno/pull/4776#issuecomment-3309948437) — moul opposes per-domain source overrides in principle and, if shipped, asked to "keep the fully qualified domain in the download path" so removing the override later re-fetches instead of reusing the earlier download; the cache does key on the full pkgPath but the marker short-circuit means the safeguard's goal is not met.
  <details><summary>details</summary>

  Three positions remain unreconciled: moul (don't ship source overrides; a package's identity is its domain), leohhhn (keep RPC config but drop `domains.<name>` for a single top-level `rpc =`, default Staging), n0izn0iz (per-domain keying is required because each chain has its own RPC namespace). The code ships the `domains.<name>` form leohhhn asked to change.

  moul's concrete condition: the download cache is keyed by pkgPath only ([`gnovm/pkg/packages/modcache.go:15-17`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/modcache.go#L15-L17) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/modcache.go#L15-L17)), and `DownloadPackageToCache` early-returns when the marker exists ([`modcache.go:53-57`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/modcache.go#L53-L57) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/modcache.go#L53-L57)). The override endpoint never enters the cache key, so bytes fetched from the override land under the canonical path and are reused after the override is removed, until the modcache is wiped. The domain is literally in the path (it is part of the pkgPath), yet the re-fetch goal moul described is not achieved. This caching is shared with the pre-existing `--remote-overrides` flag, so closing it is a broader change; surfacing it here because moul gated merge on it. Not posted as an inline comment; belongs in the design thread.
  </details>

## Nits

- [`gnovm/pkg/packages/pkgdownload/pkgfetcher.go:13-16`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/pkgdownload/pkgfetcher.go#L13-L16) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/pkgfetcher.go#L13-L16) — `RPCPackageFetcher` has no godoc; the sibling `PackageFetcher` in the same file is documented.
- [`gnovm/pkg/packages/load.go:59-62`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/load.go#L59-L62) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L59-L62) — `applyRPCOverrides` runs before the `if !conf.Deps { return }` short-circuit at [`load.go:83`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/load.go#L83) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L83), so a purely-local load that never fetches still errors on an override it would not consume. Move the check into the deps path or gate it on `conf.Deps`.
- [`gnovm/pkg/packages/gnowork.go:24`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/gnowork.go#L24) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/gnowork.go#L24) — `rpcOverrides` skips only an exactly-empty rpc; a whitespace-only value (`rpc = " "`) passes through and is used raw as the endpoint, surfacing only as an opaque client error at fetch time.

## Missing Tests

- **[real Load path and typo behavior are uncovered]** [`gnovm/pkg/packages/load.go:196-206`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/load.go#L196-L206) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/load.go#L196-L206) — the new tests cover `ParseGnowork`, `ReadGnowork`, and `applyRPCOverrides` as units, but nothing drives `Load` → `findLoaderContext` → `ReadGnowork` reading a real workspace file, which is exactly where the gnodev break and the precedence behavior live.
  <details><summary>details</summary>

  Ready-to-add white-box tests (executed green at 2f348ba64) covering: an override plus a gnodev-style non-RPC fetcher through the real `Load` (errors), an empty rpc through `Load` (no abort), gnowork-vs-flag precedence, and the go-toml silent-accept cases. Artifact: [`gnowork_rpcoverride_blue_test.go`](../../../../../reviews/pr/4xxx/4776-rpc-override-gnowork/2-2f348ba64/tests/gnowork_rpcoverride_blue_test.go). Pasted into [comment.md](comment_claude-opus-4-8.md).
  </details>

## Suggestions

- [`gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go:22-26`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L22-L26) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L22-L26) — `New` stores the caller's map by reference and `OverrideDomainsRPCs` mutates it in place ([`rpcpkgfetcher.go:69`](https://github.com/gnolang/gno/blob/2f348ba64/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L69) · [↗](../../../../../.worktrees/gno-review-4776/gnovm/pkg/packages/pkgdownload/rpcpkgfetcher/rpcpkgfetcher.go#L69)).
  <details><summary>details</summary>

  Verified: after `applyRPCOverrides`, the caller's original map passed to `New` is mutated. Benign in-tree (the only real caller, `execModDownload`, passes a fresh single-use map), but a latent aliasing footgun for any future caller that shares or reuses the map, and the seed for a data race if one `rpcpkgfetcher` is ever shared across concurrent `Load` calls with overrides (`FetchPackage` reads the map while `OverrideDomainsRPCs` writes it). Copy in `New`, or allocate in `OverrideDomainsRPCs` rather than mutate.
  </details>

## Open questions

- Schema is a one-way door: no version field, no fallback/multi-RPC (leohhhn raised this), key named `rpc` where leohhhn suggested `dep_source`, and `rpc = ""` is indistinguishable from absent, foreclosing a future "explicitly disable" value. Not posted; belongs in the design thread once direction is set.
- moul's cache-reuse concern (above) is shared with the pre-existing `--remote-overrides` flag, so it is a broader cleanup than this PR; noting for whoever settles the direction.
