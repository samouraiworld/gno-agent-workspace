// Repro: a malformed opener (a valued `open`, a misspelled attribute) is an
// invalid leaf that never goes on the details stack, so its own close tag
// pairs with the enclosing opener and ends the outer block early. A refused
// gno-frame opener, and a details opener refused by the depth cap, keep
// their close instead.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 5ad81a10214a0379b6099159e98431176732d1f7
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_malformed_inner_test.go
//	cd gno.land/pkg/gnoweb/markdown
//	go test -run 'TestDetailsMalformedInner' -count=1 -v .
//
// Expected at 5ad81a102: the open_valued and misspelled cases FAIL ("outer
// body" renders after </details>); the valid and frame cases pass.
package markdown

import (
	"strings"
	"testing"
)

func TestDetailsMalformedInner(t *testing.T) {
	for name, inner := range map[string]string{
		"valid":       `<gno-details summary="Inner">`,
		"open_valued": `<gno-details summary="Inner" open="true">`,
		"misspelled":  `<gno-details sumary="Inner">`,
	} {
		t.Run(name, func(t *testing.T) {
			src := "<gno-details summary=\"Outer\">\n" + inner + "\ninner body\n</gno-details>\nouter body\n</gno-details>\n\nafter\n"
			_, out, _ := parseDetails(t, src)
			if strings.LastIndex(out, "</details>") < strings.Index(out, "outer body") {
				t.Errorf("outer body left the Outer block:\n%s", out)
			}
		})
	}
	t.Run("frame_sibling", func(t *testing.T) {
		_, out, _ := parseDetails(t, "<gno-frame>\n<gno-frame bogus>\ninner\n</gno-frame>\nouter body\n</gno-frame>\n\nafter\n")
		if strings.Index(out, "</section>") < strings.Index(out, "outer body") {
			t.Errorf("outer body left the frame:\n%s", out)
		}
	})
}
