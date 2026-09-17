// PKGPATH: gno.land/r/demo/currangedef
package currangedef

// Pins that the `if n.Op != DEFINE` wrapper around the RangeStmt `cur` rule
// (gnovm/pkg/gnolang/preprocess.go, TRANS_LEAVE *RangeStmt) is a guard the
// callee already applies: a DEFINE key/value is declared into the RangeStmt's
// OWN block, so isCrossingCurParam resolves the written name to the RangeStmt
// node, not to a crossing FuncDecl/FuncLitExpr, and answers false on its own.
//
// Repro from a plain clone, at the reviewed sha:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 6d88deba71165b165641b65541259d00aba20377
//	cp <this file> gnovm/tests/files/zzz_b1_range_cur_define.gno
//	go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzz_b1_range_cur_define.gno$'
//	# --- PASS (the DEFINE shapes are accepted with the guard in place)
//
//	python3 - <<'PY'
//	p = "gnovm/pkg/gnolang/preprocess.go"
//	s = open(p).read()
//	old = "\t\t\t\tif n.Op != DEFINE {\n\t\t\t\t\tfor _, lh := range []Expr{n.Key, n.Value} {"
//	new = "\t\t\t\tif true {\n\t\t\t\t\tfor _, lh := range []Expr{n.Key, n.Value} {"
//	assert s.count(old) == 1
//	open(p, "w").write(s.replace(old, new))
//	PY
//	go test ./gnovm/pkg/gnolang/ -run 'TestFiles/z'
//	# --- ok  github.com/gnolang/gno/gnovm/pkg/gnolang  8.9s
//	# every zrealm_cur* filetest plus this one still passes with the DEFINE
//	# carve-out disabled, so the guard changes no outcome.
//	git checkout -- gnovm/pkg/gnolang/preprocess.go

func main(cur realm) {
	for cur := range []int{1, 2} {
		_ = cur
	}
	for _, cur := range []int{1, 2} {
		_ = cur
	}
	println("ok")
}

// Output:
// ok
