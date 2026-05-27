# PR #5070: chore: remove hardcoded examples directory name

URL: https://github.com/gnolang/gno/pull/5070
Author: moonia | Base: master | Files: 2 | +5 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `5c8ebba0b` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5070 5c8ebba0b`

**Verdict: REQUEST CHANGES** — preserves a pre-existing `"example"` vs `examples/` typo in [`pkg/dev/node.go`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node.go#L85) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node.go#L85), splits one logical constant into two divergent ones (`examplesDirName="examples"` in `app.go`, `exampleDirName="example"` in `node.go`), and misses two other hardcoded `"examples"` literals in the same package, so the refactor neither fixes the underlying TODO nor makes future centralization easier.

## Summary

The two `XXX`/TODO comments in [`app.go:154`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/app.go#L154-L155) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/app.go#L154-L155) and [`node.go:99`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node.go#L99) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node.go#L99) asked maintainers to stop hardcoding the path to the examples directory. The author addresses the symptom (two string literals) by hoisting each into a local `const` in its own file, but the original intent (per [@gfanton](https://github.com/gnolang/gno/pull/5070#discussion_r1979040580)) was to centralize the path — likely a single helper in `gnovm/pkg/gnoenv`. The result is two consts with different spellings, neither covering the other call sites in `command_local.go` and `command_staging.go`. The fix is to expose a single helper (e.g. `gnoenv.ExamplesDir()`) and route every gnodev call through it, then delete both local consts.

## Fix

Before: two string literals (`"examples"` in [`app.go:155`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/app.go#L155) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/app.go#L155), `"example"` in [`node.go:99`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node.go#L99) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node.go#L99)) each annotated with a `// XXX` requesting de-hardcoding. After: each literal is replaced by a file-local `const` of the same value, [`examplesDirName`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/app.go#L43) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/app.go#L43) (plural) in `app.go` and [`exampleDirName`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node.go#L85) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node.go#L85) (singular) in `node.go`. The `XXX` comments are removed in both files. No call sites of either constant are unified; `command_local.go:112` and `command_staging.go:79` continue to use the literal `"examples"` directly.

## Critical (must fix)

