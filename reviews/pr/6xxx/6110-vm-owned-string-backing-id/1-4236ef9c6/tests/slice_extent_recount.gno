// MAXALLOC: 500000000
// PKGPATH: gno.land/r/test
package test

import (
	"runtime"
	"strings"
)

var kept string

func main() {
	big := strings.Repeat("a", 100000)
	kept = big[0:1] // 1-byte slice of a 100000-byte backing
	big = ""        // source dies
	runtime.GC()
	println("after GC: ", runtime.MemStats())
	println(len(kept))
}

// Output:
// x
