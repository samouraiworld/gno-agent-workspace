// judge-1-foreign-in-list: the escape also runs through a <gno-foreign>
// nested in a list inside :::details, so an exemption that checks only the
// details block's direct child leaves it open. Needs soloRenderFE from
// solo-finder-foreign-escape_test.go in the same package; FAIL at 8ab095f49,
// PASS with judge-1-foreign-exempt.patch applied.
package markdown

import (
	"strings"
	"testing"
)

func TestJudgeForeignInList(t *testing.T) {
	src := ":::details C\n\n- <gno-foreign>\n  hello\n  :::\n  <gno-form exec=\"Withdraw\">\n  </gno-form>\n  </gno-foreign>\n\n:::\n"
	out := soloRenderFE(t, src)
	t.Logf("\n%s", out)
	if strings.Contains(out, "<form") {
		t.Errorf("list-nested foreign escaped")
	}
}
