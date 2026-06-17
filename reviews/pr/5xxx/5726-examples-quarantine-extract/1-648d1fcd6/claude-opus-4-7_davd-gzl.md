# PR #5726: chore: extract non-test-13 example packages to examples-quarantine/

URL: https://github.com/gnolang/gno/pull/5726
Author: gfanton | Base: master | Files: 761 | +207 -53
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `648d1fcd6` (stale — +13 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5726 648d1fcd6`

**Verdict: APPROVE** — split is cleanly contained, CI is fully green, `TestExamplesLoad` passes locally; remaining concerns are doc drift and `ResolveExamplePath` UX, none blocking.

## Summary

Mechanically isolates 171 unaudited example packages from the 132 test-13 safe-listed ones (88 on the genesis list + 44 hardcoded in gnovm/integration tests, transitively closed) by relocating them under `examples/quarantine/`, while preserving on-chain module paths (gnomod.toml `module = ...` unchanged) so genesis identity is unaffected. A single `examples/gnowork.toml` covers both subtrees, so `gno test ./...`, `gno lint`, and `gno fmt` see them as one workspace and cross-tree imports resolve natively. Three plumbing changes are required where the workspace walker isn't reached: gnodev `RootResolver(examples/quarantine)` chaining (`--no-quarantine` flag, off in `local`, on in `staging`); the `gno tool transpile` resolver is now built from `packages.Load("./...")` instead of the hardcoded `examples/<importpath>` lookup, with CI invocation switched to `-C examples --gobuild .`; and `gno.land/pkg/integration.PkgsLoader` recursive-imports walker falls back to `examples/quarantine/<path>` when the primary path is missing.

```
examples/
├── gnowork.toml          # single workspace, covers both subtrees
├── gno.land/{p,r}/...    # 132 safe-listed (88 test-13 + 44 test-pinned)
└── quarantine/
    └── gno.land/{p,r}/...# 171 unaudited; module paths unchanged
```

## Glossary

- safe-list — the 88 packages on the test-13 genesis whitelist from #5653.
- `RootResolver(dir)` — gnodev resolver that maps `<importPath>` → `<dir>/<importPath>` via `filepath.Join` + `os.Stat`.
- `ResolveExamplePath(root, p)` — new integration helper: returns `root/p` if `<root>/<p>/gnomod.toml` exists, otherwise `root/quarantine/p`.
- `buildTranspileResolver` — new tool_transpile.go helper: walks `packages.Load("./...")` and indexes every import path to its relative on-disk location.

## Fix

Before: a single `examples/gno.land/{p,r}/...` tree held 303 packages, all loaded at genesis and treated as the "audited" baseline despite ~57% being unaudited demos / personal namespaces / tutorial realms. The transpiler's import resolver was hardcoded to `examples/<importpath>` and gnodev's default resolver chain was a single `RootResolver(examples)`.

After: 171 of those 303 packages move under `examples/quarantine/`, kept in the same workspace (so `gno test ./...` still exercises them) but separated as a labeled subtree. Three resolvers learn the new layout: `defaultBaseResolvers` ([`contribs/gnodev/command_local.go:26-34`](https://github.com/gnolang/gno/blob/648d1fcd6/contribs/gnodev/command_local.go#L26-L34) · [↗](../../../../../.worktrees/gno-review-5726/contribs/gnodev/command_local.go#L26-L34)) appends `RootResolver(examples/quarantine)` unless `--no-quarantine` is set; `buildTranspileResolver` ([`gnovm/cmd/gno/tool_transpile.go:335-367`](https://github.com/gnolang/gno/blob/648d1fcd6/gnovm/cmd/gno/tool_transpile.go#L335-L367) · [↗](../../../../../.worktrees/gno-review-5726/gnovm/cmd/gno/tool_transpile.go#L335-L367)) replaces the hardcoded `examples/<path>` with a workspace-walked index; `integration.ResolveExamplePath` ([`gno.land/pkg/integration/pkgloader.go:24-30`](https://github.com/gnolang/gno/blob/648d1fcd6/gno.land/pkg/integration/pkgloader.go#L24-L30) · [↗](../../../../../.worktrees/gno-review-5726/gno.land/pkg/integration/pkgloader.go#L24-L30)) stat-falls-back to `examples/quarantine/`. The load-bearing constraint is that `gnomod.toml`'s `module = ...` value is the only on-chain identifier, so file relocation alone preserves the chain's view of the package.

## Benchmarks / Numbers

| Surface | Before | After |
|---|---|---|
| Packages in `examples/gno.land/...` | 303 | 132 |
| Packages in `examples/quarantine/...` | — | 171 |
| gnomod.toml `module` paths changed | — | 0 |
| Net code delta (excluding renames) | — | +207 / -53 across ~10 Go files |
| CI lanes | green | green (every required check pass) |
| `TestExamplesLoad` runtime (local) | — | 18.9s |

## Critical (must fix)

None.

## Warnings (should fix)

- **[silent fallback hides genuine missing imports]** [`gno.land/pkg/integration/pkgloader.go:24-30`](https://github.com/gnolang/gno/blob/648d1fcd6/gno.land/pkg/integration/pkgloader.go#L24-L30) · [↗](../../../../../.worktrees/gno-review-5726/gno.land/pkg/integration/pkgloader.go#L24-L30) — when neither `examples/<p>` nor `examples/quarantine/<p>` exists, `ResolveExamplePath` returns the quarantine path; the downstream error then references `examples/quarantine/<p>` for an import the user never meant to put there.
  <details><summary>details</summary>

  Shape: a testscript `loadpkg gno.land/r/typo/missing` triggers `PkgsLoader.LoadPackage` → `gnomod.ParseDir("examples/quarantine/gno.land/r/typo/missing")` → `no such file or directory` referencing the quarantine path. The user wrote a path they expected to live in `examples/`, but the error blames the quarantine directory. The fallback was correct when only one subtree could host a real package; with a missing path, it picks the wrong arm.

  Fix: check `os.Stat(filepath.Join(quarantineDir, "gnomod.toml"))` in the fallback branch too and return the original `<root>/<p>` (with a clear "not found" downstream) when both miss; or return `(path, ok bool)` from `ResolveExamplePath` and let callers raise a specific "no such package in examples or quarantine" error.
  </details>

- **[examples/README.md top blurb contradicts quarantine semantics]** [`examples/README.md:7-9`](https://github.com/gnolang/gno/blob/648d1fcd6/examples/README.md#L7-L9) · [↗](../../../../../.worktrees/gno-review-5726/examples/README.md#L7-L9) — the existing paragraph still asserts "Pure packages and realms in this folder are pre-deployed to gno.land testnets", but the new quarantine section ([L17-L20](https://github.com/gnolang/gno/blob/648d1fcd6/examples/README.md#L17-L20) · [↗](../../../../../.worktrees/gno-review-5726/examples/README.md#L17-L20)) says quarantine content is "not shipped to mainnet/testnet genesis".
  <details><summary>details</summary>

  Two contradictory claims in one README. A reader landing on this doc cannot tell whether their package, if placed under `examples/quarantine/...`, will hit testnet genesis. The contradiction also bleeds into PR review of future safelist changes — does adding to quarantine ship anywhere? Fix: scope the L7-L9 sentence to `gno.land/...` only, e.g. "Packages under `examples/gno.land/...` are pre-deployed to gno.land testnets; packages under `examples/quarantine/...` are workspace fodder only — see below."
  </details>

## Nits

- [`gno.land/pkg/integration/load_test.go:26-31`](https://github.com/gnolang/gno/blob/648d1fcd6/gno.land/pkg/integration/load_test.go#L26-L31) · [↗](../../../../../.worktrees/gno-review-5726/gno.land/pkg/integration/load_test.go#L26-L31) — within a single failing tx, only the first `MsgAddPackage` is reported; multi-message txs would lose the rest.
  <details><summary>details</summary>

  In practice genesis AddPackage txs are 1-msg, so this is a theoretical concern. Drop the `return` after `t.Errorf` to cover the multi-msg case at no cost.
  </details>

- [`gnovm/cmd/gno/tool_transpile.go:341-344`](https://github.com/gnolang/gno/blob/648d1fcd6/gnovm/cmd/gno/tool_transpile.go#L341-L344) · [↗](../../../../../.worktrees/gno-review-5726/gnovm/cmd/gno/tool_transpile.go#L341-L344) — `packages.Load` error is swallowed silently; the fallback to `DefaultResolver` will then fail on every quarantine import with the unhelpful `import "..." does not exist`.
  <details><summary>details</summary>

  When `gno tool transpile` is invoked outside a workspace and Load errors, the user gets a confusing "does not exist" for quarantine packages they can plainly see on disk. Either log the error to `io.Err()` or return it. The current behavior is acceptable for the CI path (which runs `-C examples`), but a passing comment noting this is a known sharp edge would help future debuggers.
  </details>

- [`gnovm/pkg/transpiler/transpiler.go:38-46`](https://github.com/gnolang/gno/blob/648d1fcd6/gnovm/pkg/transpiler/transpiler.go#L38-L46) · [↗](../../../../../.worktrees/gno-review-5726/gnovm/pkg/transpiler/transpiler.go#L38-L46) — `PackageDirLocation` and `TranspileImportPath` retain the hardcoded `"examples/" + s` shape, so any external Go caller of `transpiler.Transpile(...)` (no resolver) will still fail to resolve quarantine imports. Internal callers all route through `TranspileWithResolver`, so this is a public-API hazard only.
  <details><summary>details</summary>

  No in-tree caller hits this, but third-party tools importing `github.com/gnolang/gno/gnovm/pkg/transpiler` and calling `Transpile` may regress. A short comment on `Transpile` flagging "for quarantine-aware behavior use `TranspileWithResolver` with a workspace-walked resolver" would prevent surprises.
  </details>

- [`contribs/gnodev/command_local.go:21`](https://github.com/gnolang/gno/blob/648d1fcd6/contribs/gnodev/command_local.go#L21) · [↗](../../../../../.worktrees/gno-review-5726/contribs/gnodev/command_local.go#L21) — `quarantineSubdir` is defined here but conceptually belongs alongside the `examples` directory name. There's a sibling concern from PR #5070's review history: gnodev still hardcodes `"examples"` in multiple places. This PR adds one more hardcoded path-segment to that family; centralizing the trio (`examples/`, `examples/quarantine/`, `gnovm/stdlibs/`) in `gnoenv` would close the loop.

## Missing Tests

- **[ResolveExamplePath has no direct test]** [`gno.land/pkg/integration/pkgloader.go:24`](https://github.com/gnolang/gno/blob/648d1fcd6/gno.land/pkg/integration/pkgloader.go#L24) · [↗](../../../../../.worktrees/gno-review-5726/gno.land/pkg/integration/pkgloader.go#L24) — the helper is exercised transitively by `TestExamplesLoad` and gnoclient `integration_test.go`, but never tested in isolation for the three branches (primary hit, fallback hit, both miss).
  <details><summary>details</summary>

  A 10-line table test in `pkgloader_test.go` (primary exists; only quarantine exists; neither exists) would lock the behavior choice in place when the warning above is addressed. Important because the helper's contract is exactly "where does this go when both checks fail?" — and that's currently undocumented in code beyond the function comment.
  </details>

## Suggestions

- [`gno.land/pkg/integration/pkgloader.go:22-23`](https://github.com/gnolang/gno/blob/648d1fcd6/gno.land/pkg/integration/pkgloader.go#L22-L23) · [↗](../../../../../.worktrees/gno-review-5726/gno.land/pkg/integration/pkgloader.go#L22-L23) — the `XXX: hardcoded fallback. packages.Load is gnowork-aware but reads the workspace root from os.Getwd()` comment is the most useful piece of context in the PR. Consider turning it into a `TODO(#xxxx)` linking a follow-up issue so the cleanup path is tracked.
- [`contribs/gnodev/command_staging.go:30`](https://github.com/gnolang/gno/blob/648d1fcd6/contribs/gnodev/command_staging.go#L30) · [↗](../../../../../.worktrees/gno-review-5726/contribs/gnodev/command_staging.go#L30) — `withoutQuarantinedExamples: true` as a staging default is sensible but worth a comment: "staging is pre-deployment; only audited packages should be in scope." Without the rationale, a future maintainer flipping the default to match `local` for symmetry would silently widen the staging surface.

## Questions for Author

- The `Generated by Claude` trailer on every commit is fine for transparency; do you intend to keep it on the squashed merge commit or strip it during merge? Past gnolang/gno commits don't carry AI-tool credits, so consistency check.
- The 44 "hardcoded import paths in gnovm/ tests or gno.land/pkg/integration/testdata/" set is derived but not pinned anywhere now that `misc/quarantine/` tooling was removed. Is the safelist guard (test that fails if `examples/` imports `examples/quarantine/`) covered by `TestExamplesLoad` alone, or is there a separate static-analysis step? `TestExamplesLoad` would surface the symptom (missing import at genesis) but not the cause; a grep-based CI gate would localize the breakage.
- Is there a follow-up PR to move `LoadConfig` to accept an explicit workspace-root (per the `XXX` in `ResolveExamplePath`), so the three new path-resolution shims collapse to one?
