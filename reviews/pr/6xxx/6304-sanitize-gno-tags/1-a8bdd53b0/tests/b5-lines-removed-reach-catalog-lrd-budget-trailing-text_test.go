// scanLRDTail at PR #6304 head (merge a8bdd53b0) accepts a link reference
// definition with text after its destination whenever the walker's budget
// runs out on the whitespace before that text.
//
// 1c377bf89 rejected such a line outright ("CM §4.7: only whitespace may
// follow a definition on its line. Deleting up to the text would also leave
// that text, a fence say, at a column the fence trackers never saw it at"),
// with an unbudgeted whitespace skip. Master budgeted that skip. The merge kept
// master's budgeted skip and gated the PR's check on its ok:
//
//	end, ok = skipSpace(s, end, 0, budget)
//	if ok && end < len(s) && s[end] != '\n' { return 0, false }
//
// so out of budget the line `[a]: u ```` is taken as a definition, `[a]: u` is
// deleted and ` ```` is left at line start, a fence closeFencesAtDedent (which
// runs before the walker in Rich mode) never saw. The target is the PR's own
// `lrd-text-after-destination` case; the prefix only spends the budget, with
// definitions whose paren title walks a "`\n" tail up to a blank line.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6304/head && git checkout a8bdd53b08e988227136f8b74f4cd6571956e5f4
//	cp <this file> gnovm/stdlibs/chain/markdown/zz_lrd_budget_test.go
//	go test ./gnovm/stdlibs/chain/markdown -run TestLRDTrailingTextOutOfBudget -v -count=1
//
// Fails at a8bdd53b0. Passes with the check made fail-closed in scanLRDTail:
//
//	if !ok || end < len(s) && s[end] != '\n' { return 0, false }
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

// renderPage wraps out the way sanitize.BlockRich returns it ("\n\n" on both
// sides) between realm chrome, and renders it with CommonMark goldmark.
func renderPage(t *testing.T, out string) string {
	t.Helper()
	page := "REALM-HEADER\n" + "\n\n" + out + "\n\n" + "REALM-FOOTER-LINK\n"
	var buf bytes.Buffer
	if err := goldmark.New().Convert([]byte(page), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// footerInCode reports whether the realm footer renders inside a code block.
func footerInCode(html string) bool {
	f := strings.Index(html, "REALM-FOOTER-LINK")
	open := strings.LastIndex(html[:f], "<pre><code")
	return open >= 0 && !strings.Contains(html[open:f], "</code></pre>")
}

func TestLRDTrailingTextOutOfBudget(t *testing.T) {
	const target = "[a]: u ```\n```\n- y\n  ```\n~~~\n" // lrd-text-after-destination

	// Within budget, the PR's own expectation holds: kept, escaped, footer safe.
	if got := EscapeBlockHazardsRich(target); !strings.Contains(got, "u ```") || footerInCode(renderPage(t, got)) {
		t.Fatalf("baseline changed: %q", got)
	}

	for k := 8; k <= 10; k++ {
		for tail := 0; tail <= 1500; tail++ {
			for pad := 0; pad < 8; pad++ {
				in := strings.Repeat("[a]: u\n(x\n", k) + strings.Repeat("`\n", tail) + "\n" +
					target + "\n" + strings.Repeat("z", pad) + "\n"
				out := EscapeBlockHazardsRich(in)
				if strings.Contains(out, "u ```") {
					continue // target line kept: the check ran
				}
				html := renderPage(t, out)
				// Current (a8bdd53b0): the target line is deleted and the
				// footer renders inside the ~~~ fence.
				t.Errorf("k=%d tail=%d pad=%d: `[a]: u ```` deleted; footer in code block: %v\noutput tail: %q",
					k, tail, pad, footerInCode(html), out[max(0, len(out)-60):])
				// Fixed: the target line is kept (escaped) and the footer is not code.
				// if strings.Contains(out, "u ```") && !footerInCode(html) { ok }
				return
			}
		}
	}
}
