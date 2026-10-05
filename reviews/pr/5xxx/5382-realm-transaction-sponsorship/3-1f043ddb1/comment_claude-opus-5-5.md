# Review: [#5382](https://github.com/gnolang/gno/pull/5382)
Event: COMMENT
Verdict: REQUEST CHANGES. The branch lets `gnogenesis verify` pass a zero-fee genesis tx that a default node panics on, on chains that never enable sponsorship, and reports a sponsor-paid storage deposit as the signer's cost once the window opens.
Model: claude-opus-5-5, standard review: finders at xhigh, triage high, judges and reflector medium, writer and text pass low
Commit: 1f043ddb1 (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-5382 1f043ddb1`
Round: 3. 6 finders, one reflector, 13 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 1 refuted.

## Body

> AI review, claude-opus-5-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/3-1f043ddb1/claims.md) · Status: REQUEST CHANGES

## tm2/pkg/std/tx.go:55 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/tm2/pkg/std/tx.go#L55) · Warning

The relaxed fee check in `Tx.ValidateBasic` lets [`gnogenesis verify`](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/contribs/gnogenesis/internal/verify/verify.go#L94) accept a zero-fee genesis tx that [the ante handler rejects](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/tm2/pkg/sdk/auth/ante.go#L166) at the default window of 0. A default node then [panics in `InitChain`](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/gnoland/app.go#L391) on that genesis, since [`--skip-failing-genesis-txs`](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/cmd/gnoland/start.go#L91-L92) defaults to false.

<details><summary>repro</summary>

The test is `tests/b1-claims-zero-fee-genesis-verify_test.go`, copied into `contribs/gnogenesis/internal/verify/`. It passes at the head and fails at the merge base:

```text
tx.ValidateBasic() = <nil>
gnogenesis verify error = <nil>
--- PASS: TestB1ClaimsVerifyAcceptsZeroFeeGenesisTx
```

`gnoland start` keeps `--skip-failing-genesis-txs` false by default, so the app uses `PanicOnFailingTxResultHandler`.

</details>

## SKIP gno.land/pkg/sdk/vm/keeper.go:2547 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/sdk/vm/keeper.go#L2547) · Warning

The `sponsorShare` cap nets a release only against deposits the sponsor locked in the same tx. A 0-fee signer who grows a sponsored value and shrinks it in a later tx receives the sponsor's deposit as a refund; repeated, this drains the sponsor.

<details><summary>repro</summary>

The fixture is `tests/b3-lines-removed-reach-catalog-paystorage_cross_tx_drain.txtar`, run as `TestTestdata/paystorage_cross_tx_drain`. Both txs send `gas_fee 0ugnot`, and user1's balance rises by the refund:

```text
tx1: STORAGE FEE:    288500ugnot   user1 data: "1000000000ugnot"
tx2: STORAGE REFUND: 287500ugnot   user1 data: "1000287500ugnot"
--- PASS: TestTestdata/paystorage_cross_tx_drain
```

</details>

Not posted: the [`PayStorage` doc comment](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gnovm/stdlibs/chain/runtime/paystorage.gno#L16-L18) and the PR's Open items already name this limit, that a deposit refunds whoever frees the bytes.

## gno.land/pkg/sdk/vm/keeper.go:2488 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/sdk/vm/keeper.go#L2488) · Warning

The `StorageDepositEvent` emitted here carries no payer, so gnokey prints a sponsor-paid deposit as the signer's storage fee and total tx cost.

<details><summary>repro</summary>

The same fixture's first tx pays nothing, and gnokey still prints the deposit as its cost:

```text
STORAGE DELTA:  2885 bytes
STORAGE FEE:    288500ugnot
TOTAL TX COST:  288500ugnot
```

</details>

## gno.land/pkg/integration/testdata/paygas_credit_exhausted.txtar:12 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/integration/testdata/paygas_credit_exhausted.txtar#L12) · Missing test

Missing test: this fixture asks for exactly the credit window and checks only `out of gas`. It stays green when the ante meter follows `tx.Fee.GasWanted` instead of `MaxGasCreditPerTx`.

<details><summary>test cases</summary>

Send `-gas-wanted 100000000` and assert `stdout 'GAS USED:   100[0-9]{5}$'`. With `SetGasMeter(ctx, tx.Fee.GasWanted)` in the ante, this fixture still passes:

```text
go test -count=1 -p 1 -parallel 1 ./gno.land/pkg/integration/ -run 'TestTestdata/paygas_credit_exhausted$'
GAS WANTED: 10000000
GAS USED:   919232627
--- PASS: TestTestdata/paygas_credit_exhausted
```

</details>

## gno.land/pkg/integration/testdata/paygas_normal_fee_unaffected.txtar:18 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/integration/testdata/paygas_normal_fee_unaffected.txtar#L18) · Missing test

Missing test: the fixture funds the realm and never queries its balance. It passes when `X_payGas` debits the realm on a fee-paying tx.

<details><summary>test cases</summary>

```text
gnokey query bank/balances/g1pkhf78w28n9m9cgm4nsn3q3cggjy39nu7m3z0f
stdout '"10000000ugnot"'
```

With the gate dropped, the realm reads `"9998995ugnot"` and this assertion fails.

</details>

## gno.land/pkg/sdk/vm/keeper.go:1181 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/sdk/vm/keeper.go#L1181) · Missing test

Missing test: no txtar sends a 0-fee `addpkg` whose `init` calls PayGas or PayStorage, the second entry that `beginPayStorage` opens here.

## tm2/pkg/sdk/baseapp.go:661 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/tm2/pkg/sdk/baseapp.go#L661) · Missing test

Missing test: the `allow_zero_fee_txs=false` refusal, the validator default, has no test, since every sponsorship test and the integration harness set `AllowZeroFeeTxs` true.

## tm2/pkg/sdk/baseapp.go:1126 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/tm2/pkg/sdk/baseapp.go#L1126) · Missing test

Missing test: nothing asserts that `CheckExecute` discards handler writes from `checkState`. The CheckTx tests stay green when `WriteCheckpoint` here becomes `msCache.MultiWrite()`.

## gno.land/pkg/gnoland/sponsorship_usecase_test.go:316 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/gnoland/sponsorship_usecase_test.go#L316) · Nit

Test: the ADR cites this test for the 6M window floor, but it only logs `gasUsed` at a 30M window, so a cost rise past 6M stays green.

## gno.land/pkg/integration/testdata/paygas_cost_comparison.txtar:32 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/integration/testdata/paygas_cost_comparison.txtar#L32) · Nit

Test: the three calls each assert only `GAS USED:   \d+`, so the PayGas overhead the closing comment describes is never compared and a regression passes.

## gno.land/pkg/integration/testdata/paygas_real_multi_msg.txtar:44 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/integration/testdata/paygas_real_multi_msg.txtar#L44) · Nit

Test: user1's final balance query has no `cmp` against `user1_balance_before`, so the fixture cannot tell who paid user1's share.

## gnovm/cmd/calibrate/gen_native_table.py:187 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gnovm/cmd/calibrate/gen_native_table.py#L187) · Nit

Nit: the payGas and payStorage rows floor the base to the 12-byte sample and then charge those 12 bytes again through the slope, so the shortest path is overcharged.

<details><summary>numbers</summary>

```text
python3 gen_native_table.py sponsorship_bench_m1pro_arm64.txt --no-plot --md-out /dev/null --go-out /dev/null
Base 1097 (ns at N=12); a 12-byte path costs 1097 + 35448*12/1024 = 1512
```

</details>

## tm2/pkg/sdk/baseapp.go:901 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/tm2/pkg/sdk/baseapp.go#L901) · Nit

Nit: `OutOfGasLog` receives the credit window as gas wanted, so a sponsored tx stopped by the lower PayGas limit reports a gas used below its stated gas wanted.

## tm2/pkg/sdk/auth/ante.go:401 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/tm2/pkg/sdk/auth/ante.go#L401) · Suggestion

Suggestion: each sponsored tx reserves the whole credit window in block packing, even when PayGas caps it lower. The ADR could state that this caps sponsored txs per block at `MaxGas` over the window.

## tm2/pkg/sdk/baseapp.go:645 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/tm2/pkg/sdk/baseapp.go#L645) · Suggestion

Suggestion: `CheckTx` decodes the tx before the recover in `runTxWithDecoded`. A recover around this decode would turn a future amino decode panic into `ErrTxDecode`.

## SKIP gno.land/pkg/gnoland/app.go:251 [gh](https://github.com/gnolang/gno/blob/1f043ddb131e54e1410aebabd41d04ed49e37e1e/gno.land/pkg/gnoland/app.go#L251) · Nit

Nit: the comment above `SetBeginTxHook` says the begin and end hooks commit the gno tx stores, which `SetCommitTxHook` does.

Not posted: unverified, on the finder's read, and about a comment's wording.
