// PKGPATH: gno.land/r/test
/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/adv_decltype_realm_persist.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/adv_decltype_realm_persist.gno
go test -v -run 'TestFiles/adv_decltype_realm_persist.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_realm_persist.gno
*/
// After the fix T1 and T2 share one *StructType. Nothing in the PR stores a
// value of either half in realm state, so nothing asserts that the shared base
// serializes and that the two named types stay distinct on the way back out.
package test

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

var (
	a T1
	b T2
)

func main(cur realm) {
	a.Val = 1
	a.Next = &T2{Val: 2}
	b.Val = 3
	println(a.Val, a.Next.Val, b.Val)
	var c T1 = T1(b)
	println(c.Val)
}

// Output:
// 1 2 3
// 3
