// Reflector, round 2 on gnolang/gno#6375 head 0ee53fefc: no-panic fuzz over
// both natives with length-shaped inputs, so the fuzzer reaches the decoders.
//
// Run: copy beside gnovm/stdlibs/crypto/bls12381/bls12381.go and the PR's
// bls12381_test.go (it uses signers()), then
//   go test -count=1 -timeout 2m -run '^FuzzReflectorNatives$' -fuzz '^FuzzReflectorNatives$' -fuzztime 45s .
//
// Observed at fd7e75152 (go1.25.9, gnark-crypto v0.20.1, -parallel 4): 45s,
// 1,167,435 execs, 127 corpus entries, no panic. Cleared, not filed.
package bls12381

import "testing"

func FuzzReflectorNatives(f *testing.F) {
	msg := []byte("signing root")
	pks, sig := signers(f, 3, msg)
	f.Add(pks, sig, msg)
	f.Add(pks[:pubKeySize], sig, []byte{})
	f.Fuzz(func(t *testing.T, pk, s, m []byte) {
		if len(pk) > 4*pubKeySize {
			pk = pk[:4*pubKeySize]
		}
		pk = pk[:len(pk)/pubKeySize*pubKeySize]
		if len(s) < signatureSize {
			s = append(s, make([]byte, signatureSize-len(s))...)
		}
		s = s[:signatureSize]
		_ = X_fastAggregateVerify(pk, m, s)
		if len(pk) >= pubKeySize {
			_ = X_fastAggregateVerifyExcluding(pk[:pubKeySize], pk[pubKeySize:], m, s)
		}
	})
}
