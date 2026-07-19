/* Run: from a gno checkout:
gh pr checkout 5969 -R gnolang/gno && git checkout 7e0728bd5
curl -fsSL -o gnovm/tests/files/struct64c.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5969-defined-pointer-type-semantics/1-7e0728bd5/tests/struct64c.gno
go test -v -run 'TestFiles/struct64c.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/struct64c.gno
*/

// An alias introduces no defined type, so an embedded alias of a pointer type
// is legal in Go and must survive the new fillEmbeddedName rejection. Green at
// 7e0728bd5; it guards the rejection's other side, which no filetest covers.
package main

type D2 struct{ A int }

type P = *D2

type S struct{ P }

func main() {
	var s S
	s.P = &D2{A: 3}
	println(s.A)
}

// Output:
// 3
