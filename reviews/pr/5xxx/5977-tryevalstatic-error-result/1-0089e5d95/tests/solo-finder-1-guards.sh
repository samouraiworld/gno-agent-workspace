#!/usr/bin/env bash
# Repro, from a plain clone of gnolang/gno:
#   git fetch origin pull/5977/head && git checkout --detach 0089e5d957dd05eb3892122d050c04a0fda6a1e9
#   bash solo-finder-1-guards.sh <path-to-clone>
# Measures, at head 0089e5d95 of gnolang/gno#5977:
#   A. revert the recover() nil guard only -> TestTryEvalStatic_ErrNilOnSuccess must fail
#   B. revert the caller branch only (err == nil skip) -> unit tests vs zrealm_crossrealm17b
#   C. count *ConstExpr callees reaching the `cur` same-realm check over TestFiles
# Every mutation is restored with git checkout before the next one.
# Run C reads only the TRYEVAL_ lines: under a Go toolchain newer than go.mod (1.27.1 here)
# TestFiles exits 1 on unrelated TypeCheckError message diffs (assign33.gno, eql_0b4.gno, ...).
set -u
R=${1:?clone path}
cd "$R"
P=gnovm/pkg/gnolang/preprocess.go
restore() { git checkout -- "$P"; }
OUT=$(mktemp)

echo "== A: recover guard reverted"
python3 - "$P" <<'PY'
import sys; p=sys.argv[1]; s=open(p).read()
old="\t\t\tif r == nil {\n\t\t\t\treturn // FIX: no panic -> eval succeeded -> leave err nil\n\t\t\t}\n"
assert s.count(old)==1; open(p,"w").write(s.replace(old,""))
PY
go test -count=1 -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/ 2>&1 | grep -E 'FAIL|ok|recovered panic|--- ' | head
restore

echo "== B: caller branch reverted (err == nil skips)"
python3 - "$P" <<'PY'
import sys; p=sys.argv[1]; s=open(p).read()
old="tryEvalStatic(store, ctxpn, last, n.Func)\n\t\t\t\t\t\t\t\tif err != nil {\n"
assert s.count(old)==1; open(p,"w").write(s.replace(old,old.replace("err != nil","err == nil")))
PY
go test -count=1 -run 'TestTryEvalStatic' ./gnovm/pkg/gnolang/ 2>&1 | grep -E 'FAIL|^ok|--- ' | head
go test -count=1 -run 'TestFiles/zrealm_crossrealm17b.gno' ./gnovm/pkg/gnolang/ 2>&1 | grep -E 'FAIL|^ok|--- ' | head
restore

echo "== C: *ConstExpr callee count over TestFiles"
python3 - "$P" <<'PY'
import sys; p=sys.argv[1]; s=open(p).read()
old="ftv, err := tryEvalStatic(store, ctxpn, last, n.Func)\n"
assert s.count(old)==1
new=old+"\t\t\t\t\t\t\t\tif _, isCx := n.Func.(*ConstExpr); isCx {\n\t\t\t\t\t\t\t\t\tprintln(\"TRYEVAL_CONSTFUNC\", n.String())\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tprintln(\"TRYEVAL_ANY err=\", err != nil, \"undef=\", ftv.IsUndefined(), \"fvnil=\", ftv.GetUnboundFunc() == nil, n.Func.String())\n"
open(p,"w").write(s.replace(old,new))
PY
go test -v -count=1 -timeout 30m -run 'TestFiles$' ./gnovm/pkg/gnolang/ > "$OUT" 2>&1
echo "TestFiles exit: $?"
echo "cur-check sites hit: $(grep -c TRYEVAL_ANY "$OUT")"
echo "ConstExpr callees:   $(grep -c TRYEVAL_CONSTFUNC "$OUT")"
grep TRYEVAL_CONSTFUNC "$OUT" | sort | uniq -c | head
echo "err/undef/fvnil tally over cur-check sites:"
grep -o 'TRYEVAL_ANY err= [a-z]* undef= [a-z]* fvnil= [a-z]*' "$OUT" | sort | uniq -c
echo "sample callees with err != nil:"
grep 'TRYEVAL_ANY err= true' "$OUT" | sed 's/.*fvnil= [a-z]* //' | sort | uniq -c | sort -rn | head -12
echo "sample callees with err == nil and fv == nil:"
grep 'TRYEVAL_ANY err= false undef= [a-z]* fvnil= true' "$OUT" | sed 's/.*fvnil= [a-z]* //' | sort | uniq -c | sort -rn | head -12
restore
