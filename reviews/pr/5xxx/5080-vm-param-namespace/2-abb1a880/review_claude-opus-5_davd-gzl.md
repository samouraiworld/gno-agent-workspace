# gnolang/gno [#5080](https://github.com/gnolang/gno/pull/5080): feat(vm): control namespace enforcement via sysnames_pkgpath VM param

URL: https://github.com/gnolang/gno/pull/5080
Author: davd-gzl | Base: master | Files: 12 | +152 -121
Reviewed by: davd-gzl | Model: claude-opus-5 (xhigh) | Commit: `abb1a880` (draft, CHANGES_REQUESTED, CONFLICTING)
Open the code: [github.dev](https://github.dev/davd-gzl/gno/tree/abb1a8802) · [vscode.dev](https://vscode.dev/github/davd-gzl/gno/tree/abb1a8802) · `./scripts/review-worktrees.sh gno 5080`
Local checkout: `git -C gno worktree add --detach .worktrees/gno-review-5080 abb1a8802`

Round 2 over the same head as round 1. The head has not moved since 2026-03-24; master has. This round measures the branch against master rather than re-reading the diff, which [round 1](../1-abb1a880/claude-opus-4-7_davd-gzl.md) covers.

## Overview

The branch moves the "is namespace enforcement on" decision out of the `r/sys/names` realm, where a one-shot `Enable()` transaction flipped a package-local flag, and into the VM parameter `sysnames_pkgpath`: empty means no enforcement, a realm path means the keeper calls `IsAuthorizedAddressForNamespace` on every non-genesis `MsgAddPackage`. Master has since arrived at the same mechanism through other pull requests, while keeping the realm-side toggle and building governance on top of it. So the branch and master now agree on the mechanism and disagree on the default and on whether the realm keeps a switch of its own.

**Verdict: NEEDS DISCUSSION** — the mechanism landed on master independently, and what is left is one constant, one genesis skip and a deletion that now collides with a governance surface added since (1 discussion item, 3 measured deltas).

## Verify first

- [`gno.land/pkg/sdk/vm/params.go:538-540`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L538-L540) — `applyLegacyDefaults` rewrites an empty `SysNamesPkgPath` back to the default, so confirm by setting `sysnames_pkgpath = ""` in a genesis and reading `vm.GetParams` that no chain can currently disable enforcement through params.
- [`examples/gno.land/r/sys/names/verifier.gno:113-208`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/sys/names/verifier.gno#L113-L208) — read `Enable`, `IsEnabled`, `ProposeSetPaused` and `setPaused` together before deciding what the branch's deletion is allowed to take.

## Summary

Master carries `sysnames_pkgpath` as a VM param, gates `checkNamespacePermission` on it being non-empty, and pins it for gno.land in `genesis_params.toml`, which answers the review asks from @zivkovicmilos, @thehowl and @mvallenet. It differs from the branch on the default, `gno.land/r/sys/names` rather than `""`, on genesis, where it changes only the type-check mode and never skips the namespace check, and on the realm, which kept `Enable`/`IsEnabled` and grew a pause path and a GovDAO T1 admin around them. The branch's surviving content is therefore the default flip, the `ctx.BlockHeight() == 0` skip, and a `Params.String()` label fix.

## State

| Fact | Value |
| --- | --- |
| Commits on branch / on master since the merge base | 16 / 371 |
| Merge base | `fe3d82d23`, 2026-03-24 |
| Files conflicting on `git merge upstream/master` | 8 of 12 |
| Checks at the head | all green, last run 2026-03-24 |
| Review decision | CHANGES_REQUESTED (@moul, @zivkovicmilos, @mvallenet) |
| Unresolved inline threads | 3, all marked outdated |

Conflicting files: `examples/gno.land/r/sys/names/verifier.gno`, `verifier_test.gno`, `gno.land/pkg/integration/testdata/addpkg_namespace.txtar`, `user_journey.txtar`, `gno.land/pkg/integration/testscript_gnoland.go`, `gno.land/pkg/sdk/vm/keeper.go`, `keeper_test.go`, `params.go`.

## Superseded by master

- **The VM param and the empty-means-disabled gate.** [`keeper.go:480-484`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L480-L484) reads `params.SysNamesPkgPath` and returns nil when it is empty, which is the shape @thehowl proposed and @mvallenet seconded.
- **The genesis params entry.** [`genesis_params.toml:9`](https://github.com/gnolang/gno/blob/master/gno.land/genesis/genesis_params.toml#L9) sets `sysnames_pkgpath = "gno.land/r/sys/names"`, which is @mvallenet's ask.
- **The `sysUsersPkgParamPath` rename.** Master has no `getSysNamesPkgParam` and no per-key param path constant at all: `GetParams` reads the struct through `vm:p` and normalises in `applyLegacyDefaults`.
- **The `checkNamespacePermission` signature.** Master passes `params Params` in and [`callRealmBool`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L421) takes `chainDomain`, so the branch's call site does not compile against master and its conflict resolution is a rewrite rather than a merge.

## Still only on this branch

- **The default.** [`params.go:37`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L37) is `gno.land/r/sys/names` on master. With `applyLegacyDefaults` rewriting an explicit `""` back to it, enforcement cannot be turned off through genesis params on master at all, which is exactly @moul's objection: gnodev and other local environments should let anyone publish. Flipping the constant is a one-line change and costs gno.land nothing, since `genesis_params.toml` names the path explicitly.
- **The genesis skip.** `ctx.BlockHeight() == 0` short-circuits the check. Nothing on master waives the namespace check for a genesis transaction; `AddPackage`'s own height-0 branch only relaxes the type-check mode to `gno.TCGenesisStrict`.
- **The `String()` label.** Master still prints `SysUsersPkgPath: %q` for `p.SysNamesPkgPath` at `params.go:238`.
- **The gate ordering.** The branch validates the chain domain and the namespace regex before reading the param, so with enforcement disabled an off-domain path returns `ErrInvalidPkgPath` where master returns nil. This is a behaviour change the branch never states and no test pins.

## Open questions

- Whether the deletion of `Enable`/`IsEnabled` is still on the table now that `paused` and the GovDAO T1 admin sit beside them: the branch treats them as a redundant toggle, master treats them as one arm of a governance surface. Not posted as a finding because it is a maintainer call, not a defect.
- Whether the remaining content justifies this branch or a fresh one off master. Retargeting to the default flip and the genesis skip would be under twenty lines and would carry no conflict.
