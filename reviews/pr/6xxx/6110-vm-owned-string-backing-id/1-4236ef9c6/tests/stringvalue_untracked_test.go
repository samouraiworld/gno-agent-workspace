package gnolang

import (
	"strings"
	"testing"
)

// visitOnce runs one GC visitor over vals and returns the recounted byte
// total, mirroring alloc_test.go's visitStrings helper.
func visitOnce(alloc *Allocator, vals ...Value) int64 {
	var vc int64
	vis := GCVisitorFn(1, alloc, &vc)
	alloc.Reset()
	for _, v := range vals {
		vis(v)
	}
	_, bytes := alloc.Status()
	return bytes
}

// TestUntrackedStringPropagation pins what happens on the two edges of the
// ID-zero carve-out that alloc_test.go leaves open. The ADR calls the
// undercount "deterministic, bounded"; these are the cases that decide how
// far it travels and how large the overcount on the other side can get.
func TestUntrackedStringPropagation(t *testing.T) {
	t.Parallel()

	t.Run("slice of an untracked string stays untracked", func(t *testing.T) {
		alloc := NewAllocator(1_000_000)
		// A VM-internal string: typedString/typedRuntimeError shape.
		untracked := StringValue{Str: strings.Repeat("u", 4096)}
		tv := TypedValue{T: StringType, V: untracked}
		sliced := tv.GetSlice(alloc, 0, 4096).V.(StringValue)

		if sliced.ID != 0 || sliced.Extent != 0 {
			t.Errorf("slice of untracked: ID=%d Extent=%d, want 0,0", sliced.ID, sliced.Extent)
		}
		// The undercount propagates: 4096 live bytes recount as one header.
		if got, want := visitOnce(alloc, sliced), int64(allocString); got != want {
			t.Errorf("recount: got %d, want %d", got, want)
		}
		// And GetSlice charges the header only, where master's
		// alloc.NewString(...) charged allocStringByte per byte.
		alloc.Reset()
		tv.GetSlice(alloc, 0, 4096)
		if _, charged := alloc.Status(); charged != int64(allocString) {
			t.Errorf("GetSlice charge: got %d, want %d", charged, allocString)
		}
	})

	t.Run("empty slice of a tracked string keeps the whole backing", func(t *testing.T) {
		alloc := NewAllocator(10_000_000)
		src := alloc.NewString(strings.Repeat("x", 1_000_000))
		tv := TypedValue{T: StringType, V: src}
		empty := tv.GetSlice(alloc, 5, 5).V.(StringValue)

		if empty.Str != "" {
			t.Fatalf("content: got %q", empty.Str)
		}
		// Go's s[5:5] shares the backing pointer, so charging the full
		// extent for a zero-length value is the conservative answer — but
		// it means one surviving empty string pins 1 MB in the recount.
		if empty.ID != src.ID || empty.Extent != 1_000_000 {
			t.Errorf("empty slice identity: ID=%d Extent=%d, want %d,1000000", empty.ID, empty.Extent, src.ID)
		}
		want := int64(allocString) + allocStringByte*1_000_000
		if got := visitOnce(alloc, empty); got != want {
			t.Errorf("recount through a len-0 value: got %d, want %d", got, want)
		}
		// alloc.NewString("") is the other empty string, and it is
		// untracked — two "" values that are not interchangeable for
		// accounting.
		if e2 := alloc.NewString(""); e2.ID != 0 || e2.Extent != 0 {
			t.Errorf("NewString(\"\"): ID=%d Extent=%d, want 0,0", e2.ID, e2.Extent)
		}
	})

	t.Run("VM panic text is header-only however long", func(t *testing.T) {
		alloc := NewAllocator(1_000_000)
		// typedString and typedRuntimeError are the two production
		// constructors that mint nothing; user code reaches their values
		// through recover().
		for _, tv := range []TypedValue{
			typedString(strings.Repeat("p", 8192)),
			typedRuntimeError(strings.Repeat("r", 8192)),
		} {
			sv := tv.V.(StringValue)
			if sv.ID != 0 {
				t.Errorf("%T minted an ID", tv.V)
			}
			if got, want := visitOnce(alloc, sv), int64(allocString); got != want {
				t.Errorf("recount: got %d, want header %d", got, want)
			}
		}
	})
}
