// Repro, from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6304/head && git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	cp <this file> gno.land/pkg/gnoweb/markdown/b2_fence_mismatch_test.go
//	go test ./gno.land/pkg/gnoweb/markdown -run 'TestB2FenceMismatch|TestB2AlertInLinkSpan' -v
//
// sanitize.BlockRich escapes `[`/`]` outside links so user text cannot bind
// to a realm's own link reference definitions (`[home]` resolving to the
// realm's `[home]: ...`). Inside a span the fence tracker counts as fenced
// code, brackets are kept verbatim. Each case below builds a list item where
// goldmark and the fence tracker disagree on whether a line is fenced:
// the tracker ignores tabs in a fence line's indent (parseFenceOpener,
// isCloseFence) and ignores an HTML block opened after a list marker
// (mayOpenHTMLBlock reads the raw line only). The assertion is the fixed
// state: no user text renders as a link to the realm's [home].
package markdown

import (
	"strings"
	"testing"
)

const b2RealmLRD = "[home]: https://realm.example/home\n"

func TestB2FenceMismatch(t *testing.T) {
	cases := []struct {
		name, input string
	}{
		// Control: outside any fence the walker escapes the brackets.
		{"control-no-fence", "- a\n\n  see [home]\n"},
		// Closer indented by a tab: goldmark expands it past the item's
		// content column and closes the fence; isCloseFence counts only
		// spaces and keeps it open.
		{"tab-indented-closer", "- a\n\n  ```\n\t```\n  see [home]\n"},
		// Opener indented by a tab: goldmark opens, the tracker does not;
		// the next "  ```" closes goldmark's fence and opens the tracker's.
		{"tab-indented-opener", "- a\n\n\t```\n  ```\n  see [home]\n"},
		// HTML block after a list marker hides the fence line from
		// goldmark; the tracker opens a fence there.
		{"list-html-hides-fence", "- <div>\n  ```\n\n  see [home]\n"},
		// Control: <gno-icon> after a list marker is left unescaped, but
		// gnoweb's icon parser takes it, so it opens no HTML block.
		{"list-gno-icon-hides-fence", "- <gno-icon name=\"star\">\n  ```\n\n  see [home]\n"},
		// Full reference: the user picks the link text, the realm's
		// definition supplies the destination.
		{"tab-closer-full-reference", "- a\n\n  ```\n\t```\n  [Claim your reward][home]\n"},
		// Control inside the same mismatch: appendFenceSpans still splits
		// an LRD line, so a user definition cannot override [home].
		{"tab-closer-lrd-still-escaped", "- a\n\n  ```\n\t```\n  [home]: https://evil.example\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := callSanitizeGno(t, "BlockRich", tc.input, "", "")
			page := b2RealmLRD + "\nREALM BEFORE" + out + "REALM AFTER\n"
			html := renderMarkdown(t, page)
			t.Logf("sanitized: %q", out)
			t.Logf("html: %s", html)
			n := strings.Count(html, `href="https://realm.example/home"`)
			// Current behaviour on the mismatch cases: n == 1, the user's
			// `[home]` links to the realm's definition.
			// Fixed behaviour: n == 0, every case like the control.
			if n != 0 {
				t.Errorf("user text bound to the realm's [home] %d time(s)", n)
			}
			if !strings.Contains(html, "REALM AFTER") {
				t.Errorf("realm chrome after the block was swallowed")
			}
		})
	}
}

// A link span is kept verbatim like a fence span. findBracketSpans drops a
// multi-line link that crosses a fence-shaped or LRD-shaped line
// (fenceLines / lrdLines), but not one that crosses a blockquote alert
// line, and appendFenceSpans' `[!` split runs on fence spans only. Goldmark
// reads the blockquote line as interrupting the paragraph before any inline
// link is parsed, so the alert renders. The assertion is the fixed state.
func TestB2AlertInLinkSpan(t *testing.T) {
	cases := []struct {
		name, input string
	}{
		// Control: the same alert line on its own is escaped.
		{"control-bare-alert", "> [!WARNING] spoof\n"},
		// The alert line sits inside a two-line link text.
		{"link-text", "[a\n> [!WARNING] spoof](/u)\n"},
		// The alert line sits inside a multi-line link title.
		{"link-title", "[x](/u \"t\n> [!WARNING] spoof\n\")\n"},
		// Any block-interrupting line inside the span does the same for a
		// shortcut reference: `[home]` binds to the realm's definition.
		{"link-text-quote-ref", "[a\n> see [home]](/u)\n"},
		{"link-text-list-ref", "[a\n- see [home]](/u)\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := callSanitizeGno(t, "BlockRich", tc.input, "", "")
			html := renderMarkdown(t, b2RealmLRD+"\nREALM BEFORE"+out+"REALM AFTER\n")
			t.Logf("sanitized: %q", out)
			t.Logf("html: %s", html)
			// Current behaviour on the link cases: a gno-alert renders, or
			// `[home]` links to the realm's definition.
			// Fixed behaviour: neither, as in the control.
			if strings.Contains(html, "gno-alert") {
				t.Errorf("user content rendered a gnoweb alert")
			}
			if strings.Contains(html, `href="https://realm.example/home"`) {
				t.Errorf("user text bound to the realm's [home]")
			}
		})
	}
}
