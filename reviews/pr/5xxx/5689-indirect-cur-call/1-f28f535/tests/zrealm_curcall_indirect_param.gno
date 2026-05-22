/* Run:
cp reviews/pr/5xxx/5689-indirect-cur-call/1-f28f535/tests/zrealm_curcall_indirect_param.gno \
   .worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_param.gno
go test -C .worktrees/gno-review-5689 -v -run 'TestFiles/^zrealm_curcall_indirect_param' ./gnovm/pkg/gnolang/
rm .worktrees/gno-review-5689/gnovm/tests/files/zrealm_curcall_indirect_param.gno
*/
// PKGPATH: gno.land/r/test/indirectcurparam
package indirectcurparam

// Same-realm indirection via a function parameter. The static check
// has no value for `f` either, so it must fall through to runtime
// just like the local-variable case.

func myHandler(cur realm) {
	println("called")
}

func wrap(cur realm, f func(realm)) {
	f(cur)
}

func main(cur realm) {
	wrap(cur, myHandler)
	println("done")
}

// Output:
// called
// done
