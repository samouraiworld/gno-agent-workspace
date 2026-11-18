// topic1-cst-vectors: consensus-spec-tests v1.5.0 general BLS vectors
// (phase0 fast_aggregate_verify and verify, altair eth_fast_aggregate_verify)
// run through X_fastAggregateVerify and X_fastAggregateVerifyExcluding of
// PR 6375 at 0ee53fefc. The Excluding variant is fed the committee
// pubkeys + [G1] as its aggregate and [G1] as excluded, so its signer set
// is the vector's own; an empty signer set is fed aggregate = G1 and
// excluded = [G1], the shape of a sync committee with no participant.
//
// Run, from a plain clone of gnolang/gno at 0ee53fefc:
//
//	mkdir -p /tmp/cst && curl -sSL https://github.com/ethereum/consensus-spec-tests/releases/download/v1.5.0/general.tar.gz | tar -xz -C /tmp/cst --wildcards 'tests/general/*/bls/*'
//	cp topic1-cst-vectors_test.go gnovm/stdlibs/crypto/bls12381/
//	CST_VECTORS=/tmp/cst/tests/general go test -count=1 -timeout 2m -run TestTopic1CSTVectors -v ./gnovm/stdlibs/crypto/bls12381/
package bls12381

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	bls "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"gopkg.in/yaml.v3"
)

type topic1CSTCase struct {
	Input struct {
		Pubkeys   []string `yaml:"pubkeys"`
		Pubkey    string   `yaml:"pubkey"`
		Message   string   `yaml:"message"`
		Signature string   `yaml:"signature"`
	} `yaml:"input"`
	Output bool `yaml:"output"`
}

func TestTopic1CSTVectors(t *testing.T) {
	root := os.Getenv("CST_VECTORS")
	if root == "" {
		t.Skip("CST_VECTORS unset")
	}
	for _, handler := range []string{
		"phase0/bls/fast_aggregate_verify",
		"phase0/bls/verify",
		"altair/bls/eth_fast_aggregate_verify",
	} {
		t.Run(handler, func(t *testing.T) {
			files, _ := filepath.Glob(filepath.Join(root, handler, "bls", "*", "data.yaml"))
			if len(files) == 0 {
				t.Fatalf("no cases under %s", handler)
			}
			sort.Strings(files)
			for _, f := range files {
				raw, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				var c topic1CSTCase
				if err := yaml.Unmarshal(raw, &c); err != nil {
					t.Fatalf("%s: %v", f, err)
				}
				keysHex := c.Input.Pubkeys
				if c.Input.Pubkey != "" {
					keysHex = []string{c.Input.Pubkey}
				}
				var cat []byte
				var keys [][]byte
				for _, p := range keysHex {
					k := topic1Hex(t, p)
					keys = append(keys, k)
					cat = append(cat, k...)
				}
				msg := topic1Hex(t, c.Input.Message)
				sig := topic1Hex(t, c.Input.Signature)
				name := filepath.Base(filepath.Dir(f))

				got := X_fastAggregateVerify(cat, msg, sig)
				var ex, formed bool
				if len(keys) == 0 {
					g := g1Generator()
					gb := g.Bytes()
					ex, formed = X_fastAggregateVerifyExcluding(gb[:], gb[:], msg, sig), true
				} else {
					ex, formed = topic1Excluding(keys, msg, sig)
				}
				t.Logf("%-58s want=%v native=%v excluding=%v(formed=%v)", name, c.Output, got, ex, formed)
				if got != c.Output {
					t.Errorf("%s/%s: X_fastAggregateVerify=%v, vector says %v", handler, name, got, c.Output)
				}
				if formed && ex != c.Output {
					t.Errorf("%s/%s: X_fastAggregateVerifyExcluding=%v, vector says %v", handler, name, ex, c.Output)
				}
			}
		})
	}
	_ = bls.G1Affine{}
}
