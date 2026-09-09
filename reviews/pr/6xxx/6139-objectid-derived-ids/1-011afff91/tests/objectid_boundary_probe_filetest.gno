// PKGPATH: gno.land/r/demo/objidprobe
package objidprobe

import "chain/runtime"

func try(label string, f func() string) {
	defer func() {
		if r := recover(); r != nil {
			println(label, "-> recovered:", r)
			return
		}
	}()
	v := f()
	println(label, "-> returned", "["+v+"]")
}

var kept func() int

func main(cur realm) {
	try("nil", func() string { return runtime.ObjectID(nil) })
	try("int", func() string { return runtime.ObjectID(42) })
	try("string", func() string { return runtime.ObjectID("x") })
	try("nil map", func() string { var m map[string]int; return runtime.ObjectID(m) })
	try("live map", func() string { return runtime.ObjectID(map[string]int{"a": 1}) })
	try("func value", func() string { return runtime.ObjectID(func() {}) })
	try("plain local ptr", func() string { x := 1; return runtime.ObjectID(&x) })
	try("heap-captured local ptr", func() string {
		x := 1
		kept = func() int { return x }
		return runtime.ObjectID(&x)
	})
	try("nil pointer", func() string { var p *int; return runtime.ObjectID(p) })
	try("slice", func() string { return runtime.ObjectID([]int{1, 2, 3}) })
	try("nil slice", func() string { var s []int; return runtime.ObjectID(s) })
	try("empty slice literal", func() string { return runtime.ObjectID([]int{}) })
}
