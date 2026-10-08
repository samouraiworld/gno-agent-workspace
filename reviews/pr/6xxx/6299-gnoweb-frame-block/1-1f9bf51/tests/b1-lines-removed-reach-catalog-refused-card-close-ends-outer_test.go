// Probe: gnolang/gno#6299 at 1f9bf51d8e71902053e3bd92986f92d24431efc2.
//
// A card `<gno-frame>` in a framed grid that the parser refuses (the shared
// gno-* depth cap, reached here through two nested <gno-foreign> blocks) still
// leaves its `</gno-frame>` live: Open's `kind == frameTagClose && open` case
// pairs it with the OUTER frame, which then ends before the grid. Control: the
// same page under one <gno-foreign> keeps the grid inside the frame.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 1f9bf51d8e71902053e3bd92986f92d24431efc2
//   cp <this file> gno.land/pkg/gnoweb/markdown/probe_frame_cap_test.go
//   go test ./gno.land/pkg/gnoweb/markdown/ -run TestProbeRefusedCardCloseEndsOuter -v -count=1
package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func TestProbeRefusedCardCloseEndsOuter(t *testing.T) {
	page := "<gno-frame>\nintro\n<gno-columns>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"
	wrap := func(n int) string {
		return strings.Repeat("<gno-foreign>\n", n) + page + strings.Repeat("</gno-foreign>\n", n)
	}
	for _, n := range []int{1, 2} {
		var buf bytes.Buffer
		if err := convertGno(newGnoMarkdown(), []byte(wrap(n)), &buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		s := strings.Index(out, `<section class="gno-frame">`)
		g := strings.Index(out, `<div class="gno-columns">`)
		e := strings.Index(out[max(s, 0):], `</section>`) + max(s, 0)
		holds := s >= 0 && g > s && g < e
		t.Logf("=== foreign depth %d: outer frame holds grid=%v\n%s", n, holds, out)
		if !holds {
			t.Errorf("foreign depth %d: outer frame does not hold the grid", n)
		}
	}
}
