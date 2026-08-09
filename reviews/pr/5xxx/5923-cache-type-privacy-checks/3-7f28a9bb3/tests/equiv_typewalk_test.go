/* Run: from a gno checkout, once at each sha:
gh pr checkout 5923 -R gnolang/gno && git checkout 7f28a9bb3   # then ddb752cac
curl -fsSL -o gnovm/pkg/gnolang/equiv_typewalk_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/3-7f28a9bb3/tests/equiv_typewalk_test.go
go test ./gnovm/pkg/gnolang/ -run TestEquivTypeWalk -v 2>&1 | grep CASE
rm gnovm/pkg/gnolang/equiv_typewalk_test.go
*/

// Does the rewritten assertTypeIsPublic reject the same types as the merge
// base, for every Type kind? The file compiles at both shas: it touches only
// assertTypeIsPublic, NewStore, SetCachePackage and the Type constructors,
// none of which the PR changes. Each case gets its own store, so the head's
// process-lived typePrivacyCache carries no verdict between cases. Diff the
// two CASE listings: 42 cases, 7 differ at 7f28a9bb3.
package gnolang

import (
	"fmt"
	"testing"
)

// outcome runs assertTypeIsPublic and reports "ok" or the panic text.
func outcome(rlmPath string, register map[string]bool, t Type) (res string) {
	st := NewStore(nil, nil, nil)
	for p, priv := range register {
		st.SetCachePackage(&PackageValue{PkgPath: p, Private: priv})
	}
	rlm := NewRealm(rlmPath)
	defer func() {
		if r := recover(); r != nil {
			res = fmt.Sprintf("PANIC: %v", r)
		}
	}()
	rlm.assertTypeIsPublic(st, t, map[TypeID]struct{}{})
	return "ok"
}

