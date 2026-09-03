// MAXALLOC: 50000
// PKGPATH: gno.land/r/test
package test

import (
	"runtime"
)

var f func() string

func init() {
	s := `aaaa`
	f = func() string { return s }
}

func main() {
	s1 := f()
	runtime.GC()
	println("after GC: ", runtime.MemStats())
	println(len(s1))
}

// Output:
// after GC:  Allocator{maxBytes:50000, bytes:8548}
// 4
