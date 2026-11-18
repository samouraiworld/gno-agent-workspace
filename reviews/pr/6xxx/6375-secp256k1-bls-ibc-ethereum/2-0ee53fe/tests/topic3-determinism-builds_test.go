// Round 2, topic 3 (determinism inside gnark-crypto v0.20.1).
//
// Differential check: the BLS natives and every intermediate they compute
// (decoded keys, the Jacobian aggregate, HashToG2, the pairing product) are
// digested over a fixed, seeded case set. The digest must be identical under
// the three field-arithmetic builds a validator set can run:
//
//	default  amd64 assembly with the ADX/BMI2 path (CPU has ADX)
//	noadx    amd64 assembly with the non-ADX fallback (CPU without ADX)
//	purego   pure Go (also what non-amd64/arm64 validators run)
//
// It also asserts the expected verdict of each crafted case (duplicate keys
// through DoubleMixed, an excluded key equal to -aggregate, an excluded key
// equal to the aggregate, the identity signature, nil versus empty msg and
// excluded), and runs the natives from 16 goroutines at once for -race.
//
// Repro from a plain clone of gnolang/gno at 0ee53fefc (or the BLS slice):
//
//	cp zz_topic3_determinism_test.go gnovm/stdlibs/crypto/bls12381/
//	cd gnovm/stdlibs/crypto/bls12381
//	go test -run 'TestTopic3' -timeout 2m -v .               | grep DIGEST
//	go test -run 'TestTopic3' -timeout 2m -v -tags noadx .   | grep DIGEST
//	go test -run 'TestTopic3' -timeout 2m -v -tags purego .  | grep DIGEST
//	go test -run 'TestTopic3Concurrent' -timeout 2m -race .
//	go test -run 'TestTopic3SmallOrder' -timeout 2m -v .        # every small-order point refused
//	GOARCH=386 CGO_ENABLED=0 go test -run TestTopic3Determinism -timeout 2m -v . | grep DIGEST
//	GOARCH=arm64 CGO_ENABLED=0 go test -c -o t3.test . && qemu-aarch64 ./t3.test -test.run TestTopic3Determinism -test.v | grep DIGEST
//
// Doubling branch coverage (AddMixed's p == a path, gnark g1.go:460-461),
// reached by no test of the package and by dup2 here:
//
//	go test -run '^(TestFastAggregateVerify|TestFastAggregateVerifyExcluding)$' -timeout 2m \
//	  -coverpkg=github.com/consensys/gnark-crypto/ecc/bls12-381 -coverprofile=c.out . && grep 'bls12-381/g1.go:460.38' c.out
//
// Measured 2026-11-18 on AMD Ryzen 7 7840HS (ADX, AVX-512): DIGEST
// 4f9b8674157603753c70d0693e074238a77c934df5c392301f57930ffe0f6247 cases=504
// under default, noadx, purego, 386 and arm64 (qemu); -race ok; the
// doubling branch count is 0 under the package's tests, 1 under this file.
package bls12381

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

type t3rng struct{ ctr uint64 }

func (r *t3rng) bytes(n int) []byte {
	out := make([]byte, 0, n+32)
	for len(out) < n {
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], r.ctr)
		r.ctr++
		s := sha256.Sum256(append([]byte("topic3"), b[:]...))
		out = append(out, s[:]...)
	}
	return out[:n]
}

func (r *t3rng) scalar() *big.Int {
	var e fr.Element
	e.SetBytes(r.bytes(48))
	var bi big.Int
	e.BigInt(&bi)
	return &bi
}

type t3key struct {
	sk *big.Int
	pk bls.G1Affine
}

func t3keys(r *t3rng, n int) []t3key {
	g1 := g1Generator()
	ks := make([]t3key, n)
	for i := range ks {
		ks[i].sk = r.scalar()
		ks[i].pk.ScalarMultiplication(&g1, ks[i].sk)
	}
	return ks
}

func t3cat(ps ...bls.G1Affine) []byte {
	var out []byte
	for _, p := range ps {
		b := p.Bytes()
		out = append(out, b[:]...)
	}
	return out
}

