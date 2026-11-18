// judge-2-dst-vector: the hard-coded vector of bls12381_test.gno, run through
// the Go native. Green at fd7e75152; red once ethDST is mutated to _NUL_,
// so the .gno vector (run in CI by TestStdlibs) pins the DST.
// Run: cp this file gnovm/stdlibs/crypto/bls12381/ && go test -run TestJudgeDSTVector ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"encoding/hex"
	"testing"
)

func TestJudgeDSTVector(t *testing.T) {
	pk, _ := hex.DecodeString("820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68" +
		"b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"ac7563bb556a1687d1dfc2b5fa2d5891c0756ba960f3cfc62de348a5549ddf65121efdbc06050988704674d7820f4a3a")
	sig, _ := hex.DecodeString("857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e")
	if !X_fastAggregateVerify(pk, []byte("gno.land bls12381 test vector"), sig) {
		t.Fatal("vector of bls12381_test.gno does not verify")
	}
}
