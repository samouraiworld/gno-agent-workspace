# Findings in posting order, from round assemble: 7 to post, 6 SKIP, 0 refuted kept out

## Body

- Nit: tm2/pkg/sdk/auth/ante.go:140 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/auth/ante.go#L140), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: removed
  TL;DR: a 0-fee tx running out of gas inside the ante reports the client's gas wanted, often 0, instead of the credit window it actually ran on
  Check: Unit test in tm2/pkg/sdk/auth: consensus params MaxGasCreditPerTx=1000, MaxGas>=1000, height 1; a 0-fee tx with GasWanted 0 large enough to exceed 1000 gas in the ante; assert res.Log contains 'gas wanted (1000)': observe 'gas wanted (0)'.
  Details: For a 0-fee tx the ante sizes the meter to Block.MaxGasCreditPerTx (ante.go:118-122) and its success path reports that window (:399-401), and round 4 made baseapp's OOG recover name the real limit (baseapp.go:915-919), but the ante's own recover still passes tx.Fee.GasWanted to OutOfGasLog (:140) and res.GasWanted (:143). The log tells the user to raise a gas wanted that does not govern a sponsored tx. Log and response field only. Fix: use newCtx.GasMeter().Limit() (or the window when isZeroFeeTx) at ante.go:140 and :143.
  Evidence: read: ante.go:140 `log := store.OutOfGasLog(gasUsed, tx.Fee.GasWanted, maxGas, ex.Descriptor, true)`, :143 `res.GasWanted = tx.Fee.GasWanted`; ante.go:122 `newCtx = SetGasMeter(ctx, consParams.Block.MaxGasCreditPerTx)`; ante.go:399-401 `if isZeroFeeTx { reportedGasWanted = consParams.Block.MaxGasCreditPerTx }`; baseapp.go:917-919 `if l := ctx.GasMeter().Limit(); l > 0 && l < limit { limit = l }`
- Suggestion: gno.land/pkg/gnoland/app.go:982 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/app.go#L982), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: reach
  TL;DR: after a hard fork of a sponsorship-enabled chain, a sponsoring realm's balance differs from the source chain because replay recharges it from the new VM's gas at the genesis price
  Check: In gno.land/pkg/gnoland/replay_report_test.go add a genesis tx carrying Metadata{BlockHeight: 5, ChainID: <past chain>} that is 0-fee and calls a PayGas realm, gasReplayMode source, MaxGasCreditPerTx > 0; compare the realm's post-genesis balance with the source debit ceil(sourceGasUsed x sourcePrice).
  Details: Replay sets header.Height from metadata (app.go:964-965), so runTx marks a replayed 0-fee tx sponsored (baseapp.go:1047); in source mode the ante hands it an infinite meter (app.go:981-982, ante.go:704-705), PayGas leaves it uncapped (paygas.go:55, Limit()==0), and settlement debits ceil(gasUsed x LastGasPrice) where the price is the genesis InitialGasPrice (app.go:174, :937), capped only by MaxFee. A replayed fee-paying tx keeps its declared fee, so only sponsored txs are re-priced. Latent: no source chain with sponsorship exists yet. Fix: skip settlement (or charge a recorded source fee) for replayed txs with Metadata.BlockHeight > 0, or document that sponsored debits are recomputed at replay.
  Evidence: read: app.go:964-965 `if metadata.BlockHeight > 0 { header.Height = metadata.BlockHeight }`; :981-982 `if gasReplayMode == "source" && metadata.BlockHeight > 0 { ctx = ctx.WithValue(auth.SkipGasMeteringKey{}, true) }`; ante.go:704-705 returns NewInfiniteGasMeter; paygas.go:55 `if curLimit := m.GasMeter.Limit(); curLimit > 0`; app.go:293-301 actualCost from gasUsed and gasPrice. Balance comparison not run.

