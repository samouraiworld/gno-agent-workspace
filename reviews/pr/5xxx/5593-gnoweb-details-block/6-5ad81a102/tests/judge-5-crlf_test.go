// Judge probe for candidate 5: CRLF tag lines render as LF ones.
//
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_crlf_test.go
//	cd gno.land/pkg/gnoweb/markdown && go test -run TestJudgeCRLF -count=1 -v .
package markdown

import (
	"strings"
	"testing"
)

func TestJudgeCRLF(t *testing.T) {
	src := "<gno-details summary=\"S\" open>\nbody\n</gno-details>\n\nafter\n"
	_, lf, _ := parseDetails(t, src)
	_, crlf, _ := parseDetails(t, strings.ReplaceAll(src, "\n", "\r\n"))
	t.Logf("lf=%q\ncrlf=%q", lf, crlf)
	if strings.ReplaceAll(crlf, "\r", "") != lf {
		t.Errorf("CRLF render differs from LF")
	}
}
