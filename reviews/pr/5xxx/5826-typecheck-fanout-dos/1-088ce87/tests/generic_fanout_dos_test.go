// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a gno checkout:
gh pr checkout 5826 -R gnolang/gno && git checkout 088ce87
curl -fsSL -o gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/tests/generic_fanout_dos_test.go
go test -timeout 30s -run 'TestGenericInstantiationFanoutBounded|TestValueFanoutBounded' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go
*/

// checkTypeExpansionBound's cost() returns cost(t.X) for an *ast.IndexExpr,
// dropping the type argument, so a value-doubling chain routed through a generic
// instantiation (W[A_{n-1}]) is counted as a constant and slips past the budget.
// go/types' validType then walks it exponentially inside TypeCheckMemPackage,
// the same unmetered AddPackage path the PR's value-fan-out case rejects.
// At 088ce87 TestGenericInstantiationFanoutBounded hangs (the -timeout fires);
// TestValueFanoutBounded already passes, pinning that the value case is fixed.

package gnolang

import (
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func itoaFanout(i int) string {
	if i == 0 {
		return "0"
	}
	var d []byte
	for i > 0 {
		d = append([]byte{byte('0' + i%10)}, d...)
		i /= 10
	}
	return string(d)
}

func typeCheckFanoutPkg(t *testing.T, body string) error {
	t.Helper()
	mpkg := &std.MemPackage{
		Name: "fanout",
		Path: "gno.land/r/foo/fanout",
		Type: MPUserProd,
		Files: []*std.MemFile{
			{Name: "gnomod.toml", Body: "module = \"gno.land/r/foo/fanout\"\ngno = \"0.9\"\n"},
			{Name: "fanout.gno", Body: body},
		},
	}
	_, err := TypeCheckMemPackage(mpkg, TypeCheckOptions{Mode: TCLatestStrict})
	return err
}

// runWithDeadline runs fn; a value-fan-out DoS makes TypeCheckMemPackage hang,
// so the surrounding `go test -timeout` is what records the failure. The inner
// deadline only keeps a green run from blocking the whole suite.
func runWithDeadline(t *testing.T, body string) error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- nil // a panic is a clean rejection for this DoS-only assertion
			}
		}()
		done <- typeCheckFanoutPkg(t, body)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(25 * time.Second):
		t.Fatal("TypeCheckMemPackage did not return within 25s: type-expansion DoS reached validType")
		return nil
	}
}

// Generic instantiation doubling: A_n contains W[A_{n-1}] by value, and W holds
// its type parameter twice by value, so each level doubles validType's work.
func TestGenericInstantiationFanoutBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fanout\n")
	b.WriteString("type W[P any] struct{ a, b [0]P }\n")
	b.WriteString("type A0 struct{ v int }\n")
	prev := "A0"
	for i := 1; i <= 40; i++ {
		b.WriteString("type A" + itoaFanout(i) + " struct{ x W[" + prev + "] }\n")
		prev = "A" + itoaFanout(i)
	}
	b.WriteString("var Sink A40\n")

	err := runWithDeadline(t, b.String())
	if err == nil {
		t.Fatal("generic-instantiation fan-out was accepted; expected a type-expansion rejection")
	}
	if !strings.Contains(err.Error(), "denial-of-service") {
		t.Fatalf("expected a denial-of-service rejection, got: %v", err)
	}
}

// Baseline the PR already fixes: the same doubling expressed directly by value.
func TestValueFanoutBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fanout\ntype T0 struct{ v int }\n")
	for i := 1; i <= 40; i++ {
		b.WriteString("type T" + itoaFanout(i) + " struct{ a, b [0]T" + itoaFanout(i-1) + " }\n")
	}
	b.WriteString("var Sink T40\n")

	err := runWithDeadline(t, b.String())
	if err == nil || !strings.Contains(err.Error(), "denial-of-service") {
		t.Fatalf("expected a denial-of-service rejection for the value-fan-out baseline, got: %v", err)
	}
}
