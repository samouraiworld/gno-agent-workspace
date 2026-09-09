// PKGPATH: gno.land/r/objidrec
package objidrec

import (
	"chain/runtime"
	"errors"
)

type S struct{ n int }

func try(label string, f func() string) {
	defer func() {
		if r := recover(); r != nil {
			println(label, "PANIC-RECOVERED:", r)
		}
	}()
	println(label, "->", f())
}

func main(cur realm) {
	var np *S
	var ni interface{}
	var m map[string]int
	var fn func()
	e := errors.New("boom")
	s := S{1}

	try("int   ", func() string { return runtime.ObjectID(42) })
	try("string", func() string { return runtime.ObjectID("hello") })
	try("nilptr", func() string { return runtime.ObjectID(np) })
	try("nilifc", func() string { return runtime.ObjectID(ni) })
	try("nilmap", func() string { return runtime.ObjectID(m) })
	try("nilfn ", func() string { return runtime.ObjectID(fn) })
	try("err   ", func() string { return runtime.ObjectID(e) })
	try("struct", func() string { return runtime.ObjectID(s) })
	try("&struct", func() string { return runtime.ObjectID(&s) })
	try("method", func() string { return runtime.ObjectID(try) })
	try("closure", func() string { return runtime.ObjectID(func() {}) })
	try("slice ", func() string { return runtime.ObjectID([]int{1, 2}) })
	try("emptysl", func() string { return runtime.ObjectID([]int{}) })
	try("nilslice", func() string { var z []int; return runtime.ObjectID(z) })
	try("chanish", func() string { return runtime.ObjectID(runtime.ObjectID) })
}

// Output:
// x
