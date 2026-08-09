/* Run: from a gno checkout:
gh pr checkout 5923 -R gnolang/gno && git checkout 7f28a9bb3
curl -fsSL -o gnovm/pkg/gnolang/typeid_collision_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/3-7f28a9bb3/tests/typeid_collision_test.go
go test -v -run TestTypeHasPrivateDep_TypeIDCollides ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/typeid_collision_test.go
*/

// TypeID drops PkgPath for a struct whose field names are all exported and
// for an interface whose method names are all exported, so one memo key
// carries two packages' verdicts. The structs the first case builds are the
// ones realm_privatedep_test.go's TestTypeHasPrivateDep_PublicStruct and
// _OwnPackagePrivate already build; those pass because each gets its own
// store. Declared types are not affected and the third case pins that.

package gnolang

import "testing"

func TestTypeHasPrivateDep_TypeIDCollidesOnStruct(t *testing.T) {
	store, pubPath, privPath := newPrivateDepTestStore(t)

	pub := &StructType{PkgPath: pubPath, Fields: []FieldType{{Name: "X", Type: IntType}}}
	priv := &StructType{PkgPath: privPath, Fields: []FieldType{{Name: "X", Type: IntType}}}
	if pub.TypeID() != priv.TypeID() {
		t.Fatalf("TypeIDs differ, nothing to measure: %s vs %s", pub.TypeID(), priv.TypeID())
	}

	if typeHasPrivateDep(store, pub) {
		t.Fatal("typeHasPrivateDep(pub) = true, want false")
	}
	if !typeHasPrivateDep(store, priv) {
		t.Fatal("typeHasPrivateDep(priv) = false, want true: priv is declared in a private package")
	}
}

func TestTypeHasPrivateDep_TypeIDCollidesOnInterface(t *testing.T) {
	store, pubPath, privPath := newPrivateDepTestStore(t)

	method := func() []FieldType {
		return []FieldType{{Name: "Get", Type: &FuncType{Results: []FieldType{{Type: IntType}}}}}
	}
	pub := &InterfaceType{PkgPath: pubPath, Methods: method()}
	priv := &InterfaceType{PkgPath: privPath, Methods: method()}
	if pub.TypeID() != priv.TypeID() {
		t.Fatalf("TypeIDs differ, nothing to measure: %s vs %s", pub.TypeID(), priv.TypeID())
	}

	if typeHasPrivateDep(store, pub) {
		t.Fatal("typeHasPrivateDep(pub) = true, want false")
	}
	if !typeHasPrivateDep(store, priv) {
		t.Fatal("typeHasPrivateDep(priv) = false, want true: priv is declared in a private package")
	}
}

// A declared type carries its PkgPath in its TypeID, so the memo key is
// unique per package and no verdict crosses. This one passes at 7f28a9bb3.
func TestTypeHasPrivateDep_TypeIDCollidesNotOnDeclared(t *testing.T) {
	store, pubPath, privPath := newPrivateDepTestStore(t)

	base := func(p string) *StructType {
		return &StructType{PkgPath: p, Fields: []FieldType{{Name: "X", Type: IntType}}}
	}
	pub := &DeclaredType{PkgPath: pubPath, Name: "Data", Base: base(pubPath)}
	priv := &DeclaredType{PkgPath: privPath, Name: "Data", Base: base(privPath)}

	if typeHasPrivateDep(store, pub) {
		t.Fatal("typeHasPrivateDep(pub) = true, want false")
	}
	if !typeHasPrivateDep(store, priv) {
		t.Fatal("typeHasPrivateDep(priv) = false, want true")
	}
}
