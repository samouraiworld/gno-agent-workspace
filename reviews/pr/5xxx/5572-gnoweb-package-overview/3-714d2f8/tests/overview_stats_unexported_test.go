// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.

/* Run: from a gno checkout:
gh pr checkout 5572 -R gnolang/gno && git checkout 714d2f8
curl -fsSL -o gno.land/pkg/gnoweb/components/zz_overview_stats_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5572-gnoweb-package-overview/3-714d2f8/tests/overview_stats_unexported_test.go
go test -run TestOverviewStatsMatchRendered_Unexported ./gno.land/pkg/gnoweb/components/
rm gno.land/pkg/gnoweb/components/zz_overview_stats_test.go
*/

// qdoc is queried with unexported=true, so jdoc carries unexported symbols.
// computeStats counts every value group while buildValues renders only exported
// ones, so the sidebar "Vars" count and the Variables section disagree.
// At 714d2f8 this fails (VarCount=1, rendered=0); it passes once computeStats
// filters to exported declarations the way the render path already does.

package components

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/doc"
)

func TestOverviewStatsMatchRendered_Unexported(t *testing.T) {
	jdoc := &doc.JSONDocumentation{
		Values: []*doc.JSONValueDecl{
			{Const: false, Values: []*doc.JSONValue{{Name: "unexportedVar"}}},
		},
	}

	statVars := computeStats(nil, jdoc, nil).VarCount
	renderedVars := len(buildValues(jdoc, nil, "/r/demo/foo"))

	if statVars != renderedVars {
		t.Fatalf("Code stats VarCount=%d but rendered variables=%d; the sidebar count includes unexported symbols the page never lists", statVars, renderedVars)
	}
}
