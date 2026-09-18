#!/usr/bin/env bash
# judge-250 — CONFIRMS candidate #250's seam. Extracting the derivation out of
# BuildPlan compiles, keeps the package green, and lets a case be a Pkg literal
# instead of a directory tree. Cost: plan.go 345 -> 350 lines (+5).
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>
#
# Measured at ecf7af0 with go1.25.9: go vet silent, "ok ... 0.004s",
# TestZZPlanFromLiteral PASS. Baseline TestBuildPlan wall time: 0.005s for all
# ten subtests, so the "ten tree walks" the candidate cites cost ~4ms in total.
set -eu
cd misc/gnopreview

python3 - <<'PY'
import pathlib
p=pathlib.Path("plan.go"); t=p.read_text()
old='''func BuildPlan(root string, changed []string, maxRealms int) (*Plan, error) {
\tpkgs, err := LoadPkgs(root)
\tif err != nil {
\t\treturn nil, err
\t}
\t// dir -> package'''
new='''func BuildPlan(root string, changed []string, maxRealms int) (*Plan, error) {
\tpkgs, err := LoadPkgs(root)
\tif err != nil {
\t\treturn nil, err
\t}
\treturn planFrom(pkgs, changed, maxRealms), nil
}

// planFrom derives the plan from an already-loaded package graph.
func planFrom(pkgs map[string]*Pkg, changed []string, maxRealms int) *Plan {
\t// dir -> package'''
assert t.count(old)==1; t=t.replace(old,new)
t=t.replace('\t\t\tplan.ChangedFiles[r] = sortedKeys(f)\n\t\t}\n\t}\n\treturn plan, nil\n}',
            '\t\t\tplan.ChangedFiles[r] = sortedKeys(f)\n\t\t}\n\t}\n\treturn plan\n}')
p.write_text(t)
PY
gofmt -w plan.go && go vet ./... && wc -l plan.go

cat > zz_judge250_test.go <<'EOF'
package main

import (
	"reflect"
	"testing"
)

// The "transitive dependency pulls both dependents" case as a package-graph
// literal, plus a cap case, with no files on disk.
func TestZZPlanFromLiteral(t *testing.T) {
	pkgs := map[string]*Pkg{
		"gno.land/p/x/base/v0": {Path: "gno.land/p/x/base/v0", Dir: "examples/gno.land/p/x/base/v0"},
		"gno.land/p/x/mid/v0":  {Path: "gno.land/p/x/mid/v0", Dir: "examples/gno.land/p/x/mid/v0", Imports: []string{"gno.land/p/x/base/v0"}},
		"gno.land/r/x/leaf":    {Path: "gno.land/r/x/leaf", Dir: "examples/gno.land/r/x/leaf", Realm: true, Imports: []string{"gno.land/p/x/mid/v0"}},
		"gno.land/r/x/other":   {Path: "gno.land/r/x/other", Dir: "examples/gno.land/r/x/other", Realm: true, Imports: []string{"gno.land/p/x/base/v0"}},
	}
	got := planFrom(pkgs, []string{"examples/gno.land/p/x/base/v0/lib.gno"}, defaultMaxRealms)
	want := []string{"gno.land/r/x/leaf", "gno.land/r/x/other"}
	if !reflect.DeepEqual(got.Realms, want) {
		t.Errorf("Realms = %v; want %v", got.Realms, want)
	}
	got2 := planFrom(pkgs, []string{"examples/gno.land/p/x/base/v0/lib.gno"}, 1)
	if len(got2.Realms) != 1 || got2.Dropped != 1 {
		t.Errorf("cap=1: Realms=%v Dropped=%d", got2.Realms, got2.Dropped)
	}
}
EOF
go test ./... 2>&1 | grep -E 'FAIL|^ok'
go test -run TestZZPlanFromLiteral -v ./... 2>&1 | grep -E 'FAIL|PASS|^ok'
rm -f zz_judge250_test.go && git checkout -- .
