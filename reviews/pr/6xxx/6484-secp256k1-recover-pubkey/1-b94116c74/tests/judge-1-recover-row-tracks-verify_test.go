// Asserts the derived recoverPubkey row still covers 1.279 x the verify row at
// 64 bytes x 1.2, the derivation its comment states. Green at b94116c74; red
// once the verify row is re-fit upward (Base 225693 -> 300000) and recover is
// not re-derived, which every check in the branch leaves green.
package stdlibs

import "testing"

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
