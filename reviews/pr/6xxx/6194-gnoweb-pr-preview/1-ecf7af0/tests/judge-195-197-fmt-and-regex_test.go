// Asserts three spellings in misc/gnopreview behave as the reviewed head writes
// them: fmt.Errorf %w on a nil error, headRe against <header>, cssRe dropping
// the quotes it matched. Measured at ecf7af0f2 with go1.25.9; all three report
// the defective output below, so the assertions are written against the fix.
//
// Run: from a gno checkout:
//   gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f2
//   cp <this file> misc/gnopreview/judge_regex_test.go
//   cd misc/gnopreview && go test -run 'TestJudge' -v ./...
//   rm judge_regex_test.go

package main

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// crawl.go:599 wraps the value read off the died channel, which main.go:202
// fills with cmd.Wait() — nil when gnodev exits 0.
func TestJudgeWrapNilExit(t *testing.T) {
	var exit error // cmd.Wait() on a clean exit
	got := fmt.Errorf("gnodev exited before serving %s — see the gnodev log: %w", "/r/x/y", exit).Error()
	if strings.Contains(got, "%!w(<nil>)") {
		t.Errorf("IS:     %s", got) // bug — the diagnostic prints a verb error
	}
	// SHOULD: a clean exit reports "exited with status 0", not %!w(<nil>).
}

// crawl.go:38. setNoindex falls back to headRe when a page carries no robots
// meta; gnoweb ships <header class="b-header"> on every layout.
func TestJudgeHeadReMatchesHeader(t *testing.T) {
	headRe := regexp.MustCompile(`(?i)<head[^>]*>`) // crawl.go:38, as written
	body := `<header class="b-header">nav</header><p>fragment</p>`
	if m := headRe.FindString(body); m != "" {
		t.Errorf("IS:     headRe matched %q in a page with no <head>", m) // bug
	}
	// SHOULD: `(?i)<head[\s>]` matches the real tag and not <header ...>.
	strict := regexp.MustCompile(`(?i)<head[\s>]`)
	if strict.FindString(body) != "" {
		t.Error("strict head regex matched <header>")
	}
}

// crawl.go:41 and 359-362: the regex captures an optional quote, the
// replacement never re-emits one.
func TestJudgeCSSReDropsQuotes(t *testing.T) {
	cssRe := regexp.MustCompile(`url\(\s*"?(/public/[^)"']*)"?\s*\)`) // crawl.go:41
	in := `background:url("/public/fonts/My Font.woff2")`
	out := cssRe.ReplaceAllStringFunc(in, func(m string) string { // crawl.go:359-362
		sub := cssRe.FindStringSubmatch(m)
		return "url(" + "../" + strings.TrimPrefix(sub[1], "/") + ")"
	})
	if !strings.Contains(out, `"`) {
		t.Errorf("IS:     %s", out) // bug — unquoted url token carrying a space
	}
	// SHOULD: url("../public/fonts/My Font.woff2") — the quoting is preserved.
}
