package markdown

// Repro, from a plain clone of gnolang/gno:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6304/head && git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_b6_details_test.go
//	go test ./gno.land/pkg/gnoweb/markdown -run 'TestB6' -count=1 -v
//
// Observed at a8bdd53b0: both tests PASS.
//   TestB6AssertionsBlindToDetails logs the footer rendered inside
//   `<details class="gno-details">` while the four assertions of
//   TestExtensionAndFooterIsolation hold: the test cannot see that regression.
//   TestB6DetailsAndSiblingOpenersThroughCorpus logs `slipped=0 of 288`: the
//   sanitizer itself escapes every opener tried, so this is a test gap only.
//
// Fix in TestExtensionAndFooterIsolation (sanitize_integration_test.go:575):
//   current: class="gno-(?:form|columns|column|frame|foreign|button|alert)[" ]
//   fixed:   class="gno-(?:form|columns|column|frame|foreign|button|alert|details)[" ]
// and feed each repro shape a `<gno-details summary="x">` opener beside
// `<gno-form path="x">` and `<gno-columns>`.
//
// TestB6AssertionsBlindToDetails: a `<gno-details>` opener that survives the
// sanitizer wraps the trusted footer in a collapsed <details>, and every
// assertion TestExtensionAndFooterIsolation makes still passes, because its
// liveExtension regex has no `details` alternative and the footer check only
// asks for the <strong> text, which still renders inside the <details>.
//
// TestB6DetailsAndSiblingOpenersThroughCorpus: every repro input of
// TestExtensionAndFooterIsolation with its `<gno-form path="x">` and
// `<gno-columns>` opener swapped for each structural opener the regex names
// but the corpus never feeds, plus `<gno-details>`, through all four helpers.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	b6ChromePrefix = "# Trusted Realm Title\n\n"
	b6ChromeFooter = "\n\n---\n**Trusted Realm Footer** [home]\n\n[home]: https://realm.example/home\n"
)

// The regex TestExtensionAndFooterIsolation uses, verbatim.
var b6LiveExtension = regexp.MustCompile(`class="gno-(?:form|columns|column|frame|foreign|button|alert)[" ]`)

// The same regex with details added.
var b6LiveExtensionDetails = regexp.MustCompile(`class="gno-(?:form|columns|column|frame|foreign|button|alert|details)[" ]`)

func TestB6AssertionsBlindToDetails(t *testing.T) {
	// Simulate a sanitizer regression: the user's opener reaches the page raw.
	composed := b6ChromePrefix + "<gno-details summary=\"x\">\n" + b6ChromeFooter
	html := renderMarkdown(t, composed)
	t.Logf("rendered: %s", html)

	// The four assertions of TestExtensionAndFooterIsolation all pass...
	require.False(t, b6LiveExtension.MatchString(html))
	require.NotContains(t, html, `href="https://evil.example`)
	require.Contains(t, html, `href="https://realm.example/home"`)
	require.Contains(t, html, "<strong>Trusted Realm Footer</strong>")
	// ...while the footer sits inside a live <details> the user opened.
	require.True(t, b6LiveExtensionDetails.MatchString(html), "details did not render live")
	require.Contains(t, html, `<details class="gno-details">`)
}

func TestB6DetailsAndSiblingOpenersThroughCorpus(t *testing.T) {
	inputs := []string{
		" ~~~\n~~~\n<gno-columns>",
		" ```\nx\n```\n<gno-columns>",
		"- a\n  ```\n<gno-form path=\"x\">\n```\n",
		"- a\n\n  ```\n<gno-form path=\"x\">\n```\n",
		"- a\nb\n  ```\n<gno-columns>\n```\n",
		"- a\n  ```\n```\n<gno-columns>\n",
		"para\n2. item\n   ```\n   x\n<gno-columns>\n",
		"- x\n[foo]:\n  ```\n<gno-form path=\"x\">\n",
		"- x\n  [foo]: /u\n```\n<gno-form path=\"x\">\n```\n",
		"-\n  ```\n<gno-form path=\"x\">\n```\n",
		"```\n```\n*\n  ```\n<gno-form path=\"x\">\n```\n",
		"- ```\n  ```\n  <gno-form path=\"x\">\n",
		"- a\n  ```\n     ```\n  <gno-form path=\"x\">\n",
		"<div>\n```\n\n<gno-form path=\"x\">\n",
		"1. a\n   ```\n\t<gno-form path=\"x\">\n",
		// Container-prefixed openers, the shape the gno-icon allowlist reads.
		"- <gno-form path=\"x\">\n",
		"> <gno-form path=\"x\">\n",
		"- > <gno-form path=\"x\">\n",
	}
	openers := []string{
		`<gno-details summary="x">`,
		`<gno-frame>`,
		`<gno-button href="/r/x">x</gno-button>`,
		`<GNO-DETAILS summary="x">`,
	}
	slipped := 0
	for _, helper := range []string{"Block", "BlockRich", "Blockquote", "BlockquoteRich"} {
		sc := sanitizeCase{Func: helper}
		for _, in := range inputs {
			for _, op := range openers {
				input := strings.ReplaceAll(strings.ReplaceAll(in, `<gno-form path="x">`, op), "<gno-columns>", op)
				sanitized := applySanitize(t, sc, input)
				html := renderMarkdown(t, b6ChromePrefix+sanitized+b6ChromeFooter)
				if b6LiveExtensionDetails.MatchString(html) || !strings.Contains(html, "<strong>Trusted Realm Footer</strong>") {
					slipped++
					t.Errorf("SLIP helper=%s input=%q\nsanitized=%q\nrendered=%s", helper, input, sanitized, html)
				}
			}
		}
	}
	t.Logf("slipped=%d of %d", slipped, 4*len(inputs)*len(openers))
}