## SKIP gno.land/pkg/gnoland/sponsorship_usecase_test.go:239 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/sponsorship_usecase_test.go#L239) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: the GRC20 cases the ADR cites for its 6M floor run on a two-entry ledger, while the 64-entry avl case already needs 6.22M
Check: In grc20Realm.Setup mint to 64 distinct valid addresses (for example chain.PackageAddress("gno.land/r/h/"+strconv.Itoa(i))) before the Approve, then from gno.land/: go test -count=1 -run 'TestSponsorshipUseCaseFitsWindow/REAL_grc20_transferFrom' -v ./pkg/gnoland/ ; expect CheckTx to OOG at the 6M window or gasUsed above 6,000,000.
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/integration/testdata/paygas_insufficient_balance.txtar:15 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testdata/paygas_insufficient_balance.txtar#L15) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: the insolvent-sponsor txtar says the tx is rejected at admission but asserts only 'insufficient coins', which a delivery failure prints too | The broke-sponsor fixture cannot tell an admission refusal from a delivered tx that fails, so nothing pins CheckExecute's settlement dry run
Check: Mutate tm2/pkg/sdk/baseapp.go:1109 to `if mode == RunTxModeDeliver &&`, run go test -count=1 -p 1 -parallel 1 ./gno.land/pkg/integration/ -run 'TestTestdata/paygas_insufficient_balance$' and go test -count=1 ./tm2/pkg/sdk/: expect red, observe green. Fix: add `stderr 'check transaction failed'` after line 15 (maketx.go:355 wording). | Mutate baseapp.go:1109 to `if mode == RunTxModeDeliver &&`, then `go test -count=1 ./gno.land/pkg/integration -run 'TestTestdata/paygas_insufficient_balance$'` and `go test -count=1 -run 'CheckTx|CheckExecute|EndTxHook|Sponsor' ./tm2/pkg/sdk/ ./gno.land/pkg/gnoland/`: expect red, observe green. Read half: grep -rn SetEndTxHook --include='*_test.go' tm2 gno.land; grep -ln 'check transaction failed' gno.land/pkg/integration/testdata/pay*.txtar.
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/integration/testdata/paygas_oog_after_paygas.txtar:24 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testdata/paygas_oog_after_paygas.txtar#L24) · Missing test
State: PLAUSIBLE, band: Missing test, angle: catalog, on the finder's read only
TL;DR: The OOG-after-PayGas loop also exhausts the 10M credit window, so the fixture stays green if PayGas stops lowering the gas meter to what maxFee buys
Check: Delete paygas.go:61, then `go test -count=1 ./gno.land/pkg/integration -run 'TestTestdata/paygas_(oog_after_paygas|budget_exceeded)$'` and `go test -count=1 -run OutOfGasLogReportsPayGasLimit ./tm2/pkg/sdk/`: expect one red, observe all green. Read half: grep -rn SetLimit --include='*.go' . | grep -v _test.go (only gas meter caller is paygas.go:61); baseapp_test.go:770-771 sets the limit inside the test handler.
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/integration/testdata/paystorage_before_paygas.txtar:35 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testdata/paystorage_before_paygas.txtar#L35) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: The PayStorage-before-PayGas fixture never checks who paid msg 0's deposit, so sponsorship that silently falls back to the caller there passes
Check: Apply the keeper.go:2454 mutation above, run `go test -count=1 ./gno.land/pkg/integration -run 'TestTestdata/(paystorage_before_paygas|paygas_cross_realm_sponsors)$'`: expect a red, observe both green. Read half: grep -n 'user1_user_addr\|bank/balances' paystorage_before_paygas.txtar.
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/sdk/vm/keeper.go:2562 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/sdk/vm/keeper.go#L2562) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: no fixture frees more of the sponsor's storage than it locked in the tx, so the split refund (sponsor share plus a caller remainder sent with released=0) and gnokey's partial SponsorRefund subtraction at root.go:127-133 are unpinned
Check: grep -rn -e SpentDeposit -e SponsorRefund --include='*_test.go' . (only gnovm/cmd/calibrate/native_machine_bench_test.go hits); grep -ln sponsor_refund gno.land/pkg/integration/testdata/*.txtar (only paystorage_sponsor_grow_free_no_drain, 288500 locked vs 287500 freed); then mutate keeper.go:2564 to pass `released` instead of 0, or delete lines 2562-2567, and run go test ./gno.land/pkg/integration/ -run 'TestTestdata/pay(gas|storage)_' : expect a red, observe none
Evidence: not run: no verifier reached it; the check is still to run

## SKIP tm2/pkg/sdk/baseapp.go:795 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L795) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: decodeTx's recover, this round's fix for the round-3 CheckTx decode Suggestion, is pinned by no test: TestCheckTxMalformedTypeURL no longer panics the decoder and asserts only !IsOK
Check: Mutate: remove the defer at baseapp.go:795-799; go test -count=1 ./tm2/pkg/sdk/ -run 'CheckTx|Decode': expect red, observe green. Fix: a test Msg type registered with an UnmarshalAmino that panics, fed to CheckTx, asserting require.IsType(t, std.TxDecodeError{}, res.Error).
Evidence: not run: no verifier reached it; the check is still to run

## gno.land/pkg/integration/testscript_gnoland.go:294 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/integration/testscript_gnoland.go#L294) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the harness guards a MaxGas of -1 it can never produce, suggesting -max-gas -1 is supported when the flag ignores it
Check: sed -n '284,285p;294p' gno.land/pkg/integration/testscript_gnoland.go and sed -n 130p gno.land/pkg/integration/node_testing.go: MaxGas is either 30B or a positive flag value; no other write to Block.MaxGas sits between them
Details: TestingMinimalNodeConfig builds its genesis from DefaultTestingGenesisConfig, whose MaxGas is 30_000_000_000 (node_testing.go:130), and testscript_gnoland.go:284-285 overrides it only when *maxGas > 0 (flag default 0 at :271); nothing else writes Block.MaxGas before :294, so `bp.MaxGas != -1` is always true there. Fix: drop the `bp.MaxGas != -1 &&` conjunct.
Evidence: read: testscript_gnoland.go:284 `if *maxGas > 0 {`, :285 `cfg.Genesis.ConsensusParams.Block.MaxGas = *maxGas`, :294 `if bp.MaxGas != -1 && bp.MaxGas < bp.MaxGasCreditPerTx {`; node_testing.go:130 `MaxGas: 30_000_000_000`; node_testing.go:90 `genesis := DefaultTestingGenesisConfig(gnoroot, pk, tmconfig)`

## contribs/gnogenesis/internal/verify/verify.go:102 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/contribs/gnogenesis/internal/verify/verify.go#L102) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: gnogenesis verify passes a genesis whose replayed 0-fee tx then halts InitChain
Check: Add a case to contribs/gnogenesis/internal/verify/verify_test.go with window 1000 and a 0-fee MsgSend genesis tx with Metadata.BlockHeight 5. Assert that verify errors; expect it to pass instead. Then load the same genesis in an integration node and watch the tx fail.
Details: The new check at verify.go:102-103 fires only when MaxGasCreditPerTx == 0. With the window open, a 0-fee tx carrying Metadata.BlockHeight > 0 runs at that height (app.go:964-965), so runTx marks it sponsored (baseapp.go:1047) and fails it with 'PayGas not called in 0-fee transaction' (:1090-1091) unless a realm calls PayGas; the default GenesisTxResultHandler is PanicOnFailingTxResultHandler (app.go:61), so InitChain stops unless SkipFailingTxs is set. A 0-fee tx without metadata runs at height 0 in Deliver, is not sponsored, and is correctly accepted. Latent until a past chain carries 0-fee txs. Fix: also reject a 0-fee tx with Metadata.BlockHeight > 0 in verify, or narrow the comment to genesis-mode txs.
Evidence: read: verify.go:102-103 `if bp := genesis.ConsensusParams.Block; tx.Tx.Fee.GasFee.IsZero() && (bp == nil || bp.MaxGasCreditPerTx == 0)`; baseapp.go:1047 and :1090-1091; app.go:61 `GenesisTxResultHandler: PanicOnFailingTxResultHandler`. verify_test case not run.

## gno.land/pkg/gnoland/app.go:276 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gno.land/pkg/gnoland/app.go#L276) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: a sponsored tx's settlement transfer is billed to nobody and never counts against block gas, while a fee-paying tx pays gas for its own fee deduction
Check: sed -n '118,124p;250p' tm2/pkg/sdk/auth/ante.go (SetGasMeter, then DeductFees on newCtx) against app.go:275-276 and :309; then swap settleCtx's meter for store.NewGasMeter(math.MaxInt64) in a scratch tree and log its GasConsumed after a sponsored DeliverTx to size the unbilled work.
Details: Settlement reads gasUsed at app.go:275, then runs FeeCollectorAddress and SendCoinsUnrestricted on settleCtx with a fresh store.NewInfiniteGasMeter() (:276, :309), whose consumption reaches neither result.GasUsed nor the block meter, which charges ctx.GasMeter().GasConsumedToLimit() at baseapp.go:967. The ante's DeductFees runs on newCtx (ante.go:250), whose meter is the tx meter set at ante.go:122/124 and which bank stores meter through ctx.GasContext() (context.go:192-205), so the comment's 'like the ante's fee deduction' is wrong. The unbilled work is a fixed handful of account reads and writes per sponsored tx. Fix: either meter settlement on the tx meter after reading gasUsed (or a bounded meter whose consumption is added to block gas), or correct the comment to say settlement is deliberately unbilled.
Evidence: read: app.go:275-276 `gasUsed := ctx.GasMeter().GasConsumed(); settleCtx := ctx.WithGasMeter(store.NewInfiniteGasMeter())`; ante.go:250 `DeductFees(bank, newCtx, ...)` after ante.go:122/124 `newCtx = SetGasMeter(...)`; baseapp.go:967 block meter consumes ctx.GasMeter().GasConsumedToLimit(). Unbilled amount not measured (scratch-tree meter swap not run, read-only judge).

## gnovm/stdlibs/chain/runtime/paygas.go:44 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/gnovm/stdlibs/chain/runtime/paygas.go#L44) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: a realm passing a large maxFee to mean 'cap by the window' passes gno test and fee-paying txs, then every sponsored tx it backs aborts on chain
Check: txtar copied from paygas_basic.txtar with the realm calling runtime.PayGas(9223372036854775807, cur) in a 0-fee tx: expect success capped at the window, observe 'PayGas: maxFee * gas price overflows'
Details: X_payGas returns at paygas.go:29-33 when PayGasInfo is nil or ineligible (execctx.go:101: nil outside a tx; gnovm/pkg/test never sets it), so the overflow.Mul(maxFee, gp.Gas) panic at :44-47 fires only in a 0-fee sponsored tx. The paygas.gno doc says 'Invalid arguments always panic' and that the limit is 'capped by the credit window', so a maxFee whose product overflows is neither rejected everywhere nor capped. Fix: on overflow saturate limit to the meter's current limit (the window) instead of panicking, or reject the overflowing maxFee before the eligibility return so gno test catches it.
Evidence: read: paygas.go:29-33 `if pgi == nil || !pgi.Eligible { return }` precedes :44 `product, ok := overflow.Mul(maxFee, gp.Gas); if !ok { m.Panic("PayGas: maxFee * gas price overflows") }`; paygas.gno:11 'Invalid arguments always panic'; grep PayGasInfo in gnovm/pkg/test gnovm/tests: no hit

## tm2/pkg/sdk/baseapp.go:1109 [gh](https://github.com/gnolang/gno/blob/ba63646a512bdea71d0b1603caa64aec1cbe331b/tm2/pkg/sdk/baseapp.go#L1109) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: a wallet simulating a 0-fee tx is told it succeeds even when the sponsoring realm cannot pay, and the broadcast is then rejected
Check: A tm2/pkg/sdk unit test with a beginTxHook stamping PayGasInfo().MaxFee, an EndTxHook returning an error when MaxFee > 0, MaxGasCreditPerTx > 0 and a 0-fee tx: app.Simulate returns IsOK while app.CheckTx returns the hook's error.
Details: runTx marks the tx sponsored in every non-Deliver mode (baseapp.go:1047) and enforces the PayGas-called check in every mode (:1090), but the settlement hook runs only for Deliver and CheckExecute (:1109-1110), so an insolvent sponsor passes Simulate and fails CheckExecute. The comment above the gate records this as deliberate, to keep Simulate a cost estimate, so it is a documented tradeoff rather than an oversight; the inconsistency is that Simulate already rejects a missing PayGas but not an unpayable one. Fix: run the hook under Simulate too (its writes land in simulate's discarded cache), or document on the simulate endpoint that sponsor solvency is not checked.
Evidence: read: baseapp.go:1047 `sponsored := (ctx.BlockHeight() > 0 || mode != RunTxModeDeliver) && app.isSponsoredTx(tx)`; :1090 `if sponsored && !payGasCalled && result.IsOK()`; :1109 `if (mode == RunTxModeDeliver || mode == RunTxModeCheckExecute) && result.IsOK() && app.endTxHook != nil`; comment :1105-1107 'Simulate is deliberately excluded'
