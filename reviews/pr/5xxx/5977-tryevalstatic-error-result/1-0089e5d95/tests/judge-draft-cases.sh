#!/usr/bin/env bash
# Runs the two test cases the draft proposes, from a clone of gnolang/gno at 0089e5d95:
#   bash judge-draft-cases.sh <clone>
# 1. TestTryEvalStatic_ErrSetOnEvalFailure passes at head, and fails once tryEvalStatic
#    drops every panic but EvalStatic's not-preprocessed guard, while
#    TestTryEvalStatic_ErrSetOnFailure keeps passing under that mutation.
# 2. zrealm_curcall_constnil.gno, a *ConstExpr callee at the cur-call check, passes at head.
set -u
R=${1:?clone path}
cd "$R"
P=gnovm/pkg/gnolang/preprocess.go
T=gnovm/pkg/gnolang/preprocess_tryevalstatic_test.go
F=gnovm/tests/files/zrealm_curcall_constnil.gno
cat >> "$T" <<'EOF'

// TestTryEvalStatic_ErrSetOnEvalFailure: a preprocessed expression whose
// evaluation fails must report a non-nil err.
func TestTryEvalStatic_ErrSetOnEvalFailure(t *testing.T) {
	t.Parallel()
	db := memdb.NewMemDB()
	tm2 := dbadapter.StoreConstructor(db, stypes.StoreOptions{})
	st := NewStore(nil, tm2, tm2)
	m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.land/r/test/tryevaliface", Store: st})
	defer m.Release()
	pn, _ := m.RunMemPackage(&std.MemPackage{
		Type:  MPUserProd,
		Name:  "tryevaliface",
		Path:  "gno.land/r/test/tryevaliface",
		Files: []*std.MemFile{{Name: "a.gno", Body: "package tryevaliface\n\ntype Fooer interface{ Foo(cur realm) }\n\nvar fooer Fooer\n"}},
	}, true)

	// An interface method has no concrete func until call time.
	x := Preprocess(st, pn, Sel(Nx("fooer"), "Foo")).(Expr)
	_, err := tryEvalStatic(st, pn, pn, x)

	require.Error(t, err, "err must be non-nil when evaluating a preprocessed expression fails")
}
EOF
echo "== 1a: head"
go test -count=1 -v -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/ 2>&1 | grep -E -- '--- |^ok|^FAIL' | head
echo "== 1b: head, every panic but the not-preprocessed guard dropped"
python3 - "$P" <<'PY'
import sys; p=sys.argv[1]; s=open(p).read()
old="\t\t\tif r == nil {\n\t\t\t\treturn // FIX: no panic -> eval succeeded -> leave err nil\n\t\t\t}\n"
assert s.count(old)==1
new=old+"\t\t\tif !strings.Contains(fmt.Sprint(r), \"not yet preprocessed\") {\n\t\t\t\treturn\n\t\t\t}\n"
open(p,"w").write(s.replace(old,new))
PY
go test -count=1 -v -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/ 2>&1 | grep -E -- '--- |^ok|^FAIL' | head
git checkout -- "$P" "$T"

echo "== 2: filetest at head"
cat > "$F" <<'EOF'
// PKGPATH: gno.land/r/test/curnil
package curnil

// A *ConstExpr callee reaches the preprocess cur-call check; its nil
// func value defers the call to the runtime.
func Bar(cur realm) {
	(func(cur realm))(nil)(cur)
}

func main(cur realm) {
	Bar(cur)
}

// Error:
// runtime error: call of nil function
EOF
go test -count=1 -v -run 'TestFiles/zrealm_curcall_constnil.gno$' ./gnovm/pkg/gnolang/ 2>&1 | grep -E -- '--- |^ok|^FAIL|Expected|^\s+[-+]' | head
rm -f "$F"
