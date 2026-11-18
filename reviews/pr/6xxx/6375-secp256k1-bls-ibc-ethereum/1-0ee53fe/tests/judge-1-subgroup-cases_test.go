// judge-1-subgroup-cases: the test cases the Missing test finding on
// bls12381.go:50 proposes, run at fd7e75152: an on-curve G1 point outside
// the prime-order subgroup, as a key and as an excluded key, must not verify.
// Run: cp this file gnovm/stdlibs/crypto/bls12381/ && go test -run TestJudgeSubgroup ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/stretchr/testify/require"
)

// offSubgroupG1 returns the encoding of a point on y^2 = x^3 + 4 that is not
// in the r-order subgroup.
func offSubgroupG1(t *testing.T) []byte {
	var p bls.G1Affine
	var rhs, four fp.Element
	four.SetUint64(4)
	for i := uint64(1); ; i++ {
		p.X.SetUint64(i)
		rhs.Square(&p.X).Mul(&rhs, &p.X).Add(&rhs, &four)
		if p.Y.Sqrt(&rhs) == nil {
			continue
		}
		require.True(t, p.IsOnCurve())
		if !p.IsInSubGroup() {
			b := p.Bytes()
			return b[:]
		}
	}
}

func TestJudgeSubgroup(t *testing.T) {
	msg := make([]byte, 32)
	pubkeys, aggregate, sig := committee(t, 4, msg, func(int) bool { return true })
	bad := offSubgroupG1(t)
	var pk bls.G1Affine
	_, err := pk.SetBytes(bad)
	require.Error(t, err)
	require.False(t, X_fastAggregateVerify(append(append([]byte{}, pubkeys[0]...), bad...), msg, sig))
	require.False(t, X_fastAggregateVerifyExcluding(aggregate, bad, msg, sig))
	t.Logf("off-subgroup key %x refused: %v", bad[:8], err)
}
