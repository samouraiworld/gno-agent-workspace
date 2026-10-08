// Probe: gnolang/gno#6299 at 1f9bf51d8e71902053e3bd92986f92d24431efc2.
//
// frameHTMLBlockParser.Continue ends an HTML block on a `</gno-frame>` line
// and on any columns line, but not on a card's `<gno-frame>` open line. An
// HTML line directly above a card in a framed grid swallows the card's open,
// and the card's `</gno-frame>` then closes the OUTER frame, which wrapFrames
// ends just before the grid: the outer frame renders empty, the grid outside it.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 1f9bf51d8e71902053e3bd92986f92d24431efc2
//   cp <this file> gno.land/pkg/gnoweb/markdown/probe_frame_html_test.go
//   go test ./gno.land/pkg/gnoweb/markdown/ -run TestProbeFrameHTMLBlockHidesCardOpen -v -count=1
package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func probeRender(t *testing.T, src string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := convertGno(newGnoMarkdown(), []byte(src), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// The outer frame must hold the grid: `<section class="gno-frame">` is
// followed by the grid's `<div class="gno-columns">` before any `</section>`.
func outerFrameHoldsGrid(out string) bool {
	s := strings.Index(out, `<section class="gno-frame">`)
	g := strings.Index(out, `<div class="gno-columns">`)
	e := strings.Index(out, `</section>`)
	return s >= 0 && g > s && (e < 0 || g < e)
}

func TestProbeFrameHTMLBlockHidesCardOpen(t *testing.T) {
	cases := []struct {
		name, src string
	}{
		// control: no HTML line, the grid sits inside the frame
		{"control_no_html", "<gno-frame>\n<gno-columns>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		// control: HTML line directly before the card CLOSE, which the wrapper defends
		{"control_html_before_close", "<gno-frame>\n<gno-columns>\n<gno-frame>\ncard\n<br>\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		// control: blank line between the HTML line and the card open
		{"control_blank_after_html", "<gno-frame>\n<gno-columns>\n<br>\n\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		// defect: HTML line directly before the card OPEN
		{"html_before_card_open", "<gno-frame>\n<gno-columns>\n<br>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		{"div_before_card_open", "<gno-frame>\n## Apps\n<gno-columns>\n<div></div>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
	}
	for _, c := range cases {
		out := probeRender(t, c.src)
		t.Logf("=== %s holds=%v\n%s", c.name, outerFrameHoldsGrid(out), out)
		if !outerFrameHoldsGrid(out) {
			t.Errorf("%s: outer frame does not hold the grid", c.name)
		}
	}
}
