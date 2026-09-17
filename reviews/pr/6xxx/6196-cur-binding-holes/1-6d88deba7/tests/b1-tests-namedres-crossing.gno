// PKGPATH: gno.land/r/demo/curnamedres
package curnamedres

// The cell zrealm_cur_other_legal.gno leaves out. That file pins "a realm-typed
// `cur` that is not a crossing parameter keeps compiling" with a named result
// of a NON-crossing function; the same shape inside a CROSSING function is
// still refused, because isCrossingCurParam tests the declaring node's
// signature and not whether the written name is that signature's first
// parameter. An unnamed first realm parameter is what separates the two.
//
// Repro, from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/files/zrealm_cur_namedres.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_namedres.gno$'
//
// Head 6d88deba7 and merge base b7845fb2d both FAIL the same way, so the
// refusal predates the branch; the branch narrows the rule around it and its
// new positive control does not cover this cell:
//
//   unexpected panic: gno.land/r/demo/curnamedres/zrealm_cur_namedres.gno:27:2-11:
//   cannot reassign the crossing `cur` parameter: it names the realm this frame
//   is executing as, and that binding is fixed for the life of the frame

func F(_ realm) (cur realm) {
	cur = nil
	return cur
}

func main() {
	println("ok")
}

// Output:
// ok
