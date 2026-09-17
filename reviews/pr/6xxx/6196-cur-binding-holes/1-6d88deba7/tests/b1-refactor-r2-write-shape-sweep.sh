#!/bin/sh
# Which write spellings reach the crossing `cur` slot, and which rule refuses
# each. Run from a plain clone of gnolang/gno at the reviewed sha:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git checkout 6d88deba71165b165641b65541259d00aba20377
#   sh <this file>
#
# Observed at 6d88deba7 (2026-09-18):
#
#   var    -> gno.land/r/demo/curvardecl/...: cur redeclared in this block
#             previous declaration at zzz_b1_var_cur_decl.gno:13:11
#             (refused by ParseMemPackageAsType, before preprocess; no write
#             rule involved)
#   decomp -> zzz_b1_decompose_cur.gno:17:2-19: cannot reassign the crossing
#             `cur` parameter: it names the realm this frame is executing as,
#             and that binding is fixed for the life of the frame
#             (the decompose rewrite replaces n.Lhs with .decompose_ names, but
#             the ASSIGN it splits out is re-Preprocessed and the AssignStmt
#             rule fires on the original Lhs)
#
# So neither shape escapes: the two holes this probe was written to look for
# are closed. What it does show is the shape of the rule set -- one refusal per
# statement kind, in four places, none of which knows about the others.
set -e
cat > gnovm/tests/files/zzz_b1_var_cur_decl.gno <<'GNO'
// PKGPATH: gno.land/r/demo/curvardecl
package curvardecl

func probe(cur realm) { _ = cur.IsCurrent() }

func main(cur realm) {
	var cur realm = cur.Previous()
	probe(cur)
}
GNO
cat > gnovm/tests/files/zzz_b1_decompose_cur.gno <<'GNO'
// PKGPATH: gno.land/r/demo/curdecompose
package curdecompose

func two(cur realm) (realm, int) { return cur.Previous(), 1 }

func probe(cur realm) { _ = cur.IsCurrent() }

func main(cur realm) {
	var x int
	cur, x = two(cur)
	_ = x
	probe(cur)
}
GNO
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzz_b1' 2>&1 | grep -E 'unexpected panic|--- FAIL' || true
rm gnovm/tests/files/zzz_b1_var_cur_decl.gno gnovm/tests/files/zzz_b1_decompose_cur.gno
