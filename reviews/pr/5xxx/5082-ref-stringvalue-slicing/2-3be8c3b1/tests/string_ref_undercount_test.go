// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a local clone of gnolang/gno:
gh pr checkout 5082 -R gnolang/gno && git checkout 3be8c3b1a
curl -fsSL -o gnovm/pkg/gnolang/string_ref_undercount_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5082-ref-stringvalue-slicing/2-3be8c3b1/tests/string_ref_undercount_test.go
go test -v -run 'TestStringSliceChargeVsRetention|TestStringRefMarshalRoundTripChangesSize' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/string_ref_undercount_test.go
*/

// Baselined against master 27b5b8e24: the identical test there accounts 49
// bytes post-GC and pins the same parent, so slice retention is NOT introduced
// by this PR. The round-2 claim that it was is retracted. What the diff moves
// is the allocation-time charge and the size of a ref across persistence.
//
//  1. Slice charge: master charges 48+len for a slice it never copied, which
//     over-charges in the copy sense but bounds how much a transaction can
//     retain by slicing. This PR charges a flat 48. Neither revision tracks the
//     retained backing, so that per-byte charge is the only bound there is.
//
//  2. Round-trip drift: a ref-mode StringValue reports 48 bytes via
//     GetShallowSize; after MarshalAmino/UnmarshalAmino it comes back as owner
//     mode (ref=false) and reports 48 + len(data). Master has no ref mode, so
//     the same value reports 54 on both sides.

package gnolang

import (
	"strings"
	"testing"
)

func TestStringSliceChargeVsRetention(t *testing.T) {
	const parentLen = 64 << 20
	alloc := NewAllocator(1 << 30)
	tv := TypedValue{T: StringType, V: alloc.NewString(strings.Repeat("a", parentLen))}
	_, before := alloc.Status()

	// Half the parent. No bytes are copied on either revision: Go substrings
	// share the backing array, so the parent stays alive through the slice.
	slice := tv.GetSlice(alloc, 0, 32<<20)
	_, after := alloc.Status()
	charged := after - before

	// IS (this PR): a flat allocStringRef, whatever the slice length.
	if charged != allocStringRef {
		t.Fatalf("slice charge: got %d, want %d (allocStringRef)", charged, allocStringRef)
	}
	t.Logf("32 MiB slice charged %d bytes; master charges 33554480 for the same slice", charged)

	// SHOULD, once backing-range tracking lands (see PR 4885): the retained
	// backing is charged once per cycle rather than not at all. Flip to assert:
	// if charged != allocStringRef+allocStringByte*int64(32<<20) {
	//     t.Fatalf("slice charge: got %d, want the retained length", charged)
	// }
	_ = slice
}

func TestStringRefMarshalRoundTripChangesSize(t *testing.T) {
	const data = "abcdef"
	ref := NewStringValueRef(data)
	beforeSize := ref.GetShallowSize()

	repr, err := ref.MarshalAmino()
	if err != nil {
		t.Fatalf("MarshalAmino: %v", err)
	}
	var after StringValue
	if err := after.UnmarshalAmino(repr); err != nil {
		t.Fatalf("UnmarshalAmino: %v", err)
	}
	afterSize := after.GetShallowSize()

	if beforeSize == afterSize {
		t.Fatalf("expected size to change across amino round-trip, got %d == %d",
			beforeSize, afterSize)
	}

	t.Logf("pre-roundtrip=%d (ref) post-roundtrip=%d (owner) — same logical value, different accounting",
		beforeSize, afterSize)

	// IS: 48 (ref) -> 48+len(data) (owner). Same string, two sizes.
	if beforeSize != allocStringRef {
		t.Fatalf("pre-roundtrip: got %d, want allocStringRef=%d", beforeSize, allocStringRef)
	}
	expectedOwner := allocString + allocStringByte*int64(len(data))
	if afterSize != expectedOwner {
		t.Fatalf("post-roundtrip: got %d, want owner=%d", afterSize, expectedOwner)
	}

	// SHOULD: either preserve ref across persistence, or use one accounting
	// rule everywhere. Flip:
	// if beforeSize != afterSize {
	//     t.Fatalf("round-trip should preserve GetShallowSize: %d vs %d",
	//         beforeSize, afterSize)
	// }
}
