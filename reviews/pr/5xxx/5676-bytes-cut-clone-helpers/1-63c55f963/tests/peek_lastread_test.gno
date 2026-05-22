// Adversarial test for PR #5676: confirm Peek does not break a pending
// UnreadByte. Upstream Go's Peek deliberately doesn't touch lastRead so an
// intervening Peek between Read and Unread* still allows the unread to fire.
package bytes_test

import (
	"bytes"
	"io"
	"testing"
)

func TestPeekPreservesLastRead(t *testing.T) {
	buf := bytes.NewBufferString("abcdef")

	c, err := buf.ReadByte()
	if err != nil || c != 'a' {
		t.Fatalf("ReadByte() = %q, %v; want 'a', nil", c, err)
	}

	// Peek should not invalidate the read op recorded by ReadByte.
	p, err := buf.Peek(2)
	if err != nil {
		t.Fatalf("Peek(2) err = %v; want nil", err)
	}
	if string(p) != "bc" {
		t.Fatalf("Peek(2) = %q; want %q", p, "bc")
	}

	// UnreadByte must still succeed because Peek did not touch lastRead.
	if err := buf.UnreadByte(); err != nil {
		t.Fatalf("UnreadByte after Peek: got %v; want nil", err)
	}

	// We should re-read 'a' (the unread byte).
	c, err = buf.ReadByte()
	if err != nil || c != 'a' {
		t.Fatalf("ReadByte() after UnreadByte = %q, %v; want 'a', nil", c, err)
	}
}

// TestPeekEOFShortBuffer verifies the documented EOF return: when n > Len(),
// Peek returns whatever unread bytes remain together with io.EOF (not nil).
func TestPeekEOFShortBuffer(t *testing.T) {
	buf := bytes.NewBufferString("xy")
	p, err := buf.Peek(5)
	if err != io.EOF {
		t.Fatalf("Peek(5) err = %v; want io.EOF", err)
	}
	if string(p) != "xy" {
		t.Fatalf("Peek(5) on short buffer = %q; want %q", p, "xy")
	}
	// Buffer must not have been advanced.
	if buf.Len() != 2 {
		t.Fatalf("buf.Len after Peek = %d; want 2", buf.Len())
	}
}

// TestPeekEmptyN zero asks for zero bytes from an empty buffer; upstream
// returns ([], nil) because Len() (0) is not less than n (0).
func TestPeekZeroEmpty(t *testing.T) {
	var buf bytes.Buffer
	p, err := buf.Peek(0)
	if err != nil {
		t.Fatalf("Peek(0) on empty buf err = %v; want nil", err)
	}
	if len(p) != 0 {
		t.Fatalf("Peek(0) on empty buf len = %d; want 0", len(p))
	}
}

// TestAvailableBufferAliasing verifies the documented contract: the slice
// returned by AvailableBuffer aliases unused capacity of b.buf. Appending to
// it and then Write-ing the result should not double-copy or corrupt data.
func TestAvailableBufferAliasing(t *testing.T) {
	var buf bytes.Buffer
	buf.Grow(64) // ensure some capacity
	buf.WriteString("hi")

	avail := buf.AvailableBuffer()
	if len(avail) != 0 {
		t.Fatalf("AvailableBuffer len = %d; want 0", len(avail))
	}
	if cap(avail) != buf.Available() {
		t.Fatalf("cap(AvailableBuffer) = %d; want Available()=%d", cap(avail), buf.Available())
	}

	avail = append(avail, "world"...)
	buf.Write(avail)

	if got := buf.String(); got != "hiworld" {
		t.Fatalf("buf = %q; want %q", got, "hiworld")
	}
}

// TestCloneIndependence confirms the clone does not share its backing array
// with the input (mutating the input after Clone must not affect the clone).
// gno lacks unsafe.SliceData, but we can detect aliasing behaviorally by
// mutating one side and re-reading the other.
func TestCloneIndependence(t *testing.T) {
	src := []byte{1, 2, 3, 4, 5}
	cp := bytes.Clone(src)
	src[0] = 99
	if cp[0] == 99 {
		t.Fatalf("Clone shares backing array: mutating src changed clone")
	}
	cp[1] = 88
	if src[1] == 88 {
		t.Fatalf("Clone shares backing array: mutating clone changed src")
	}
}

// TestCutAliasing confirms the documented contract: Cut returns slices
// aliasing s, not copies. Mutating s should propagate.
func TestCutAliasing(t *testing.T) {
	s := []byte("foo=bar")
	before, after, found := bytes.Cut(s, []byte("="))
	if !found {
		t.Fatal("Cut: found=false; want true")
	}
	s[0] = 'F'
	if before[0] != 'F' {
		t.Fatalf("Cut before is a copy, not an alias: before[0]=%q", before[0])
	}
	s[4] = 'B'
	if after[0] != 'B' {
		t.Fatalf("Cut after is a copy, not an alias: after[0]=%q", after[0])
	}
}
