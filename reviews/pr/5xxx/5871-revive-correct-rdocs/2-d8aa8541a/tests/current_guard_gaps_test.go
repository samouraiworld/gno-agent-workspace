/* Run: from a gno checkout:
gh pr checkout 5871 -R gnolang/gno && git checkout d8aa8541a
curl -fsSL -o misc/audit-pattern-harness/internal/auditpattern/current_guard_gaps_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5871-revive-correct-rdocs/2-d8aa8541a/tests/current_guard_gaps_test.go
(cd misc/audit-pattern-harness && go test ./internal/auditpattern -run TestCurrentGuard)
rm misc/audit-pattern-harness/internal/auditpattern/current_guard_gaps_test.go
*/

// Both tests assert what expected/current-guard.yaml claims the rule detects: a
// secondary realm parameter read before its own IsCurrent(). Both fail at
// d8aa8541a.

package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

// scanSource runs current_guard over one .gno file and returns the hit count.
func scanSource(t *testing.T, src string) int {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.gno"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err := RunRule("current_guard", dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(hits)
}

// TestCurrentGuardScansFuncLiterals pairs the same unguarded read written as a
// top-level func and as a func literal. The literal is the shape
// p/demo/tokens/grc20/tellers.gno uses for its caller resolver.
func TestCurrentGuardScansFuncLiterals(t *testing.T) {
	const decl = "package x\n\n" +
		"func accountFn(_ int, rlm realm) address {\n" +
		"\treturn rlm.Previous().Address()\n" +
		"}\n"
	const lit = "package x\n\n" +
		"func teller() *fnTeller {\n" +
		"\treturn &fnTeller{\n" +
		"\t\taccountFn: func(_ int, rlm realm) address {\n" +
		"\t\t\treturn rlm.Previous().Address()\n" +
		"\t\t},\n" +
		"\t}\n" +
		"}\n"
	if got := scanSource(t, decl); got != 1 {
		t.Fatalf("declared func: expected 1 hit, got %d", got)
	}
	if got := scanSource(t, lit); got != 1 {
		t.Fatalf("func literal: expected 1 hit, got %d", got)
	}
}

// TestCurrentGuardCoversIdentityAccessors walks the realm methods that read an
// identity off the value or mint a capability from it.
func TestCurrentGuardCoversIdentityAccessors(t *testing.T) {
	for _, accessor := range []string{
		"Previous()", "Address()", "PkgPath()", "String()",
		"Sub(\"treasury\")", "Subpath()",
		"IsUser()", "IsUserCall()", "IsUserRun()", "IsCode()",
	} {
		src := "package x\n\nfunc f(_ int, rlm realm) {\n\t_ = rlm." + accessor + "\n}\n"
		if got := scanSource(t, src); got != 1 {
			t.Errorf("rlm.%s: expected 1 hit, got %d", accessor, got)
		}
	}
}
