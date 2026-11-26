// Repro: in Rich mode, a document-level fence opened 1-3 spaces in, whose
// content sits left of the opener and whose closer sits at column 0, turns
// every line the user writes after that closer into code.
//
// closeFencesAtDedent (gnovm/stdlibs/chain/markdown/markdown.go) writes a
// closer before the first dedented content line. The user's own column-0
// closer then opens a new fence, which nothing dedents from, so it runs to
// BlockRich's EOF autoclose and holds the rest of the user's post. The ADR
// (gno.land/adr/pr6304_sanitize_gno_tags.md, Consequences) and the thread
// describe the cost as "two empty code blocks" / "three blocks"; with any
// text after the closer the third block is not empty.
//
// Alternative measured at a8bdd53b0: in closeFencesAtDedent, re-indent the
// dedented line to the opener's column (write fenceOpen[:indent], skip the
// line's own leading spaces, inFence = !closes) instead of writing a closer
// before it. All three defect cases then render as goldmark alone does (one
// code block, prose after it), TestExtensionAndFooterIsolation (41 inputs x 4
// helpers, idempotency included) and TestBlockRichSetextCannotReachRealm stay
// green, and only the goldens that pin the split change:
// blockrich-gno-form-in-list-fence{,-blank-line} and
// blockrich-gno-button-after-fence-in-list-item (the tag now sits in the
// list item's code block) plus TestEscapeBlockHazardsRich.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_dedent_closer_test.go
//	go test ./gno.land/pkg/gnoweb/markdown -run TestRichDedentCloserSwallowsProse -count=1 -v
package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRichDedentCloserSwallowsProse(t *testing.T) {
	cases := []struct {
		name, helper, in string
		defect           bool
	}{
		{"backtick-1-space", "BlockRich", " ```\ncode\n```\n\nprose after\n\n## heading after\n", true},
		{"tilde-2-spaces", "BlockRich", "  ~~~\ncode\n~~~\n\nprose after\n", true},
		{"quote-rich", "BlockquoteRich", " ```\ncode\n```\n\nprose after\n", true},
		// Control: content at the opener's column is not dedented, so
		// nothing is written and the prose stays prose.
		{"control-content-at-opener", "BlockRich", " ```\n code\n ```\n\nprose after\n", false},
		// Control: strict mode does not run the dedent pass.
		{"control-strict", "Block", " ```\ncode\n```\n\nprose after\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// goldmark alone: one code block holding "code", then prose.
			raw := renderMarkdown(t, c.in)
			require.Contains(t, raw, "<pre><code>code\n</code></pre>")
			require.Contains(t, raw, "<p>prose after</p>")

			out := applySanitize(t, sanitizeCase{Func: c.helper}, c.in)
			got := renderMarkdown(t, out)
			t.Logf("sanitized=%q\nrendered=%q", out, got)
			if !c.defect {
				require.Contains(t, got, "<p>prose after</p>")
				return
			}
			// Current (defect): the user's prose after the closer renders as code.
			require.NotContains(t, got, "<p>prose after</p>")
			require.Regexp(t, `(?s)<pre><code>.*prose after.*</code></pre>`, got)
			// Fixed: the prose after the user's closer stays a paragraph.
			// require.Contains(t, got, "<p>prose after</p>")
		})
	}
}
