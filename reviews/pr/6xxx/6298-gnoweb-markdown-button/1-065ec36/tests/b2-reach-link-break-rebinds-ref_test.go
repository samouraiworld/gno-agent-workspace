package markdown

// Repro for gnolang/gno#6298 at 065ec369b (reach finder, round 1).
//
// From a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6298/head && git checkout --detach 065ec369b321c207815f82b0c1dbf2b0f9f10b01
//   cp <this file> gno.land/pkg/gnoweb/markdown/zz_b2reach_test.go
//   go test ./gno.land/pkg/gnoweb/markdown -run 'TestB2Reach' -v -count=1
//
// It drives the real Gno sanitize package through the VM (callSanitizeGno,
// the golden harness's own helper) and renders with gnoweb's goldmark stack
// (renderMarkdown), appending realm chrome after the sanitized user content.

import (
	"strings"
	"testing"
)

const realmChrome = "\n\n[evil]: https://realm.example/home\n\n[^evil]: realm footnote\n"

// A pointy link destination containing a space is a link to the bracket
// walker (scanLinkURL) and to goldmark, so the walker leaves its brackets
// alone; escapeGnoButtonTags then inserts `\` before `<gno-button`, which
// turns `(<...>)` into a non-pointy destination that may not contain a space,
// so the link no longer parses and its bare brackets bind to the realm's
// reference definitions.
func TestB2ReachLinkBreakRebindsRealmRef(t *testing.T) {
	for _, fn := range []string{"Block", "BlockRich"} {
		for _, in := range []string{
			"[evil](<gno-button x>)",
			"[a](<gno-button [evil]>)",
			"![a](<gno-button [evil]>)",
			"[a](<gno-button [^evil]>)",
		} {
			out := callSanitizeGno(t, fn, in, "", "")
			html := renderMarkdown(t, out+realmChrome)
			raw := renderMarkdown(t, "\n\n"+in+"\n\n"+realmChrome)
			bound := strings.Contains(html, `href="https://realm.example/home"`) || strings.Contains(html, `class="footnote-ref"`)
			rawBound := strings.Contains(raw, `href="https://realm.example/home"`) || strings.Contains(raw, `class="footnote-ref"`)
			twice := callSanitizeGno(t, fn, out, "", "")
			t.Logf("%s(%q)\n  out=%q\n  bound-to-realm-ref=%v (unsanitized input bound=%v)\n  idempotent=%v twice=%q\n  html=%s", fn, in, out, bound, rawBound, twice == out, twice, strings.TrimSpace(html))
		}
	}
}
