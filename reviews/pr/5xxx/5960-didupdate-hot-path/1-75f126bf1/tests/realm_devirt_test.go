/* Run: from a gno checkout:
gh pr checkout 5960 -R gnolang/gno && git checkout 75f126bf1
curl -fsSL -o gnovm/pkg/gnolang/realm_devirt_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5960-didupdate-hot-path/1-75f126bf1/tests/realm_devirt_test.go
go test -v -run 'TestObjectInfoAccessorsAreNotOverridden' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/realm_devirt_test.go
*/

// DidUpdate now reads and writes only through *ObjectInfo, so an override of
// any accessor on an Object implementation would be bypassed there and honored
// everywhere else. Passes at 75f126bf1; fails if StructValue overrides GetIsReal.

package gnolang

import "testing"

// allObjectImpls returns one fresh instance of every type that
// satisfies Object. The eight value types carry ObjectInfo directly;
// the twelve BlockNode types reach it through StaticBlock -> Block ->
// ObjectInfo, so they satisfy Object too and are equally able to
// shadow an accessor. Confirmed complete with a go/types sweep of the
// package scope at 75f126bf1 (20 concrete types).
func allObjectImpls() []Object {
	return []Object{
		// carry ObjectInfo directly (the `_ Object = &X{}` block in ownership.go)
		&ArrayValue{}, &StructValue{}, &FuncValue{}, &BoundMethodValue{},
		&MapValue{}, &PackageValue{}, &Block{}, &HeapItemValue{},
		// reach ObjectInfo via StaticBlock -> Block
		&BlockStmt{}, &FileNode{}, &ForStmt{}, &FuncDecl{},
		&FuncLitExpr{}, &IfCaseStmt{}, &IfStmt{}, &PackageNode{},
		&RangeStmt{}, &SelectCaseStmt{}, &SwitchClauseStmt{}, &SwitchStmt{},
	}
}

// TestObjectInfoAccessorsAreNotOverridden pins the invariant that
// DidUpdate's devirtualization rests on: every Object implementation
// inherits the ObjectInfo accessors from the embedded ObjectInfo, so
// reading/writing through the Object interface and through the
// *ObjectInfo returned by GetObjectInfo() are indistinguishable.
//
// DidUpdate reads and writes exclusively through *ObjectInfo. If a
// future Object implementation overrides any of these methods, DidUpdate
// would silently bypass the override while every other caller keeps
// seeing it, a divergence with no test to catch it today.
func TestObjectInfoAccessorsAreNotOverridden(t *testing.T) {
	t.Parallel()
	for _, oo := range allObjectImpls() {
		oi := oo.GetObjectInfo()
		if oi == nil {
			t.Fatalf("%T: GetObjectInfo() returned nil", oo)
		}

		// GetObjectInfo must be idempotent and identity-stable: DidUpdate
		// caches the pointer for the whole call.
		if oo.GetObjectInfo() != oi {
			t.Errorf("%T: GetObjectInfo() not identity-stable", oo)
		}

		// DidUpdate reads poi.ID.PkgID where it used to call
		// po.GetObjectID().PkgID.
		pid := PkgIDFromPkgPath("gno.land/r/devirt")
		oi.SetObjectID(ObjectID{PkgID: pid, NewTime: 7})
		if oo.GetObjectID() != oi.ID {
			t.Errorf("%T: GetObjectID() != oi.ID (%v vs %v)", oo, oo.GetObjectID(), oi.ID)
		}
		if !oo.GetObjectID().PkgID.eq(oi.ID.PkgID) {
			t.Errorf("%T: GetObjectID().PkgID diverges from oi.ID.PkgID", oo)
		}

		// Boolean flag accessors read by DidUpdate / mark*.
		flags := []struct {
			name string
			set  func(bool)
			viaO func() bool
			viaI func() bool
		}{
			{"IsReal", func(b bool) {
				if b {
					oi.SetNewTime(7)
				} else {
					oi.SetNewTime(0)
				}
			}, oo.GetIsReal, oi.GetIsReal},
			{"IsDirty", func(b bool) { oi.SetIsDirty(b, 42) }, oo.GetIsDirty, oi.GetIsDirty},
			{"IsEscaped", oi.SetIsEscaped, oo.GetIsEscaped, oi.GetIsEscaped},
			{"IsNewReal", oi.SetIsNewReal, oo.GetIsNewReal, oi.GetIsNewReal},
			{"IsNewEscaped", oi.SetIsNewEscaped, oo.GetIsNewEscaped, oi.GetIsNewEscaped},
			{"IsNewDeleted", oi.SetIsNewDeleted, oo.GetIsNewDeleted, oi.GetIsNewDeleted},
			{"IsDeleted", oi.SetIsDeleted, oo.GetIsDeleted, oi.GetIsDeleted},
		}
		for _, f := range flags {
			for _, want := range []bool{true, false} {
				f.set(want)
				if f.viaO() != f.viaI() {
					t.Errorf("%T: Get%s() via Object = %v, via *ObjectInfo = %v",
						oo, f.name, f.viaO(), f.viaI())
				}
				if f.viaI() != want {
					t.Errorf("%T: Set/Get%s round-trip lost value %v", oo, f.name, want)
				}
			}
		}

		// Refcount ops: DidUpdate calls coi.IncRefCount()/xoi.DecRefCount()
		// where it used to call co.IncRefCount()/xo.DecRefCount().
		oi.RefCount = 0
		if got, want := oi.IncRefCount(), oo.GetRefCount(); got != want {
			t.Errorf("%T: IncRefCount via *ObjectInfo = %d, GetRefCount via Object = %d", oo, got, want)
		}
		if got, want := oo.IncRefCount(), oi.GetRefCount(); got != want {
			t.Errorf("%T: IncRefCount via Object = %d, GetRefCount via *ObjectInfo = %d", oo, got, want)
		}
		if got, want := oi.DecRefCount(), oo.GetRefCount(); got != want {
			t.Errorf("%T: DecRefCount via *ObjectInfo = %d, GetRefCount via Object = %d", oo, got, want)
		}

		// SetOwner/GetOwner: markNewReal's debugAssert reads oi.GetOwner().
		owner := &StructValue{}
		owner.SetObjectID(ObjectID{PkgID: pid, NewTime: 1})
		oi.SetOwner(owner)
		if oo.GetOwner() != oi.GetOwner() {
			t.Errorf("%T: GetOwner() via Object diverges from *ObjectInfo", oo)
		}
		if oo.GetOwnerID() != oi.OwnerID {
			t.Errorf("%T: GetOwnerID() via Object diverges from oi.OwnerID", oo)
		}
	}
}

// TestHashletLayoutMatchesHashSize is a build-time-ish guard for the
// 8+8+4 word layout that PkgID.eq and Hashlet.IsZero hard-code. A
// HashSize below 20 is already a compile error (constant slice bound out
// of range); a HashSize above 20 compiles fine and silently makes both
// functions ignore the trailing bytes, which for eq means two distinct
// PkgIDs comparing equal in the cross-realm write guards.
func TestHashletLayoutMatchesHashSize(t *testing.T) {
	t.Parallel()
	if HashSize != 8+8+4 {
		t.Fatalf("HashSize = %d, but PkgID.eq and Hashlet.IsZero hard-code an 8+8+4 = 20 byte layout; "+
			"update both (realm.go, hash_image.go) before changing HashSize", HashSize)
	}
}
