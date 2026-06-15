# Review: PR #5826
Event: REQUEST_CHANGES

## Body
The value-containment fan-out fix is correct and the guard is genuinely linear, but the same exponential `validType` DoS is still reachable through one type shape the guard mishandles. Verified on 088ce87: reverting the new guard reproduces the value-fan-out hang it fixes, and a generic-instantiation fan-out hangs the real `TypeCheckMemPackage` past 20s at depth 24 on the same unmetered `MsgAddPackage` path.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/typecheck_bound.go:143-146 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L143)
A value fan-out routed through a generic instantiation (`W[A{n-1}]`) is counted as `cost(W)`, a constant, because this arm drops the type argument, so the guard accepts it and `go/types` validType still hangs the unmetered deploy path. The base type does not bound the cost when the type argument drives the expansion. Fix: reject or conservatively over-count generic instantiations whose type arguments can drive value-containment fan-out, so a generic doubling chain hits the budget like the direct-value one.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5826 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go <<'EOF'
package gnolang

import (
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func itoaF(i int) string {
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

func tcFanout(body string) error {
	mpkg := &std.MemPackage{
		Name: "fanout", Path: "gno.land/r/foo/fanout", Type: MPUserProd,
		Files: []*std.MemFile{
			{Name: "gnomod.toml", Body: "module = \"gno.land/r/foo/fanout\"\ngno = \"0.9\"\n"},
			{Name: "fanout.gno", Body: body},
		},
	}
	_, err := TypeCheckMemPackage(mpkg, TypeCheckOptions{Mode: TCLatestStrict})
	return err
}

func run(t *testing.T, body string) error {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if recover() != nil {
				done <- nil
			}
		}()
		done <- tcFanout(body)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(25 * time.Second):
		t.Fatal("TypeCheckMemPackage did not return within 25s: type-expansion DoS reached validType")
		return nil
	}
}

// generic instantiation doubling: each An holds W[A{n-1}], W holds its param twice by value.
func TestGenericInstantiationFanoutBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fanout\ntype W[P any] struct{ a, b [0]P }\ntype A0 struct{ v int }\n")
	prev := "A0"
	for i := 1; i <= 40; i++ {
		b.WriteString("type A" + itoaF(i) + " struct{ x W[" + prev + "] }\n")
		prev = "A" + itoaF(i)
	}
	b.WriteString("var Sink A40\n")
	if err := run(t, b.String()); err == nil || !strings.Contains(err.Error(), "denial-of-service") {
		t.Fatalf("expected a denial-of-service rejection, got: %v", err)
	}
}

// baseline already fixed: the same doubling expressed directly by value.
func TestValueFanoutBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fanout\ntype T0 struct{ v int }\n")
	for i := 1; i <= 40; i++ {
		b.WriteString("type T" + itoaF(i) + " struct{ a, b [0]T" + itoaF(i-1) + " }\n")
	}
	b.WriteString("var Sink T40\n")
	if err := run(t, b.String()); err == nil || !strings.Contains(err.Error(), "denial-of-service") {
		t.Fatalf("expected a denial-of-service rejection for the value baseline, got: %v", err)
	}
}
EOF
go test -timeout 60s -run 'TestGenericInstantiationFanoutBounded|TestValueFanoutBounded' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go
```

```
=== RUN   TestGenericInstantiationFanoutBounded
    zz_generic_fanout_dos_test.go:49: TypeCheckMemPackage did not return within 25s: type-expansion DoS reached validType
--- FAIL: TestGenericInstantiationFanoutBounded (25.00s)
=== RUN   TestValueFanoutBounded
--- PASS: TestValueFanoutBounded (0.00s)
FAIL
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	25.02s
```
The value baseline is rejected (PR fix works); the generic shape hangs until the 25s deadline (the hole). Both assert the desired post-fix `denial-of-service` rejection.
</details>

*(AI Agent)*
