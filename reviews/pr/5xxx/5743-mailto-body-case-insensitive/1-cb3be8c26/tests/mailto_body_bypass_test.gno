// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5743 -R gnolang/gno
cat > examples/gno.land/p/nt/markdown/sanitize/v0/zz_bypass_test.gno <<'EOF'
package sanitize

import "testing"

func TestMailtoPercentEncodedBodyBypass(t *testing.T) {
	got := URL("mailto:a@b.com?%62ody=phish")
	if got != "" {
		t.Errorf("BYPASS: percent-encoded body= survived, got %q", got)
	}
}
EOF
gno test -v -run TestMailtoPercentEncodedBodyBypass ./examples/gno.land/p/nt/markdown/sanitize/v0/
rm examples/gno.land/p/nt/markdown/sanitize/v0/zz_bypass_test.gno
*/

// linkSchemeAllowed rejects ?body= / &body= via a lowercased substring match.
// It does not percent-decode, and PercentEncodeURL preserves valid %XX escapes,
// so mailto:a@b.com?%62ody=phish passes the filter and renders an href that the
// browser decodes to body=phish at click time. The PR's case-insensitivity fix
// does not close this hole; @thehowl flagged it in review.
//
// Observed: URL("mailto:a@b.com?%62ody=phish") returns the input unchanged
// ("mailto:a@b.com?%62ody=phish") instead of "". Test FAILS, proving the bypass.
// Flip: once the filter percent-decodes the query before matching, got becomes ""
// and the assertion passes.

package sanitize

import "testing"

func TestMailtoPercentEncodedBodyBypass(t *testing.T) {
	got := URL("mailto:a@b.com?%62ody=phish")
	if got != "" {
		t.Errorf("BYPASS: percent-encoded body= survived, got %q", got)
	}
}
