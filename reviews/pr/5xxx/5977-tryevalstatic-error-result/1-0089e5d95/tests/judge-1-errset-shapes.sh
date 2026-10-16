#!/usr/bin/env bash
# Judge check for candidates 1 and 1002, from a clone of gnolang/gno at 0089e5d95:
#   bash judge-1-errset-shapes.sh <clone>
# 1. logs the err TestTryEvalStatic_ErrSetOnFailure's input produces, beside the err a
#    preprocessed callee the sole caller can meet produces;
# 2. makes tryEvalStatic swallow every panic except EvalStatic's not-preprocessed guard,
#    then reruns the PR's two tests: both PASS means the negative control pins only that guard.
set -u
R=${1:?clone path}
cd "$R"
P=gnovm/pkg/gnolang/preprocess.go
T=gnovm/pkg/gnolang/judge_tryeval_test.go
cat > "$T" <<'EOF'
package gnolang

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	stypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestJudgeTryEvalErrShapes(t *testing.T) {
	db := memdb.NewMemDB()
	tm2 := dbadapter.StoreConstructor(db, stypes.StoreOptions{})
	st := NewStore(nil, tm2, tm2)
	m := NewMachineWithOptions(MachineOptions{PkgPath: "gno.land/r/test/judgeeval", Store: st})
	defer m.Release()
	body := "package judgeeval\n\ntype Fooer interface{ Foo(cur realm) }\n\nvar fooer Fooer\n\nfunc Foo(cur realm) {}\n\nfunc get() func(cur realm) { return Foo }\n"
	pn, _ := m.RunMemPackage(&std.MemPackage{
		Type: MPUserProd, Name: "judgeeval", Path: "gno.land/r/test/judgeeval",
		Files: []*std.MemFile{{Name: "a.gno", Body: body}},
	}, true)
	_, err := tryEvalStatic(st, pn, pn, Nx("Undeclared"))
	t.Logf("JUDGE unpreprocessed Nx(Undeclared): err=%v", err)
	x := Preprocess(st, pn, Call(Nx("get"))).(Expr)
	tv, err := tryEvalStatic(st, pn, pn, x)
	t.Logf("JUDGE preprocessed get(): err=%v undefined=%v", err, tv.IsUndefined())
	y := Preprocess(st, pn, Sel(Nx("fooer"), "Foo")).(Expr)
	tv, err = tryEvalStatic(st, pn, pn, y)
	t.Logf("JUDGE preprocessed fooer.Foo: err=%v undefined=%v", err, tv.IsUndefined())
}
EOF
echo "== 1: err per input shape"
go test -count=1 -v -run 'TestJudgeTryEvalErrShapes' ./gnovm/pkg/gnolang/ 2>&1 | grep -E 'JUDGE|FAIL|^ok|panic' | head
rm -f "$T"

echo "== 2: swallow every panic but the not-preprocessed guard, rerun the PR's tests"
python3 - "$P" <<'PY'
import sys; p=sys.argv[1]; s=open(p).read()
old="\t\t\tif r == nil {\n\t\t\t\treturn // FIX: no panic -> eval succeeded -> leave err nil\n\t\t\t}\n"
assert s.count(old)==1
new=old+"\t\t\tif !strings.Contains(fmt.Sprint(r), \"not yet preprocessed\") {\n\t\t\t\treturn\n\t\t\t}\n"
open(p,"w").write(s.replace(old,new))
PY
go test -count=1 -v -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/ 2>&1 | grep -E -- '--- |FAIL|^ok' | head
git checkout -- "$P"
