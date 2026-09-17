// PKGPATH: gno.land/r/demo/curpkglevel
package curpkglevel

// The package-level arm of fix 2, which no committed fixture covers.
//
// Both rewritten guards (preprocess.go:2583 address-of, preprocess.go:3011
// assign) dropped the `evalStaticTypeOf(ne) == gRealmType` test and now key on
// the declaring block node, so a realm-typed `cur` declared at package level
// became writable and addressable. The base's guards keyed on name plus type,
// so both lines below panic there. gnovm/tests/files/zrealm_cur_other_legal.gno
// claims to pin "a package-level var" but declares it as `pkgCur` (line 9), a
// name neither guard ever keyed on, so that arm is green on the base too; the
// two address-of fixtures (zrealm_cur_addr.gno, zrealm_cur_addr_arg.gno) pin
// only the parameter refusal.
//
// Repro from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp b1-reach-r2-pkglevel-cur-untested.gno \
//      gnovm/tests/files/zrealm_cur_pkglevel_probe.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_pkglevel_probe.gno$'
//
// Head 6d88deba7: ok  github.com/gnolang/gno/gnovm/pkg/gnolang  0.038s
// Merge base b7845fb2d: expected to panic at line 20 with
//   "cannot reassign the crossing `cur` parameter: ..."
// which is the compat break fix 2 says it repairs.
//
// The setter's first parameter is an int on purpose: a function whose first
// parameter is realm is crossing, and a crossing function's first realm
// parameter must be named `cur`.

var cur realm

func set(x int, r realm) {
	cur = r
	p := &cur
	_ = p
}

func main() {
	set(0, nil)
	println("ok")
}

// Output:
// ok