- **[wrong directory name perpetuated]** [@davd-gzl](https://github.com/gnolang/gno/pull/5070#discussion_r1979037537) [`contribs/gnodev/pkg/dev/node.go:85`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node.go#L85) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node.go#L85) — `exampleDirName = "example"` is singular; the actual on-disk directory is `examples/`. Hoisting the typo into a named constant locks the bug in place instead of fixing it.
  <details><summary>details</summary>

  The default loader built in `DefaultNodeConfig` joins `gnoenv.RootDir()` with `"example"` and hands the result to `packages.NewRootResolver`. [`resolver_root.go:28-32`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/packages/resolver_root.go#L28-L32) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/packages/resolver_root.go#L28-L32) calls `os.Stat` on `<root>/<path>` and returns `ErrResolverPackageNotFound` when missing — so every resolution against this default loader silently fails. The bug is masked in production because `setup_node.go:106` immediately overwrites `config.Loader` ([`setup_node.go:106`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/setup_node.go#L106) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/setup_node.go#L106)), and `TestNewNode_NoPackages` ([`node_test.go:36-44`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node_test.go#L36-L44) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node_test.go#L36-L44)) asserts zero packages — which is exactly what a broken loader returns. The PR title is "remove hardcoded examples directory name" but this commit makes the wrong name harder to spot, not easier. Fix: change the literal to `"examples"` (or, better, replace with a centralized helper — see next finding).
  </details>

- **[two sources of truth, divergent values]** [@gfanton](https://github.com/gnolang/gno/pull/5070#discussion_r1979038920) [`contribs/gnodev/app.go:43`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/app.go#L43) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/app.go#L43), [`contribs/gnodev/pkg/dev/node.go:85`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node.go#L85) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node.go#L85) — same logical concept, two consts in two packages, two different spellings.
  <details><summary>details</summary>

  `app.go` defines `examplesDirName = "examples"` lumped into the `LogName` constant block; `node.go` defines `exampleDirName = "example"` as a standalone const. Both refer to the same directory inside `GNOROOT`. The original TODO asked to stop hardcoding the path, not to duplicate it. The right shape — confirmed by both review comments on the PR — is one helper in `gnovm/pkg/gnoenv` (e.g. `gnoenv.ExamplesDir()`), routing every caller through it. That also picks up the two literals the PR missed (next finding). Fix: delete both consts, add `gnoenv.ExamplesDir()` returning `filepath.Join(RootDir(), "examples")`, and replace all four call sites with one call.
  </details>

## Warnings (should fix)

- **[incomplete refactor]** [`contribs/gnodev/command_local.go:112`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/command_local.go#L112) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/command_local.go#L112), [`contribs/gnodev/command_staging.go:79`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/command_staging.go#L79) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/command_staging.go#L79) — two more hardcoded `"examples"` literals in the same `contribs/gnodev` package, untouched by this PR.
  <details><summary>details</summary>

  `command_local.go:112` builds `filepath.Join(gnoroot, "examples")` and feeds it to `NewRootResolver`; `command_staging.go:79` does the same. If "remove hardcoded examples directory name" is the goal of the PR, these two call sites are in scope and missing. They also justify the "centralize in `gnoenv`" approach over a file-local const, since a const in `app.go` is not visible from `command_local.go`/`command_staging.go` without an import that's ugly. Fix: route all four sites through `gnoenv.ExamplesDir()` (or equivalent) in a single follow-up.
  </details>

- **[stale doc strings]** [`contribs/gnodev/command_local.go:63`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/command_local.go#L63) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/command_local.go#L63), [`contribs/gnodev/command_staging.go:54`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/command_staging.go#L54) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/command_staging.go#L54), [`contribs/gnodev/README.md:104`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/README.md#L104) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/README.md#L104), [`contribs/gnodev/README.md:148`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/README.md#L148) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/README.md#L148) — user-facing help text still says `"example"` folder (singular).
  <details><summary>details</summary>

  Help output and README both refer to "the `example` folder from gnoroot". Same root cause as the const typo in `node.go`. While not introduced by this PR, anyone touching the topic should fix the docs in the same pass — they will mislead users who copy-paste the path. Fix: replace `"example"` with `"examples"` in the four doc strings.
  </details>

- **[const placement obscures intent]** [@gfanton](https://github.com/gnolang/gno/pull/5070#discussion_r1979038920) [`contribs/gnodev/app.go:43`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/app.go#L43) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/app.go#L43) — `examplesDirName` is added to a `const (...)` block whose other members are all `*LogName` strings.
  <details><summary>details</summary>

  The block at [`app.go:35-44`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/app.go#L35-L44) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/app.go#L35-L44) groups `NodeLogName`, `WebLogName`, ..., `ProxyLogName`, then `examplesDirName`. The naming convention and the surrounding members signal "log group names"; the new const is a path component. A reader scanning the block will misread the intent. Fix: move `examplesDirName` to a separate const block (or delete it entirely if centralized in `gnoenv`).
  </details>

## Nits

- [`contribs/gnodev/pkg/dev/node.go:85`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node.go#L85) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node.go#L85) — naming inconsistency: `examplesDirName` (plural) in `app.go` vs `exampleDirName` (singular) in `node.go`. Even after fixing the value, pick one spelling.

## Missing Tests

- **[no coverage of default loader]** [`contribs/gnodev/pkg/dev/node_test.go:36`](https://github.com/gnolang/gno/blob/5c8ebba0b/contribs/gnodev/pkg/dev/node_test.go#L36) · [↗](../../../../../.worktrees/gno-review-5070/contribs/gnodev/pkg/dev/node_test.go#L36) — `TestNewNode_NoPackages` is the only test that uses the loader built inside `DefaultNodeConfig`, and it only asserts `Len(ListPkgs()) == 0`. That assertion passes equally for "loader works but no packages requested" and "loader points at a non-existent directory". A test that asks the default loader to resolve a known package (e.g. `gno.land/p/demo/avl`) would have caught the `"example"` vs `"examples"` typo years ago.
  <details><summary>details</summary>

  Concrete shape: in `pkg/dev/node_test.go`, add a test calling `cfg.Loader.Load("gno.land/p/demo/avl")` against the default config and asserting success. The bug shows up immediately as `ErrResolverPackageNotFound`. Not strictly the PR author's responsibility, but worth raising because the refactor would have been caught by the test.
  </details>

## Suggestions

- [`gnovm/pkg/gnoenv/gnoroot.go`](https://github.com/gnolang/gno/blob/5c8ebba0b/gnovm/pkg/gnoenv/gnoroot.go) · [↗](../../../../../.worktrees/gno-review-5070/gnovm/pkg/gnoenv/gnoroot.go) — add a sibling helper `ExamplesDir()` returning `filepath.Join(RootDir(), "examples")`. Single source of truth for both gnodev and any future caller (`gno test`, `gnokey`, tooling). The original `XXX` comment in `node.go` explicitly hints at this: "we should avoid having to hardcoding this here" — implying it should be sourced from somewhere central.

## Questions for Author

- Is the PR scope intentionally limited to two files, or would you fold in `command_local.go` / `command_staging.go` (and the README/help-text typos) in the same change? The current shape leaves the refactor half-done.
- Have you confirmed the `"example"` (singular) in `node.go` is reachable in production? If `setup_node.go:106` always overwrites `config.Loader`, the default loader's path may be entirely dead — worth either deleting the default loader construction or fixing the typo and writing a test that exercises it.
