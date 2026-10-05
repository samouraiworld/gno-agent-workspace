// Reflector measurement for gnolang/gno PR 6271 at 2a84c1dcd: the ADR's
// "Verify, 256 KiB block (sha2-256): 2.4M" row. Run with
// gno test -v -print-runtime-metrics -run TestRefl; Verify's cost is
// TestReflVerify256K minus TestReflParseAndBlock.
package cid

import "testing"

const reflCID = "bafkreiekhhjkxu4ztk3tyng3er3ijhg56mb44oe3gwbgquhzu4afrg2ksa" // Raw, sha2-256 of 262144 zero bytes

func TestReflParseAndBlock(t *testing.T) {
	c, _ := Parse(reflCID)
	block := make([]byte, 262144)
	_, _ = c, block
}

func TestReflVerify256K(t *testing.T) {
	c, _ := Parse(reflCID)
	block := make([]byte, 262144)
	if err := c.Verify(block); err != nil {
		t.Fatal(err)
	}
}
