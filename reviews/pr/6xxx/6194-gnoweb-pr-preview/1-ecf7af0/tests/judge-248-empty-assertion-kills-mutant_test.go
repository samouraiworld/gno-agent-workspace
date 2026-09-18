package main

// judge-248 — REFUTES candidate #248. plan_test.go:152-154 is the ONLY test of
// Plan.Empty(), which main.go:111 and comment.go:24 both branch on. Deleting it
// leaves a mutated Empty() undetected.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cd misc/gnopreview
//	# mutant: drop the Realms half of Empty()
//	sed -i 's|return !p.Gnoweb && len(p.Realms) == 0|return !p.Gnoweb|' plan.go
//	go test -run TestBuildPlan ./...              # RED, five subtests, plan_test.go:153
//	# now delete the three lines the candidate calls dead weight
//	python3 - <<'PY'
//	import pathlib; q=pathlib.Path("plan_test.go"); s=q.read_text()
//	q.write_text(s.replace('\t\t\tif got.Empty() != (tc.wantMode == "none") {\n\t\t\t\tt.Errorf("Empty = %v; want %v", got.Empty(), tc.wantMode == "none")\n\t\t\t}\n',''))
//	PY
//	go test -run TestBuildPlan ./...              # GREEN with the mutant still in
//	git checkout -- .
//
// Measured at ecf7af0 with go1.25.9:
//	lines kept    (278-line plan_test.go): FAIL, plan_test.go:153 "Empty = true; want false"
//	                                       one_realm, transitive_dependency_pulls_both_dependents,
//	                                       a_changed_test_realm_is_still_previewed,
//	                                       cap_keeps_the_changed_realm_and_reports_the_rest
//	lines deleted (275-line plan_test.go): ok  github.com/gnolang/gno/misc/gnopreview  0.014s
//
// The test below is the same mutation expressed in-process: it fails at this head
// (Empty() is correct) and is the assertion the deletion would remove.

import "testing"

// TestJudge248EmptyIsTheOnlyGuard pins the half of Empty() that only
// plan_test.go:152-154 reaches: a plan with realms and no gnoweb is not empty.
func TestJudge248EmptyIsTheOnlyGuard(t *testing.T) {
	t.Parallel()
	p := &Plan{Realms: []string{"gno.land/r/x/leaf"}}
	if p.Empty() {
		t.Errorf("Empty() = true for a plan with %d realms; comment.go:24 would emit no comment", len(p.Realms))
	}
	// The mutant `return !p.Gnoweb` fails exactly here, and nowhere else in the
	// package once the table assertion is gone.
}
