/* Run: from a gno checkout:
gh pr checkout 5844 -R gnolang/gno && git checkout da956ea64
curl -fsSL -o misc/audit-pattern-harness/internal/auditpattern/repair_compile_gap_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5844-audit-repair-contracts/1-da956ea64/tests/repair_compile_gap_test.go
go test -v -run 'TestRepairContractAcceptsNonCompilingTarget' ./misc/audit-pattern-harness/internal/auditpattern/
rm misc/audit-pattern-harness/internal/auditpattern/repair_compile_gap_test.go
*/

// TestRepairContracts checks the to_fixture only via RunRule, a text scan, so a
// "repaired" fixture that produces zero pattern hits passes even when it does
// not compile. This writes a syntactically-broken but hit-free target and shows
// the repair invariant ("target fixture removes the pattern") stays green.
// When the harness compiles repair targets, this test fails at the setup step.
package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepairContractAcceptsNonCompilingTarget(t *testing.T) {
	root := filepath.Clean(filepath.Join(pkgDir, "..", ".."))
	rec, err := LoadRecord(filepath.Join(root, "expected", "current-guard.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	var to Fixture
	for _, f := range rec.Fixtures {
		if f.Name == rec.Repair.ToFixture {
			to = f
		}
	}
	target := filepath.Join(to.Path, "admin.gno")

	orig, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.WriteFile(target, orig, 0o644) })

	// Same exported func names, differs from the vulnerable fixture, zero
	// current_guard hits, but references an undeclared identifier: will not compile.
	broken := []byte(`package admin

var owner = address("g1qz8e0fz3y0pl9y4dq9d7c5dwnyu6qf04hs7z0a")

func TransferOwnership(cur realm, next address) {
	owner = thisDoesNotExist
}

func Owner() address {
	return owner
}
`)
	if err := os.WriteFile(target, broken, 0o644); err != nil {
		t.Fatal(err)
	}

	toHits, err := RunRule(rec.Rule, to.Path)
	if err != nil {
		t.Fatal(err)
	}
	// The repair invariant is "target has no hits". The broken target satisfies
	// it, so TestRepairContracts would pass here despite the target not compiling.
	if len(toHits) != 0 {
		t.Fatalf("expected the broken target to have zero pattern hits, got %+v", toHits)
	}
}
