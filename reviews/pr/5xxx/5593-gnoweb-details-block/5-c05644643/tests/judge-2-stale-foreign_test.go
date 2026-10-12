// Judge repro, candidate 2: an unclosed <gno-foreign> whose blockquote or list
// closes it keeps the details' own ":::" from closing the block. Expects the
// footer outside </details>. Fails at c05644643 for the first three; the
// closed_control row passes at both heads.
package markdown

import (
	"strings"
	"testing"
)

func TestZZJudgeStaleForeign(t *testing.T) {
	cases := map[string]string{
		"quote_blank":    ":::details A\n> <gno-foreign>\n> text\n\n:::\n\nfooter\n",
		"quote_noblank":  ":::details A\n> <gno-foreign>\n> text\n:::\n\nfooter\n",
		"list_blank":     ":::details A\n- <gno-foreign>\n  text\n\n:::\n\nfooter\n",
		"closed_control": ":::details A\n> <gno-foreign>\n> text\n> </gno-foreign>\n\n:::\n\nfooter\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, out, _ := parseDetails(t, src)
			if !strings.HasSuffix(out, "</details>\n<p>footer</p>\n") {
				t.Errorf("footer folded into the details:\n%s", out)
			}
		})
	}
}
