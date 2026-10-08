// Repro for gnolang/gno#6231 at dd40ef3dc, omnisearch resolve_chain.go.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6231/head && git checkout dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp <this file> gno.land/pkg/gnoweb/feature/omnisearch/zz_b9_resolve_chain_test.go
//	go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB9' -count=1 -v
//	rm gno.land/pkg/gnoweb/feature/omnisearch/zz_b9_resolve_chain_test.go
//
// Each test asserts the behaviour the page should have; a FAIL is the defect.
package omnisearch

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/doc"
)

// capResults cuts func/type/file/import results at MaxResults and leaves
// Group.Truncated false, while discover sets it for the same situation.
func TestB9FuncResultsCappedReportTruncation(t *testing.T) {
	t.Parallel()

	jdoc := &doc.JSONDocumentation{}
	for i := range MaxResults + 5 {
		jdoc.Funcs = append(jdoc.Funcs, &doc.JSONFunc{
			Name:      fmt.Sprintf("Vote%d", i),
			Crossing:  true,
			Signature: fmt.Sprintf("func Vote%d(cur realm)", i),
			File:      "boards.gno",
			Line:      i + 1,
		})
	}
	h := newHandler(t, &mockClient{doc: jdoc}, nil)
	groups, _ := h.Search(context.Background(), mustQuery(t, h, "func:Vote", "/r/demo/boards"))
	if len(groups) != 1 {
		t.Fatalf("groups = %+v, want one", groups)
	}
	g := groups[0]
	t.Logf("matching funcs = %d, results shown = %d, Truncated = %v", MaxResults+5, len(g.Results), g.Truncated)
	if len(g.Results) < MaxResults+5 && !g.Truncated {
		t.Errorf("%d of %d matches dropped and Group.Truncated is false: the page and the JSON cannot tell the list is partial",
			MaxResults+5-len(g.Results), MaxResults+5)
	}
}

// resolveFuncs tags every top-level realm function except Render as an
// "action", whatever its export or crossing status. qdoc is built with
// unexported=true (keeper.go QueryDoc), and MsgCall refuses a non-crossing
// function (keeper.go: "is non-crossing and cannot be called with MsgCall").
func TestB9ActionTagOnlyOnCallableFuncs(t *testing.T) {
	t.Parallel()

	jdoc := &doc.JSONDocumentation{Funcs: []*doc.JSONFunc{
		{Name: "helper", Signature: "func helper() int", File: "x.gno", Line: 3},
		{Name: "GetBoard", Signature: "func GetBoard(id int) string", File: "x.gno", Line: 7},
		{Name: "CreateBoard", Crossing: true, Signature: "func CreateBoard(cur realm, name string)", File: "x.gno", Line: 11},
	}}
	h := newHandler(t, &mockClient{doc: jdoc}, nil)
	for _, tc := range []struct {
		term       string
		wantAction bool
	}{
		{"helper", false},     // unexported: not callable at all
		{"GetBoard", false},   // exported, non-crossing: MsgCall panics
		{"CreateBoard", true}, // exported, crossing: an action
	} {
		groups, _ := h.Search(context.Background(), mustQuery(t, h, "func:"+tc.term, "/r/demo/boards"))
		if len(groups) != 1 || len(groups[0].Results) != 1 {
			t.Fatalf("func:%s: groups = %+v, want one result", tc.term, groups)
		}
		r := groups[0].Results[0]
		got := slices.Contains(r.Tags, "action")
		t.Logf("func:%s tags=%v href=%s", tc.term, r.Tags, r.Href)
		if got != tc.wantAction {
			t.Errorf("func:%s: action tag = %v, want %v", tc.term, got, tc.wantAction)
		}
	}
}
