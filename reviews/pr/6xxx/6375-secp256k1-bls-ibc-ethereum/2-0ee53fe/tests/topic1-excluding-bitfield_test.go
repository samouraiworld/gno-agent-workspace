// topic1-excluding-bitfield: FastAggregateVerifyExcluding with both
// obligations its doc states (bls12381.gno:50-53) met, every excluded key a
// proven committee member and every member's possession proven, still
// verifies a message the committee never signed when the participation the
// caller thresholds on comes from the update's sync_committee_bits, as in
// Ethereum's validate_light_client_update, rather than from excluded.
//
// One committee member (sync committee members are ordinary validators)
// signs a forged header alone, sets every participation bit, and lists the
// other 15 members, real members with valid Merkle branches, as excluded.
// The doc names no obligation binding excluded to the clear bits, so a realm
// that meets the two it names passes the 2/3 threshold on 16 of 16 bits.
//
// Run, from a plain clone of gnolang/gno at 0ee53fefc:
//
//	cp topic1-excluding-bitfield_test.go gnovm/stdlibs/crypto/bls12381/
//	go test -count=1 -timeout 2m -run TestTopic1ExcludingBitfield -v ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"math/big"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

func TestTopic1ExcludingBitfield(t *testing.T) {
	const n = 16
	g1 := g1Generator()
	var keys [][]byte
	var sks []big.Int
	var aggJ bls.G1Jac
	for range n {
		var sk fr.Element
		if _, err := sk.SetRandom(); err != nil {
			t.Fatal(err)
		}
		var skInt big.Int
		sk.BigInt(&skInt)
		var pk bls.G1Affine
		pk.ScalarMultiplication(&g1, &skInt)
		b := pk.Bytes()
		keys = append(keys, b[:])
		sks = append(sks, skInt)
		aggJ.AddMixed(&pk)
	}
	var agg bls.G1Affine
	agg.FromJacobian(&aggJ)
	aggregate := agg.Bytes()

	// The committee's membership set, standing in for the Merkle branch of
	// each key against the committee's pubkeys root.
	member := map[string]bool{}
	for _, k := range keys {
		member[string(k)] = true
	}

	// Member 0 alone signs a header the committee never saw.
	forged := []byte("forged attested header signing root")
	h, err := bls.HashToG2(forged, ethDST)
	if err != nil {
		t.Fatal(err)
	}
	var s bls.G2Affine
	s.ScalarMultiplication(&h, &sks[0])
	sig := s.Bytes()

	// The update: every bit set, the other members listed as excluded.
	bits := make([]bool, n)
	for i := range bits {
		bits[i] = true
	}
	var excluded []byte
	for _, k := range keys[1:] {
		if !member[string(k)] {
			t.Fatal("excluded key not a member")
		}
		excluded = append(excluded, k...)
	}

	participants := 0
	for _, b := range bits {
		if b {
			participants++
		}
	}
	thresholdMet := participants*3 >= n*2
	ok := X_fastAggregateVerifyExcluding(aggregate[:], excluded, forged, sig[:])
	t.Logf("membership of all %d excluded keys: proven; bits set: %d/%d; 2/3 threshold met: %v; X_fastAggregateVerifyExcluding: %v; signers by excluded: %d",
		len(excluded)/pubKeySize, participants, n, thresholdMet, ok, n-len(excluded)/pubKeySize)
	if ok && thresholdMet {
		t.Errorf("a single member's signature over a forged header passes as %d/%d participation with both documented obligations met", participants, n)
	}
}
