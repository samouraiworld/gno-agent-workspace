# Review: PR [#5382](https://github.com/gnolang/gno/pull/5382)
Event: REQUEST_CHANGES

## Body
Verified on a0226c4: `Fee.SponsorStorage=true` survives an amino binary round-trip (proto field 3) and JSON, so the sign-bytes and the wire agree and no sponsored tx silently fails signature verification.

One item that gates enabling the feature rather than merging it: a sponsored tx's compute is charged to the block gas meter (`baseapp.go:868-870`) and the next block's price is derived from block gas consumed (`auth/keeper.go:363`), so 0-fee traffic raises `LastGasPrice`, the floor normal fee-payers must meet, while contributing no fees itself. Deterministic, so not a fork, but worth settling before `MaxGasCreditPerTx` is ever set above 0: should sponsored gas feed the dynamic price, or be excluded from it?

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/1-a0226c4/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoland/app.go:278 [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L278)
In a SponsorStorage tx this pays every realm's freed-storage deposit refund to the PayStorage sponsor (`psi.RealmAddr`), while the no-sponsor branch just below routes refunds to `ctx.TxCaller()`. So a realm that sponsors storage also collects the deposit refunds for any storage the tx frees, including storage a user or an unrelated realm funded in an earlier tx. Route freed-storage refunds to the tx caller, as the no-sponsor path and pre-PR gno both do.

## gno.land/pkg/gnoland/app.go:287 [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/gnoland/app.go#L287)
This grow-without-PayStorage rejection only runs in the endTxHook, which fires in DeliverTx alone, so a SponsorStorage 0-fee tx that grows storage without calling PayStorage passes CheckTx admission and fails only at block time. Any user can get one admitted, so each executes to completion and burns up to the credit window of validator compute for free before failing. The PayGas-not-called check in `runTx` (`baseapp.go:983`) runs in every mode and is caught at admission; this one should move there too.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5382 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/ss_admission_gap.txtar <<'EOF'
loadpkg gno.land/r/test/ss_payer $WORK/payer
adduserfrom user1 'success myself purchase tray reject demise scene little legend someone lunar hope media goat regular test area smart save flee surround attack rapid smoke'
gnoland start
gnokey maketx send -send 10000000ugnot -to g1sel0lekwpwm6fmjnfeqfwwgrrnymjx97ujha04 -gas-fee 1000000ugnot -gas-wanted 2000000 -broadcast -chainid=tendermint_test test1
stdout OK!
gnokey sign -tx-path $WORK/grow.tx -chainid=tendermint_test -account-number $user1_account_num -account-sequence $user1_account_seq user1
stdout 'Tx successfully signed'
! gnokey broadcast $WORK/grow.tx -quiet=false
stdout 'HEIGHT:     3'
stdout 'GAS USED:'
stderr 'deliver transaction failed'
stderr 'unauthorized error'

-- payer/gnomod.toml --
module = "ss_payer"
gno = "0.9"

-- payer/payer.gno --
package ss_payer

import "chain/runtime"

var data []string

func Grow(cur realm) string {
	runtime.PayGas(5000000)
	data = append(data, "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	return "grown"
}

-- grow.tx --
{"msg":[{"@type":"/vm.m_call","caller":"g1c0j899h88nwyvnzvh5jagpq6fkkyuj76nld6t0","send":"","pkg_path":"gno.land/r/test/ss_payer","func":"Grow","args":null}],"fee":{"gas_wanted":"10000000","gas_fee":"0ugnot","sponsor_storage":true},"signatures":null,"memo":""}
EOF
go test ./gno.land/pkg/integration/ -run 'TestTestdata/ss_admission_gap$' -v 2>&1 | tail -6
rm gno.land/pkg/integration/testdata/ss_admission_gap.txtar
```

```
        GAS USED:   1427195
        HEIGHT:     3
        [stderr] "gnokey" error: unauthorized error
        deliver transaction failed: log:msg:0,success:true
--- PASS: TestTestdata/ss_admission_gap (3.27s)
ok      github.com/gnolang/gno/gno.land/pkg/integration
```
The tx is included at HEIGHT 3, the message succeeds, ~1.4M gas is spent, then the tx fails at DeliverTx: it passed mempool admission.
</details>

## tm2/pkg/sdk/auth/ante.go:50 [↗](../../../../../.worktrees/gno-review-5382/tm2/pkg/sdk/auth/ante.go#L50)
`isZeroFeeTx` omits the `ctx.BlockHeight() > 0` guard that `runTx`'s `zeroFeeCreditTx` and both SponsorStorage rejections carry. It is harmless only because `SetGasMeter` forces an infinite meter at height 0; a genesis with `MaxGasCreditPerTx > 0`, or a refactor of that height-0 branch, would silently cap or misclassify genesis txs. Add `&& ctx.BlockHeight() > 0` for parity.

## gno.land/pkg/sdk/vm/keeper.go:1989 [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1989)
The `depositAmt <= 0 → DefaultDeposit` fallback is unreachable (both callers pass `maxBudget > 0`), and if it were reached it would disable the budget cap and charge the sponsor up to `DefaultDeposit`. Assert `maxBudget > 0` instead of falling back silently.

## gno.land/pkg/sdk/vm/keeper.go:1800 [↗](../../../../../.worktrees/gno-review-5382/gno.land/pkg/sdk/vm/keeper.go#L1800)
The doc comment on this newly-exported `ProcessStorageDeposit` says it charges and refunds the caller, but under sponsorship the caller is the PayStorage realm, and for restricted denoms the refund goes to `StorageFeeCollector`. Update the comment to state who actually pays and receives now that it is public API.

## gnovm/stdlibs/chain/runtime/paygas.go:9 [↗](../../../../../.worktrees/gno-review-5382/gnovm/stdlibs/chain/runtime/paygas.go#L9)
Missing test: patch coverage is ~21% and the natives are exercised only through txtar, leaving several money/consensus branches without a focused Go test.

<details><summary>test cases</summary>

Three paste-ready tests, each passing at a0226c4 as a regression guard:

- `Tx.ValidateBasic` zero-fee accept/reject matrix, canonical zero and valid coins accepted, bad-denom / empty-denom-with-amount / negative rejected: [tx_validatebasic_fee_test.go](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/1-a0226c4/tests/tx_validatebasic_fee_test.go).
- `ValidateConsensusParams` rejects `MaxGasCreditPerTx < 0` and `> MaxGas` unless `MaxGas == -1`: [params_maxgascredit_test.go](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/1-a0226c4/tests/params_maxgascredit_test.go).
- The admission gap above, red-to-green once the storage check moves into `runTx`: [paygas_sponsorstorage_admission_gap.txtar](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/1-a0226c4/tests/paygas_sponsorstorage_admission_gap.txtar).

Worth adding too: an ante test pinning that a 0-fee tx's meter and reported `GasWanted` are the credit window, not the client `GasWanted`, with a normal-tx regression guard.
</details>
