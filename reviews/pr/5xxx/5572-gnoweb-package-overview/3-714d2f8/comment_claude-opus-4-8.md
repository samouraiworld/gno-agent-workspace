# Review: PR #5572
Event: APPROVE

## Body
Solid work. Verified on 714d2f8 against a live node: the `$source` overview renders, and its symbol links deep-link to the correct source file and line.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5572-gnoweb-package-overview/3-714d2f8/review_claude-opus-4-8_davd-gzl.md [↗](./review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/gnoweb/components/overview_build.go:41-48 [↗](../../../../../.worktrees/gno-review-5572/gno.land/pkg/gnoweb/components/overview_build.go#L41)
The Code stats sidebar counts unexported types, consts and vars, but the page lists only exported symbols, so the counts don't match what's shown. On `/r/gnoland/blog$source` the sidebar shows "Vars 3" while no Variables section renders, because all three are unexported. Count only exported declarations in computeStats, matching the filter the render path already applies.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5572 -R gnolang/gno

cat > gno.land/pkg/gnoweb/components/zz_stats_repro_test.go <<'EOF'
package components

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/doc"
)

func TestStatsCountsUnexported(t *testing.T) {
	// qdoc returns unexported symbols; one unexported var group here.
	jdoc := &doc.JSONDocumentation{Values: []*doc.JSONValueDecl{
		{Values: []*doc.JSONValue{{Name: "unexportedVar"}}},
	}}
	stat := computeStats(nil, jdoc, nil).VarCount
	rendered := len(buildValues(jdoc, nil, "/r/demo/foo"))
	t.Logf("sidebar VarCount=%d, rendered variables=%d", stat, rendered)
	if stat != rendered {
		t.Fatalf("mismatch: sidebar shows %d vars, page renders %d", stat, rendered)
	}
}
EOF

go test -run TestStatsCountsUnexported ./gno.land/pkg/gnoweb/components/
rm gno.land/pkg/gnoweb/components/zz_stats_repro_test.go
```

```
--- FAIL: TestStatsCountsUnexported (0.00s)
    zz_stats_repro_test.go:16: sidebar VarCount=1, rendered variables=0
    zz_stats_repro_test.go:18: mismatch: sidebar shows 1 vars, page renders 0
FAIL
FAIL	github.com/gnolang/gno/gno.land/pkg/gnoweb/components	0.0Xs
```
</details>

*(AI Agent)*