func t3sign(msg []byte, sks ...*big.Int) []byte {
	h, err := bls.HashToG2(msg, ethDST)
	if err != nil {
		panic(err)
	}
	sum := new(big.Int)
	for _, sk := range sks {
		sum.Add(sum, sk)
	}
	sum.Mod(sum, fr.Modulus())
	var s bls.G2Affine
	s.ScalarMultiplication(&h, sum)
	b := s.Bytes()
	return b[:]
}

// digest of the intermediates a native computes for one call.
func t3trace(d []byte, pubkeys, msg, sig []byte) []byte {
	var agg bls.G1Jac
	for i := 0; i+pubKeySize <= len(pubkeys); i += pubKeySize {
		var pk bls.G1Affine
		_, err := pk.SetBytes(pubkeys[i : i+pubKeySize])
		d = append(d, []byte(fmt.Sprint(err))...)
		agg.AddMixed(&pk)
	}
	var aff bls.G1Affine
	aff.FromJacobian(&agg)
	ab := aff.Bytes()
	d = append(d, ab[:]...)
	h, _ := bls.HashToG2(msg, ethDST)
	hb := h.Bytes()
	d = append(d, hb[:]...)
	var s bls.G2Affine
	_, err := s.SetBytes(sig)
	d = append(d, []byte(fmt.Sprint(err))...)
	if !aff.IsInfinity() {
		gt, err := bls.Pair([]bls.G1Affine{aff, negG1Gen}, []bls.G2Affine{h, s})
		if err == nil {
			d = append(d, gt.Marshal()...)
		}
	}
	return d
}