func TestEquivTypeWalk(t *testing.T) {
	const pub = "gno.land/r/pub"
	const priv = "gno.land/r/priv"
	reg := map[string]bool{pub: false, priv: true}

	// privS is a struct declared in the private package; anything that
	// reaches it must be rejected.
	privS := func() Type {
		return &StructType{PkgPath: priv, Fields: []FieldType{{Name: "X", Type: IntType}}}
	}
	// privD is the declared-type form of the same.
	privD := func() Type {
		d := &DeclaredType{PkgPath: priv, Name: "D"}
		d.Base = &StructType{PkgPath: priv, Fields: []FieldType{{Name: "X", Type: IntType}}}
		return d
	}

	cases := []struct {
		name string
		rlm  string
		typ  func() Type
	}{
		// ---- *FuncType ------------------------------------------------
		{"FuncType/named-param-private", pub, func() Type {
			return &FuncType{Params: []FieldType{{Name: "a", Type: privD()}}}
		}},
		{"FuncType/UNNAMED-param-private", pub, func() Type {
			return &FuncType{Params: []FieldType{{Name: "", Type: privD()}}}
		}},
		{"FuncType/named-result-private", pub, func() Type {
			return &FuncType{Results: []FieldType{{Name: "r", Type: privD()}}}
		}},
		{"FuncType/UNNAMED-result-private", pub, func() Type {
			return &FuncType{Results: []FieldType{{Name: "", Type: privD()}}}
		}},
		{"FuncType/all-public", pub, func() Type {
			return &FuncType{Params: []FieldType{{Name: "", Type: IntType}}}
		}},

		// ---- FieldType as the root ------------------------------------
		{"FieldType/named-private", pub, func() Type {
			return FieldType{Name: "f", Type: privD()}
		}},
		{"FieldType/UNNAMED-private", pub, func() Type {
			return FieldType{Name: "", Type: privD()}
		}},

		// ---- element wrappers -----------------------------------------
		{"SliceType/private-elem", pub, func() Type { return &SliceType{Elt: privD()} }},
		{"ArrayType/private-elem", pub, func() Type { return &ArrayType{Len: 3, Elt: privD()} }},
		{"PointerType/private-elem", pub, func() Type { return &PointerType{Elt: privD()} }},
		{"SliceType/public-elem", pub, func() Type { return &SliceType{Elt: IntType} }},

		// ---- *tupleType -----------------------------------------------
		{"tupleType/private-elt", pub, func() Type {
			return &tupleType{Elts: []Type{IntType, privD()}}
		}},
		{"tupleType/public", pub, func() Type { return &tupleType{Elts: []Type{IntType}} }},

		// ---- *MapType -------------------------------------------------
		{"MapType/private-key", pub, func() Type { return &MapType{Key: privD(), Value: IntType} }},
		{"MapType/private-value", pub, func() Type { return &MapType{Key: IntType, Value: privD()} }},
		{"MapType/public", pub, func() Type { return &MapType{Key: IntType, Value: StringType} }},

		// ---- *InterfaceType -------------------------------------------
		{"InterfaceType/own-pkg-private", pub, func() Type {
			return &InterfaceType{PkgPath: priv, Methods: []FieldType{}}
		}},
		{"InterfaceType/method-named-result-private", pub, func() Type {
			return &InterfaceType{PkgPath: pub, Methods: []FieldType{
				{Name: "M", Type: &FuncType{Results: []FieldType{{Name: "r", Type: privD()}}}},
			}}
		}},
		{"InterfaceType/method-UNNAMED-result-private", pub, func() Type {
			return &InterfaceType{PkgPath: pub, Methods: []FieldType{
				{Name: "M", Type: &FuncType{Results: []FieldType{{Name: "", Type: privD()}}}},
			}}
		}},
		{"InterfaceType/public", pub, func() Type {
			return &InterfaceType{PkgPath: pub, Methods: []FieldType{
				{Name: "M", Type: &FuncType{Results: []FieldType{{Name: "", Type: IntType}}}},
			}}
		}},

		// ---- *StructType ----------------------------------------------
		{"StructType/own-pkg-private", pub, func() Type { return privS() }},
		{"StructType/private-field", pub, func() Type {
			return &StructType{PkgPath: pub, Fields: []FieldType{{Name: "P", Type: privD()}}}
		}},
		{"StructType/embedded-private", pub, func() Type {
			return &StructType{PkgPath: pub, Fields: []FieldType{
				{Name: "D", Embedded: true, Type: privD()},
			}}
		}},
		{"StructType/public", pub, func() Type {
			return &StructType{PkgPath: pub, Fields: []FieldType{{Name: "X", Type: IntType}}}
		}},

		// ---- *DeclaredType ---------------------------------------------
		{"DeclaredType/own-pkg-private", pub, func() Type { return privD() }},
		{"DeclaredType/base-private", pub, func() Type {
			d := &DeclaredType{PkgPath: pub, Name: "Wrap"}
			d.Base = &StructType{PkgPath: pub, Fields: []FieldType{{Name: "P", Type: privD()}}}
			return d
		}},
		{"DeclaredType/method-T-private-named", pub, func() Type {
			d := &DeclaredType{PkgPath: pub, Name: "WithM"}
			d.Base = &StructType{PkgPath: pub, Fields: []FieldType{{Name: "X", Type: IntType}}}
			d.Methods = []TypedValue{{
				T: &FuncType{Params: []FieldType{{Name: "a", Type: privD()}}},
				V: nil,
			}}
			return d
		}},
		{"DeclaredType/method-T-private-UNNAMED", pub, func() Type {
			d := &DeclaredType{PkgPath: pub, Name: "WithM2"}
			d.Base = &StructType{PkgPath: pub, Fields: []FieldType{{Name: "X", Type: IntType}}}
			d.Methods = []TypedValue{{
				T: &FuncType{Params: []FieldType{{Name: "", Type: privD()}}},
				V: nil,
			}}
			return d
		}},
		{"DeclaredType/methodValue-Type-private", pub, func() Type {
			d := &DeclaredType{PkgPath: pub, Name: "WithMV"}
			d.Base = &StructType{PkgPath: pub, Fields: []FieldType{{Name: "X", Type: IntType}}}
			ft := &FuncType{Params: []FieldType{{Name: "a", Type: IntType}}}
			d.Methods = []TypedValue{{
				T: ft,
				V: &FuncValue{Type: &FuncType{Params: []FieldType{{Name: "b", Type: privD()}}}},
			}}
			return d
		}},
		{"DeclaredType/self-cycle-public", pub, func() Type {
			d := &DeclaredType{PkgPath: pub, Name: "Node"}
			d.Base = &StructType{PkgPath: pub, Fields: []FieldType{
				{Name: "Next", Type: &PointerType{Elt: d}},
			}}
			return d
		}},
		{"DeclaredType/self-cycle-with-private", pub, func() Type {
			d := &DeclaredType{PkgPath: pub, Name: "Node2"}
			d.Base = &StructType{PkgPath: pub, Fields: []FieldType{
				{Name: "Next", Type: &PointerType{Elt: d}},
				{Name: "P", Type: privD()},
			}}
			return d
		}},

		// ---- leaf kinds -------------------------------------------------
		{"PrimitiveType", pub, func() Type { return IntType }},
		{"TypeType", pub, func() Type { return gTypeType }},
		{"PackageType", pub, func() Type { return &PackageType{} }},
		{"blockType", pub, func() Type { return blockType{} }},
		{"heapItemType", pub, func() Type { return heapItemType{} }},
		{"RefType-ptr", pub, func() Type { return &RefType{ID: TypeID("x")} }},
		{"RefType-value", pub, func() Type { return RefType{ID: TypeID("x")} }},

		// ---- realm-exemption paths --------------------------------------
		{"own-private-realm-struct", priv, func() Type { return privS() }},
		{"own-private-realm-declared", priv, func() Type { return privD() }},
		{"own-private-realm-reaching-other-private", priv, func() Type {
			return &StructType{PkgPath: priv, Fields: []FieldType{{Name: "X", Type: IntType}}}
		}},
	}

	for _, c := range cases {
		got := outcome(c.rlm, reg, c.typ())
		t.Logf("CASE %-46s => %s", c.name, got)
	}
}

// TestEquivTypeWalkUnregisteredOwnPkg pins the one case where the realm's
// OWN package is not in the store. The merge base short-circuits on
// pkgPath != rlm.Path and never calls isPkgPrivateFromPkgPath; the head's
// pre-walk has no such exemption.
func TestEquivTypeWalkUnregisteredOwnPkg(t *testing.T) {
	const own = "gno.land/r/unregistered"
	got := outcome(own, map[string]bool{}, &StructType{
		PkgPath: own, Fields: []FieldType{{Name: "X", Type: IntType}},
	})
	t.Logf("CASE %-46s => %s", "unregistered-own-pkg", got)
}
