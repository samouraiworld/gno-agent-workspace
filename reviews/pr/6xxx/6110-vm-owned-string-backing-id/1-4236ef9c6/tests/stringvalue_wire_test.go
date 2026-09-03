package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
)

// TestStringValueWireIsMintIndependent pins the migration and rollback
// claim in gnovm/adr/pr6110_string_backing_id.md — "Persisted format
// unchanged (plain string); no store migration" — for values that carry a
// live mint.
//
// parity_test.go's two StringValue cases both hold ID 0 and Extent 0, so
// they cannot tell a codec that persists the mint serial from one that
// does not, and aminotest.AssertCodecParity cannot be reused here: its
// invariant (4), roundtrip fidelity, is deliberately false for this type
// (a decoded StringValue has ID 0 by design). This test asserts
// AssertCodecParity's invariants (1) encoder parity, (2) size, and (3)
// cross-decoder agreement, and replaces (4) with the intended lossiness.
func TestStringValueWireIsMintIndependent(t *testing.T) {
	t.Parallel()

	cdc := amino.NewCodec()
	cdc.RegisterPackage(Package)
	cdc.Seal()

	alloc := NewAllocator(1_000_000)
	sliceOf := func(s string, low, high int) StringValue {
		tv := TypedValue{T: StringType, V: alloc.NewString(s)}
		return tv.GetSlice(alloc, low, high).V.(StringValue)
	}

	cases := []struct {
		name string
		sv   StringValue
	}{
		{"minted", alloc.NewString("hello world")},
		{"minted/second-mint-same-content", alloc.NewString("hello world")},
		{"sliced/mid", sliceOf("hello worldXX", 0, 11)},
		{"untracked", StringValue{Str: "hello world"}},
		{"empty/untracked", StringValue{}},
		{"empty/sliced-from-tracked", sliceOf("abcdefghij", 3, 3)},
	}

	// The pre-struct encoding: the content string and nothing else.
	reference := func(s string) []byte {
		bz, err := cdc.MarshalReflect(&StringValue{Str: s})
		if err != nil {
			t.Fatalf("marshal reference %q: %v", s, err)
		}
		return bz
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sv := c.sv

			// (1) Encoder parity: reflect codec and genproto2 fast path.
			bzReflect, err := cdc.MarshalReflect(&sv)
			if err != nil {
				t.Fatalf("MarshalReflect: %v", err)
			}
			bzBinary2, err := cdc.MarshalBinary2(&sv)
			if err != nil {
				t.Fatalf("MarshalBinary2: %v", err)
			}
			if string(bzReflect) != string(bzBinary2) {
				t.Fatalf("encoder parity: reflect %X != binary2 %X", bzReflect, bzBinary2)
			}

			// (2) Size invariant.
			size, err := sv.SizeBinary2(cdc)
			if err != nil {
				t.Fatalf("SizeBinary2: %v", err)
			}
			if size != len(bzBinary2) {
				t.Errorf("size invariant: SizeBinary2=%d, len(MarshalBinary2)=%d", size, len(bzBinary2))
			}

			// Master's bytes for the same content, byte for byte. This is
			// what a node running this code writes into a store master
			// wrote, and what a reverted node reads back out of it.
			if want := reference(sv.Str); string(bzReflect) != string(want) {
				t.Errorf("wire bytes carry mint state:\n got %X\nwant %X", bzReflect, want)
			}

			// (3) Cross-decoder parity, and (4') intended lossiness: the
			// mint never survives a round trip, so the ID space is free to
			// differ across nodes and across restarts.
			var viaReflect, viaBinary2 StringValue
			if err := cdc.UnmarshalReflect(bzReflect, &viaReflect); err != nil {
				t.Fatalf("UnmarshalReflect: %v", err)
			}
			if err := viaBinary2.UnmarshalBinary2(cdc, bzBinary2, 0); err != nil {
				t.Fatalf("UnmarshalBinary2: %v", err)
			}
			if viaReflect != viaBinary2 {
				t.Errorf("cross-decoder parity: %+v != %+v", viaReflect, viaBinary2)
			}
			if want := (StringValue{Str: sv.Str}); viaReflect != want {
				t.Errorf("decoded %+v, want %+v", viaReflect, want)
			}
		})
	}
}

// TestObjectHashIsMintIndependent closes the consensus half of the same
// claim. Object hashes are HashBytes(amino.MustMarshalAny(o2)) — store.go
// 584 and 665 — so two nodes that minted different serials for the same
// content must produce the same object hash, or they fork. Nothing in the
// PR asserts this at the hash level.
func TestObjectHashIsMintIndependent(t *testing.T) {
	t.Parallel()

	alloc := NewAllocator(1_000_000)
	build := func(sv StringValue) *HeapItemValue {
		return &HeapItemValue{Value: TypedValue{T: StringType, V: sv}}
	}

	// Same content, three different provenances: two separate mints and a
	// value straight off the wire (ID 0).
	a := build(alloc.NewString("gno.land/r/demo/boards"))
	b := build(alloc.NewString("gno.land/r/demo/boards"))
	c := build(StringValue{Str: "gno.land/r/demo/boards"})

	if a.Value.V.(StringValue).ID == b.Value.V.(StringValue).ID {
		t.Fatal("test setup: the two mints must differ")
	}

	ha := HashBytes(amino.MustMarshalAny(a))
	hb := HashBytes(amino.MustMarshalAny(b))
	hc := HashBytes(amino.MustMarshalAny(c))
	if ha != hb || ha != hc {
		t.Errorf("object hash depends on the mint serial:\n mint1 %X\n mint2 %X\n wire  %X", ha, hb, hc)
	}
}