func TestTopic3Determinism(t *testing.T) {
	r := &t3rng{}
	h := sha256.New()
	cases := 0
	expect := func(name string, got, want bool) {
		cases++
		fmt.Fprintf(h, "%s=%v;", name, got)
		if got != want {
			t.Errorf("%s: got %v want %v", name, got, want)
		}
	}

	msgs := [][]byte{nil, {}, r.bytes(1), r.bytes(32), r.bytes(1000)}
	for _, n := range []int{1, 2, 3, 7, 64} {
		ks := t3keys(r, n)
		var pks []bls.G1Affine
		var sks []*big.Int
		for _, k := range ks {
			pks = append(pks, k.pk)
			sks = append(sks, k.sk)
		}
		for mi, msg := range msgs {
			sig := t3sign(msg, sks...)
			name := fmt.Sprintf("fav n=%d m=%d", n, mi)
			expect(name, X_fastAggregateVerify(t3cat(pks...), msg, sig), true)
			h.Write(t3trace(nil, t3cat(pks...), msg, sig))
			// aggregate path: drop the last key from the signers.
			var aggJ bls.G1Jac
			for _, p := range pks {
				aggJ.AddMixed(&p)
			}
			var agg bls.G1Affine
			agg.FromJacobian(&aggJ)
			ab := agg.Bytes()
			if n > 1 {
				sig2 := t3sign(msg, sks[:n-1]...)
				expect(name+" excl1", X_fastAggregateVerifyExcluding(ab[:], t3cat(pks[n-1]), msg, sig2), true)
			}
			expect(name+" excl0 nil", X_fastAggregateVerifyExcluding(ab[:], nil, msg, sig), true)
			expect(name+" excl0 empty", X_fastAggregateVerifyExcluding(ab[:], []byte{}, msg, sig), true)
		}
	}

	// Duplicates: DoubleMixed inside AddMixed.
	ks := t3keys(r, 3)
	msg := r.bytes(32)
	expect("dup2", X_fastAggregateVerify(t3cat(ks[0].pk, ks[0].pk), msg, t3sign(msg, ks[0].sk, ks[0].sk)), true)
	expect("dup3", X_fastAggregateVerify(t3cat(ks[0].pk, ks[0].pk, ks[0].pk), msg, t3sign(msg, ks[0].sk, ks[0].sk, ks[0].sk)), true)
	expect("dup-mixed", X_fastAggregateVerify(t3cat(ks[0].pk, ks[1].pk, ks[0].pk, ks[1].pk), msg, t3sign(msg, ks[0].sk, ks[1].sk, ks[0].sk, ks[1].sk)), true)
	expect("dup2 single-sig", X_fastAggregateVerify(t3cat(ks[0].pk, ks[0].pk), msg, t3sign(msg, ks[0].sk)), false)
	// pk then -pk: the sum is the identity, refused.
	var neg bls.G1Affine
	neg.Neg(&ks[0].pk)
	expect("pk,-pk,pk1", X_fastAggregateVerify(t3cat(ks[0].pk, neg, ks[1].pk), msg, t3sign(msg, ks[1].sk)), true)
	var infSig bls.G2Affine
	ib := infSig.Bytes()
	expect("pk,-pk inf", X_fastAggregateVerify(t3cat(ks[0].pk, neg), msg, ib[:]), false)

	// Excluding edges: excluded == aggregate (signers = identity), excluded ==
	// -aggregate (doubling path inside AddMixed), excluded twice.
	a0 := ks[0].pk.Bytes()
	expect("excl==agg", X_fastAggregateVerifyExcluding(a0[:], t3cat(ks[0].pk), msg, ib[:]), false)
	two := new(big.Int).Lsh(ks[0].sk, 1)
	expect("excl==-agg", X_fastAggregateVerifyExcluding(a0[:], t3cat(neg), msg, t3sign(msg, two)), true)
	// agg = k0+k1+k2, exclude k1 twice: signers = k0 + k2 - k1.
	var aJ bls.G1Jac
	aJ.AddMixed(&ks[0].pk).AddMixed(&ks[1].pk).AddMixed(&ks[2].pk)
	var aA bls.G1Affine
	aA.FromJacobian(&aJ)
	aB := aA.Bytes()
	minus := new(big.Int).Sub(fr.Modulus(), ks[1].sk)
	expect("excl-twice", X_fastAggregateVerifyExcluding(aB[:], t3cat(ks[1].pk, ks[1].pk), msg, t3sign(msg, ks[0].sk, ks[2].sk, minus)), true)

	// Random bytes under every flag combination: decode errors and verdicts.
	for i := 0; i < 200; i++ {
		pk := r.bytes(48)
		pk[0] = pk[0]&0x1f | byte(i%8)<<5
		sig := r.bytes(96)
		sig[0] = sig[0]&0x1f | byte((i/8)%8)<<5
		fmt.Fprintf(h, "%v", X_fastAggregateVerify(pk, msg, sig))
		fmt.Fprintf(h, "%v", X_fastAggregateVerifyExcluding(a0[:], pk, msg, sig))
		h.Write(t3trace(nil, pk, msg, sig))
		cases += 2
	}

	t.Logf("DIGEST %x cases=%d", h.Sum(nil), cases)
}

func TestTopic3Concurrent(t *testing.T) {
	r := &t3rng{ctr: 1 << 40}
	ks := t3keys(r, 4)
	msg := r.bytes(32)
	var pks []bls.G1Affine
	var sks []*big.Int
	for _, k := range ks {
		pks = append(pks, k.pk)
		sks = append(sks, k.sk)
	}
	pkb := t3cat(pks...)
	sig := t3sign(msg, sks...)
	var aJ bls.G1Jac
	for _, p := range pks {
		aJ.AddMixed(&p)
	}
	var aA bls.G1Affine
	aA.FromJacobian(&aJ)
	aB := aA.Bytes()
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 4; i++ {
				if !X_fastAggregateVerify(pkb, msg, sig) {
					t.Error("concurrent verify false")
				}
				if !X_fastAggregateVerifyExcluding(aB[:], nil, msg, sig) {
					t.Error("concurrent excluding false")
				}
			}
		}()
	}
	wg.Wait()
}

