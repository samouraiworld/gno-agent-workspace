// A pointy link destination that starts a quote's continuation line:
//
//	> [home](
//	> <gno-a b>)
//
// The bracket walker keeps `[home](<gno-a b>)` as an inline link (CM §6.3
// allows one line ending before the destination). escapeGnoExtDelimiters then
// runs in the line loop, after the walker, sees `<gno-` at block start after
// the `>` container marker, and writes `\<`: the destination breaks, and
// `[home]` becomes a shortcut reference that binds to the realm's own
// `[home]:` definition. The `<gno-button` escape was moved before the walker
// for this exact shape (`escapeBlockHazardsImpl`, `[a](<gno-button x>)`); the
// generic `<gno-…>` escape, which this PR extends past `>` and list markers,
// was not.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6304/head && git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_pointy_dest_quote_test.go
//	go test ./gno.land/pkg/gnoweb/markdown -run TestPointyDestOnQuoteContinuation -v -count=1
//
// Run the same at the merge base d25b308c1a22ec04f429a2826e681249126320cd to
// see the line-start rows already bind there (pre-existing) and the `>` rows
// render the user's own link.
package markdown

import (
	"fmt"
	"strings"
	"testing"
)

func TestPointyDestOnQuoteContinuation(t *testing.T) {
	const (
		chromePrefix = "# Trusted Realm Title\n\n"
		chromeFooter = "\n\n---\n**Trusted Realm Footer** [home]\n\n[home]: https://realm.example/home\n"
		realmHref    = `href="https://realm.example/home"`
	)
	inputs := []string{
		"> [home](\n> <gno-a b>)\n",      // quote continuation: newly escaped by this PR
		"> [home](\n><gno-a b>)\n",       // same, no space after `>`
		"[home](\n<gno-a b>)\n",          // column 0: escaped on master too
		"- [home](\n  <gno-a b>)\n",      // list continuation, 2-space indent: escaped on master too
		"> [home](<gno-a b>)\n",          // control: one line, mid-line tag, never escaped
	}
	for _, helper := range []string{"Block", "BlockRich", "Blockquote", "BlockquoteRich"} {
		for _, in := range inputs {
			sanitized := applySanitize(t, sanitizeCase{Func: helper}, in)
			html := renderMarkdown(t, chromePrefix+sanitized+chromeFooter)
			// The footer's own [home] accounts for one realm link; a second
			// one is the user's text bound to the realm's definition.
			bound := strings.Count(html, realmHref) > 1
			own := strings.Contains(html, "gno-a%20b")
			// Expected after a fix: bound=false, own=true wherever the
			// one-line control renders own=true.
			fmt.Printf("%-14s %-28q bound-to-realm=%-5v own-dest=%-5v sanitized=%q\n", helper, in, bound, own, sanitized)
		}
	}
}
