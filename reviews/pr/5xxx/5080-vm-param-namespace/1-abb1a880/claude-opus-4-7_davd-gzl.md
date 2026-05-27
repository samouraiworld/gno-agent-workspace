# PR #5080: feat(vm): control namespace enforcement via sysnames_pkgpath VM param

URL: https://github.com/gnolang/gno/pull/5080
Author: davd-gzl | Base: master | Files: 12 | +152 -121
Reviewed by: davd-gzl (self-review by PR author) | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5080 abb1a880` (then `gh -R gnolang/gno pr checkout 5080` inside it)

**Verdict: REQUEST CHANGES** — gnoland1 betanet genesis script ([`misc/deployments/gnoland1/govdao_prop1.gno:104`](https://github.com/gnolang/gno/blob/abb1a880/misc/deployments/gnoland1/govdao_prop1.gno#L104) · [↗](../../../../../.worktrees/gno-review-5080/misc/deployments/gnoland1/govdao_prop1.gno#L104)) still calls the deleted `names.Enable(cross)` and will fail typecheck at genesis; fix that, then revisit the unconditional cache clear in `checkNamespacePermission` and the integration test default-param overwrite.

## Summary

Replaces the realm-level `Enable()`/`IsEnabled()` toggle in `r/sys/names` with a VM-level parameter `sysnames_pkgpath`. When empty (the default), namespace enforcement is disabled; when set to a realm path (e.g. `gno.land/r/sys/names`), the VM keeper calls `IsAuthorizedAddressForNamespace` on every `MsgAddPackage` outside genesis. The change is the right shape — namespace enforcement is a protocol decision and belongs in genesis params, not in a one-shot realm tx. Deletes 5 lines of dead workaround code in `process.go` (the `patchpkg` admin-address override).

## Glossary

- **r/sys/names** — system realm implementing namespace authorization (`IsAuthorizedAddressForNamespace`).
- **PA namespace** — "Personal Address" namespace; allows deploying under `gno.land/{r,p}/<your-own-address>/**`.
- **sysnames_pkgpath** — VM param naming the realm whose `IsAuthorizedAddressForNamespace(address, string) bool` gets called on every non-genesis `MsgAddPackage`.
- **MsgRun** — message type that executes a Gno script in a temporary realm; used at genesis for govdao bootstrap.

## Fix

Before: `r/sys/names.Enable()` flipped a package-local `enabled` boolean and dropped Ownable; `checkNamespacePermission` always called `IsAuthorizedAddressForNamespace` and the realm itself decided whether to enforce. After: the realm is stateless (no `enabled`, no `Ownable`), and the keeper gate is `sysnames_pkgpath != "" && store.GetPackage(sysnames_pkgpath) != nil && ctx.BlockHeight() != 0`. The flag is wired through `genesis_params.toml` ([`gno.land/genesis/genesis_params.toml:9`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/genesis/genesis_params.toml#L9) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/genesis/genesis_params.toml#L9)) and a new testscript flag `-sysnames-pkgpath` ([`gno.land/pkg/integration/testscript_gnoland.go:288`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/integration/testscript_gnoland.go#L288) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/integration/testscript_gnoland.go#L288)).

## Critical (must fix)

- **[breaks gnoland1 betanet genesis]** [`misc/deployments/gnoland1/govdao_prop1.gno:104`](https://github.com/gnolang/gno/blob/abb1a880/misc/deployments/gnoland1/govdao_prop1.gno#L104) · [↗](../../../../../.worktrees/gno-review-5080/misc/deployments/gnoland1/govdao_prop1.gno#L104) — calls deleted `names.Enable(cross)`; `gen-genesis.sh` will fail.
  <details><summary>details</summary>

  The gnoland1 production deploy script signs `govdao_prop1.gno` as a MsgRun tx and embeds it in `genesis_txs.jsonl` ([`misc/deployments/gnoland1/gen-genesis.sh:190-216`](https://github.com/gnolang/gno/blob/abb1a880/misc/deployments/gnoland1/gen-genesis.sh#L190-L216) · [↗](../../../../../.worktrees/gno-review-5080/misc/deployments/gnoland1/gen-genesis.sh#L190-L216)). This PR deletes `Enable` and `IsEnabled` from `examples/gno.land/r/sys/names/verifier.gno`. Re-running `gen-genesis.sh` after this merges will fail at the `gnokey maketx run` step with a typecheck error on `names.Enable(cross)`. Even if signed transactions in the existing `genesis_txs.jsonl` are reused, replaying genesis on a fresh node will fail the same way during InitChainer.

  Fix: in the same PR, replace lines 101-104 of `govdao_prop1.gno` with a comment explaining that namespace enforcement is now driven by the `sysnames_pkgpath` VM param (set in `genesis.json` via `genesis_params.toml`), and drop the `gno.land/r/sys/names` import on line 26. Verify by running `make -C misc/deployments/gnoland1` after the change.
  </details>

## Warnings (should fix)

- **[unnecessary cache clear when enforcement disabled]** [`gno.land/pkg/sdk/vm/keeper.go:422`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L422) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L422) — `getGnoTransactionStore(ctx)` is called before the `sysNamesPkgPath == ""` early return, clearing the object cache on every `AddPackage` even when the param is unset.
  <details><summary>details</summary>

  `getGnoTransactionStore` calls `ClearObjectCache` as a side effect ([`keeper.go:355-359`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L355-L359) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L355-L359)). The author's own PR comment on the earlier draft explained this ordering: "package couldn't be reuploaded anymore in some situation when `gnomod.toml` private == true". Acknowledged — but the resulting code now clears the cache on every `AddPackage` even when `sysnames_pkgpath` is empty (the default), making the default path slower and entangling cache invalidation with a code path that doesn't otherwise need the store. `AddPackage` already calls `getGnoTransactionStore(ctx)` at line 518 to obtain `gnostore`; passing `gnostore` into `checkNamespacePermission` would let the param-empty path skip the second clear entirely.

  Fix: take `store gno.TransactionStore` as a parameter to `checkNamespacePermission`, drop the second `getGnoTransactionStore` call, and reorder so the early returns for `sysNamesPkgPath == ""` and `sysNamesPkg == nil` happen before any work that depends on the store.
  </details>

- **[silent param leak in integration tests]** [`gno.land/pkg/integration/testscript_gnoland.go:308`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/integration/testscript_gnoland.go#L308) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/integration/testscript_gnoland.go#L308) — `genesis.VM.Params = tsGenesis.VM.Params` is now unconditional, so every txtar inherits `sysnames_pkgpath = "gno.land/r/sys/names"` from `genesis_params.toml` whether or not it intends to enable enforcement.
  <details><summary>details</summary>

  Previously the integration framework never copied `tsGenesis.VM.Params` into the runtime genesis, so the in-test default was `SysNamesPkgPath=""`. With this PR the default `genesis_params.toml` is now `sysnames_pkgpath = "gno.land/r/sys/names"`, which the unconditional copy propagates to every test. The only thing currently saving `addpkg_namespace_disabled.txtar` is that the test doesn't `loadpkg gno.land/r/sys/names`, so the `store.GetPackage(...) == nil` fallback fires — a fragile invariant. Any future txtar that incidentally loads `r/sys/names` and then runs `gnoland start` (without `-sysnames-pkgpath`) will silently flip into enforcement mode and confuse the reader.

  Fix: gate the param copy behind `-sysnames-pkgpath` (only set `genesis.VM.Params.SysNamesPkgPath` when the flag is non-empty, leave the rest of `genesis.VM.Params` at the keeper default), or set `sysnames_pkgpath = ""` in `genesis_params.toml` and rely on the flag in tests that need it. The latter is cleaner: the testscript framework already has a working knob.
  </details>

- **[lost defensive bech32 check]** [`examples/gno.land/r/sys/names/verifier.gno:14`](https://github.com/gnolang/gno/blob/abb1a880/examples/gno.land/r/sys/names/verifier.gno#L14) · [↗](../../../../../.worktrees/gno-review-5080/examples/gno.land/r/sys/names/verifier.gno#L14) — replaced `!address_XXX.IsValid()` with `address_XXX.String() == ""`, so malformed bech32 addresses no longer reject.
  <details><summary>details</summary>

  `address.IsValid()` ([`gnovm/pkg/gnolang/uverse.go:997-1012`](https://github.com/gnolang/gno/blob/abb1a880/gnovm/pkg/gnolang/uverse.go#L997-L1012) · [↗](../../../../../.worktrees/gno-review-5080/gnovm/pkg/gnolang/uverse.go#L997-L1012)) decodes bech32 and checks `len(addr) == 20`. The new emptiness check accepts any non-empty string, including malformed bech32. From the keeper this doesn't matter because `creator crypto.Address` is already validated upstream, but `IsAuthorizedAddressForNamespace` is a public realm function callable via `vm/qeval` with arbitrary input. Today the regression is benign — the function only checks string equality against `namespace`, so a malformed address can only match itself — but the defensive layer is gone.

  Fix: keep `!address_XXX.IsValid()` as the first guard; the `String() == ""` case is subsumed because empty bech32 is invalid.
  </details>

- **[AI-assisted PR with no ADR]** PR description hints AI involvement (commit format, multi-rewrite history) and `AGENTS.md` requires "every non-trivial AI-assisted PR must include an ADR" — none was added under `gno.land/adr/`.
  <details><summary>details</summary>

  The protocol-level decision "namespace enforcement is a VM param, not a realm toggle, and genesis skips enforcement entirely" is exactly the kind of decision an ADR is meant to capture. Future contributors looking at why `r/sys/names` shrank to 8 lines and why `BlockHeight()==0` short-circuits enforcement will have to reconstruct the rationale from PR commentary. Add a brief `gno.land/adr/pr5080_sysnames_pkgpath.md` covering: (1) why this moved out of `r/sys/names`, (2) why genesis is unconditionally skipped, (3) the bootstrap order (sysnames realm must be deployed before the first non-genesis MsgAddPackage), (4) alternatives considered (a genesis flag, a CLI flag, an Enable tx).
  </details>

## Nits

- [`gno.land/pkg/sdk/vm/keeper.go:418`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L418) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L418) — typo "permssion" in doc comment.
- [`gno.land/pkg/sdk/vm/keeper.go:424-435`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L424-L435) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L424-L435) — the `HasPrefix(chainDomain+"/")` check and `reNamespace` match are now redundant with the same checks in `AddPackage` ([`keeper.go:535`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L535) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L535)) and `IsRealmPath`/`IsPPackagePath` validation. Not harmful, but the function could just take `namespace string` directly.
- [`gno.land/pkg/sdk/vm/keeper.go:601`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L601) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L601) — comment "Check namespace permission via r/sys/names." is now slightly misleading since the realm path is configurable; consider "Check namespace permission via the configured sysnames realm."

## Missing Tests

- **[no test for the namespace-realm-not-yet-deployed bootstrap path]** [`gno.land/pkg/sdk/vm/keeper.go:443-446`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L443-L446) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L443-L446) — `checkCLASignature` has a dedicated `TestVMKeeperCLASignature_RealmNotDeployed` test ([`keeper_test.go:1631`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper_test.go#L1631) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper_test.go#L1631)); the equivalent bootstrap path for `sysnames` (param set, package not yet deployed) is not covered.
  <details><summary>details</summary>

  The path is exercised implicitly by `addpkg_namespace_disabled.txtar` but only because that test doesn't `loadpkg` `r/sys/names`. A dedicated Go-level unit test would lock in the contract that the sysnames realm itself can be deployed even when the param points to it — analogous to the CLA bootstrap test the PR author clearly drew from.
  </details>

- **[no test for the BlockHeight==0 skip on a real genesis tx flow]** [`gno.land/pkg/sdk/vm/keeper.go:450-452`](https://github.com/gnolang/gno/blob/abb1a880/gno.land/pkg/sdk/vm/keeper.go#L450-L452) · [↗](../../../../../.worktrees/gno-review-5080/gno.land/pkg/sdk/vm/keeper.go#L450-L452) — the new unit test exercises `checkNamespacePermission` directly but not an end-to-end genesis-tx `MsgAddPackage` with an unauthorized creator.
  <details><summary>details</summary>

  `TestCheckNamespacePermission_GenesisSkipsEnforcement` calls the unexported function directly. The real risk is that some future caller of `checkNamespacePermission` introduces a `BlockHeight()` change that breaks the skip. A txtar test that deploys `r/sys/names` via `loadpkg`, sets `sysnames_pkgpath` via `genesis_params.toml`, embeds a `MsgAddPackage` for `gno.land/r/random/foo` signed by an unauthorized user in `genesis_txs.jsonl`, and asserts the genesis succeeds, would catch end-to-end regressions.
  </details>

## Suggestions

- `examples/gno.land/r/sys/names/render.gno:5-9` — render output mentions the param by name; consider also surfacing the current effective state by reading it via `std.GetParam("vm:p:sysnames_pkgpath", ...)` (if exposed) so `gno query vm/qrender --data "gno.land/r/sys/names:"` shows operators whether enforcement is on. Improves the "verify `IsEnable == true` for production chains" check the PR thread already flagged as a docs gap.

## Questions for Author

- Why skip namespace enforcement at `BlockHeight() == 0` for `MsgAddPackage` but not for the rest of `AddPackage` (typecheck modes, draft gating)? Genesis already runs every tx through the ante handler that auto-creates signer accounts — adding a "trust genesis" bypass to namespace enforcement, but no equivalent bypass elsewhere, is asymmetric. Worth a sentence in the ADR.
- Should `Params.Validate` reject `sysnames_pkgpath` pointing to a non-existent realm at param-set time, or is the "skip if not deployed" behavior intentional for GovDAO param updates? Current behavior is permissive — a bad GovDAO proposal silently disables enforcement instead of failing validation.
- The PR is currently ~12 commits behind master and includes a merge. Was the intent to keep the merge in history, or rebase before merging? The merge commit (`abb1a8802`) brings in ~2300 unrelated file changes that obscure `git log --stat` for this PR.
