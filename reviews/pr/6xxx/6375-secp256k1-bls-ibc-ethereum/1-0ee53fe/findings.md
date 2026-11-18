# Findings in posting order, from round assemble: 5 to post, 0 SKIP, 1 refuted kept out

## gnovm/stdlibs/native_gas.go:196 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L196) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: msg of both bls12381 natives is neither bounded in bls12381.gno nor sloped in its gas row, so gas per call stays flat while the native's hash-to-curve work grows with len(msg)
Check: gno test -v on tests/solo-finder-2-msg-unmetered_test.gno: per-call gas (x5-x1)/4 identical at 32 B, 1 MiB, 16 MiB, 64 MiB; go test -bench BenchmarkMsgGrowth from tests/solo-finder-2-msg-unmetered_test.go
Details: The fastAggregateVerify row slopes on SlopeIdx 0 (pubkeys) and the Excluding row on SlopeIdx 1 (excluded); msg is param 1 and 2 respectively and carries no Slope2. bls12381.gno:37 and :65 check pubkeys, aggregate, excluded and sig lengths only. verifyAggregate passes msg whole to bls.HashToG2 (bls12381.go:123), which runs expand_message_xmd over every byte, and the generated dispatcher copies msg into Go first. adr/native_input_bounds.md:82 requires a variable-length parameter to be bounded in the wrapper or sloped; the wrapper comment at bls12381.gno:35-36 cites that ADR. A realm that builds one large msg (the allocation is near-free in gas: 16 MiB cost 24 gas in the run) and calls in a loop buys native work that grows about 25x from 32 B to 64 MiB at the same gas.
Evidence: judge rerun of tests/solo-finder-2-msg-unmetered_test.gno: 32B x1 4712724 x5 13235880 (2130789 per call); 64MiB x1 4712748 x5 13236382 (2130908 per call). Rerun of BenchmarkMsgGrowth: growth over msg=32 is 1.3x at 1MiB, 7.2x at 16MiB, 25.2x at 64MiB for fastAggregateVerify; 1.3x, 6.8x, 25.0x for Excluding. Merge base has no gnovm/stdlibs/crypto/bls12381 (ls of the base crypto dir). Parent: gno.land/pkg/sdk/vm/keeper.go:50 maxAllocTx = 500_000_000, so a 64 MiB msg fits one transaction. moul measured the msg dimension flat to about 5% at 64 KiB (discussion_r4335346193); the growth starts past that, 1.3x at 1 MiB. Parent reran tests/solo-finder-2-msg-unmetered_test.gno at head: per-call gas identical to the judge run at every size.
Artifact: tests/solo-finder-2-msg-unmetered_test.gno

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:78 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L78) · Missing test
State: CONFIRMED, band: Missing test, angle: claims
TL;DR: no test feeds an on-curve point outside the r-subgroup, or an x >= p encoding, though the godocs and the ADR state both are rejected
Check: grep -n -i 'subgroup\|canonical' gnovm/stdlibs/crypto/bls12381/*_test.*
Details: grep -i 'subgroup|canonical' counts 0 in bls12381_test.go and bls12381_test.gno. The subtests are wrong message, missing signer, malformed lengths, infinity pubkey, identity aggregate, infinity signature, garbage signature (byte 10 xor 0xff, off-curve rather than off-subgroup), and for Excluding wrong excluded set, wrong message, nobody signed, malformed. gnark's SetBytes runs the subgroup check today; nothing pins it.
Evidence: grep -c -i 'subgroup\|canonical' over both test files: 0 and 0; subtest list read at bls12381_test.go:64-104 and 182-206 Parent, after re-anchoring on the test file where the case goes: the draft TestOutsideSubgroup block run at head in a scratch copy of the BLS slice, go test -run TestOutsideSubgroup: --- PASS, ok 0.003s.

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:197 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L197) · Missing test
State: CONFIRMED, band: Missing test, angle: tests
TL;DR: no Excluding test pairs an every-member exclusion with the identity signature, so deleting verifyAggregate's identity guard leaves TestFastAggregateVerifyExcluding green while an arbitrary message verifies
Check: delete bls12381.go:116-118, run go test -run 'TestJudgeExcludingIdentity|TestFastAggregateVerifyExcluding$' with tests/judge-2001-excluding-identity_test.go
Details: 'nobody signed' passes the committee's real sig; with a non-identity signature the pairing fails whatever the guard does. With bls12381.go:116-118 deleted, Excluding(aggregate, every member, "any message", identity signature) returns true.
Evidence: lines 116-118 deleted: --- PASS: TestFastAggregateVerifyExcluding, --- FAIL: TestJudgeExcludingIdentity; at head TestJudgeExcludingIdentity ok
Artifact: tests/judge-2001-excluding-identity_test.go

## gnovm/adr/pr6375_bls12381_native.md:72 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/adr/pr6375_bls12381_native.md#L72) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: the ADR's 'about 47M gas' for 512 keys is the slope term alone; the row charges 49.1M with its base
Check: gno test -v on tests/judge-r1-gas-512_test.gno
Details: 512 keys are 24576 bytes: 1962000*24576/1024 = 47088000, plus Base 2024000 = 49112000. The measured charge matches the formula.
Evidence: tests/judge-r1-gas-512_test.gno: TestKeys512 GAS 49486865 minus TestKeys0 GAS 360230 = 49126635
Artifact: tests/judge-r1-gas-512_test.gno

## gnovm/stdlibs/native_gas.go:195 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L195) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: the bls12381 rows say to regenerate with the rest of the table, but the calibrate benches and gen_native_table.py know no bls12381 native, so a regeneration cannot emit either row
Check: grep -rln -i 'bls12381\|bls12-381' gnovm/cmd/calibrate/ at 0ee53fefc
Details: grep -rln 'bls12381|bls12-381' gnovm/cmd/calibrate/ returns no file at 0ee53fefc; gen_native_table.py's IBC specs (217-225) list bn254 and cometbls, and emit_go_table writes calibratedNativeGas from the fitted rows of native_bench_output.txt alone. The same table comment already records that rows with no samples (chain/params Get*) are dropped by a re-run. The bls12381 rows are the only IBC rows with no bench behind them.
Evidence: grep -rln -i 'bls12381\|bls12-381' gnovm/cmd/calibrate/: no output; grep -c -i bls hits are cometbls only; gen_native_table.py:521-530 emit_go_table header 'Code generated by gen_native_table.py from native_bench_output.txt'