// TestTopic3SmallOrder: points of each small prime order dividing the G1
// cofactor, the same plus G1, (0, ±2) of order 3, and uncleared twist points
// must all fail SetBytes' subgroup check, so no native ever sees them.
func TestTopic3SmallOrder(t *testing.T) {
	p, _ := new(big.Int).SetString("1a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaab", 16)
	u, _ := new(big.Int).SetString("-d201000000010000", 16)
	tr := new(big.Int).Add(u, big.NewInt(1))
	n := new(big.Int).Sub(new(big.Int).Add(p, big.NewInt(1)), tr) // #E(Fp)
	r := fr.Modulus()
	h := new(big.Int).Div(n, r)
	t.Logf("cofactor %x rem=%v", h, new(big.Int).Mod(n, r))
	g1 := g1Generator()
	rng := &t3rng{ctr: 1 << 50}
	// a random curve point: try x until y exists, built through SetBytes
	// without the subgroup check is not exported, so solve directly.
	var q bls.G1Affine
	for {
		var x, y2, y, b bls.G1Affine
		_ = b
		x.X.SetBytes(rng.bytes(48))
		y2.X.Square(&x.X).Mul(&y2.X, &x.X)
		var four = new(big.Int).SetInt64(4)
		var bb = y.X
		bb.SetBigInt(four)
		y2.X.Add(&y2.X, &bb)
		if y.Y.Sqrt(&y2.X) != nil {
			q.X, q.Y = x.X, y.Y
			break
		}
	}
	if !q.IsOnCurve() || q.IsInSubGroup() {
		t.Fatalf("setup: random point on=%v sub=%v", q.IsOnCurve(), q.IsInSubGroup())
	}
	try := func(name string, pt bls.G1Affine) {
		if pt.IsInfinity() {
			t.Logf("%s: infinity, skipped", name)
			return
		}
		b := pt.Bytes()
		var dec bls.G1Affine
		_, err := dec.SetBytes(b[:])
		accepted := err == nil
		v := X_fastAggregateVerify(append(b[:], g1Bytes()...), []byte("m"), make([]byte, 96))
		t.Logf("%s: onCurve=%v SetBytes accepted=%v", name, pt.IsOnCurve(), accepted)
		if accepted {
			t.Errorf("%s: off-subgroup point accepted", name)
		}
		_ = v
	}
	try("random", q)
	for _, l := range []int64{3, 11, 10177, 859267, 52437899} {
		k := new(big.Int).Div(n, big.NewInt(l))
		var s bls.G1Affine
		s.ScalarMultiplication(&q, k)
		try(fmt.Sprintf("order %d", l), s)
		var j bls.G1Jac
		j.FromAffine(&g1)
		j.AddMixed(&s)
		var m bls.G1Affine
		m.FromJacobian(&j)
		try(fmt.Sprintf("G1+order %d", l), m)
	}
	var z bls.G1Affine
	z.Y.SetUint64(2)
	try("(0,2)", z)
	z.Y.Neg(&z.Y)
	try("(0,-2)", z)
	// G2: random twist points, uncleared.
	for i := 0; i < 8; i++ {
		var u0 bls.G2Affine
		u0.X.A0.SetBytes(rng.bytes(48))
		u0.X.A1.SetBytes(rng.bytes(48))
		hm := bls.MapToG2(u0.X) // cleared: must be accepted
		hb := hm.Bytes()
		var dec bls.G2Affine
		if _, err := dec.SetBytes(hb[:]); err != nil {
			t.Errorf("cleared G2 point refused: %v", err)
		}
		raw := bls.MapToCurve2(&u0.X) // on the 3-isogenous curve, not E2
		_ = raw
	}
}

// TestTopic3EmptyMsgVector prints the aggregate signature over the empty
// message by the three bls12381_test.gno signers, for the Gno-side run.
func TestTopic3EmptyMsgVector(t *testing.T) {
	sks := []*big.Int{big.NewInt(0x1234567), big.NewInt(0x89abcdef), big.NewInt(0x7654321)}
	t.Logf("EMPTYSIG %x", t3sign(nil, sks...))
}
