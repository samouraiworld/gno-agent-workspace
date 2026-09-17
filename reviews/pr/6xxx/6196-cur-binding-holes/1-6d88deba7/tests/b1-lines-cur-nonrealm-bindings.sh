#!/usr/bin/env bash
# b1-lines-cur-nonrealm-bindings.sh — PR 6196, bundle 1, angle "lines".
#
# What it measures
#   isCrossingCurParam (gnovm/pkg/gnolang/preprocess.go:6111) resolves the written
#   name `cur` to its declaring block node and refuses the write when that node is a
#   crossing FuncDecl/FuncLitExpr. It does not test the binding's static type, and it
#   does not test that the crossing function's first parameter is actually named `cur`.
#   A crossing function whose leading realm parameter is unnamed (`func F(_ realm, ...)`,
#   56 occurrences in examples/ + gnovm/ at this sha) therefore has every `cur` declared
#   at its own block level treated as the frame identity, whatever its type.
#
#   Four shapes, run at the head and at the merge base:
#     b1local   func f(_ realm, cur int)      { cur = cur + 1 }   second param, int
#     b1define  func g(_ realm)               { cur := 41 }       body-level local, int
#     b1ref     func h(_ realm, cur int) *int { return &cur }     address-of, int
#     b1result  func k(_ realm) (cur realm)   { cur = nil }       realm-typed named result
#   The first three pass on the base and fail at the head: a compile regression.
#   The fourth fails on both: pre-existing, and the member the PR's own fix 2
#   ("named results like func F(x int) (cur realm)") did not reach — its fixture
#   gnovm/tests/files/zrealm_cur_other_legal.gno only spells the non-crossing form.
#
# Repro from a plain clone
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6196/head:pr6196
#   git worktree add --detach /tmp/h 6d88deba71165b165641b65541259d00aba20377
#   git worktree add --detach /tmp/b b7845fb2da9a21bfb895e1c008f016776f8342d0
#   # pin the toolchain the tree asks for: grep -m1 '^go ' go.mod  ->  go1.25.9
#   bash b1-lines-cur-nonrealm-bindings.sh /tmp/h      # 3 regressions + 1 pre-existing
#   bash b1-lines-cur-nonrealm-bindings.sh /tmp/b      # only b1result fails
#
# Usage: b1-lines-cur-nonrealm-bindings.sh <path to a gno worktree>
set -u
REPO=${1:?usage: $0 <gno worktree>}
cd "$REPO" || exit 1
D=gnovm/tests/files

write() { # write <name> <body>
	cat > "$D/zrealm_cur_$1.gno" <<EOF
// PKGPATH: gno.land/r/demo/$1
package $1

$2

func main() {
	println("ok")
}

// Output:
// ok
EOF
}

write b1local 'func f(_ realm, cur int) int {
	cur = cur + 1
	return cur
}'
write b1define 'func g(_ realm) int {
	cur := 41
	cur = cur + 1
	return cur
}'
write b1ref 'func h(_ realm, cur int) *int {
	return &cur
}'
write b1result 'func k(_ realm) (cur realm) {
	cur = nil
	return cur
}'

printf '%-10s %-8s %s\n' shape verdict message
for n in b1local b1define b1ref b1result; do
	out=$(go test ./gnovm/pkg/gnolang/ -run "TestFiles/zrealm_cur_$n.gno\$" 2>&1)
	if printf '%s' "$out" | grep -q '^ok '; then
		printf '%-10s %-8s %s\n' "$n" COMPILES -
	else
		msg=$(printf '%s' "$out" | grep -m1 -o 'cannot [^"]*' | cut -c1-60)
		printf '%-10s %-8s %s\n' "$n" REFUSED "$msg"
	fi
	rm -f "$D/zrealm_cur_$n.gno"
done
