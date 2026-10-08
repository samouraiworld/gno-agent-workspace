// judge-2001: the button escape now runs before the bracket walker, so a
// reference definition whose pointy destination holds `<gno-button x>` is
// stripped only up to the space and leaves `x>` behind (ab27ce5c0: stripped whole).
// Run: cp to gnovm/stdlibs/chain/markdown/, go test ./gnovm/stdlibs/chain/markdown -run TestJudgeLRD -v
package markdown

import "testing"

func TestJudgeLRD(t *testing.T) {
	for _, in := range []string{
		"[evil]: <gno-button>\n",
		"[evil]: <gno-button x>\n",
		"[evil]: <gno-button> \"t\"\n",
		"[evil]:\n<gno-button>\n",
		"[evil](<gno-button>)\n",
		"![evil](<gno-button x>)\n",
	} {
		t.Logf("%q -> B=%q R=%q", in, EscapeBlockHazards(in), EscapeBlockHazardsRich(in))
	}
}
