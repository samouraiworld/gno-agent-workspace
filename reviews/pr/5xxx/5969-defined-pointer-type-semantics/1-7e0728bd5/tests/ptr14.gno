/* Run: from a gno checkout:
gh pr checkout 5969 -R gnolang/gno && git checkout 7e0728bd5
curl -fsSL -o gnovm/tests/files/ptr14.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5969-defined-pointer-type-semantics/1-7e0728bd5/tests/ptr14.gno
go test -v -run 'TestFiles/ptr14.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/ptr14.gno
*/

// A pointer to a defined pointer type promotes nothing in Go: the (*x).f
// shorthand only applies when the operand type is not itself a pointer type.
// At 7e0728bd5 this panics "should not happen" in applyPointerDeref, because
// findEmbeddedFieldType strips the outer pointer before the walk and the
// fields-only crossing is entered one level too late.
package main

type D2 struct{ A int }

type D1 *D2

func main() {
	var p *D1
	_ = p.A
}

// Error:
// main/ptr14.gno:22:6-9: missing field A in *main.D1

// TypeCheckError:
// main/ptr14.gno:22:8: p.A undefined (type *D1 has no field or method A)
