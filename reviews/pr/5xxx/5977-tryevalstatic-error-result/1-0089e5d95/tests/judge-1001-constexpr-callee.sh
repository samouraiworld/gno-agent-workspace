#!/usr/bin/env bash
# Judge check for candidates 2 and 1001, from two clones of gnolang/gno, one at the
# PR head 0089e5d95 and one at its merge base 27b5b8e24:
#   bash judge-1001-constexpr-callee.sh <head clone> <base clone>
# 1. builds a *ConstExpr callee at the cur-call check, `(func(cur realm))(nil)(cur)`,
#    and runs it at head and base with a print marking the *ConstExpr callee;
# 2. at head, restores the base's skip for a *ConstExpr callee and reruns the PR's
#    tests and the zrealm_crossrealm filetests: all green means no test pins the path.
set -u
H=${1:?head clone}
B=${2:?base clone}
P=gnovm/pkg/gnolang/preprocess.go
F=gnovm/tests/files/zrealm_curcall_constnil.gno

run_filetest() {
	cd "$1"
	cat > "$F" <<'EOF'
// PKGPATH: gno.land/r/test/curnil
package curnil

func Bar(cur realm) {
	(func(cur realm))(nil)(cur)
}

func main(cur realm) {
	Bar(cur)
}

// Error:
// JUDGE_PLACEHOLDER
EOF
	python3 - "$P" <<'PY'
import sys; p=sys.argv[1]; s=open(p).read()
old="ftv, err := tryEvalStatic(store, ctxpn, last, n.Func)\n"
assert s.count(old)==1
new=old+"\t\t\t\t\t\t\t\tif _, isCx := n.Func.(*ConstExpr); isCx {\n\t\t\t\t\t\t\t\t\tprintln(\"JUDGE_CONSTFUNC err=\", err != nil, \"undef=\", ftv.IsUndefined(), \"fvnil=\", ftv.GetUnboundFunc() == nil)\n\t\t\t\t\t\t\t\t}\n"
open(p,"w").write(s.replace(old,new))
PY
	go test -count=1 -v -run 'TestFiles/zrealm_curcall_constnil.gno$' ./gnovm/pkg/gnolang/ 2>&1 \
		| grep -E 'JUDGE_CONSTFUNC|Error|error|panic|nil|--- |^ok|^FAIL' | grep -v PLACEHOLDER | head -12
	git checkout -- "$P"
	rm -f "$F"
}

echo "== 1a: head"
run_filetest "$H"
echo "== 1b: base"
run_filetest "$B"

echo "== 2: head with the base's *ConstExpr skip restored"
cd "$H"
python3 - "$P" <<'PY'
import sys; p=sys.argv[1]; s=open(p).read()
old="ftv, err := tryEvalStatic(store, ctxpn, last, n.Func)\n"
assert s.count(old)==1
new=old+"\t\t\t\t\t\t\t\tif _, isCx := n.Func.(*ConstExpr); isCx {\n\t\t\t\t\t\t\t\t\terr = fmt.Errorf(\"judge: skip ConstExpr callee\")\n\t\t\t\t\t\t\t\t}\n"
open(p,"w").write(s.replace(old,new))
PY
go test -count=1 -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/ 2>&1 | grep -E 'FAIL|^ok' | head -5
go test -count=1 -v -run 'TestFiles/zrealm_crossrealm' ./gnovm/pkg/gnolang/ 2>&1 | grep -cE -- '--- PASS'
go test -count=1 -v -run 'TestFiles/zrealm_crossrealm' ./gnovm/pkg/gnolang/ 2>&1 | grep -E -- '--- FAIL|^ok|^FAIL' | head -5
git checkout -- "$P"
