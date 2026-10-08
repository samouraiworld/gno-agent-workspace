// Probe: gnolang/gno#6299 at 1f9bf51d8e71902053e3bd92986f92d24431efc2.
//
// frameHTMLBlockParser.Continue (ext_frame.go:270) cuts every HTML block type
// at a gno-columns or </gno-frame> line while a frame is open, comments
// (type 2) included. Inside a frame, a commented-out <gno-columns> line ends
// the comment, opens a live grid that closes the frame early, and the rest of
// the comment, "-->" included, renders as text. Outside a frame the same
// comment stays hidden.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 1f9bf51d8e71902053e3bd92986f92d24431efc2
//   cp <this file> gno.land/pkg/gnoweb/markdown/probe_frame_comment_test.go
//   go test ./gno.land/pkg/gnoweb/markdown/ -run TestProbeFrameCommentHidesColumns -v -count=1
//   rm gno.land/pkg/gnoweb/markdown/probe_frame_comment_test.go
// IS:     in_frame output carries <div class="gno-columns"> and "--&gt;"
// SHOULD: in_frame output keeps the comment as <!-- raw HTML omitted -->, like no_frame
package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func TestProbeFrameCommentHidesColumns(t *testing.T) {
	render := func(src string) string {
		var buf bytes.Buffer
		if err := convertGno(newGnoMarkdown(), []byte(src), &buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	for name, src := range map[string]string{
		"in_frame": "<gno-frame>\nintro\n<!--\ndraft\n<gno-columns>\nhidden\n-->\n</gno-frame>\n",
		"no_frame": "<!--\ndraft\n<gno-columns>\nhidden\n-->\n",
	} {
		out := render(src)
		t.Logf("=== %s\n%s", name, out)
		if strings.Contains(out, "gno-columns\">") || strings.Contains(out, "--&gt;") {
			t.Errorf("%s: commented-out lines render live", name)
		}
	}
}
