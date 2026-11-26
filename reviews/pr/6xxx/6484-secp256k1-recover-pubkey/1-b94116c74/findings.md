# Findings in posting order, from round assemble: 2 to post, 0 SKIP, 0 refuted kept out

## gnovm/stdlibs/native_gas.go:130 [gh](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/stdlibs/native_gas.go#L130) · Missing test
State: CONFIRMED, band: Missing test, angle: claims
TL;DR: No test ties the derived recoverPubkey Base to the verify row it is computed from, so a verify re-fit leaves 350,000 stale with every check green.
Check: set the crypto/secp256k1 verify Base to 300000, go test ./gnovm/stdlibs -run 'TestRecoverPubkey|TestEveryRow' -timeout 2m: green; add tests/judge-1-recover-row-tracks-verify_test.go: red mutated, green at head
Details: The row comment derives Base 350000 as 1.279 x verify at 64 B (227,309) x 1.2 = 348,874, and the verify row above it is marked PRELIMINARY, to be re-fit before merge. recoverPubkeyCharge in secp256k1_recover_test.go reads only the recover row and checks that it is flat; native_gas_test.go has no secp256k1 test; the txtar's EXACT_GAS calls only Recover. Raising the verify Base to 300000 leaves TestRecoverPubkey*, TestEveryRow*, Mirrors and TestModExpRow* green. A floor test in the shape of TestModExpRowCoversMeasuredCalibrationPoint goes red on that mutation (want 462,920, have 350,000) and green at the head.
Evidence: mutation verify Base 225693->300000, go test -run 'TestRecoverPubkey|TestEveryRow|Mirrors|TestModExpRow' ./gnovm/stdlibs: ok 0.083s. tests/judge-1-recover-row-tracks-verify_test.go: mutated FAIL 'recoverPubkey Base = 350000, below 1.279 x verify at 64 B (301616) x 1.2 = 462920'; head PASS.
Artifact: tests/judge-1-recover-row-tracks-verify_test.go
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: Same gap as candidate 1: nothing ties the recoverPubkey Base to the verify row it is derived from.
Check: set the verify row's Base to 300000 and run go test ./gnovm/stdlibs -run 'RecoverPubkey|EveryRow|Mirrors' -timeout 2m: green
Details: One defect with candidate 1, one edit closes both; carried by row 1. Same mutation and the same judge test settle it.
Evidence: see candidate 1: mutated run green across the existing tests, judge test red mutated and green at head.
Artifact: tests/judge-1-recover-row-tracks-verify_test.go

## gnovm/stdlibs/crypto/secp256k1/recover.gno:32 [gh](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/stdlibs/crypto/secp256k1/recover.gno#L32) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: A recovery ID above 1 is refused only inside the native, after the 350,000 flat charge, where the wrapper could refuse it beside the length check.
Check: read recover.go:21, native_gas.go:130, op_call.go:612, and the 'recovery ID 2' input in TestRecoverPubkeyFlatCharge
Details: doOpCallNativeBody (op_call.go:610-612) charges the row before fv.nativeBody runs; X_recoverPubkey refuses recoveryID > 1 as its first statement (recover.go:21), measured at 621 ns (RefuseID, secp256k1_recover_bench_m1pro.txt:46). TestRecoverPubkeyFlatCharge pins nativeCharges == 1 for the 'recovery ID 2' input, so the flat charge on that input is deliberate. The cost falls only on a caller whose call returns false either way, and r or s out of range is refused after the charge too, so a wrapper check would cover one refusal of several.
Evidence: read: op_call.go:610-612 chargeNativeGas then nativeBody; recover.go:21 'if recoveryID > 1 {'; secp256k1_recover_test.go:252 'recovery ID 2' case with nativeCharges != 1 failing the test.
