# Review: [#6484](https://github.com/gnolang/gno/pull/6484)
Posted: https://github.com/gnolang/gno/pull/6484#pullrequestreview-6079360653
Event: APPROVE
Verdict: APPROVE, nothing blocks: the one posted finding asks for a test tying the derived `recoverPubkey` gas row to the `verify` row it comes from.
Model: claude-opus-5-5, standard review; finders at xhigh, judge and writer at high
Commit: b94116c74 (latest)
Overview: [overview](../overview.md)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6484 b94116c74`
Round: 1. 2 finders, one reflector, 14 candidates, 11 of them settled on the finder's own read and the rest judged by an agent that was not their finder.

## Body

This can go in as it is: `RecoverPubkey` [refuses every malformed input](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/stdlibs/crypto/secp256k1/recover.go#L21-L33) with a zero key and false, and the VM [charges its flat 350,000 gas before the Go code runs](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/pkg/gnolang/op_call.go#L611-L612). The test asked for on its gas-table row is optional.

## gnovm/stdlibs/native_gas.go:130 [gh](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/stdlibs/native_gas.go#L130) · [↗](../../../../../.worktrees/gno-review-6484/gnovm/stdlibs/native_gas.go#L130) · Missing test [posted](https://github.com/gnolang/gno/pull/6484#discussion_r4712541366)

Missing test: no test ties this 350,000 charge to the [`verify` row](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/stdlibs/native_gas.go#L129), so raising that row to 300,000 gas leaves recovery at 350,000 where the formula in its comment gives 462,920.

**Fix:** add a floor test in the shape of [`TestModExpRowCoversMeasuredCalibrationPoint`](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/stdlibs/native_gas_test.go#L49), for instance the one below.

<details><summary>test cases</summary>

```go
func TestRecoverPubkeyRowTracksVerify(t *testing.T) {
	t.Parallel()
	var verify, recover *nativeGasEntry
	for i := range calibratedNativeGas {
		e := &calibratedNativeGas[i]
		if e.Pkg == "crypto/secp256k1" && e.Fn == "verify" {
			verify = e
		}
		if e.Pkg == "crypto/secp256k1" && e.Fn == "recoverPubkey" {
			recover = e
		}
	}
	if verify == nil || recover == nil {
		t.Fatal("crypto/secp256k1 verify or recoverPubkey row missing")
	}
	// The slowest recovery measured 1.279x a 64-byte verify
	// (cmd/calibrate/secp256k1_recover_bench_m1pro.txt), with 1.2 headroom.
	verifyAt64 := verify.Base + verify.Slope*64/1024
	floor := verifyAt64 * 1279 * 12 / 10000
	if recover.Base < floor {
		t.Errorf("recoverPubkey Base = %d, below 1.279 x verify at 64 B (%d) x 1.2 = %d: re-derive it from the verify row",
			recover.Base, verifyAt64, floor)
	}
}
```

With the `verify` Base set to 300000, `go test -run 'TestRecoverPubkey|TestEveryRow' ./gnovm/stdlibs` stays green, and the test above fails:

```
recoverPubkey Base = 350000, below 1.279 x verify at 64 B (301616) x 1.2 = 462920: re-derive it from the verify row
FAIL
```

At the head it passes.

</details>

## SKIP gnovm/stdlibs/crypto/secp256k1/recover.gno:32 [gh](https://github.com/gnolang/gno/blob/b94116c7472000c864176ce44489cefa8890205b/gnovm/stdlibs/crypto/secp256k1/recover.gno#L32) · [↗](../../../../../.worktrees/gno-review-6484/gnovm/stdlibs/crypto/secp256k1/recover.gno#L32) · Suggestion

Suggestion: a recovery ID above 1 passes this check and pays the 350,000 flat charge, though a `signature[64] > 1` test here would refuse it for a few VM ops.

**Fix:** for instance, return false on `signature[64] > 1` beside the length check, and expect no native charge for the `recovery ID 2` input of `TestRecoverPubkeyFlatCharge`.

Not posted: the charge falls only on a call that returns false either way, `TestRecoverPubkeyFlatCharge` pins the flat charge on that input on purpose, and an r or s out of range is refused after the charge too.
