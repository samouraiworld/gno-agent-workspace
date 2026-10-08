// b1-lines: a label made only of invisible format characters still renders a
// button. buttonLabel strips through chainmd.StripBidiAndZeroWidth, whose set
// misses U+061C ARABIC LETTER MARK (a Bidi_Control character, like the LRM and
// RLM it does strip), U+2060 WORD JOINER and U+180E MONGOLIAN VOWEL SEPARATOR
// (zero width). The label == "" gate in newButtonLink then passes, and the page
// shows an empty button, which the golden hostile_label_bidi_zero_width
// ("Only invisible") says is rejected.
//
// Repro from a plain clone (red at the head, green once the label path strips
// these code points):
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6298/head && git checkout 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//   cp <this file> gno.land/pkg/gnoweb/markdown/b1_lines_invisible_label_test.go
//   go test -count=1 -run 'TestButtonInvisibleOnlyLabel' -v ./gno.land/pkg/gnoweb/markdown/
package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestButtonInvisibleOnlyLabel(t *testing.T) {
	for name, label := range map[string]string{
		"word joiner entity":    "&#x2060;",
		"arabic letter mark":    "؜",
		"mongolian vowel sep":   "᠎",
		"all three, raw":        "؜⁠᠎",
	} {
		out := renderMarkdown(t, `A <gno-button href="/r/x" label="`+label+`" />`+"\n")
		require.NotContains(t, out, `class="gno-button"`, name)
	}
}
