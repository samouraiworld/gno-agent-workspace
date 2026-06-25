/* Run: from a gno checkout:
gh pr checkout 5835 -R gnolang/gno && git checkout 34ac1e7cd
curl -fsSL -o misc/audit-pattern-harness/internal/auditpattern/zz_brace_counting_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5835-audit-pattern-harness/2-34ac1e7cd/tests/brace_counting_test.go
(cd misc/audit-pattern-harness && go test -run TestCurrentGuard_BraceIn ./internal/auditpattern/)
rm misc/audit-pattern-harness/internal/auditpattern/zz_brace_counting_test.go

The brace-depth scanners count { and } inside string literals and comments, so a "}"
in a string ends the function scope early. At 34ac1e7cd the two NoFalsePositive tests
fail: a correctly guarded cur.Previous() is flagged. Stripping string and line-comment
spans before counting makes them pass, while StillCatchesRealBug keeps the genuinely
unguarded case flagged so the fix does not over-blind the rule.
*/
package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

func writeGno(t *testing.T, src string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "a.gno"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

// A correctly guarded function: cur.IsCurrent() before cur.Previous(). A "}"
// inside a string literal must not be read as the function's closing brace.
func TestCurrentGuard_BraceInString_NoFalsePositive(t *testing.T) {
	hits, err := RunRule("current_guard", writeGno(t, `package x

func F(cur realm) {
	if !cur.IsCurrent() {
		panic("no")
	}
	msg := "}"
	_ = cur.Previous()
	_ = msg
}
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("guarded function flagged (brace-in-string miscounted): %+v", hits)
	}
}

// Same, for a brace inside a line comment.
func TestCurrentGuard_BraceInComment_NoFalsePositive(t *testing.T) {
	hits, _ := RunRule("current_guard", writeGno(t, `package x

func F(cur realm) {
	if !cur.IsCurrent() { panic("no") }
	// closing brace in a comment: }
	_ = cur.Previous()
}
`))
	if len(hits) != 0 {
		t.Fatalf("guarded function flagged (brace-in-comment miscounted): %+v", hits)
	}
}

// The fix must not blind the rule: an unguarded cur.Previous() after a
// string-literal brace must still be flagged.
func TestCurrentGuard_BraceInString_StillCatchesRealBug(t *testing.T) {
	hits, _ := RunRule("current_guard", writeGno(t, `package x

func F(cur realm) {
	msg := "}"
	_ = cur.Previous()
	_ = msg
}
`))
	if len(hits) != 1 {
		t.Fatalf("unguarded Previous() not flagged, want 1 hit, got %+v", hits)
	}
}
