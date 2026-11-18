// topic1-eth-vectors: the Ethereum BLS vectors of ethereum/bls12-381-tests
// v0.1.2 (the source of consensus-spec-tests general/phase0/bls) run through
// the natives of PR 6375 and through gnark-crypto v0.20.1, the version the
// head pins.
//
//   - fast_aggregate_verify, verify: X_fastAggregateVerify, and
//     X_fastAggregateVerifyExcluding fed aggregate = sum(pubkeys) + G1 with
//     excluded = [G1], so the signers' key is the vector's own sum.
//   - sign: sk * HashToG2(msg, ethDST) compressed, which pins the POP DST
//     the natives hash with against Ethereum's own signatures.
//   - hash_to_G2: bls.HashToG2 under the RFC 9380 QUUX DST the vectors use.
//   - deserialization_G1/G2: the decode the natives run on each 48- or
//     96-byte chunk, the length guard included.
//
// Run, from a plain clone of gnolang/gno at 0ee53fefc:
//
//	mkdir -p /tmp/v && curl -sSL https://github.com/ethereum/bls12-381-tests/releases/download/v0.1.2/bls_tests_json.tar.gz | tar -xz -C /tmp/v
//	cp topic1-eth-vectors_test.go gnovm/stdlibs/crypto/bls12381/
//	BLS_VECTORS=/tmp/v go test -count=1 -timeout 2m -run TestTopic1EthVectors -v ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
)

func topic1Hex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		t.Fatalf("hex %q: %v", s, err)
	}
	return b
}

func topic1Files(t *testing.T, dir string) []string {
	t.Helper()
	root := os.Getenv("BLS_VECTORS")
	if root == "" {
		t.Skip("BLS_VECTORS unset")
	}
	fs, err := filepath.Glob(filepath.Join(root, dir, "*.json"))
	if err != nil || len(fs) == 0 {
		t.Fatalf("no vectors under %s/%s: %v", root, dir, err)
	}
	sort.Strings(fs)
	return fs
}

