// PKGPATH: gno.land/r/objidalias
package objidalias

import (
	"chain/runtime"

	tests "gno.land/r/tests/vm"
)

type leaf struct{ n int }

type holder struct {
	A leaf
	B leaf
}

var (
	h   holder
	arr [2]leaf
)

func main(cur realm) {
	// force a realm finalize so object IDs are stamped, the same trick
	// grc20reg.setEntry uses.
	_ = tests.CurrentRealmPath(cross(cur))

	println("h    ", runtime.ObjectID(&h))
	println("h.A  ", runtime.ObjectID(&h.A))
	println("h.B  ", runtime.ObjectID(&h.B))
	println("arr  ", runtime.ObjectID(&arr))
	println("arr0 ", runtime.ObjectID(&arr[0]))
	println("arr1 ", runtime.ObjectID(&arr[1]))
	println("A==B ", runtime.ObjectID(&h.A) == runtime.ObjectID(&h.B))
	println("a0==a1", runtime.ObjectID(&arr[0]) == runtime.ObjectID(&arr[1]))
}

// Output:
// x
