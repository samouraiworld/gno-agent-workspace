// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a local clone of gnolang/gno:
gh pr checkout 5082 -R gnolang/gno && git checkout 3be8c3b1a
curl -fsSL -o gnovm/pkg/gnolang/string_ref_undercount_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5082-ref-stringvalue-slicing/2-3be8c3b1/tests/string_ref_undercount_test.go
go test -v -run 'TestStringRefAccountingAsymmetry' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/string_ref_undercount_test.go
*/

// Demonstrates two asymmetries introduced by ref-mode StringValue:
//
//  1. Pre/post-GC accounting drift: a parent string allocated at owner cost
//     (24 + N) is dropped while only a tiny slice remains live. After GC the
//     allocator's `bytes` counter reflects only the slice's 24-byte ref cost,
//     but the Go runtime still pins the parent's N-byte backing array because
//     the slice header references it. The bigger the parent vs slice, the
//     bigger the gap between "GnoVM thinks we use" and "Go heap actually
//     holds".
//
//  2. Round-trip drift (no GC needed): a single ref-mode StringValue reports
//     24 bytes via GetShallowSize today; after MarshalAmino/UnmarshalAmino it
//     is reconstructed as owner mode (ref=false) and reports 24 + len(data).
//     Same logical value, two different accounted sizes depending on whether
//     it was just deserialized or still in-memory from a slice.
//
// Flip the post-fix asserts to verify a fix (e.g. either keep the ref flag
// across persistence or charge owner cost in GC recount).

package gnolang

import (
	"strings"
	"testing"
)

func TestStringRefAccountingAsymmetry(t *testing.T) {
	const parentLen = 100_000
	parent := strings.Repeat("a", parentLen)

	alloc := NewAllocator(1024 * 1024 * 1024)

	// Owner: charges 24 + parentLen.
	owner := alloc.NewString(parent)
	_, afterOwner := alloc.Status()

	// Slice 1 byte. With this PR: charges only allocStringRef (24).
	ownerTV := TypedValue{T: StringType, V: owner}
	sliceTV := ownerTV.GetSlice(alloc, 0, 1)
	_, afterSlice := alloc.Status()

	sliceCost := afterSlice - afterOwner
	if sliceCost != allocStringRef {
		t.Fatalf("slice alloc cost: got %d, want %d (allocStringRef)", sliceCost, allocStringRef)
	}

	// Simulate the GC re-walk on a "live set" that contains only the slice
	// (caller has dropped the original owner string). The allocator counter
	// is rebuilt from GetShallowSize.
	alloc.Reset()
	sliceVal := sliceTV.V.(StringValue)
	alloc.Recount(sliceVal.GetShallowSize())
	_, afterGC := alloc.Status()

	// IS (with this PR): GnoVM accounts only the ref struct (24 bytes),
	// even though the slice still pins the full parent backing array in Go.
	if afterGC != allocStringRef {
		t.Fatalf("post-GC bytes: got %d, want %d", afterGC, allocStringRef)
	}

	t.Logf("post-GC accounted=%d bytes, Go-pinned≈%d bytes (parent kept alive by slice)",
		afterGC, parentLen)

	// SHOULD: post-GC bytes ≈ allocString + parentLen — covers what is
	// actually retained. Flip the comparison to assert a fix:
	// if afterGC != allocString+allocStringByte*int64(parentLen) {
	//     t.Fatalf("post-GC bytes: got %d, want owner-equivalent %d",
	//         afterGC, allocString+allocStringByte*int64(parentLen))
	// }
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

	// IS: 24 (ref) -> 24+len(data) (owner). Same string, two sizes.
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
