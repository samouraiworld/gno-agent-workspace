/* Run:
cp reviews/pr/5xxx/5689-indirect-cur-call/1-f28f535/tests/zrealm_curcall_indirect_field.gno \
   .worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_field.gno
go test -C .worktrees/gno-review-5689 -v -run 'TestFiles/^zrealm_curcall_indirect_field' ./gnovm/pkg/gnolang/
rm .worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_field.gno
*/
// PKGPATH: gno.land/r/test/indirectcurfield
package indirectcurfield

// Same-realm indirection via a struct field. Static check cannot
// resolve s.H to a FuncValue, so V==nil and we fall through.

type handler struct {
	H func(realm)
}

func myHandler(cur realm) {
	println("called")
}

func main(cur realm) {
	s := handler{H: myHandler}
	s.H(cur)
	println("done")
}

// Output:
// called
// done
