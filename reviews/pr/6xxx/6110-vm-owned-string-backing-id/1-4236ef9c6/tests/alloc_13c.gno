// MAXALLOC: 100000
// PKGPATH: gno.land/r/test

// Companion to alloc_13b: ONLY the slice survives. alloc_13b keeps a full
// copy (s1) beside the slice, so the backing stays reachable through a
// value whose Extent is the full length — the slice's own Extent is never
// load-bearing there. Here nothing but the slice is left, so the recount
// depends on GetSlice inheriting both ID and Extent.
//
//	s  := base + base   // 1024B backing, minted through NewString
//	s2 := s[1:11]       // slice, shares the backing
//	s, base = "", ""    // every full-length value dies
//
// After GC the full 1024B backing must still be counted exactly once,
// reached only through a 10-byte slice.
package test

import (
	"runtime"
)

func main() {
	base := `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd
eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee
ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
gggggggggggggggggggggggggg` // 512 bytes
	s := base + base // 1024B fresh backing: charged and tracked
	s2 := s[1:11]    // slice shares the backing; the only survivor
	s = ""           // source dies
	base = ""        // literal ref dies too
	runtime.GC()
	println("after GC: ", runtime.MemStats())
	println(len(s2))
}

// Output:
// after GC:  Allocator{maxBytes:100000, bytes:8872}
// 10
