// Judge test for candidate 1 of gnolang/gno PR 6271 at 2a84c1dcd: the
// CIDv0/CIDv1 discriminators. Passes at the head; fails once isV0 drops
// `b[0] == 0x12` or Parse's v0 branch drops `s[:2] == "Qm"`, which the
// package's own suite does not notice.
package cid

import (
	"testing"

	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestJudgeCodec0x20(t *testing.T) {
	// A CIDv1 whose codec varint is 0x20: 01 20 12 20 <32 bytes>.
	c, err := NewV1(0x20, Sum(Raw, []byte("hello")).Multihash())
	urequire.NoError(t, err)
	d, err := Decode(c.Bytes())
	urequire.NoError(t, err)
	uassert.True(t, d == c)
	f, n, err := DecodeFirst(c.Bytes())
	urequire.NoError(t, err)
	uassert.Equal(t, 36, n)
	uassert.True(t, f == c)
}

func TestJudgeBase32Len46(t *testing.T) {
	// Raw + identity digest of 24 bytes: 28 bytes, 46 characters in base32.
	mh, err := NewMultihash(Identity, []byte("abcdefghijklmnopqrstuvwx"))
	urequire.NoError(t, err)
	c, err := NewV1(Raw, mh)
	urequire.NoError(t, err)
	uassert.Equal(t, 46, len(c.String()))
	p, err := Parse(c.String())
	urequire.NoError(t, err)
	uassert.True(t, p == c)
}
