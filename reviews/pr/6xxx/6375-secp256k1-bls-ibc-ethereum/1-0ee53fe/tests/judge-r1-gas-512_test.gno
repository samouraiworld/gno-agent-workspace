// judge-r1-gas-512: gas charged for a 512-key FastAggregateVerify call against
// a 1-key call. The charge is taken before the native runs, so zero bytes of
// the right lengths price the call without valid keys. Compare the 512-key
// figure with the ADR's "about 47M gas"; at fd7e75152 it reads ~49.1M.
package blsgas

import (
	"crypto/bls12381"
	"testing"
)

func call(keys int) {
	bls12381.FastAggregateVerify(make([]byte, keys*48), make([]byte, 32), make([]byte, 96))
}

func TestKeys1(t *testing.T)   { call(1) }
func TestKeys512(t *testing.T) { call(512) }
func TestKeys0(t *testing.T)   { _ = make([]byte, 512*48); _ = make([]byte, 32); _ = make([]byte, 96) }
