// Repro, from a plain clone of github.com/gnolang/gno:
//
//	mkdir -p examples/gno.land/r/demo/objidslice
//	printf 'module = "gno.land/r/demo/objidslice"\ngno = "0.9"\n' \
//	    > examples/gno.land/r/demo/objidslice/gnomod.toml
//	cp objectid_slice_alias_filetest.gno examples/gno.land/r/demo/objidslice/
//	cd examples/gno.land/r/demo/objidslice
//	GNOROOT=$(git rev-parse --show-toplevel) go run ../../../../../gnovm/cmd/gno test -v .
//
// A []grc20.Token aliases as much as a two-field struct: the slice and its
// first element answer with one address. Three separate package-level vars
// answer with three, so the collapse is bounded to containers and slices. It
// passes at the reviewed head; the equal pair above is the defect.

// PKGPATH: gno.land/r/demo/objidslice
package objidslice

import (
	"chain/runtime"

	"gno.land/p/demo/tokens/grc20"
	tests "gno.land/r/tests/vm"
)

type leaf struct{ n int }

var (
	sl         []grc20.Token
	iA, iB, sA leaf
)

func init(cur realm) {
	a, _ := grc20.NewToken("Alpha", "ALPHA", 4, cur)
	b, _ := grc20.NewToken("Beta", "BETA", 4, cur)
	sl = []grc20.Token{*a, *b}
}

func main(cur realm) {
	// Force a realm finalize so the object IDs are stamped, the same trick
	// grc20reg.setEntry uses.
	_ = tests.CurrentRealmPath(cross(cur))

	println("slice     ", runtime.ObjectID(sl))
	println("slice[0]  ", runtime.ObjectID(&sl[0]))
	println("slice==&0 ", runtime.ObjectID(sl) == runtime.ObjectID(&sl[0]))
	println("iA        ", runtime.ObjectID(&iA))
	println("iB        ", runtime.ObjectID(&iB))
	println("sA        ", runtime.ObjectID(&sA))
	println("three differ:", runtime.ObjectID(&iA) != runtime.ObjectID(&iB) &&
		runtime.ObjectID(&iB) != runtime.ObjectID(&sA) &&
		runtime.ObjectID(&iA) != runtime.ObjectID(&sA))
}

// Output:
// slice      g13n8ye4wt2xc74lz50z2sv9wj7xk87kv95wv6as
// slice[0]   g13n8ye4wt2xc74lz50z2sv9wj7xk87kv95wv6as
// slice==&0  true
// iA         g1hdh4gwtcq7w2v0e8sfjj2wsvdmdmss5ca5xdfe
// iB         g1jllccjf4klsl6yldzlr4j88459tt7cvujhjuvu
// sA         g194xmlx0d34y90n0m2kdwzwrx89l9lc252zfszf
// three differ: true
