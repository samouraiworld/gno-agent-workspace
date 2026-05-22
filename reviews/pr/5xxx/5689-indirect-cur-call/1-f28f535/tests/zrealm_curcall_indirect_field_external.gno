/* Run:
cp reviews/pr/5xxx/5689-indirect-cur-call/1-f28f535/tests/zrealm_curcall_indirect_field_external.gno \
   .worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_field_external.gno
go test -C .worktrees/gno-review-5689 -v -run 'TestFiles/^zrealm_curcall_indirect_field_external' ./gnovm/pkg/gnolang/
rm .worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_field_external.gno
*/
// PKGPATH: gno.land/r/test/indirectcurfieldext
package indirectcurfieldext

// Cross-realm indirection via a struct field. Static check is
// inconclusive (V==nil); runtime must reject with the canonical
// error message.

import tests "gno.land/r/tests/vm"

type handler struct {
	H func(realm)
}

func main(cur realm) {
	s := handler{H: tests.IncCounter}
	s.H(cur)
}

// Error:
// cannot cur-call to external realm function gno.land/r/tests/vm.IncCounter from gno.land/r/test/indirectcurfieldext
