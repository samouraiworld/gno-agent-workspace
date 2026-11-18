package bls12381

import (
	"math/big"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/stretchr/testify/require"
)

// keyFrom returns the compressed public key of secret s and s*H(msg).
func keyFrom(tb testing.TB, s int64, msg []byte) (pk []byte, sig bls.G2Affine) {
	tb.Helper()
	g1 := g1Generator()
	h, err := bls.HashToG2(msg, ethDST)
	require.NoError(tb, err)
	var p bls.G1Affine
	p.ScalarMultiplication(&g1, big.NewInt(s))
	b := p.Bytes()
	sig.ScalarMultiplication(&h, big.NewInt(s))
	return b[:], sig
}

// sigBytes returns the compressed sum of points.
func sigBytes(points ...bls.G2Affine) []byte {
	var sum bls.G2Jac
	for i := range points {
		sum.AddMixed(&points[i])
	}
	var a bls.G2Affine
	a.FromJacobian(&sum)
	b := a.Bytes()
	return b[:]
}

// keySum returns the compressed sum of compressed keys.
func keySum(tb testing.TB, keys ...[]byte) []byte {
	tb.Helper()
	var sum bls.G1Jac
	for _, k := range keys {
		var p bls.G1Affine
		_, err := p.SetBytes(k)
		require.NoError(tb, err)
		sum.AddMixed(&p)
	}
	var a bls.G1Affine
	a.FromJacobian(&sum)
	b := a.Bytes()
	return b[:]
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// A: (bls12381_test.go:58) the same key twice in pubkeys.
func TestR2RepeatedKey(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	q, sq := keyFrom(t, 0x89abcdef, msg)
	require.True(t, X_fastAggregateVerify(cat(pk, pk, q), msg, sigBytes(s, s, sq)))
	require.False(t, X_fastAggregateVerify(cat(pk, pk, q), msg, sigBytes(s, sq)))
}

// B: (bls12381_test.go:185) a member twice in the committee, absent twice.
func TestR2RepeatedExcluded(t *testing.T) {
	msg := []byte("signing root")
	k1, s1 := keyFrom(t, 0x1234567, msg)
	k2, _ := keyFrom(t, 0x89abcdef, msg)
	k3, s3 := keyFrom(t, 0x7654321, msg)
	aggregate := keySum(t, k1, k2, k2, k3)
	require.True(t, X_fastAggregateVerifyExcluding(aggregate, cat(k2, k2), msg, sigBytes(s1, s3)))
	require.False(t, X_fastAggregateVerifyExcluding(aggregate, k2, msg, sigBytes(s1, s3)))
}

// C: (bls12381_test.go:76) a trailing byte on the signature or the aggregate.
func TestR2TrailingByte(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	q, sq := keyFrom(t, 0x89abcdef, msg)
	sig := sigBytes(s)
	require.True(t, X_fastAggregateVerify(pk, msg, sig))
	require.False(t, X_fastAggregateVerify(pk, msg, cat(sig, []byte{0})))
	aggregate := keySum(t, pk, q)
	require.True(t, X_fastAggregateVerifyExcluding(aggregate, q, msg, sig))
	require.False(t, X_fastAggregateVerifyExcluding(cat(aggregate, []byte{0}), q, msg, sig))
	require.False(t, X_fastAggregateVerifyExcluding(aggregate, q, msg, cat(sig, []byte{0})))
	_ = sq
}

// D: (bls12381_test.go:104) non-canonical encodings of a key and a signature.
func TestR2NonCanonical(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	sig := sigBytes(s)
	modulus := fp.Modulus().FillBytes(make([]byte, fp.Bytes))

	xIsP := append([]byte{}, modulus...) // x = p, compressed flag set
	xIsP[0] |= 0x80
	infPayload := make([]byte, pubKeySize) // infinity flag with a nonzero x
	infPayload[0], infPayload[pubKeySize-1] = 0xc0, 0x01
	badMask := make([]byte, pubKeySize) // compressed, infinity and sign all set
	badMask[0] = 0xe0
	for name, key := range map[string][]byte{"x = p": xIsP, "infinity payload": infPayload, "invalid mask": badMask} {
		require.False(t, X_fastAggregateVerify(cat(pk, key), msg, sig), name)
	}

	sigXIsP := cat(modulus, make([]byte, fp.Bytes)) // x.A1 = p
	sigXIsP[0] |= 0x80
	sigInfPayload := make([]byte, signatureSize)
	sigInfPayload[0], sigInfPayload[signatureSize-1] = 0xc0, 0x01
	sigBadMask := make([]byte, signatureSize)
	sigBadMask[0] = 0xe0
	for name, bad := range map[string][]byte{"x = p": sigXIsP, "infinity payload": sigInfPayload, "invalid mask": sigBadMask} {
		require.False(t, X_fastAggregateVerify(pk, msg, bad), name)
	}
}

// E: (bls12381_test.go:205) an identity aggregate with -pk_a excluded.
func TestR2IdentityAggregate(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	var p, neg bls.G1Affine
	_, err := p.SetBytes(pk)
	require.NoError(t, err)
	neg.Neg(&p)
	nb := neg.Bytes()
	var inf bls.G1Affine
	ib := inf.Bytes()
	require.False(t, X_fastAggregateVerifyExcluding(ib[:], nb[:], msg, sigBytes(s)))
}
