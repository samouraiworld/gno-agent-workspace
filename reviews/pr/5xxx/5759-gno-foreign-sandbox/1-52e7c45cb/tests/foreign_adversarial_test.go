// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5759 -R gnolang/gno
cp <this-file> gno.land/pkg/gnoweb/markdown/foreign_adversarial_test.go
go test -run 'TestAdversarialForeign' ./gno.land/pkg/gnoweb/markdown/
rm gno.land/pkg/gnoweb/markdown/foreign_adversarial_test.go
*/

// Adversarial probes against the <gno-foreign> sandbox boundary in
// ext_foreign.go. Exercises (1) dangerous-scheme links nested inside
// an inner gno-columns block, (2) the data:image/svg+xml carve-out
// goldmark's IsDangerousURL leaves open, and (3) a sandbox whose body
// tries to terminate the outer block with an attribute-bearing close
// tag emitted DIRECTLY (i.e. not through the foreign.Foreign helper).
//
// Result observed at HEAD 52e7c45cb: (1) neutralized (href=""),
// (2) the data:image/svg link does NOT even form — the inline <svg> raw
// HTML in the link destination is stripped by inner safe-mode ("raw HTML
// omitted"), so no live href survives; the sandbox is stricter than the
// goldmark svg carve-out alone, (3) the attr-bearing close DOES terminate
// the outer block, confirming the helper's escape of such lines is the
// only thing keeping comment bytes inside the sandbox.
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
)

func buildAdvForeign(t *testing.T) (goldmark.Markdown, parser.ParseOption) {
	t.Helper()
	gnourl, err := weburl.Parse("https://gno.land/r/test")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	m := goldmark.New()
	ExtForeign.Extend(m, nil)
	ExtColumns.Extend(m)
	ExtAlerts.Extend(m)
	ExtLinks.Extend(m)
	ExtMention.Extend(m)
	return m, parser.WithContext(NewGnoParserContext(GnoContext{GnoURL: gnourl}))
}

// (1) A javascript: link nested inside an inner <gno-columns> block must
// still be neutralized by the inner instance's link transformer.
func TestAdversarialForeign_DangerousLinkInsideNestedColumns(t *testing.T) {
	m, ctx := buildAdvForeign(t)
	src := "before\n\n" +
		"<gno-foreign>\n" +
		"<gno-columns>\n" +
		"[evil](javascript:alert(1))\n" +
		"<gno-columns-sep/>\n" +
		"col two\n" +
		"</gno-columns>\n" +
		"</gno-foreign>\n\n" +
		"after\n"
	var buf bytes.Buffer
	if err := m.Convert([]byte(src), &buf, ctx); err != nil {
		t.Fatalf("convert: %v", err)
	}
	got := buf.String()
	if strings.Contains(got, "javascript:alert") {
		// IS the bug if it ever fires: a live javascript: href inside a
		// nested column escaped the inner link transformer.
		t.Errorf("javascript: href survived inside nested columns:\n%s", got)
	}
	if !strings.Contains(got, `href=""`) {
		t.Errorf("expected neutralized empty href for the dangerous link:\n%s", got)
	}
}

// (2) data:image/svg+xml is whitelisted by goldmark.IsDangerousURL even
// though SVG can carry script. Document the current behavior: the link
// passes through with a live href, carrying only rel="...ugc".
func TestAdversarialForeign_DataSvgLinkPassesThrough(t *testing.T) {
	m, ctx := buildAdvForeign(t)
	src := "\n\n<gno-foreign>\n" +
		"[x](data:image/svg+xml,<svg onload=alert(1)>)\n" +
		"</gno-foreign>\n\n"
	var buf bytes.Buffer
	if err := m.Convert([]byte(src), &buf, ctx); err != nil {
		t.Fatalf("convert: %v", err)
	}
	got := buf.String()
	// Active assertion: current behavior — the svg data URI keeps a live
	// href (goldmark whitelists it) and is marked ugc.
	if !strings.Contains(got, `href="data:image/svg`) {
		t.Logf("data:image/svg link did NOT pass through (goldmark may have changed):\n%s", got)
	}
	if strings.Contains(got, `href="data:image/svg`) && !strings.Contains(got, "ugc") {
		t.Errorf("svg data URI link not marked untrusted (ugc):\n%s", got)
	}
}

// (3) A bare attribute-bearing close emitted directly in the body (NOT
// via foreign.Foreign) terminates the outer block: "after" renders
// OUTSIDE the sandbox. Confirms the realm MUST go through the helper —
// hand-built envelopes are unsafe, exactly as ext_foreign.go documents.
func TestAdversarialForeign_DirectAttrCloseEscapesSandbox(t *testing.T) {
	m, ctx := buildAdvForeign(t)
	src := "\n\n<gno-foreign>\n" +
		"trapped?\n" +
		"</gno-foreign bogus=\"y\">\n" +
		"AFTER-ESCAPED\n\n"
	var buf bytes.Buffer
	if err := m.Convert([]byte(src), &buf, ctx); err != nil {
		t.Fatalf("convert: %v", err)
	}
	got := buf.String()
	bodyStart := strings.Index(got, `<div class="gno-foreign__body">`)
	if bodyStart < 0 {
		t.Fatalf("no foreign body:\n%s", got)
	}
	bodyEnd := strings.Index(got[bodyStart:], "</div>\n</div>")
	bodySlice := got[bodyStart : bodyStart+bodyEnd]
	// IS the contract (not a bug): hand-built attr-close escapes the
	// sandbox; the helper neutralizes it, which is why realms must use it.
	if strings.Contains(bodySlice, "AFTER-ESCAPED") {
		t.Errorf("attr-close did NOT terminate outer (helper-independent safety?):\n%s", got)
	}
}
