/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/adv_decltype_basekinds.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/adv_decltype_basekinds.gno
go test -v -run 'TestFiles/adv_decltype_basekinds.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_basekinds.gno
*/
// The PR pins one base kind, struct. fillTypeInPlace also copies slice, map,
// array, func, interface and pointer bases; nothing asserts them. All six pass
// at 093c32be0 and all six are rejected by 0397fc87f with "should not happen".
package main

type SL1 []*SL2

type SL2 SL1

type MP1 map[string]*MP2

type MP2 MP1

type AR1 [3]*AR2

type AR2 AR1

type FN1 func(*FN2) int

type FN2 FN1

type IF1 interface {
	M() int
}

type IF2 IF1

type PS struct{ Val int }

type PT1 *PT2

type PT2 PS

type impl struct{ v int }

func (i impl) M() int { return i.v }

func main() {
	var sl SL1
	sl = append(sl, &SL2{})
	var sl2 SL2
	sl2 = append(sl2, nil, nil)
	println(len(sl), len(sl2))

	mp := MP1{}
	mp["x"] = &MP2{}
	mp2 := MP2{}
	mp2["y"] = nil
	println(len(mp), len(mp2))

	var ar AR1
	ar[0] = &AR2{}
	var ar2 AR2
	println(len(ar), len(ar2))

	var fn FN1 = func(x *FN2) int { return 1 }
	var fn2 FN2 = func(x *FN2) int { return 2 }
	println(fn(nil), fn2(nil))

	var if1 IF1 = impl{3}
	var if2 IF2 = impl{4}
	println(if1.M(), if2.M())

	p := PT2{Val: 5}
	var pt PT1 = &p
	println(pt.Val)
}

// Output:
// 1 2
// 1 1
// 3 3
// 1 2
// 3 4
// 5
