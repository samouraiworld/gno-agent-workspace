// Round 2, solo-finder-1: a frame tag line alone inside an HTML comment in
// a frame now ends the comment, so the rest of the comment renders as text.
//
// Repro from a plain clone (Go 1.25.9):
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6299/head && git checkout 11a5d796a07a92ece6344770ca86e6baac84ebbb
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_comment_opener_leak_test.go
//	cd gno.land && go test ./pkg/gnoweb/markdown -run TestFrameCommentHidesOpener -count=1 -v
//
// At 11a5d796a it fails: "secret" and "--&gt;" render inside the frame.
// At 96fcf8a2b (round 1's head 1f9bf51d8 merged with master 47d19f8a3;
// ext_frame.go is byte-identical to 1f9bf51d8) the same test passes: the
// comment stays one "raw HTML omitted" block. Cutting on an opener or invalid frame tag
// only for HTML block types 6 and 7 (ast.HTMLBlockType6/7) keeps the whole
// package green, html_block_before_card_open included, and passes this test.
package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func TestFrameCommentHidesOpener(t *testing.T) {
	for name, in := range map[string]string{
		"opener":  "<gno-frame>\nintro\n<!--\n<gno-frame>\nsecret\n-->\n</gno-frame>\nafter\n",
		"invalid": "<gno-frame>\nintro\n<!--\n<gno-frame x=\"1\">\nsecret\n-->\n</gno-frame>\nafter\n",
	} {
		var buf bytes.Buffer
		if err := convertGno(newGnoMarkdown(), []byte(in), &buf); err != nil {
			t.Fatal(err)
		}
		if out := buf.String(); strings.Contains(out, "secret") || strings.Contains(out, "--&gt;") {
			t.Errorf("%s: commented-out text rendered:\n%s", name, out)
		}
	}
}
