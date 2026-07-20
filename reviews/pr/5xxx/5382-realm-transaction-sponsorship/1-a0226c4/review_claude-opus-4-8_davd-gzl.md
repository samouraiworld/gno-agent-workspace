# PR [#5382](https://github.com/gnolang/gno/pull/5382): feat: realm transaction sponsorship (PayGas + PayStorage)

URL: https://github.com/gnolang/gno/pull/5382
Author: omarsy | Base: master | Files: 56 | +3330 -100
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep, multi-lens) | Commit: a0226c4 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5382 a0226c4`
Overview: [visual overview](../overview.html) · [↗](../overview.html)

**TL;DR:** Lets a realm pay a user's gas and storage deposits out of its own balance, so a user holding zero gnot can still transact. The realm decides mid-execution whether to sponsor by calling the new `runtime.PayGas` / `runtime.PayStorage` natives, gated behind a consensus credit window and a per-validator opt-in, both off by default.

**Verdict: REQUEST CHANGES** — two SponsorStorage-path defects to resolve (freed-storage refunds are paid to the sponsor rather than the tx caller; a grow-without-PayStorage tx passes mempool admission and only fails at block time) plus one fee-market question to settle before the feature is ever enabled. Core paths (consensus determinism, wire compat, gas/coin math, backward compat) are clean.

## Summary
A 0-fee tx runs on a bounded gas "credit window" (`Block.MaxGasCreditPerTx`, a new consensus param); the realm runs its own logic then calls `PayGas(maxFee)` to commit to paying, and settlement at end-of-tx charges the realm `ceil(gasUsed × gasPrice)` capped at `maxFee`. A tx that never calls `PayGas` is rejected in every mode, so a proposer cannot force-include a free run. `PayStorage(maxDeposit)` does the same for storage deposits, and `Fee.SponsorStorage` defers a multi-message tx's storage to one end-of-tx settlement. The feature needs both `MaxGasCreditPerTx > 0` (consensus, deterministic) and a per-validator `AllowZeroFeeTxs` mempool opt-in; with both off the node behaves exactly as master. Gas/storage math is overflow-guarded and money is globally conserved on the multi-message and two-realm-sponsor paths. The remaining issues all live in the `SponsorStorage` deferred path and the economics of enabling the feature.

## Glossary
- ante handler: pre-execution stage; verifies sigs, sequence, fee, gas bounds; its writes survive a failed tx.
- storage deposit: per-realm refundable charge for on-chain storage, locked on growth, refunded on the realm's original deposit ratio.
- app hash: per-block Merkle commitment to app state; a mismatch between honest nodes halts the chain.

## Fix
The two code Warnings both sit in the deferred storage settlement. Freed-storage refunds should return to the tx caller ([`app.go:291`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/gnoland/app.go#L291) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L291) already does this in the no-sponsor branch), not to the storage sponsor. The grow-without-PayStorage rejection should move from the DeliverTx-only `endTxHook` into `runTx` alongside the PayGas-not-called check ([`baseapp.go:984`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/baseapp.go#L984) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L984)) so it runs at CheckExecute admission too.

## Critical (must fix)
None.

## Warnings (should fix)

- **[sponsor pockets refunds it never funded]** [`gno.land/pkg/gnoland/app.go:278`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/gnoland/app.go#L278) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L278) — In a `SponsorStorage` tx, every realm's freed-storage deposit refund is paid to the PayStorage sponsor, including realms the sponsor never grew or funded.
  <details><summary>details</summary>

  The deferred settlement passes `psi.RealmAddr` (the PayStorage sponsor) as the refund receiver for the whole tx's accumulated diffs at [`app.go:278`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/gnoland/app.go#L278) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L278), and the release branch sends the unlocked deposit to that address ([`keeper.go:2065`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/sdk/vm/keeper.go#L2065) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L2065); same in the per-message sponsored path, [`keeper.go:1825`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/sdk/vm/keeper.go#L1825) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1825) reassigns `caller = psi.RealmAddr`, [`keeper.go:1947`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/sdk/vm/keeper.go#L1947) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1947) sends there). Pre-PR gno routes a storage refund to the message caller; the no-sponsor deferred branch still does ([`app.go:291`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/gnoland/app.go#L291) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L291) uses `ctx.TxCaller()`). So a realm advertising storage sponsorship silently collects, in one tx, the deposit refunds of any storage that tx frees, funded in a prior tx by the user or an unrelated realm.

  Money stays globally conserved, but it is redistributed from the original depositor to the sponsor. A run (realmA sponsors gas, realmB sponsors storage `MaxDeposit=100000`; one tx grows `r/B` +400 bytes and frees `r/C` −500 bytes where `r/C` held a pre-existing 50000 deposit) settles realmB at net +10000: it pays 40000 for its own growth yet receives `r/C`'s entire 50000 refund. Fix: route freed-storage refunds to `ctx.TxCaller()`, mirroring the no-sponsor branch and the pre-PR behavior; keep charging growth to the sponsor.
  </details>

- **[guaranteed-fail tx enters the mempool]** [`gno.land/pkg/gnoland/app.go:285-292`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/gnoland/app.go#L285-L292) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L285) — A `SponsorStorage` 0-fee tx that calls PayGas and grows storage but never calls PayStorage passes CheckTx admission and only fails at DeliverTx.
  <details><summary>details</summary>

  The grow-without-PayStorage rejection lives in the `endTxHook` ([`app.go:287`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/gnoland/app.go#L287) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L287)), which `runTx` invokes only in DeliverTx ([`baseapp.go:1007`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/baseapp.go#L1007) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L1007)). The PayGas-not-called rejection, by contrast, runs in `runTx` for all modes ([`baseapp.go:983-984`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/baseapp.go#L983-L984) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L983)) so it is caught at CheckExecute admission. The storage case is not, so any user (not only a block proposer force-including) can get such a tx admitted; each one executes to completion and burns up to the credit window of validator compute before failing, with the submitter paying nothing, so it is a free mempool CPU-amplification vector on opt-in validators as well as wasted block gas.

  Confirmed at a0226c4: the tx reaches HEIGHT 3, the message succeeds, ~1.4M gas is consumed, then the tx fails with `unauthorized error` at DeliverTx (`tests/paygas_sponsorstorage_admission_gap.txtar`). This narrows the design's "CheckTx filters invalid 0-fee txs before mempool entry" guarantee. Fix: move the grew-without-PayStorage rejection into `runTx` (all modes), mirroring the PayGas check.
  </details>

- **[gasless load raises everyone's fee floor]** [`tm2/pkg/sdk/baseapp.go:868-870`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/baseapp.go#L868-L870) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L868) — A sponsored tx's compute is charged to the block gas meter, which feeds the dynamic gas-price update, so 0-fee traffic raises the price normal users pay while contributing no fees.
  <details><summary>details</summary>

  DeliverTx charges the tx's consumed gas to the block gas meter ([`baseapp.go:868-870`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/baseapp.go#L868-L870) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L868)), and the next block's price is derived from `BlockGasMeter().GasConsumed()` ([`auth/keeper.go:363`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/auth/keeper.go#L363) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/auth/keeper.go#L363)). So gasless-tx load pushes up `LastGasPrice`, both the minimum a normal fee-payer must meet and the price at which sponsored gas itself settles, while 0-fee txs pay nothing into the fee market. It is deterministic (identical on every node, no fork), so not a safety bug, but it is a real economic coupling. This is a decision to settle before the feature is enabled on any live chain, not a code change to land in this PR; noting it as the load-bearing gate on the verdict.
  </details>

## Nits

- [`tm2/pkg/sdk/auth/ante.go:50`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/auth/ante.go#L50) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/auth/ante.go#L50) — `isZeroFeeTx` omits the `ctx.BlockHeight() > 0` guard that `runTx`'s `zeroFeeCreditTx` and both SponsorStorage rejections carry. Harmless only because `SetGasMeter` forces an infinite meter at height 0; a genesis with `MaxGasCreditPerTx > 0`, or any refactor of that height-0 branch, would silently cap or misclassify genesis txs. Add `&& ctx.BlockHeight() > 0` for parity.
- [`tm2/pkg/store/types/gas.go:248`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/store/types/gas.go#L248) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/store/types/gas.go#L248) — `basicGasMeter.SetLimit` accepts any non-negative value, so the "PayGas may only shrink the limit" invariant that keeps a 0-fee tx bounded by the credit window is enforced solely by the single caller. A future caller that raises the limit would let consumption exceed the reported `GasWanted` and overflow block packing. Defense-in-depth: reject a raise in the meter, or flag the invariant loudly at the type.
- [`gno.land/pkg/sdk/vm/keeper.go:1989`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/sdk/vm/keeper.go#L1989) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1989) — `ProcessStorageDepositFromDiffs`'s `depositAmt <= 0 → DefaultDeposit` fallback is unreachable (both call sites pass `maxBudget > 0`), and if it were reached it would silently disable the budget cap and charge up to `DefaultDeposit`. Prefer asserting `maxBudget > 0` over a silent fallback.
- [`gno.land/pkg/sdk/vm/keeper.go:1878`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/sdk/vm/keeper.go#L1878) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1878) — The per-message sponsored path checks the budget twice (the `maxStorageBudget` cap and the `depositAmt < requiredDeposit` check are algebraically identical when sponsored); `FromDiffs` does the same. Redundant, not wrong; collapse for clarity.
- [`gno.land/pkg/sdk/vm/keeper.go:1800`](https://github.com/gnolang/gno/blob/a0226c4/gno.land/pkg/sdk/vm/keeper.go#L1800) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1800) — The doc comment on the newly-exported `ProcessStorageDeposit` says it charges and refunds "the caller", but under sponsorship the caller is overridden to the PayStorage realm and, for restricted denoms, the refund goes to `StorageFeeCollector`. Now that the function is public API, update the comment to state who actually pays and receives.

## Missing Tests

- **[consensus-affecting branches uncovered]** [`gnovm/stdlibs/chain/runtime/paygas.go:9`](https://github.com/gnolang/gno/blob/a0226c4/gnovm/stdlibs/chain/runtime/paygas.go#L9) · [↗](../../../../../.worktrees/gno-review-5382/gnovm/stdlibs/chain/runtime/paygas.go#L9) — Patch coverage is ~21% (the natives 0% unit-covered, exercised only via txtar); several money/consensus branches have no direct Go test.
  <details><summary>details</summary>

  The txtar suite is thorough on happy/rejection paths, but the consensus-affecting units lack focused tests. Three ready-to-add, paste-runnable tests (each passes at a0226c4, guarding against regression):
  - `tests/tx_validatebasic_fee_test.go` — the zero-fee accept/reject matrix in `Tx.ValidateBasic` (canonical zero and valid coins pass; bad-denom, empty-denom-with-amount, negative reject).
  - `tests/params_maxgascredit_test.go` — `ValidateConsensusParams` rejects `MaxGasCreditPerTx < 0` and `> MaxGas` (unless `MaxGas == -1`).
  - `tests/paygas_sponsorstorage_admission_gap.txtar` — the W-B admission gap, red→green when the check moves into `runTx`.

  Also worth adding: an ante test pinning that a 0-fee tx's meter and reported `GasWanted` are the credit window (not the client `GasWanted`), with a normal-tx regression guard.
  </details>

## Open questions
- Fee-market feedback (W-C above) is the one item that should block enabling the feature, not merging the code. Whether sponsored compute should feed the dynamic gas price, or be excluded from it, is a policy call for the maintainers; surfaced because it changes the answer to "is this safe to turn on."
- Proposer force-inclusion of a PayGas-not-called 0-fee tx was checked and is correctly rejected at DeliverTx with message writes reverted and gas bounded by the credit window (only the ante sequence write persists); no finding. Recorded so the next round need not re-derive it.
- `MaxGasCreditPerTx` is memoized at InitChain / node load and never reloaded, so activating the feature must be a coordinated restart, never a live governance tx (a mid-flight divergence in the `zeroFeeCreditTx` classification would fork). Pre-existing consensus-param memoization; the PR only adds the field. Worth confirming governance tooling cannot set it live.
- First-time admission of every distinct 0-fee tx runs the full VM ([`baseapp.go:610`](https://github.com/gnolang/gno/blob/a0226c4/tm2/pkg/sdk/baseapp.go#L610) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L610) admits via `RunTxModeCheckExecute`) with no fee and no per-account admission rate limit, so even a flood of valid sponsored txs is a free CPU-amplification vector on opt-in validators. Rate-limiting is deferred by design; not posted, but a real load ceiling to add before enabling at scale.

## Verification
Verified on a0226c4 (checks CI does not cover):
- `Fee.SponsorStorage=true` survives an amino binary round-trip (`08904e1801`, proto field 3) and JSON, so the sign-bytes and wire paths agree; `Result` gained no wire field and stays compatible with `abci.ResponseDeliverTx`.
- The W-B admission gap reproduces: the grow-without-PayStorage tx is included at HEIGHT 3 and fails only at DeliverTx.
- The full `paygas_*` / `paystorage_*` txtar suite (24 subtests) and the baseapp/ante sponsorship unit tests pass.
- With the feature off (`MaxGasCreditPerTx = 0`, `AllowZeroFeeTxs = false`, both default), the ante and `runTx` classification (`isZeroFeeTx`, `zeroFeeCreditTx`) is false and the non-sponsored path is unchanged.

Independent multi-lens pass (correctness, blue-team, red-team, plus a claim-verification gate and a verdict/severity critic) confirmed every finding above, added the two Nits/Open-question items noted here, and returned no verdict change and no new blocker: W-A and W-B were each reproduced by more than one lens (W-A: a sponsored refund routes 203400 ugnot to the sponsor with the depositor receiving 0; W-B: live txtar at HEIGHT 3 with the account sequence advanced, proving delivery not check-rejection).
