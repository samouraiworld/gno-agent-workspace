// PKGPATH: gno.land/r/demo/reflectcross2
package reflectcross2

// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/6214/head && git checkout 58b2692067d6545410b99c71f33bdf506caadaa0
//   cp <this file> gnovm/tests/files/zrealm_reflect_zprobe_foreign.gno
//   cd gnovm/pkg/gnolang && go test . -run 'TestFiles/zrealm_reflect_(foreign_finalize|zprobe_foreign)' -v
// Expect both PASS. Line 1 below prints `theirs stamped before SetFooer: true`:
// the PR's zrealm_reflect_foreign_finalize.gno hands crossrealm an object that
// its own init finalize already stamped, so the foreign branch never runs.
// Mutation that must turn a foreign-finalize test red, realm.go:2038:
//   sed -i '2038s/targetRlm = rlm.touchForeignRealm(store, oid.PkgID)/_ = rlm.touchForeignRealm(store, oid.PkgID)/' gnovm/pkg/gnolang/realm.go
// Re-run: zrealm_reflect_foreign_finalize.gno still PASSES; this file FAILS
// (fresh takes tick :47 from crossrealm's clock instead of :10 from its own).
// Restore: git checkout -- gnovm/pkg/gnolang/realm.go

import (
	"chain/reflect"

	crossrealm "gno.land/r/tests/vm/crossrealm"
)

type fooer struct{ n int }

func (fooer) Foo(cur realm) {}
func (fooer) Bar()          {}

var theirs = &fooer{n: 2}

var keep *fooer

func stash(cur realm, v *fooer) { keep = v }

func main(cur realm) {
	t0, ok0 := reflect.Of(theirs)
	println("theirs stamped before SetFooer:", ok0, t0.ID())

	fresh := &fooer{n: 3}
	crossrealm.SetFooer(cross(cur), fresh)
	f, fok := reflect.Of(fresh)
	println("fresh stamped by foreign finalize:", fok, f.ID(), f.PkgPath())

	other := &fooer{n: 4}
	stash(cross(cur), other)
	o, ook := reflect.Of(other)
	println("other stamped by own finalize:", ook, o.ID(), o.PkgPath())
	println("ids differ:", f.ID() != o.ID())
}

// Output:
// theirs stamped before SetFooer: true 02889f73cd78d086728994511fad24d4807b6296:4
// fresh stamped by foreign finalize: true 02889f73cd78d086728994511fad24d4807b6296:10 gno.land/r/demo/reflectcross2
// other stamped by own finalize: true 02889f73cd78d086728994511fad24d4807b6296:12 gno.land/r/demo/reflectcross2
// ids differ: true
