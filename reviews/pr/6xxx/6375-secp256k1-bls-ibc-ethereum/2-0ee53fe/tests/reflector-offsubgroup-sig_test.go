// Reflector, round 2 on gnolang/gno#6375 head 0ee53fefc: off-subgroup G2
// signature and off-subgroup G1 aggregate, the two decodes round 1's
// judge-1-subgroup-cases_test.go did not feed (it covered pubkeys and excluded).
//
// Run: copy beside gnovm/stdlibs/crypto/bls12381/bls12381.go and the PR's
// bls12381_test.go (it uses signers()), then
//   go test -count=1 -timeout 2m -run 'TestReflectorOffSubgroup' -v .
// Mutation: replace `s.SetBytes(sig)` in verifyAggregate with
//   bls.NewDecoder(bytes.NewReader(sig), bls.NoSubgroupChecks()).Decode(&s)
// and rerun both this test and the PR's TestFastAggregateVerify.
//
// Observed at fd7e75152 (go1.25.9, gnark-crypto v0.20.1): PASS at head, and
// PASS again under the mutation, the PR's tests included. SetBytes refuses
// the point ("invalid point: subgroup check failed"), and sig+T with
// T = [r]Q fails the pairing equation before any decode, so the signature's
// subgroup check is not observable through the verdict for this shape. The
// off-subgroup aggregate is refused too. Cleared, not filed.
package bls12381

import (
	"crypto/rand"
	"math/big"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

// offSubgroupG2 returns a point of the twist E2: y^2 = x^3 + 4(1+i) that is
// on the curve and outside the r-order subgroup.
func offSubgroupG2(t *testing.T) bls.G2Affine {
	t.Helper()
	var p bls.G2Affine
	for i := uint64(1); i < 1000; i++ {
		p.X.A0.SetUint64(i)
		p.X.A1.SetUint64(1)
		rhs := p.X
		rhs.Square(&rhs).Mul(&rhs, &p.X)
		b := p.X
		b.A0.SetUint64(4)
		b.A1.SetUint64(4)
		rhs.Add(&rhs, &b)
		if rhs.Legendre() != 1 {
			continue
		}
		p.Y.Sqrt(&rhs)
		if p.IsOnCurve() && !p.IsInSubGroup() {
			return p
		}
	}
	t.Fatal("no off-subgroup point found")
	return p
}

// mulPlain is double-and-add with no endomorphism, valid off the subgroup.
func mulPlain(q *bls.G2Affine, k *big.Int) bls.G2Affine {
	var acc, base bls.G2Jac
	base.FromAffine(q)
	for i := k.BitLen() - 1; i >= 0; i-- {
		acc.DoubleAssign()
		if k.Bit(i) == 1 {
			acc.AddAssign(&base)
		}
	}
	var r bls.G2Affine
	r.FromJacobian(&acc)
	return r
}

func TestReflectorOffSubgroup(t *testing.T) {
	msg := make([]byte, 32)
	_, _ = rand.Read(msg)
	pubkeys, sig := signers(t, 4, msg)

	q := offSubgroupG2(t)
	qb := q.Bytes()

	// T = [r]Q: a nonzero torsion point outside G2.
	tor := mulPlain(&q, fr.Modulus())
	if tor.IsInfinity() || !tor.IsOnCurve() || tor.IsInSubGroup() {
		t.Fatalf("torsion point: inf=%v oncurve=%v insub=%v", tor.IsInfinity(), tor.IsOnCurve(), tor.IsInSubGroup())
	}
	var s bls.G2Affine
	if _, err := s.SetBytes(sig); err != nil {
		t.Fatal(err)
	}
	var sj bls.G2Jac
	sj.FromAffine(&s)
	sj.AddMixed(&tor)
	var sT bls.G2Affine
	sT.FromJacobian(&sj)
	sTb := sT.Bytes()

	// Pairing level, no decode: does sig+T satisfy the equation?
	var aggJ bls.G1Jac
	for i := 0; i < len(pubkeys); i += pubKeySize {
		var pk bls.G1Affine
		_, _ = pk.SetBytes(pubkeys[i : i+pubKeySize])
		aggJ.AddMixed(&pk)
	}
	var agg bls.G1Affine
	agg.FromJacobian(&aggJ)
	h, _ := bls.HashToG2(msg, ethDST)
	okT, errT := bls.PairingCheck([]bls.G1Affine{agg, negG1Gen}, []bls.G2Affine{h, sT})
	t.Logf("pairing equation with sig+T (no decode): ok=%v err=%v", okT, errT)

	var dq bls.G2Affine
	_, errQ := dq.SetBytes(qb[:])
	t.Logf("SetBytes(off-subgroup G2): err=%v", errQ)

	// G1 point (0, 2): on y^2 = x^3 + 4, order 3, outside G1.
	var g bls.G1Affine
	g.X.SetZero()
	g.Y.SetUint64(2)
	if !g.IsOnCurve() || g.IsInSubGroup() {
		t.Fatalf("(0,2): oncurve=%v insub=%v", g.IsOnCurve(), g.IsInSubGroup())
	}
	gb := g.Bytes()

	gotQ := X_fastAggregateVerify(pubkeys, msg, qb[:])
	gotT := X_fastAggregateVerify(pubkeys, msg, sTb[:])
	gotAgg := X_fastAggregateVerifyExcluding(gb[:], nil, msg, sig)
	t.Logf("verify(off-subgroup sig)=%v verify(sig+T)=%v excluding(off-subgroup aggregate)=%v", gotQ, gotT, gotAgg)
	if gotQ || gotT || gotAgg {
		t.Fatal("an off-subgroup point was accepted")
	}
}
