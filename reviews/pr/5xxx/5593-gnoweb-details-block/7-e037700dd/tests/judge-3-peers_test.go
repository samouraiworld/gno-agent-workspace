package markdown

import "testing"

func TestJudgePeers(t *testing.T) {
	for _, c := range []struct{ name, src string }{
		{"button_bare", "<gno-button href=\"/r/x\" label=\"L\" />\n"},
		{"button_in_details", "<gno-details>\n<gno-button href=\"/r/x\" label=\"L\" />\n</gno-details>\n"},
		{"math_list_interrupt", "$$\na\n- x\nb\n$$\n"},
		{"math_details_interrupt", "$$\na\n<gno-details>\nb\n$$\n"},
	} {
		out, _ := parseDetails(t, c.src)
		t.Logf("== %s\n%s", c.name, out)
	}
}
