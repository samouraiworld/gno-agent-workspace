/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/adv_decltype_embed.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/adv_decltype_embed.gno
go test -v -run 'TestFiles/adv_decltype_embed.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_embed.gno
*/
// Field promotion through an embedded value and an embedded pointer, read from
// both halves of the cycle. The shared base pointer must not merge the two
// types' promoted-field caches.
package main

type Inner struct {
	Deep int
}

type T1 struct {
	Inner
	Next *T2
	Val  int
}

type T2 T1

type E1 struct {
	*E2
	Val int
}

type E2 struct {
	Deep int
}

type E3 E1

func main() {
	var a T1
	a.Deep = 1
	a.Next = &T2{Val: 2}
	a.Next.Deep = 3
	println(a.Deep, a.Next.Deep, a.Next.Val)

	e := E1{E2: &E2{Deep: 11}, Val: 1}
	println(e.Deep, e.Val)
	var f E3
	f.E2 = &E2{Deep: 22}
	f.Val = 2
	println(f.Deep, f.Val)
}

// Output:
// 1 3 2
// 11 1
// 22 2
