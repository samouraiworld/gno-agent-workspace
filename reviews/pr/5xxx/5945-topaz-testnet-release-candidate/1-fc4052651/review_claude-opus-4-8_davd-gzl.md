# PR [#5945](https://github.com/gnolang/gno/pull/5945): feat: topaz testnet release candidate

URL: https://github.com/gnolang/gno/pull/5945
Author: aeddi | Base: master | Files: 8 | +1278 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: fc4052651 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5945 fc4052651`

**TL;DR:** Adds a self-contained folder that builds the `genesis.json` for topaz, a brand-new gno test chain. One shell script compiles the tools, packs 90 example packages plus four setup transactions into a genesis file, works out how much each signer must be given so it ends up with nothing left, and checks the result against a recorded fingerprint.

**Verdict: REQUEST CHANGES** — the genesis this produces, and the one already published on the release page, hands nine accounts a billion GNOT each instead of leaving them empty; the balance-measurement step that was meant to prevent that reads every balance as zero and its own verification pass cannot detect it (1 Critical, 2 Warnings, 3 Nits, 4 Suggestions).

## Summary

`gen-genesis.sh` builds the whole topaz genesis in one pass: resolve the curated `examples/` set, `addpkg` all 90 packages, append a GovDAO bootstrap `MsgRun`, a `names.Enable` `MsgCall` and two `valopers.Register` calls, then credit balances and verify. The design goal for balances is exact-burn funding: each genesis-transaction fee payer is credited precisely what its fees cost, so it lands at zero and, per [`README.md:21`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/README.md?plain=1#L21) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/README.md#L21), the final state holds only the ten faucet balances.

That step does not work. The measurement node is declared ready as soon as an `auth/accounts` query answers, roughly four seconds before `bank/balances` returns anything, and [`query_balance`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L895) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L895) turns that empty answer into `0`. Every fee payer is therefore recorded as having spent its full over-provisioned float, and all nine are credited `INITIAL_BALANCE` = 1e15 ugnot in the real genesis. The second pass, which exists to catch exactly this, re-runs the same broken measurement and reads zero again, which is what it expects, so it prints `All balances zero — fee-payer costs verified` and the build succeeds.

The scale is 8,999,999,775,461,906 ugnot (≈9 billion GNOT) sitting on nine accounts after genesis, seven of which are third-party addresses pulled in by `[addpkg] creator` overrides in `gnomod.toml` (moul's personal address, the `@gnoland`/`@sys`/`@gov`/`@nt` namespace addresses, leon, samcrew) plus the gnoland1 GovDAO T1 multisig. The topaz operators hold none of those keys.

Everything else checked out. The build is reproducible to the locked digest, the published release asset is byte-identical to a local rebuild, genesis replay is 94/94, namespace enforcement is on, `AllowedDAOs` is locked, and both founding valoper profiles resolve.

## Glossary

- addpkg: the transaction (`maketx addpkg`) that uploads a package or realm to the chain.
- genesis tx: a transaction embedded in `GenesisDoc.AppState.Txs`, executed by `InitChainer` at height 0 before any block; fees are charged normally, so every signer needs a genesis balance.
- valoper: a validator's operator-keyed profile in `gno.land/r/gnops/valopers`, the management plane (signing-key rotation, profile edits, opt-out) that `r/sys/validators/v3` reads.
- namespace verifier: `gno.land/r/sys/names`, which gates `addpkg` on namespace ownership once `Enable` has run.

## Benchmarks / Numbers

| Quantity | ugnot |
| --- | --- |
| Total fee burn across the 9 genesis fee payers (the intended credit) | 224,538,094 |
| Credited by the shipped genesis (9 × `INITIAL_BALANCE`) | 9,000,000,000,000,000 |
| Left on those 9 accounts after genesis executes | 8,999,999,775,461,906 |
| Total faucet allocation (10 × `FAUCET_BALANCE`) | 10,000,000,000,000,000,000 |
| `int64` maximum | 9,223,372,036,854,775,807 |

## Critical (must fix)

- **[launch genesis hands out unowned funds]** `misc/deployments/topaz.gno.land/gen-genesis.sh:861` — the genesis credits all nine fee payers `INITIAL_BALANCE` instead of their fee burn, leaving ≈9 billion GNOT on addresses topaz does not control.
  <details><summary>details</summary>

  [`start_temp_node`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L861) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L861) treats the node as ready once `auth/accounts/<deployer>` answers. On a rebuild here that happened at 10s, while `bank/balances/<addr>` still returned `data: ""` and only returned real figures at 14s. Every balance in the measure pass is therefore read as zero, so [`final=$((INITIAL_BALANCE - remaining))`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L910) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L910) records the full float as spent.

  The result reaches the real genesis through [the copy into the final balance sheet](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L934) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L934): on a rebuild here that sheet listed all nine addresses at exactly 1000000000000000ugnot, and the produced `genesis.json` carries those nine entries alongside the ten faucets. Booting a node on it, the deployer holds 999999876075544 ugnot rather than 0; the nine together hold 8,999,999,775,461,906 ugnot against an actual burn of 224,538,094. Seven of the nine come from `[addpkg] creator` overrides in `gnomod.toml` and belong to third parties: `g1manfred47kzduec920z88wfr64ylksmdcedlf5`, `g1g3lsfxhvaqgdv4ccemwpnms4fv6t3aq3p5z6u7`, `g1g73v2anukg4ej7axwqpthsatzrxjsh0wk797da`, `g1r929wt2qplfawe4lvqv9zuwfdcz4vxdun7qh8l`, `g15ge0ae9077eh40erwrn2eq0xw6wupwqthpv34l`, `g125em6arxsnj49vx35f0n0z34putv5ty3376fg5`, `g1kfd9f5zlvcvy6aammcmqswa7cyjpu2nyt9qfen`; the eighth is the gnoland1 GovDAO T1 multisig used as the names admin.

  A local rebuild reproduces the published digest `2dd049f97…` exactly, and the `genesis.json` asset on the `chain/topaz` release carries the same sha256, so the released artifact has this state too. [repro](comment_claude-opus-4-8.md)

  Fix: gate readiness on the query the measurement actually uses, re-run the build, and re-lock `CHECKSUMS_DATA` and the digest in `VALIDATOR.md`.
  </details>

## Warnings (should fix)

- **[verification pass cannot fail]** `misc/deployments/topaz.gno.land/gen-genesis.sh:895` — an unreadable balance is returned as `0`, which is the value the verify pass asserts, so the two-run cross-check passes on a measurement that never happened.
  <details><summary>details</summary>

  [`echo "${r:-0}"`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L895) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L895) collapses "the node returned no balance" and "the account is empty" into the same result. Run 1 then records the full float as spent and run 2 re-reads zero, which is exactly what [the all-zero assertion](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L921) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L921) wants to see.

  On the run here that produced the published digest, `work/deployers_balances.txt` held nine entries at `INITIAL_BALANCE` while the log printed `All balances zero — fee-payer costs verified`. Because both passes share the measurement, no amount of repetition catches the failure; only distinguishing "no data" from "zero" does.

  Fix: treat a `bank/balances` response with no `ugnot` figure as an error rather than as zero.
  </details>

- **[stale premise under a safety claim]** `misc/deployments/topaz.gno.land/gen-genesis.sh:947-949` — the comment states the fee payers are the deployer and the names admin and that no address can appear in both sheets, but the set is nine addresses derived at runtime and the no-overlap property is never checked.
  <details><summary>details</summary>

  [The comment](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L947-L949) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L947-L949) is repeated at [`gen-genesis.sh:786`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L786) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L786) and in [`README.md:21`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/README.md?plain=1#L21) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/README.md#L21). The build reports `Found 9 unique creator/caller addresses`: seven arrive from `[addpkg] creator` fields in `gnomod.toml`, so the set grows silently whenever a package with a creator override enters `FILTERED_PACKAGES`.

  The sheets are concatenated and parsed into a map by [`LoadFromEntries`](https://github.com/gnolang/gno/blob/fc4052651/gno.land/pkg/gnoland/balance.go#L203) · [↗](../../../../../.worktrees/gno-review-5945/gno.land/pkg/gnoland/balance.go#L203), which assigns without checking for an existing key. A faucet address that also appeared as a creator would silently overwrite its exact-burn entry, with no diagnostic.

  Fix: derive the claim from the data rather than restating two addresses, and reject a duplicate address across the two sheets instead of asserting the absence in a comment.
  </details>

## Nits

- **[launch timestamp already elapsed]** `misc/deployments/topaz.gno.land/gen-genesis.sh:53` — [`GENESIS_TIME=1783868400`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L53) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L53) is 2026-07-12 15:00 UTC, already in the past, so block 1 carries a backdated timestamp. Changing it changes the genesis digest, so it has to move together with `CHECKSUMS_DATA` and the sha in `VALIDATOR.md`.
- **[comments describe a pipeline that no longer exists]** `misc/deployments/topaz.gno.land/.gitignore:1` — [the header](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/.gitignore#L1) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/.gitignore#L1) and [line 4](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/.gitignore#L4) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/.gitignore#L4) mention "phase-1 + phase-2 intermediates" and `work/phase-2/`, left over from the abandoned hardfork design; the script is single-phase and writes `genesis.json` straight from `work/`.
- **[usage line names a file that is not there]** `misc/deployments/topaz.gno.land/govdao-exec.sh:3` — [the usage line](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/govdao-exec.sh#L3) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/govdao-exec.sh#L3) says `./govdao`, carried over from [gnoland1's copy](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/gnoland1/govdao#L3) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/gnoland1/govdao#L3), but the topaz file is `govdao-exec.sh`.

## Suggestions

- **[combined faucet supply is unspendable in one account]** `misc/deployments/topaz.gno.land/gen-genesis.sh:105` — ten faucets at 1e18 sum to 1e19 ugnot, above the `int64` ceiling that `Coin.Amount` uses.
  <details><summary>details</summary>

  [The comment](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L105) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L105) reads "~9.2x headroom under int64 max", which holds per faucet but not for the total: 1e19 exceeds 9,223,372,036,854,775,807. [`Coin.Add`](https://github.com/gnolang/gno/blob/fc4052651/tm2/pkg/std/coin.go#L128-L136) · [↗](../../../../../.worktrees/gno-review-5945/tm2/pkg/std/coin.go#L128-L136) panics on overflow, so a transfer that would push one account past the ceiling aborts.

  No current trigger: [`SDKBanker.TotalCoin`](https://github.com/gnolang/gno/blob/fc4052651/gno.land/pkg/sdk/vm/builtins.go#L42-L44) · [↗](../../../../../.worktrees/gno-review-5945/gno.land/pkg/sdk/vm/builtins.go#L42-L44) panics as unimplemented, so nothing on chain sums the supply, and the failure would abort a single transaction rather than the chain. It only bites whoever tries to consolidate the faucets.
  </details>

- **[two validators means no fault tolerance]** `misc/deployments/topaz.gno.land/gen-genesis.sh:82` — with two validators at power 60, +2/3 of 120 needs 81, so both must sign every block and either one going down halts the chain.
  <details><summary>details</summary>

  [The comment](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L82) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L82) addresses the power value rather than the set size. gnoland1 launched with [seven validators and stated the threshold explicitly](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/gnoland1/gen-genesis.sh#L29) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/gnoland1/gen-genesis.sh#L29). Recovery on topaz requires a GovDAO proposal through `r/sys/validators/v3`, which itself needs blocks, so a single-node outage is not self-healing.
  </details>

- **[test-only packages ship in the launch set]** `misc/deployments/topaz.gno.land/gen-genesis.sh:643` — [`deplist -test-dep`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L643) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L643) takes the set from 79 to 90 packages.
  <details><summary>details</summary>

  The eleven added are `p/demo/nestedpkg`, `p/jeronimoalbi/expect`, `p/moul/typeutil`, `p/nt/testutils/v0`, `p/nt/uassert/v0`, `p/nt/urequire/v0`, `p/onbloc/diff`, `p/sunspirit/table`, `r/demo/defi/grc20factory`, `r/tests/vm` and `r/tests/vm/subtests`. The last two are the VM's adversarial cross-realm probes: the package carries an [`exploit.gno`](https://github.com/gnolang/gno/blob/fc4052651/examples/gno.land/r/tests/vm/exploit.gno#L1) · [↗](../../../../../.worktrees/gno-review-5945/examples/gno.land/r/tests/vm/exploit.gno#L1) and its entry points are built on [`chain/runtime/unsafe`](https://github.com/gnolang/gno/blob/fc4052651/examples/gno.land/r/tests/vm/tests.gno#L5) · [↗](../../../../../.worktrees/gno-review-5945/examples/gno.land/r/tests/vm/tests.gno#L5).

  gnoland1 uses the same flag, so this is inherited rather than new, and the deploys succeed. The point is that neither `README.md` nor the PR body says the set includes test dependencies, so a reader auditing the 90 packages has no signal that eleven of them are there for tests.
  </details>

- **[deployed package list is not reviewable in-tree]** `misc/deployments/topaz.gno.land/.gitignore:2` — [`/work/`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/.gitignore#L2) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/.gitignore#L2) hides `packages.gen.txt`, so the exact deployed set has to be regenerated to be read; gnoland1 [commits its copy](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/gnoland1/packages.gen.txt#L1) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/gnoland1/packages.gen.txt#L1), which also makes set changes diffable across regenerations. Not posted; recorded here as a follow-up rather than a change needed for this PR.

## Verified

- The rebuild is reproducible and matches what shipped: running `gen-genesis.sh` on this worktree produced `genesis.json` with sha256 `2dd049f973b82858727440df9aff5722cb0b322fd00890f40f2b0688276898ff` in 1m09s, equal to [`CHECKSUMS_DATA`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L163) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L163), to [`VALIDATOR.md:45`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/VALIDATOR.md?plain=1#L45) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/VALIDATOR.md#L45), and to the digest GitHub reports for the `genesis.json` asset on the `chain/topaz` release. Every intermediate artifact matched its locked hash.
- Booting a node on that genesis, the strict replay report is `total: 94, ok: 94, ok_gas_differs: 0, failed: 0, skipped_failed: 0`, so all 90 addpkgs and the four setup transactions execute.
- The bootstrap transactions take effect on chain: `gno.land/r/gov/dao.AllowedDAOs()` returns `["gno.land/r/gov/dao/v3/impl"]`, `gno.land/r/sys/names.IsEnabled()` returns `true`, and `valopers.GetByAddr` resolves both operator addresses to `gno-core-val-01` and `gno-core-val-02`.
- The nine genesis fee payers hold 8,999,999,775,461,906 ugnot after genesis rather than zero, against a measured burn of 224,538,094 ugnot; the deployer alone keeps 999999876075544 of the 1e15 it was credited.
- The readiness gap is timed, not inferred: on a re-boot of the same measurement genesis, `auth/accounts` answered at 10s while `bank/balances` returned `data: ""` until 14s, then the true figure.
- Faucet allocation lands as intended: each of the ten faucet addresses holds 1000000000000000000 ugnot post-genesis.

## Open questions

- `r/sys/users/init.Bootstrap` is never called at genesis, so `r/sys/namereg/v1` is the only whitelisted registration controller (it self-registers in its own `init`). Post-genesis names are therefore limited to the `nym-[a-z]{5,13}\d{3}` shape, and namespaces like `moul` or `gnops` can only be granted by a GovDAO proposal. That looks deliberate for a testnet, so not posted.
- [`README.md:24`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/README.md?plain=1#L24) · [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/README.md#L24) lists CLA and minimum fee as unset at genesis. With unrestricted transfers and 1e18-ugnot faucets, a zero minimum fee leaves no spam floor; worth a launch-day decision, but it is a chain-policy call rather than a defect in this diff.
