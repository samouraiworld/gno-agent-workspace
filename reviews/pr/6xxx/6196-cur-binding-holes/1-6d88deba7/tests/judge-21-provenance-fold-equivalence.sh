#!/usr/bin/env bash
# judge-21: the 2155 provenance switch and isCrossingCurParam decide the same
# case identically, so folding one into the other changes no behaviour.
# Asserts: head and b1-refactor-provenance-fold.patch produce byte-identical
# output on `func F(_ realm) (cur realm) { Target(cur) }`, the one shape where
# the two copies could disagree. Passes at the reviewed head.
#
# Run from a plain clone of gnolang/gno:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6196/head && git checkout 6d88deba71165b165641b65541259d00aba20377
#   curl -fsSL -o /tmp/fold.patch \
#     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6196-cur-binding-holes/1-6d88deba7/tests/b1-refactor-provenance-fold.patch
#   bash <this file> /tmp/fold.patch
set -u
PATCH="${1:?usage: judge-21-provenance-fold-equivalence.sh <path to b1-refactor-provenance-fold.patch>}"
F=gnovm/tests/files/zjudge_curfold.gno

cat > "$F" <<'EOF'
// PKGPATH: gno.land/r/demo/curfoldjudge
package curfoldjudge

func Target(cur realm) {
	_ = cur.IsCurrent()
}

// F is crossing (first param is realm) but that param is unnamed, so `cur`
// here is the named result. Both copies of the declaration test resolve it to
// this crossing FuncDecl and let it through the provenance check.
func F(_ realm) (cur realm) {
	Target(cur)
	return
}

func main(cur realm) {
	F(cross(cur))
	println("unreached")
}

// Output:
// JUDGE_PLACEHOLDER
EOF

run() { go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zjudge_curfold.gno$' 2>&1 |
	grep -oE "(unexpected panic:|Error:).*"; }

run > /tmp/judge21-head.txt
git apply "$PATCH" || { echo "patch failed"; exit 2; }
run > /tmp/judge21-patched.txt
git checkout -- gnovm/pkg/gnolang/preprocess.go
rm -f "$F"

# IS:     both trees accept Target(cur) at preprocess and panic at runtime.
diff /tmp/judge21-head.txt /tmp/judge21-patched.txt &&
	grep -q "was entered without cross()" /tmp/judge21-head.txt &&
	echo "PASS: identical, both reach the runtime stale-capture panic"
# SHOULD (once ltzmaxwell's `&& ft.Params[0].Name == \"cur\"` lands, in the one
# folded site instead of four arms): both refuse at preprocess with
# "only the `cur` argument of a containing crossing function maybe passed by cross-call".
