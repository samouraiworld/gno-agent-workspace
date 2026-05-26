// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5301 -R gnolang/gno && git checkout 8ef338b94
curl -fsSL -o tm2/pkg/amino/trailing_reserved_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5301-amino-reserved-fields/1-8ef338b94/tests/trailing_reserved_test.go
go test -v -run 'TestReservedField(Trailing|MiddleControl)' ./tm2/pkg/amino/
rm tm2/pkg/amino/trailing_reserved_test.go
*/

// The reflection decoder (binary_decode.go:1058) rejects any wire bytes that
// remain after iterating info.Fields. info.Fields excludes reserved slots, so
// when the reserved fnum is the LAST in the struct, old encoded bytes carrying
// the removed field are still in bz when the loop exits and trip the
// "unknown field number N for <Type>" error — defeating the backward-compat
// promise of amino:"reserved".
//
// The generated UnmarshalBinary2 path (genproto2) emits a top-level switch
// keyed on fnum, so trailing reserved fnums dispatch to the per-typ3 skip
// stub and decode correctly. The bug is reflection-path-only.
//
// Result: TestReservedFieldTrailing fails on this commit with
// "unknown field number 2 for amino_test.NewStruct".
// TestReservedFieldMiddleControl passes; it's the same migration shape with
// a trailing C field after the reserved slot, isolating the cause.
// To assert the fix: TestReservedFieldTrailing's `assert.NoError` flips to
// pass once binary_decode.go treats trailing fnums > last declared
// field.BinFieldNum as reserved-skippable rather than rejecting them.

package amino_test

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/stretchr/testify/assert"
)

func TestReservedFieldTrailing(t *testing.T) {
	type OldStruct struct {
		A string
		B string
	}

	type NewStruct struct {
		A string
		_ [0]struct{} `amino:"reserved"`
	}

	cdc := amino.NewCodec()

	bz, err := cdc.Marshal(OldStruct{A: "hello", B: "removed"})
	assert.NoError(t, err)

	var got NewStruct
	err = cdc.Unmarshal(bz, &got)
	assert.NoError(t, err, "trailing-reserved=err middle-reserved=ok")
	assert.Equal(t, "hello", got.A)
}

func TestReservedFieldMiddleControl(t *testing.T) {
	type OldStruct struct {
		A string
		B string
		C string
	}

	type NewStruct struct {
		A string
		_ [0]struct{} `amino:"reserved"`
		C string
	}

	cdc := amino.NewCodec()
	bz, err := cdc.Marshal(OldStruct{A: "hello", B: "removed", C: "world"})
	assert.NoError(t, err)

	var got NewStruct
	err = cdc.Unmarshal(bz, &got)
	assert.NoError(t, err, "middle-reserved=ok control for trailing-reserved=err")
	assert.Equal(t, "hello", got.A)
	assert.Equal(t, "world", got.C)
}
