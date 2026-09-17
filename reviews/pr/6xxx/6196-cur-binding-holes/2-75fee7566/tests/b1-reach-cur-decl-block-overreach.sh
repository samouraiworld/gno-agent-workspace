#!/usr/bin/env bash
# b1-reach-cur-decl-block-overreach.sh
#
# What it measures
#   isCrossingCurParam (gnovm/pkg/gnolang/preprocess.go:6111 at 6d88deba7) keys the
#   three `cur` write rules on the DECLARING BLOCK NODE of the name, not on the
#   binding. Any name `cur` declared directly in a crossing FuncDecl / FuncLitExpr
#   block is refused, whatever its type and whatever it is: a named result, a
#   plain local, a method receiver. The shape that reaches it is a crossing
#   function whose first realm parameter is unnamed (`_ realm`), which frees the
#   name `cur` for something else in the same block.
#
#   Six shapes, run at the PR head and at the merge base. Five compile and run at
#   the base and are rejected at the head; the sixth (a named result) is rejected
#   at both and is pre-existing.
#
# Exact repro from a plain clone
#   git clone https://github.com/gnolang/gno
#   cd gno
#   bash <this file>                       # uses $PWD as the checkout
#   HEAD_SHA=6d88deba71165b165641b65541259d00aba20377 \
#   BASE_SHA=b7845fb2da9a21bfb895e1c008f016776f8342d0 \
#     bash <this file>
#
#   Or by hand, in a worktree at either sha:
#     cp <the .gno files this script writes> gnovm/tests/files/
#     go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzreach'
#
# Measured 2026-09-18, go1.25.9.
#
# Result at 6d88deba7 (head)   6 / 6 FAIL
# Result at b7845fb2d (base)   1 / 6 FAIL  (zzreach_cur_named_result.gno only)

set -u

HEAD_SHA="${HEAD_SHA:-6d88deba71165b165641b65541259d00aba20377}"
BASE_SHA="${BASE_SHA:-b7845fb2da9a21bfb895e1c008f016776f8342d0}"
REPO="${REPO:-$PWD}"
TMP="${TMP:-$(mktemp -d)}"

write_cases() {
	d="$1/gnovm/tests/files"

	# 1. A plain int local named `cur`. The crossing parameter is `_ realm`, so
	#    `cur` is free. Base: prints 5. Head: "cannot reassign the crossing `cur`
	#    parameter".
	cat >"$d/zzreach_cur_local_int.gno" <<'EOF'
// PKGPATH: gno.land/r/demo/curreach2
package curreach2

func localIntCur(_ realm) int {
	var cur int
	cur = 5
	return cur
}

func main(cur realm) {
	println(localIntCur(cross(cur)))
}

// Output:
// 5
EOF

	# 2. The address of that same int local. Base: prints 7. Head: "cannot take
	#    the address of a realm-typed `cur`" — the value is an int.
	cat >"$d/zzreach_cur_addr_int.gno" <<'EOF'
// PKGPATH: gno.land/r/demo/curreach3
package curreach3

func addrIntCur(_ realm) int {
	var cur int
	p := &cur
	*p = 7
	return cur
}

func main(cur realm) {
	println(addrIntCur(cross(cur)))
}

// Output:
// 7
EOF

	# 3. The same int local as a range assign target. Base: prints 9. Head:
	#    "cannot assign to a realm-typed `cur` in a range clause".
	cat >"$d/zzreach_cur_range_int.gno" <<'EOF'
// PKGPATH: gno.land/r/demo/curreach4
package curreach4

func rangeIntCur(_ realm) int {
	var cur int
	for _, cur = range []int{1, 2, 9} {
	}
	return cur
}

func main(cur realm) {
	println(rangeIntCur(cross(cur)))
}

// Output:
// 9
EOF

	# 4. A method receiver named `cur` on a crossing method. Base: prints 100.
	#    Head: "cannot reassign the crossing `cur` parameter".
	cat >"$d/zzreach_cur_recv.gno" <<'EOF'
// PKGPATH: gno.land/r/demo/curreach5
package curreach5

type counter struct{ n int }

func (cur *counter) Bump(_ realm) int {
	cur.n++
	cur = &counter{n: 100}
	return cur.n
}

func main(cur realm) {
	c := &counter{}
	println(c.Bump(cross(cur)))
}

// Output:
// 100
EOF

	# 5. The same int local inside a crossing function LITERAL. Base: prints 3.
	#    Head: "cannot reassign the crossing `cur` parameter".
	cat >"$d/zzreach_cur_lit_int.gno" <<'EOF'
// PKGPATH: gno.land/r/demo/curreach6
package curreach6

func main(cur realm) {
	f := func(_ realm) int {
		var cur int
		cur = 3
		return cur
	}
	println(f(cross(cur)))
}

// Output:
// 3
EOF

	# 6. Control: a realm-typed named result of a crossing function. Rejected at
	#    BOTH revisions, so it is pre-existing and not a regression of this diff.
	#    It is the shape the open reviewer thread raises.
	cat >"$d/zzreach_cur_named_result.gno" <<'EOF'
// PKGPATH: gno.land/r/demo/curreach1
package curreach1

func namedResultCrossing(_ realm) (cur realm) {
	cur = nil
	return cur
}

func main(cur realm) {
	_ = namedResultCrossing(cross(cur))
	println("ok")
}

// Output:
// ok
EOF
}

run_at() {
	sha="$1"
	label="$2"
	wt="$TMP/$label"
	git -C "$REPO" worktree add --detach "$wt" "$sha" >/dev/null 2>&1 || {
		echo "cannot create worktree at $sha"
		return 1
	}
	write_cases "$wt"
	echo "===== $label ($sha) ====="
	(cd "$wt" && go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzreach' 2>&1) |
		grep -E '^\s*(---|    ---|        files_test.go:135)'
	git -C "$REPO" worktree remove --force "$wt" >/dev/null 2>&1
}

run_at "$BASE_SHA" base
run_at "$HEAD_SHA" head
