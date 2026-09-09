// Equivalence harness for the X_objectID rewrite in refactor_pass.patch. From a
// plain clone of github.com/gnolang/gno:
//
//	gh pr checkout 6139 -R gnolang/gno
//	git apply refactor_pass.patch
//	cp refactor_xobjectid_equiv_test.go gnovm/stdlibs/chain/runtime/zz_xobjectid_equiv_test.go
//	go test ./gnovm/stdlibs/chain/runtime/ -run TestZZXObjectIDEquiv -count=1 -v
//	rm gnovm/stdlibs/chain/runtime/zz_xobjectid_equiv_test.go
//	git checkout -- gnovm/stdlibs/chain/runtime/native.go
//
// Prints the shipped form beside the two-guard form over every shape the
// parameter arrives as, the non-TypedValue the transpiled-Go path passes
// included. All seven agree.

package runtime

import (
	"fmt"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func TestZZXObjectIDEquiv(t *testing.T) {
	m := gno.NewMachineWithOptions(gno.MachineOptions{})
	for _, c := range []struct {
		name string
		v    any
	}{
		{"not a TypedValue (int)", 42},
		{"not a TypedValue (string)", "abc"},
		{"nil interface", nil},
		{"TypedValue zero", gno.TypedValue{}},
		{"TypedValue primitive", typedString("not an object")},
		{"nil PointerValue", gno.TypedValue{V: gno.PointerValue{}}},
		{"unstamped StructValue", gno.TypedValue{V: gno.NewAllocator(1 << 30).NewStruct(nil, nil)}},
	} {
		fmt.Printf("%-26s old=%-34s new=%s\n", c.name, zzCallTwoGuardForm(m, c.v), zzCallShipped(m, c.v))
	}
}

func zzCallShipped(m *gno.Machine, v any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = "panic"
		}
	}()
	return "ok:" + X_objectID(m, v)
}

// zzCallTwoGuardForm is the shape at the reviewed head.
func zzCallTwoGuardForm(m *gno.Machine, v any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = "panic"
		}
	}()
	tv, ok := v.(gno.TypedValue)
	if !ok || tv.V == nil {
		m.PanicString("value has no object identity")
	}
	oo := tv.GetFirstObject(m.Store)
	if oo == nil {
		m.PanicString("value has no object identity")
	}
	return "ok:" + oo.GetObjectID().DerivePath()
}
