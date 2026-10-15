# Review: [#5382](https://github.com/gnolang/gno/pull/5382)
Event: APPROVE
Verdict: APPROVE. No finding sits above Nit on the consensus path; the open items are a log field in the ante's out-of-gas recover, unbilled settlement work and three latent replay and simulate gaps, none blocking.
Model: claude-opus-5-5, standard review
Commit: ba63646a5 (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-5382 ba63646a5`
Round: 4. 6 finders, one reflector, 7 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted, none of them above Nit.

## Body

- Nit: the ante's out-of-gas recover passes [`tx.Fee.GasWanted`](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/auth/ante.go#L140) to `OutOfGasLog` and to [`res.GasWanted`](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/auth/ante.go#L143). A 0-fee tx that runs out of gas in the ante reports its declared gas wanted, often 0, rather than the [credit window](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/auth/ante.go#L122) it ran on.
- Related suggestion: a replayed 0-fee tx keeping its source height is [marked sponsored](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L1047) and gets an [unmetered gas meter](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/app.go#L982) in source replay mode. Settlement then debits the sponsoring realm from the new VM's gas at the [genesis price](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/app.go#L937), so its balance after a hard fork differs from the source chain's. No chain with sponsorship enabled exists yet.

## SKIP gno.land/pkg/gnoland/sponsorship_usecase_test.go:239 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/sponsorship_usecase_test.go#L239) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/sponsorship_usecase_test.go#L239) · Missing test

Missing test: the GRC20 cases use a two-entry ledger, while a 64-entry avl case already needs 6.22M gas against the 6M window floor.

Not posted: PLAUSIBLE on the finder's read; the 64-holder mint check is still to run.

## SKIP gno.land/pkg/integration/testdata/paygas_insufficient_balance.txtar:15 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testdata/paygas_insufficient_balance.txtar#L15) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/integration/testdata/paygas_insufficient_balance.txtar#L15) · Missing test

Missing test: the fixture asserts only `insufficient coins`, which a delivered tx failing prints too, so it stays green with the settlement dry run in CheckTx removed.

Not posted: PLAUSIBLE on the finder's read; the baseapp.go:1109 mutation is still to run.

## SKIP gno.land/pkg/integration/testdata/paygas_oog_after_paygas.txtar:46 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testdata/paygas_oog_after_paygas.txtar#L46) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/integration/testdata/paygas_oog_after_paygas.txtar#L46) · Missing test

Missing test: the `SmallBudgetHeavyWork` loop also exhausts the 10M credit window, so the fixture stays green when PayGas stops lowering the gas limit to what maxFee buys.

Not posted: PLAUSIBLE on the finder's read; deleting paygas.go:61 and running the fixture is still to do.

## SKIP gno.land/pkg/integration/testdata/paystorage_before_paygas.txtar:35 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testdata/paystorage_before_paygas.txtar#L35) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/integration/testdata/paystorage_before_paygas.txtar#L35) · Missing test

Missing test: the fixture never checks who paid the first message's storage deposit, so a sponsorship that falls back to the caller passes.

Not posted: PLAUSIBLE on the finder's read; the keeper.go:2454 mutation is still to run.

## SKIP gno.land/pkg/sdk/vm/keeper.go:2562 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/sdk/vm/keeper.go#L2562) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L2562) · Missing test

Missing test: no fixture frees more of the sponsor's storage than the tx locked, so the split refund and the partial `SponsorRefund` subtraction in gnokey are unpinned.

Not posted: PLAUSIBLE on the finder's read; the keeper.go:2564 mutation is still to run.

## SKIP tm2/pkg/sdk/baseapp.go:795 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L795) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L795) · Missing test

Missing test: no test reaches the recover in `decodeTx`, since `TestCheckTxMalformedTypeURL` no longer panics the decoder and asserts only that the result is not OK.

Not posted: PLAUSIBLE on the finder's read; removing the defer and running the CheckTx tests is still to do.

## gno.land/pkg/integration/testscript_gnoland.go:294 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testscript_gnoland.go#L294) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/integration/testscript_gnoland.go#L294) · Nit

Nit: the `bp.MaxGas != -1` conjunct is always true, since the flag [replaces the 30B genesis value](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testscript_gnoland.go#L284) only when positive, so it implies a `-max-gas -1` mode that does not exist.

## contribs/gnogenesis/internal/verify/verify.go:102 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/contribs/gnogenesis/internal/verify/verify.go#L102) · [↗](../../../../../.worktrees/gno-review-5382/contribs/gnogenesis/internal/verify/verify.go#L102) · Suggestion

Suggestion: the check fires only when `MaxGasCreditPerTx` is 0, so with the window open a 0-fee genesis tx at a replayed height passes verify. That tx then [fails as sponsored without PayGas](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L1090) and [halts InitChain](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/app.go#L61).

## gno.land/pkg/gnoland/app.go:276 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/app.go#L276) · [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L276) · Suggestion

Suggestion: `settleCtx` runs the sponsor's settlement transfer on an unmetered gas meter. That work counts in neither the tx's gas used nor the [block gas](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L967), unlike the ante's [`DeductFees`](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/auth/ante.go#L250) which the comment cites as its model.

## gnovm/stdlibs/chain/runtime/paygas.go:44 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gnovm/stdlibs/chain/runtime/paygas.go#L44) · [↗](../../../../../.worktrees/gno-review-5382/gnovm/stdlibs/chain/runtime/paygas.go#L44) · Suggestion

Suggestion: the overflow panic on `maxFee * gas price` sits after the [eligibility return](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gnovm/stdlibs/chain/runtime/paygas.go#L29). A huge maxFee passes `gno test`, then aborts every sponsored tx on chain.

## tm2/pkg/sdk/baseapp.go:1109 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L1109) · [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/baseapp.go#L1109) · Suggestion

Suggestion: the settlement hook skips Simulate, while the [missing-PayGas check](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L1090) runs there. A wallet simulating a 0-fee tx sees success for a sponsor that cannot pay, and the broadcast is then rejected.
