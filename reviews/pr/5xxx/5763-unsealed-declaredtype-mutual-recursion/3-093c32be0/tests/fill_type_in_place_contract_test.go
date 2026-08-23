// Run: from a gno checkout:
// gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
// curl -fsSL -o gnovm/pkg/gnolang/fill_type_in_place_contract_test.go \
//   https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/fill_type_in_place_contract_test.go
// go test -v -run 'TestFillTypeInPlace' ./gnovm/pkg/gnolang/
// rm gnovm/pkg/gnolang/fill_type_in_place_contract_test.go

package gnolang

import "testing"

// TestFillTypeInPlaceSameKind pins the documented behaviour: a same-kind call
// mutates the existing dst pointer and returns true.
func TestFillTypeInPlaceSameKind(t *testing.T) {
	dst := &StructType{}
	src := &StructType{Fields: []FieldType{{Name: "Val", Type: IntType}}}
	if !fillTypeInPlace(dst, src) {
		t.Fatal("same-kind fill returned false")
	}
	if len(dst.Fields) != 1 || dst.Fields[0].Name != "Val" {
		t.Fatalf("dst not filled: %v", dst.Fields)
	}
}

// TestFillTypeInPlaceKindMismatch pins what a mismatched call does. It returns
// false, leaves dst untouched, and reports nothing. The caller in the TypeDecl
// TRANS_LEAVE handler reads that false as "value kind, nothing to fill" and
// keeps the fresh base, which is the pre-fix behaviour that orphans a dependent
// type. A future caller passing a mismatched pair gets that fallback with no
// diagnostic.
func TestFillTypeInPlaceKindMismatch(t *testing.T) {
	dst := &StructType{}
	src := &SliceType{Elt: IntType}
	if fillTypeInPlace(dst, src) {
		t.Fatal("cross-kind fill returned true")
	}
	if len(dst.Fields) != 0 {
		t.Fatalf("dst mutated by a cross-kind call: %v", dst.Fields)
	}
}

// TestFillTypeInPlaceChanBase records that *ChanType has no case. gno rejects
// `chan` at parse ("channels are not permitted"), so no gno source reaches this
// today; the false is a silent fallback if that ever changes.
func TestFillTypeInPlaceChanBase(t *testing.T) {
	dst := &ChanType{}
	src := &ChanType{Dir: BOTH, Elt: IntType}
	if fillTypeInPlace(dst, src) {
		t.Fatal("chan fill returned true")
	}
	if dst.Elt != nil {
		t.Fatalf("dst mutated: %v", dst.Elt)
	}
}

// TestFillTypeInPlaceReturnIgnorable shows the return value carries the whole
// contract. Nothing stops a caller from dropping it, and a dropped false leaves
// dst empty with no error and no panic.
func TestFillTypeInPlaceReturnIgnorable(t *testing.T) {
	dst := &MapType{}
	src := &StructType{}
	_ = fillTypeInPlace(dst, src) // return dropped, as a future caller may
	if dst.Key != nil || dst.Value != nil {
		t.Fatalf("dst mutated: %v %v", dst.Key, dst.Value)
	}
}
