/* Run: from a gno checkout:
gh pr checkout 5969 -R gnolang/gno && git checkout 7e0728bd5
curl -fsSL -o gnovm/tests/files/method51.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5969-defined-pointer-type-semantics/1-7e0728bd5/tests/method51.gno
go test -v -run 'TestFiles/method51.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/method51.gno
*/

// A defined pointer type has an empty method set, so it satisfies no
// interface. VerifyImplementedBy shares the embedded lookup, so this is the
// assignability face of the same change. Master panics "should not happen"
// here; at 7e0728bd5 the assignment is rejected with the normal message.
package main

type D2 struct{}

func (D2) Foo() string { return "m" }

type D1 *D2

type Fooer interface{ Foo() string }

func main() {
	d2 := D2{}
	var x D1 = &d2
	var f Fooer = x
	println(f.Foo())
}

// Error:
// main/method51.gno:26:6-17: main.D1 does not implement main.Fooer (missing method Foo)

// TypeCheckError:
// main/method51.gno:26:16: cannot use x (variable of pointer type D1) as Fooer value in variable declaration: D1 does not implement Fooer (missing method Foo)
