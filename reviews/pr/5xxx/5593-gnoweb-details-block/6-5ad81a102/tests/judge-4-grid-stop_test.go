// Judge probe for candidate 4: a details block stops at a gno-columns or
// gno-frame marker. Run with the break at ext_details.go:230 removed, or
// with the transformer moved ahead of the frame transformer, the package
// suite stays green while these cases go red.
//
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_grid_stop_test.go
//	cd gno.land/pkg/gnoweb/markdown && go test -run TestJudgeGridStop -count=1 -v .
package markdown

import (
	"strings"
	"testing"
)

func TestJudgeGridStop(t *testing.T) {
	for name, src := range map[string]string{
		"in_column": "<gno-columns>\n<gno-details summary=\"X\">\nA\n<gno-columns-sep />\nB\n</gno-columns>\n",
		"before_frame": "<gno-details summary=\"X\">\nA\n\n<gno-frame>\nB\n</gno-frame>\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, out, _ := parseDetails(t, src)
			t.Logf("%s", out)
			if i, j := strings.Index(out, "</details>"), strings.Index(out, "<p>B</p>"); i < 0 || j < i {
				t.Errorf("details took B across a grid or frame marker")
			}
		})
	}
}
