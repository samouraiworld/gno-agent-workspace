# Claims: gnolang/gno#6484 feat(stdlibs): add crypto/secp256k1.RecoverPubkey round 1, b94116c74, claude-opus-5-5, standard review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`native_gas.go:130`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/native_gas.go#L130) | 1 | Missing test | CONFIRMED | differs | [posted](https://github.com/gnolang/gno/pull/6484#discussion_r4712541366) |
| [`native_gas.go:130`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/native_gas.go#L130) | 1001 | Missing test | CONFIRMED | differs | with row 1 |
| [`recover.gno:32`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/crypto/secp256k1/recover.gno#L32) | 1002 | Suggestion | CONFIRMED | differs | SKIP |

### 1 · Missing test · CONFIRMED · `gnovm/stdlibs/native_gas.go:130`

No test ties the derived recoverPubkey Base to the verify row it is computed from, so a verify re-fit leaves 350,000 stale with every check green.

- Check: set the crypto/secp256k1 verify Base to 300000, go test ./gnovm/stdlibs -run 'TestRecoverPubkey\|TestEveryRow' -timeout 2m: green; add tests/judge-1-recover-row-tracks-verify_test.go: red mutated, green at head
- Observed: mutation verify Base 225693->300000, go test -run 'TestRecoverPubkey\|TestEveryRow\|Mirrors\|TestModExpRow' ./gnovm/stdlibs: ok 0.083s. tests/judge-1-recover-row-tracks-verify_test.go: mutated FAIL 'recoverPubkey Base = 350000, below 1.279 x verify at 64 B (301616) x 1.2 = 462920'; head PASS.
- Test: [`tests/judge-1-recover-row-tracks-verify_test.go`](tests/judge-1-recover-row-tracks-verify_test.go)

### 1001 · Missing test · CONFIRMED · `gnovm/stdlibs/native_gas.go:130`

Same gap as candidate 1: nothing ties the recoverPubkey Base to the verify row it is derived from.

- Check: set the verify row's Base to 300000 and run go test ./gnovm/stdlibs -run 'RecoverPubkey\|EveryRow\|Mirrors' -timeout 2m: green
- Observed: see candidate 1: mutated run green across the existing tests, judge test red mutated and green at head.
- Test: [`tests/judge-1-recover-row-tracks-verify_test.go`](tests/judge-1-recover-row-tracks-verify_test.go)

### 1002 · Suggestion · CONFIRMED · `gnovm/stdlibs/crypto/secp256k1/recover.gno:32`

A recovery ID above 1 is refused only inside the native, after the 350,000 flat charge, where the wrapper could refuse it beside the length check.

- Check: read recover.go:21, native_gas.go:130, op_call.go:612, and the 'recovery ID 2' input in TestRecoverPubkeyFlatCharge
- Observed: read: op_call.go:610-612 chargeNativeGas then nativeBody; recover.go:21 'if recoveryID > 1 {'; secp256k1_recover_test.go:252 'recovery ID 2' case with nativeCharges != 1 failing the test.

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`secp256k1_recover_bench_test.go:41`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/cmd/calibrate/secp256k1_recover_bench_test.go#L41) | claims | The price rests on the dense-s input being near the most point additions decred's GLV and NAF variable-base multiplication can do. A slower input could exceed the 1.2 margin. | A Python run of the secp256k1 GLV split (the splitK constants) and the NAF weight of u2 = s/r: 'slowS (128, 127)' against 's (86, 127)', 'random adds median 86 max of 20000 101 doubl max 128'. 128 additions is the ceiling for two ~127-bit NAFs, so dense-s is the worst case. |
| [`native_gas.go:97`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/native_gas.go#L97) | claims | The '74 entries — exhaustive coverage' header after adding the recoverPubkey row. | comm of the table's rows against generated.go's nativeFuncs: 73 single-line rows plus the multi-line crypto/modexp.modExp row against 74 natives. The only difference is modExp's formatting. |
| [`secp256k1_recover_test.go:37`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/secp256k1_recover_test.go#L37) | lines | The fitter drops the derived recoverPubkey row on regeneration, and a missing row panics chargeNativeGas on the first metered call. | CI catches a dropped row: recoverPubkeyCharge fails with t.Fatal("no gas row for crypto/secp256k1.recoverPubkey"), and the txtar runs a metered call. |
| [`recover_test.go:125`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/crypto/secp256k1/recover_test.go#L125) | claims | The recovery-ID test reads 'recovery-ids-pure-go.json'. go-ethereum's pure-Go recovery accepts IDs 4 and 5, which the native refuses. | recover_test.go:125 compares against ids[seed]["evm_precompile"], which accepts only 0 and 1 (rejected_count 254). The 'recovery' block that accepts 4 and 5 is never asserted. |
| [`recover.go:21`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/crypto/secp256k1/recover.go#L21) | lines | The native's refusal set compared with decred RecoverCompact v4.4.1: does any reachable error path leave out a documented refusal, or panic? | Read of the module cache's ecdsa/signature.go RecoverCompact: the length is fixed, codes 27 and 28 only, R and S overflow and zero, R+N>=P only for codes 2 and 3 (refused first), DecompressY, then the infinity check. That matches the godoc list. The sandbox runs of go test ./stdlibs/crypto/secp256k1 -run TestRecoverPubkey and ./stdlibs -run 'TestRecoverPubkey\|TestNativeWrapperBoundsCopy/secp256k1\|TestEveryRowValidates' print 'ok'. |
| [`vectors.json:1`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/crypto/secp256k1/testdata/vectors.json#L1) | claims | The ADR counts 28 vectors, 19 of them fixed-size. | python3 json count: '28 19'. |
| [`op_call.go:612`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/pkg/gnolang/op_call.go#L612) | reach | the native could start before its flat charge is levied | sed -n 610,616p op_call.go: `gi := m.chargeNativeGas(fv)` precedes `fv.nativeBody(m)`; machine.go:1455 incrCPU calls chargeCPUGas, which runs gm.ConsumeGas immediately (machine.go:1472), so an out-of-gas panic fires before argument conversion and recovery |
| [`native_gas.go:130`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/native_gas.go#L130) | removed | a fitter regeneration drops the derived row and RecoverPubkey then panics 'no calibrated gas entry' on chain | secp256k1_recover_test.go:25 recoverPubkeyCharge ends in t.Fatal("no gas row for crypto/secp256k1.recoverPubkey"), reached by TestRecoverPubkeyGasBoundary and TestRecoverPubkeyFlatCharge in CI |
| [`secp256k1_recover_bench_test.go:1`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/cmd/calibrate/secp256k1_recover_bench_test.go#L1) | reach | the dense-s input might not be the slowest recovery, leaving a constructed input above the flat charge | decred secp256k1/v4@v4.4.1 curve.go ScalarMultNonConst: `k1NAF, k2NAF := naf(k1Bytes[:]), naf(k2Bytes[:])`, a width-2 NAF over the GLV halves, so additions top out at density 1/2 against a mean of 1/3, and ScalarBaseMultNonConst does one table add per byte. The measured dense-s/typical ratio, 1.279/1.089 = 1.17, sits at that ceiling for the u2R share, and the row adds x1.2 on top |
| [`gonative.go:120`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/pkg/gnolang/gonative.go#L120) | reach | the returned [65]byte could come back List-backed, storing as a different shape than a Gno-built key | gonative.go:119-120: a reflect.Array of Uint8 goes to alloc.NewDataArray, the same Data backing a Gno [65]byte has |
| [`native_gas.go:97`](https://github.com/gnolang/gno/blob/b94116c74/gnovm/stdlibs/native_gas.go#L97) | removed | the rewritten '74 entries' count or the 'trailing 10 IBC-crypto entries' could be wrong after the insertion | awk over calibratedNativeGas: 74 `Pkg: "` rows at head (72 at base), 10 after the IBC marker |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Missing test | gnovm/stdlibs/native_gas.go:130 | set the crypto/secp256k1 verify Base to 300000, go test ./gnovm/stdlibs -run 'TestRecoverPubkey\|TestEveryRow' -timeout 2m: green; add tests/judge-1-recover-row-tracks-verify_test.go: red mutated, green at head | hot |  |
| 1001 | CONFIRMED | Missing test | gnovm/stdlibs/native_gas.go:130 | set the verify row's Base to 300000 and run go test ./gnovm/stdlibs -run 'RecoverPubkey\|EveryRow\|Mirrors' -timeout 2m: green | hot |  |
| 1002 | CONFIRMED | Suggestion | gnovm/stdlibs/crypto/secp256k1/recover.gno:32 | read recover.go:21, native_gas.go:130, op_call.go:612, and the 'recovery ID 2' input in TestRecoverPubkeyFlatCharge | warm |  |

</details>

Hit rate per tier, from the record: hot 2/2 confirmed over 5 files, warm 1/1 confirmed over 7 files, cold 0/0 confirmed over 13 files.

## Completeness

Catalog walked against the diff. Determinism: `recover.go` calls decred v4 directly with no build tags and no package state, and `misc/multiarch-determinism/corpus.gno` adds recovery cases. Gas: `doOpCallNativeBody` (op_call.go:610-612) runs `chargeNativeGas` before `fv.nativeBody`, so the flat charge precedes the work; the derived row's missing tie to `verify` is row 1, its mutation run and judge test in `tests/judge-1-recover-row-tracks-verify_test.go`. Error and panic: every `RecoverCompact` error maps to false, and the wrapper refuses wrong lengths before the native. VM-fault: no index or nil path in `recover.gno`, lengths checked before the copies. Realm state, caller identity, coin, storage deposit, twin input paths, older stored data and VM semantics are untouched: the change adds a pure function over fixed arrays.

Sibling sweep: the edited `deepfill_invariant_test.go` list and `native_input_bounds_test.go` table carry both new array parameters and both `RecoverPubkey` slices; no other native in `generated.go` takes an array parameter, and `Verify`'s rows belong to gnolang/gno#5688. The reflector added no candidate.

PR body and threads: no review and no thread on the pull request; the bot's checks summary only. The body's claims (lengths, recovery IDs above 1 refused, high-S accepted, flat 350,000 derived from `verify`) match `recover.gno`, `recover.go` and `native_gas.go:130`.

## Parent checks

- Row 1 re-run by the parent at b94116c74, in the sandbox: `go test -count=1 -timeout 2m ./gnovm/stdlibs/ -run 'TestRecoverPubkeyRowTracksVerify|TestEveryRowValidates'` with the judge's test added prints `ok`; with the `verify` Base set to 300000, `-run 'TestRecoverPubkey|TestEveryRow|Mirrors|TestModExpRow'` fails only `TestRecoverPubkeyRowTracksVerify`, `recoverPubkey Base = 350000, below 1.279 x verify at 64 B (301616) x 1.2 = 462920`.
- `stdlib_secp256k1_recover.txtar` pins `EXACT_GAS=3692706` at line 20 and asserts it on `GAS USED` at line 38, which the overview's Determinism section states.
- tm2's `secp256k1_cgo.go` and `secp256k1_nocgo.go` split on `//go:build libsecp256k1`, which the overview's wrapper section states for `Verify`.

## Retro

- What failed: setup refused `--own-commits` for this stacked pull request, since `b94116c74` edits `native_input_bounds_test.go`, which reached the branch through the master merge `d44697199`; the round reviewed the whole diff against master instead, `Verify` from gnolang/gno#5688 included and scoped out through the notes. The disk held 803 MB before the launch, under the 1024 MB floor, until the user cleared 17 GB of stale caches. No agent died, no cap hit, no refusal; `./scripts/review-retro.py --model claude-opus-5-5` exit 0.
- What worked: the solo-wider shape over a 2,024-line diff whose code is about 220 lines: 12.1 minutes against the 21 projected, $7.42 measured against $7 projected. The gas-path brief produced the one posted finding.
- Hit rate per tier: hot 2/2 over 5 files, warm 1/1 over 7, cold 0/0 over 13.
- Upgrade, proposed and not applied this turn: `--own-commits` in `./scripts/review-setup.sh` could take a merge commit and diff it against its first parent, so a stacked pull request that merged master keeps its own-commit scope; estimate one setup re-run and the parent's scoping notes saved per stacked round.
- Agents: finder 2 at xhigh, 66 turns, 89k output; judge and writer 1 at high, 16 turns, 23k output; 10.0M cache read.
