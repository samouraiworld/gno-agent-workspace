// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a local clone of gnolang/gno:
gh pr checkout 5082 -R gnolang/gno && git checkout 3be8c3b1a
curl -fsSL -o gnovm/pkg/gnolang/string_ref_roundtrip_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5082-ref-stringvalue-slicing/2-3be8c3b1/tests/string_ref_undercount_test.go
go test -v -run 'TestStringRefMarshalRoundTripChangesSize' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/string_ref_roundtrip_test.go
*/

// One asymmetry, introduced by this diff: a ref-mode StringValue reports 48
// bytes via GetShallowSize; after MarshalAmino/UnmarshalAmino it comes back as
// owner mode (ref=false) and reports 48 + len(data). Master has no ref mode, so
// the same value reports 54 on both sides. The GC rebuilds the allocator's
// counter from GetShallowSize, so one string contributes two different amounts
// depending on whether it was just deserialized or still in memory from a slice.
//
// The round-2 slice-undercount claim that lived here is retracted. It was tested
// against master, not argued: retention is len(backing) and the slice charge
// tracks len(slice); the two are independent. Master charges 49, 1072, and
// 33,554,480 for 1 B, 1 KiB, and 32 MiB slices of a 64 MiB parent while
// retention stays 64 MiB throughout, so the charge never bounded retention. The
// flat 48 here is more accurate, not less: the parent already carries its own
// 48 + len charge, and master double-counts those bytes for a copy it never made.

package gnolang

import "testing"

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
