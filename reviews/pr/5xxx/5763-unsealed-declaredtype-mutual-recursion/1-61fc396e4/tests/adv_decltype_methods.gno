// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno && git checkout 61fc396e4
curl -fsSL -o gnovm/tests/files/adv_decltype_methods.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/1-61fc396e4/tests/adv_decltype_methods.gno
go test -v -run 'TestFiles/adv_decltype_methods.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/adv_decltype_methods.gno
*/

// Control: T1 (in a mutual cycle with T2) carries a method; `type T2 T1`
// must drop it (Go semantics). T1 has NO data field beyond *T2, so its
// base does not collapse — this is exactly the no-data-field shape that
// masks the empty-base bug in decltype_derived_empty_base.gno. Passes on
// 61fc396e4 ("t1\nok").

package main

// Mutual cycle where the source type T1 carries a method. After both
// seal, T2 (defined as `type T2 T1`) must NOT inherit T1's methods
// (Go semantics: defining a new type from a named type drops methods).

type T1 struct {
	Next *T2
}

func (t T1) Hello() string { return "t1" }

type T2 T1

func main() {
	var t T1
	t.Next = &T2{}
	println(t.Hello())
	println("ok")
}

// Output:
// t1
// ok
