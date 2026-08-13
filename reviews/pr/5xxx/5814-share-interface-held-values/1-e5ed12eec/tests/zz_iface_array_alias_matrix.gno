/* Run: from a gno checkout:
gh pr checkout 5814 -R gnolang/gno && git checkout e5ed12eec
curl -fsSL -o gnovm/tests/files/zz_iface_array_alias_matrix.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5814-share-interface-held-values/1-e5ed12eec/tests/zz_iface_array_alias_matrix.gno
go test -v -run 'TestFiles/zz_iface_array_alias_matrix.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_iface_array_alias_matrix.gno
*/

// Every mutation below is rejected by go/types and executed by the VM, which
// has no addressability guard of its own. At e5ed12eec cases 2 and 6 to 10
// write through to the source array; at merge base 754780601 they do not, and
// the file prints the SHOULD block instead.

package main

type S struct{ F int }
type W struct{ A [2]int }
type I interface{ Get() int }
type T struct{ N int }

func (t T) Get() int { return t.N }

func main() {
	// 1. primitive: held in N, never copied through V
	var a1 [1]any
	a1[0] = 1
	b1 := a1
	println("1 plain:", a1[0].(int), b1[0].(int))

	// 2. struct: the case the diff changes
	var a2 [1]any
	a2[0] = S{1}
	b2 := a2
	b2[0].(S).F = 9
	println("2 struct:", a2[0].(S).F, b2[0].(S).F)

	// 3. pointer: PointerValue, shared on both sides of the diff
	var a3 [1]any
	a3[0] = &S{1}
	b3 := a3
	b3[0].(*S).F = 9
	println("3 pointer:", a3[0].(*S).F, b3[0].(*S).F)

	// 4. slice header: *SliceValue, shared on both sides
	var a4 [1]any
	a4[0] = []int{1, 2}
	b4 := a4
	b4[0].([]int)[0] = 9
	println("4 slice:", a4[0].([]int)[0], b4[0].([]int)[0])

	// 5. map: *MapValue, shared on both sides
	var a5 [1]any
	a5[0] = map[string]int{"k": 1}
	b5 := a5
	b5[0].(map[string]int)["k"] = 9
	println("5 map:", a5[0].(map[string]int)["k"], b5[0].(map[string]int)["k"])

	// 6. nested array
	var a6 [1]any
	a6[0] = [2]int{1, 2}
	b6 := a6
	b6[0].([2]int)[0] = 9
	println("6 array:", a6[0].([2]int)[0], b6[0].([2]int)[0])

	// 7. struct whose field is an array
	var a7 [1]any
	a7[0] = W{[2]int{1, 2}}
	b7 := a7
	b7[0].(W).A[0] = 9
	println("7 structarr:", a7[0].(W).A[0], b7[0].(W).A[0])

	// 8. named interface element type, not just any
	var a8 [1]I
	a8[0] = T{1}
	b8 := a8
	b8[0].(T).N = 9
	println("8 namediface:", a8[0].Get(), b8[0].Get())

	// 9. the array reached through a struct copy, not an array assignment
	type H struct{ Arr [1]any }
	h1 := H{}
	h1.Arr[0] = S{1}
	h2 := h1
	h2.Arr[0].(S).F = 9
	println("9 structfield:", h1.Arr[0].(S).F, h2.Arr[0].(S).F)

	// 10. the copy is a by-value function parameter
	var a10 [1]any
	a10[0] = S{1}
	mut(a10)
	println("10 argcopy:", a10[0].(S).F)
}

func mut(x [1]any) { x[0].(S).F = 9 }

// TypeCheckError:
// main/zz_iface_array_alias_matrix.gno:34:2: cannot assign to b2[0].(S).F (neither addressable nor a map index expression); main/zz_iface_array_alias_matrix.gno:62:2: cannot assign to b6[0].([2]int)[0] (neither addressable nor a map index expression); main/zz_iface_array_alias_matrix.gno:69:2: cannot assign to b7[0].(W).A[0] (neither addressable nor a map index expression); main/zz_iface_array_alias_matrix.gno:76:2: cannot assign to b8[0].(T).N (neither addressable nor a map index expression); main/zz_iface_array_alias_matrix.gno:84:2: cannot assign to h2.Arr[0].(S).F (neither addressable nor a map index expression); main/zz_iface_array_alias_matrix.gno:94:22: cannot assign to x[0].(S).F (neither addressable nor a map index expression)

// Output:
// 1 plain: 1 1
// 2 struct: 9 9
// 3 pointer: 9 9
// 4 slice: 9 9
// 5 map: 9 9
// 6 array: 9 9
// 7 structarr: 9 9
// 8 namediface: 9 9
// 9 structfield: 9 9
// 10 argcopy: 9

// IS:     the block above, at e5ed12eec.
// SHOULD: the block below, at merge base 754780601, and the value semantics a
//         reader takes from Go for a by-value array copy.
//
// 1 plain: 1 1
// 2 struct: 1 9
// 3 pointer: 9 9
// 4 slice: 9 9
// 5 map: 9 9
// 6 array: 1 9
// 7 structarr: 1 9
// 8 namediface: 1 9
// 9 structfield: 1 9
// 10 argcopy: 1
