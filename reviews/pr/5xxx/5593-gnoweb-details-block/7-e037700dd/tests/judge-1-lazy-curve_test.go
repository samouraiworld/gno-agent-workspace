package markdown

import (
	"strings"
	"testing"
)

func TestJudgeLazyCurve(t *testing.T) {
	for _, n := range []int{10, 100, 1000} {
		lazy, _ := parseDetails(t, strings.Repeat("> <gno-details>\n> p\n<i>x</i>\n", n))
		ctl, _ := parseDetails(t, strings.Repeat("> <gno-details>\n> p\nplain\n", n))
		t.Logf("n=%d lazy opened=%d control opened=%d", n,
			strings.Count(lazy, `<details class="gno-details">`), strings.Count(ctl, `<details class="gno-details">`))
	}
}