func topic1Load(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// topic1Excluding runs X_fastAggregateVerifyExcluding over the same signer
// set: aggregate = sum(pubkeys) + G1, excluded = [G1]. ok is false where the
// sum cannot be formed (an undecodable key) or is the identity.
func topic1Excluding(pubkeys [][]byte, msg, sig []byte) (got bool, ok bool) {
	g := g1Generator()
	var sum bls.G1Jac
	sum.FromAffine(&g)
	for _, b := range pubkeys {
		var pk bls.G1Affine
		if _, err := pk.SetBytes(b); err != nil || len(b) != pubKeySize {
			return false, false
		}
		sum.AddMixed(&pk)
	}
	var agg bls.G1Affine
	agg.FromJacobian(&sum)
	if agg.IsInfinity() {
		return false, false
	}
	ab := agg.Bytes()
	gb := g.Bytes()
	return X_fastAggregateVerifyExcluding(ab[:], gb[:], msg, sig), true
}

func TestTopic1EthVectors(t *testing.T) {
	t.Run("fast_aggregate_verify", func(t *testing.T) {
		for _, f := range topic1Files(t, "fast_aggregate_verify") {
			var v struct {
				Input struct {
					Pubkeys   []string `json:"pubkeys"`
					Message   string   `json:"message"`
					Signature string   `json:"signature"`
				} `json:"input"`
				Output bool `json:"output"`
			}
			topic1Load(t, f, &v)
			var cat []byte
			var keys [][]byte
			for _, p := range v.Input.Pubkeys {
				k := topic1Hex(t, p)
				keys = append(keys, k)
				cat = append(cat, k...)
			}
			msg := topic1Hex(t, v.Input.Message)
			sig := topic1Hex(t, v.Input.Signature)
			got := X_fastAggregateVerify(cat, msg, sig)
			ex, exOK := topic1Excluding(keys, msg, sig)
			t.Logf("%-62s want=%v native=%v excluding=%v(formed=%v)", filepath.Base(f), v.Output, got, ex, exOK)
			if got != v.Output {
				t.Errorf("%s: X_fastAggregateVerify=%v, vector says %v", filepath.Base(f), got, v.Output)
			}
			if exOK && ex != v.Output {
				t.Errorf("%s: X_fastAggregateVerifyExcluding=%v, vector says %v", filepath.Base(f), ex, v.Output)
			}
		}
	})

	t.Run("verify", func(t *testing.T) {
		for _, f := range topic1Files(t, "verify") {
			var v struct {
				Input struct {
					Pubkey    string `json:"pubkey"`
					Message   string `json:"message"`
					Signature string `json:"signature"`
				} `json:"input"`
				Output bool `json:"output"`
			}
			topic1Load(t, f, &v)
			pk := topic1Hex(t, v.Input.Pubkey)
			msg := topic1Hex(t, v.Input.Message)
			sig := topic1Hex(t, v.Input.Signature)
			got := X_fastAggregateVerify(pk, msg, sig)
			ex, exOK := topic1Excluding([][]byte{pk}, msg, sig)
			t.Logf("%-62s want=%v native=%v excluding=%v(formed=%v)", filepath.Base(f), v.Output, got, ex, exOK)
			if got != v.Output {
				t.Errorf("%s: X_fastAggregateVerify=%v, vector says %v", filepath.Base(f), got, v.Output)
			}
			if exOK && ex != v.Output {
				t.Errorf("%s: X_fastAggregateVerifyExcluding=%v, vector says %v", filepath.Base(f), ex, v.Output)
			}
		}
	})

	t.Run("sign_pins_pop_dst", func(t *testing.T) {
		for _, f := range topic1Files(t, "sign") {
			var v struct {
				Input struct {
					Privkey string `json:"privkey"`
					Message string `json:"message"`
				} `json:"input"`
				Output *string `json:"output"`
			}
			topic1Load(t, f, &v)
			sk := new(big.Int).SetBytes(topic1Hex(t, v.Input.Privkey))
			if v.Output == nil {
				t.Logf("%-62s want=null (zero key, nothing to sign)", filepath.Base(f))
				continue
			}
			h, err := bls.HashToG2(topic1Hex(t, v.Input.Message), ethDST)
			if err != nil {
				t.Fatal(err)
			}
			var s bls.G2Affine
			s.ScalarMultiplication(&h, sk)
			sb := s.Bytes()
			want := topic1Hex(t, *v.Output)
			match := hex.EncodeToString(sb[:]) == hex.EncodeToString(want)
			t.Logf("%-62s sk*HashToG2(msg, ethDST) matches=%v", filepath.Base(f), match)
			if !match {
				t.Errorf("%s: signature under ethDST differs", filepath.Base(f))
			}
		}
	})

	t.Run("hash_to_G2", func(t *testing.T) {
		quux := []byte("QUUX-V01-CS02-with-BLS12381G2_XMD:SHA-256_SSWU_RO_")
		for _, f := range topic1Files(t, "hash_to_G2") {
			var v struct {
				Input struct {
					Msg string `json:"msg"`
				} `json:"input"`
				Output struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"output"`
			}
			topic1Load(t, f, &v)
			h, err := bls.HashToG2([]byte(v.Input.Msg), quux)
			if err != nil {
				t.Fatal(err)
			}
			el := func(s string) [2]fp.Element {
				parts := strings.Split(s, ",")
				var e [2]fp.Element
				for i := range parts {
					if _, err := e[i].SetString(strings.TrimSpace(parts[i])); err != nil {
						t.Fatal(err)
					}
				}
				return e
			}
			x, y := el(v.Output.X), el(v.Output.Y)
			match := h.X.A0.Equal(&x[0]) && h.X.A1.Equal(&x[1]) && h.Y.A0.Equal(&y[0]) && h.Y.A1.Equal(&y[1])
			t.Logf("%-62s HashToG2 matches=%v", filepath.Base(f), match)
			if !match {
				t.Errorf("%s: HashToG2 differs", filepath.Base(f))
			}
		}
	})

	deser := func(t *testing.T, dir, field string, size int, decode func([]byte) error) {
		for _, f := range topic1Files(t, dir) {
			var v struct {
				Input  map[string]string `json:"input"`
				Output bool              `json:"output"`
			}
			topic1Load(t, f, &v)
			b := topic1Hex(t, v.Input[field])
			// The natives' own guard: whole chunks of exactly size bytes.
			got := len(b) == size && decode(b) == nil
			t.Logf("%-62s want=%v natives-decode=%v", filepath.Base(f), v.Output, got)
			if got != v.Output {
				t.Errorf("%s: decode=%v, vector says %v", filepath.Base(f), got, v.Output)
			}
		}
	}
	t.Run("deserialization_G1", func(t *testing.T) {
		deser(t, "deserialization_G1", "pubkey", pubKeySize, func(b []byte) error {
			var p bls.G1Affine
			_, err := p.SetBytes(b)
			return err
		})
	})
	t.Run("deserialization_G2", func(t *testing.T) {
		deser(t, "deserialization_G2", "signature", signatureSize, func(b []byte) error {
			var p bls.G2Affine
			_, err := p.SetBytes(b)
			return err
		})
	})
}
