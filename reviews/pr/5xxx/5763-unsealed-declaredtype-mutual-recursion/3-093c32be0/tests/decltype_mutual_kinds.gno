/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/decltype_mutual_kinds.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/decltype_mutual_kinds.gno
go test -v -run 'TestFiles/decltype_mutual_kinds.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_mutual_kinds.gno
*/

// Mutual type-decl recursion on the six base kinds decltype_mutual.gno leaves
// untested. Every // Output: line is the stdout of the identical Go program
// under go1.26.5. Passes at 093c32be0, panics `should not happen` at the
// merge-base 0397fc87f.

package main

// Struct base.
type S1 struct {
	Next *S2
	Val  int
}

type S2 S1

// Slice base.
type L1 []L2

type L2 L1

// Map base.
type M1 map[string]M2

type M2 M1

// Func base.
type F1 func(int) F2

type F2 F1

// Pointer base.
type P1 *P2

type P2 P1

// Array-of-pointer base.
type A1 [2]*A2

type A2 A1

func main() {
	var s S1
	s.Next = &S2{Val: 9}
	println(s.Next.Val)

	var l L1
	l = append(l, L2{})
	l[0] = append(l[0], L2{})
	println(len(l), len(l[0]))

	m := M1{}
	m["x"] = M2{"y": nil}
	println(len(m), len(m["x"]))

	var f F1 = func(i int) F2 { return nil }
	println(f(3) == nil)

	var p P1
	println(p == nil)

	var a A1
	a[0] = &A2{}
	println(len(a), a[1] == nil)
}

// Output:
// 9
// 1 1
// 1 1
// true
// true
// 2 true
