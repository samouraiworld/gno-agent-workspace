// judge-2001-excluding-identity: FastAggregateVerifyExcluding with every
// member excluded and the identity signature must fail for any message.
// Green at fd7e75152; red once verifyAggregate's pk.IsInfinity() guard is
// deleted, while the PR's own TestFastAggregateVerifyExcluding stays green.
// Run: cp this file gnovm/stdlibs/crypto/bls12381/ && go test -run TestJudgeExcludingIdentity ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

func TestJudgeExcludingIdentity(t *testing.T) {
	msg := make([]byte, 32)
	pubkeys, aggregate, _ := committee(t, 4, msg, func(int) bool { return true })
	var inf bls.G2Affine
	infSig := inf.Bytes()
	var all []byte
	for _, pk := range pubkeys {
		all = append(all, pk...)
	}
	if X_fastAggregateVerifyExcluding(aggregate, all, []byte("any message"), infSig[:]) {
		t.Fatal("every member excluded plus the identity signature verifies an arbitrary message")
	}
}
