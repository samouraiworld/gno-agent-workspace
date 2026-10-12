// Reflector probe: a details opener left unclosed in a column ends at the
// next column marker, but its gno-* depth level is held until the end of
// the page, so a later grid holding an alert loses the alert.
//
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_column_depth_test.go
//	cd gno.land/pkg/gnoweb/markdown && go test -run TestJudgeColumnDepth -count=1 -v .
package markdown

import (
	"strings"
	"testing"
)

func TestJudgeColumnDepth(t *testing.T) {
	tail := "\n<gno-columns>\n> [!NOTE]\n> x\n</gno-columns>\n"
	for name, cell := range map[string]string{
		"closed":   "<gno-details>\nA\n</gno-details>\n<gno-columns-sep />\n",
		"unclosed": "<gno-details>\nA\n<gno-columns-sep />\n",
	} {
		t.Run(name, func(t *testing.T) {
			src := "<gno-columns>\n" + strings.Repeat(cell, 3) + "Z\n</gno-columns>\n" + tail
			_, out, depth := parseDetails(t, src)
			if !strings.Contains(out, "gno-alert") {
				t.Errorf("later alert refused; depth at EOF %d", depth)
			}
		})
	}
}
