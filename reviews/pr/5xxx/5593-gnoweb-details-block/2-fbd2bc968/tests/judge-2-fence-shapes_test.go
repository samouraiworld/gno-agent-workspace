// Judge probe for candidates 2 and 4: shapes the fenced-code exemption in
// detailsParser.Continue must keep, rendered with the production extension chain.
// From a plain clone of gnolang/gno at fbd2bc968, beside solo_shapes_test.go:
//	cp <this file> gno.land/pkg/gnoweb/markdown/judge_fence_shapes_test.go
//	go test ./gno.land/pkg/gnoweb/markdown/ -run TestJudgeFenceShapes -v -count=1
package markdown

import "testing"

func TestJudgeFenceShapes(t *testing.T) {
	shapes := []struct{ name, src string }{
		{"cols_fence_unclosed", "<gno-columns>\n```\nleft\n</gno-columns>\n\n## Footer\n"},
		{"fence_in_list", ":::details A\n- item\n\n  ```\n  :::\n  ```\n:::\n\n## Footer\n"},
		{"fence_unclosed_in_details", ":::details A\n```\ncode\n:::\n\n## Footer\n"},
	}
	for _, s := range shapes {
		t.Logf("=== %s\n--- input\n%s--- output\n%s", s.name, s.src, soloRender(t, s.src))
	}
}
