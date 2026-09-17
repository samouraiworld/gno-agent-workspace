// Repro: installInheritedCur's identity guard is a bare Go panic, so gno
// `recover()` never sees it. The deferred recover in F below does not run; the
// panic unwinds past the VM's op loop. Invariant catalog, VM-fault
// recoverability: a fault user .gno can trigger must surface through the VM
// Exception machinery.
//
// Head (6d88deba7): the filetest reports "unexpected panic: crossing function
// gno.land/r/demo/curzz3.Target was entered without cross()...", and nothing
// from the deferred recover is printed.
// Pre-existing from #6193; this PR rewrites that panic's message.
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba7
//   cp <this file> gnovm/tests/files/zzb1recover.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zzb1recover.gno$'

// PKGPATH: gno.land/r/demo/curzz3
package curzz3

func Target(cur realm) {
	println("target ran")
}

func F(_ realm) (cur realm) {
	defer func() {
		if r := recover(); r != nil {
			println("recovered")
		}
	}()
	Target(cur)
	return
}

func main(cur realm) {
	F(cross(cur))
	println("done")
}

// Output:
// recovered
// done
