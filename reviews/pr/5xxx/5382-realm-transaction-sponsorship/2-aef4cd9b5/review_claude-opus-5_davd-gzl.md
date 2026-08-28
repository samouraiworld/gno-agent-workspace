# PR [#5382](https://github.com/gnolang/gno/pull/5382): feat: realm transaction sponsorship (PayGas + PayStorage)

URL: https://github.com/gnolang/gno/pull/5382
Author: omarsy | Base: master | Files: 61 | +4847 -132
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: aef4cd9b5 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5382 aef4cd9b5`
Overview: [visual overview](../overview.html)

**TL;DR:** A realm can pay a user's gas and storage, so a transaction can carry a zero fee and still be included. Since round 1 the author closed the admission hole and the genesis-exemption parity gap, and added a benchmark suite for the DoS question. What is left is where the money goes: a sponsoring realm collects storage refunds that belong to the tx caller, and sponsored gas still feeds the fee market that normal payers must clear.

**Verdict: REQUEST CHANGES** — the settlement invariant is now validated at admission, but the refund recipient diverges from the unsponsored path and two consensus-adjacent validations have no test (1 Warning, 2 Missing tests, 2 Nits).

## What moved since 1-a0226c4

Five of the PR's own commits landed on top of a master merge.

| Commit | Effect |
|---|---|
| [`14f99add0`](https://github.com/gnolang/gno/commit/14f99add08c25f9d14884d23b689a4aa7017a948) | `EndTxHook` runs in `RunTxModeCheckExecute` as a dry run, so settlement is validated before admission; settlement events reach the response; the genesis exemption is keyed on mode rather than raw height; the storage balance is read in the deposit denom |
| `c454fb373`, `d0d2763e7`, `22cefd7d8`, `aef4cd9b5` | CheckTx admission benchmarks, load tests, and a GRC20 paymaster path measured end to end |

| Round 1 finding | State at aef4cd9b5 |
|---|---|
| `app.go:278` — freed-storage refunds go to the sponsor | open, re-anchored to `app.go:298`, now stated as intended in the [comment above it](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/gnoland/app.go#L296-L297) |
| `app.go:287` — grow-without-PayStorage rejected only at DeliverTx | fixed by `14f99add0`, with [`paygas_sponsor_storage_no_paystorage_rejected.txtar`](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/integration/testdata/paygas_sponsor_storage_no_paystorage_rejected.txtar) as the regression guard |
| `ante.go:50` — `isZeroFeeTx` missing the height guard | fixed by `14f99add0`, differently and better: enforcement is [keyed on mode](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/sdk/auth/ante.go#L59), which also closed a hole where every CheckTx admission before the first Commit skipped PayGas enforcement |
| `keeper.go:1989` — `DefaultDeposit` fallback | open, re-anchored to `keeper.go:2050-2052`, now carrying a comment that calls it defensive |
| `keeper.go:1800` — `ProcessStorageDeposit` doc comment | open, re-anchored to `keeper.go:1861-1870`, and the comment is detached from the function by a blank line |
| `paygas.go:9` — patch coverage ~21% | dropped as stated; the [codecov comment](https://github.com/gnolang/gno/pull/5382#issuecomment-4148195096) has not been updated since 2026-03-29 and four test commits landed after it, so the number is not measurable from here. Replaced by two specific gaps, both confirmed by reading the test files |

## Round 1 reached the author through a third party

Round 1 was never posted from this account: `gh api repos/gnolang/gno/pulls/5382/reviews` and `.../comments` return nothing for `davd-gzl` at this head. The admission-gap finding reached the PR as [a comment from moul](https://github.com/gnolang/gno/pull/5382#issuecomment-4885804942) marked `[bot] Review finding:`, carrying the same mechanism and the same suggested fix. The author [took the stronger of the two options offered](https://github.com/gnolang/gno/pull/5382#issuecomment-5359211924) and validates the whole settlement invariant at admission rather than the `grewStorage => MaxDeposit > 0` implication alone. Nothing else from round 1 was relayed, so the four remaining findings are reaching the author for the first time and none is auto-SKIPped as already raised.

## Summary

A 0-fee transaction is admitted on the strength of a realm's promise to pay, made during execution. `PayGas` inside a realm transfers the fee from the realm to the fee collector; `PayStorage` commits a budget for the transaction's storage growth. Two things had to move for that to be safe: the ante handler must stop rejecting a zero fee outright, and settlement must happen where a doomed transaction can still be refused. This round closes the second. `EndTxHook` now runs under `RunTxModeCheckExecute` against the discarded message cache, so sponsor solvency, the storage budget, and the grew-without-`PayStorage` case are all decided before the transaction is gossiped. The gno-store commit moved out to a delivery-only `CommitTxHook`, because it writes through the node-level cache and cannot run in a dry run.

## Critical (must fix)

None.

## Warnings (should fix)

- **[a sponsor collects refunds the tx caller earns]** `gno.land/pkg/gnoland/app.go:298` — the sponsored branch names the PayStorage realm as the recipient of both charges and refunds; the unsponsored branch four lines down names the tx caller.
  <details><summary>details</summary>

  In [the `psi.MaxDeposit > 0` branch](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/gnoland/app.go#L298) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L298) the `caller` argument is `psi.RealmAddr`. Inside [`ProcessStorageDepositFromDiffs`](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/sdk/vm/keeper.go#L2042) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L2042) that one argument serves both directions: a positive diff [locks a deposit from it](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/sdk/vm/keeper.go#L2091), and a negative diff [refunds to it](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/sdk/vm/keeper.go#L2126-L2130), the amount being the realm's own locked deposit prorated by the freed bytes rather than anything this transaction paid. The [default branch](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/gnoland/app.go#L311) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L311), reached when a SponsorStorage tx only frees storage, passes `ctx.TxCaller()` instead, and pre-PR gno routed every refund the same way. So the recipient of a freed-storage refund depends on whether the same transaction also grew something, which is not a property a user or a realm controls. A realm sponsoring one byte of growth collects the release of deposits locked by earlier transactions from other accounts. The author's comment above the call states the behaviour as intended; the finding is that it diverges from the sibling branch and from master for a reason the design document does not give. Fix: pass `ctx.TxCaller()` for refunds in both branches, or state in the HLD why sponsorship transfers the refund claim.
  </details>

## Missing Tests

- **[the mempool fee gate has no test]** `tm2/pkg/std/tx.go:55` — no test in `tm2/pkg/std` calls `Tx.ValidateBasic`.
  <details><summary>details</summary>

  The PR relaxes the fee check from `!IsValid()` to [`!IsValid() && GasFee != (Coin{})`](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/std/tx.go#L55) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/std/tx.go#L55), which is what lets a 0-fee transaction reach the ante handler at all, and the comment above it names the case the looser `!IsZero() && !IsValid()` form would have admitted. `grep -rn ValidateBasic tm2/pkg/std/*_test.go` at this head returns one hit, on `MemPackage`, and there is no `tx_test.go`. Ready-to-add table in [tests/tx_validatebasic_fee_test.go](tests/tx_validatebasic_fee_test.go), run at aef4cd9b5: `ok github.com/gnolang/gno/tm2/pkg/std 0.069s`.
  </details>

- **[both new consensus-param rejections are untested]** `tm2/pkg/bft/types/params.go:80-92` — `makeParams` never sets the field the two branches read.
  <details><summary>details</summary>

  `ValidateConsensusParams` gains a [`MaxGasCreditPerTx < 0` rejection](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/bft/types/params.go#L80-L82) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/bft/types/params.go#L80) and a [`> MaxGas` rejection with a `MaxGas == -1` escape](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/bft/types/params.go#L90-L92) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/bft/types/params.go#L90). The package's only caller of `ValidateConsensusParams` is [`TestConsensusParamsValidation`](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/bft/types/params_test.go#L44-L46), whose fixtures come from [`makeParams`](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/bft/types/params_test.go#L51-L67), which sets four block fields and not this one, so every case runs with the credit window at zero. Ready-to-add table in [tests/params_maxgascredit_test.go](tests/params_maxgascredit_test.go), run at aef4cd9b5: `ok github.com/gnolang/gno/tm2/pkg/bft/types 0.125s`.
  </details>

## Nits

- **[a newly-exported symbol reads as undocumented]** `gno.land/pkg/sdk/vm/keeper.go:1861-1870` — a blank line at 1870 separates the comment from `func (vm *VMKeeper) ProcessStorageDeposit`, and the text describes the pre-sponsorship recipients.
  <details><summary>details</summary>

  Go associates a doc comment with the declaration immediately below it, so [the block at 1861-1869](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/sdk/vm/keeper.go#L1861-L1870) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1861) is a free-floating comment and `go doc` prints the symbol bare. Both lines of its content also went stale in this PR: under sponsorship the charged party is the PayStorage realm rather than the caller, and for a restricted denom the refund goes to `params.StorageFeeCollector` rather than back to anyone. Deleting the blank line and naming the two recipients fixes both in one edit.
  </details>

- **[a defensive fallback that raises the cap it defends]** `gno.land/pkg/sdk/vm/keeper.go:2050-2052` — reaching it swaps the sponsor's committed budget for `DefaultDeposit`.
  <details><summary>details</summary>

  [`depositAmt := maxBudget`, then `if depositAmt <= 0 { depositAmt = DefaultDeposit }`](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/sdk/vm/keeper.go#L2049-L2052) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L2049). The comment above it states the invariant the endTxHook guarantees and calls the fallback defensive, which is the right reading of the current callers: both pass a positive budget. The objection is only to the direction of the failure. `ProcessStorageDepositFromDiffs` is exported, and the branch that fires when the invariant breaks is the one that spends more of someone's money, not less. Returning an error costs the same line and fails toward not charging.
  </details>

## Verified

- Booted nothing: every finding this round is source-visible, and each anchor was read in the worktree at aef4cd9b5 before it was written.
- Both proposed tests were copied into the worktree, run, and removed. `go test -run TestValidateConsensusParamsMaxGasCreditPerTx ./tm2/pkg/bft/types/` and `go test -run TestTxValidateBasicFeeMatrix ./tm2/pkg/std/` both pass at this head with go1.25.9. `git status --short` in the worktree is empty afterwards.
- The admission fix is real, not just described: [`EndTxHook`'s guard](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/sdk/baseapp.go#L1072) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L1072) now admits `RunTxModeCheckExecute`, and the grow-without-PayStorage rejection it reaches is [the same typed error](https://github.com/gnolang/gno/blob/aef4cd9b5/gno.land/pkg/gnoland/app.go#L307) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L307) round 1 could only trip at DeliverTx.
- The genesis-exemption parity finding is closed by a different construction than the one proposed: [`notGenesis`](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/sdk/auth/ante.go#L59) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/auth/ante.go#L59) reads `BlockHeight() > 0 || Mode() != RunTxModeDeliver`, so adding `&& ctx.BlockHeight() > 0` as round 1 asked would have reopened the CheckTx window the author found.
- The fee-market claim was re-read at this head, not carried: [`ConsumeGas` on the block meter](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/sdk/baseapp.go#L920-L923) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L920) is unconditional in `RunTxModeDeliver`, and [`UpdateGasPrice`](https://github.com/gnolang/gno/blob/aef4cd9b5/tm2/pkg/sdk/auth/keeper.go#L361-L363) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/auth/keeper.go#L361) reads that meter with no filter on how the transaction paid.
- The refund arithmetic was read, not assumed: the released amount is prorated from the realm's own locked deposit rather than from `released * price`, so a governance price change cannot make it panic or under-refund. That part is correct; only the recipient is the finding.

## Open questions

- moul's [DoS-boundary comment](https://github.com/gnolang/gno/pull/5382#issuecomment-4886454488) is open and the author is answering it with the four benchmark commits in this range. Not posted: it is another reviewer's thread, and the fee-market bullet in the Body is a distinct mechanism, an honest payer's floor rising rather than validator CPU.
- The HLD records a divergence the fix cannot close: CheckTx cannot see pending transactions' message effects, so a sponsor gating on a one-shot resource can still admit N transactions of which one succeeds. Not posted: the author documented it in the same commit that fixed the admission hole, and no line in the diff carries an action.
- The invariant catalog walk applies to the `chain/runtime` natives rather than to the tm2 plumbing; `PayGas` and `PayStorage` both read the executing realm from `ExecContext` rather than from an argument, so no caller-identity class is reachable from the added surface.
