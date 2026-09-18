#!/usr/bin/env bash
# judge-233 — REFUTES candidate #233's rewrite. plan.go:261 `plan.Realms = realms`
# overwrites whatever the Plan literal set, so `Realms: []string{}` in the literal
# cannot carry the "realms": [] guarantee: the one-line sortedKeys still turns the
# same four TestBuildPlan subtests red.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>
#
# Measured at ecf7af0 with go1.25.9. Baseline: "ok github.com/gnolang/gno/misc/gnopreview".
set -u
cd misc/gnopreview

echo "== A1: the candidate's fix, whole: literal init + one-line sortedKeys =="
python3 - <<'PY'
import pathlib
p=pathlib.Path("plan.go"); t=p.read_text()
old='\tplan := &Plan{ChangedFiles: map[string][]string{}}\n'
new='\tplan := &Plan{ChangedFiles: map[string][]string{}, Realms: []string{}, Dirs: []string{}}\n'
assert t.count(old)==1; t=t.replace(old,new)
old2='''func sortedKeys(m map[string]bool) []string {
\tout := make([]string, 0, len(m))
\tfor k := range m {
\t\tout = append(out, k)
\t}
\tsort.Strings(out)
\treturn out
}'''
assert t.count(old2)==1
t=t.replace(old2,'func sortedKeys(m map[string]bool) []string {\n\treturn slices.Sorted(maps.Keys(m))\n}')
t=t.replace('\t"io/fs"\n','\t"io/fs"\n\t"maps"\n',1)
p.write_text(t)
PY
gofmt -w plan.go
go test ./... 2>&1 | grep -E 'FAIL|^ok|plan_test.go:'
git checkout -- .
# Observed: 345 -> 341 lines and
#   --- FAIL: TestBuildPlan/nothing_relevant                          plan_test.go:144: Realms = []; want []
#   --- FAIL: TestBuildPlan/gnoweb_alone                              plan_test.go:144: Realms = []; want []
#   --- FAIL: TestBuildPlan/quarantined_packages_never_appear         plan_test.go:144: Realms = []; want []
#   --- FAIL: TestBuildPlan/tests_and_filetests_do_not_trigger_a_preview  plan_test.go:144: Realms = []; want []

echo "== A2: the literal's Realms value is dead — a sentinel never reaches the test =="
python3 - <<'PY'
import pathlib
p=pathlib.Path("plan.go"); t=p.read_text()
old='\tplan := &Plan{ChangedFiles: map[string][]string{}}\n'
new='\tplan := &Plan{ChangedFiles: map[string][]string{}, Realms: []string{"SENTINEL"}, Dirs: []string{"SENTINEL"}}\n'
assert t.count(old)==1; p.write_text(t.replace(old,new))
PY
go test -run 'TestBuildPlan/nothing_relevant' ./... 2>&1 | grep -E 'FAIL|^ok'
git checkout -- .
# Observed: ok  github.com/gnolang/gno/misc/gnopreview  0.003s
# The sentinel is discarded at plan.go:261, `plan.Realms = realms`.

echo "== A3: the only plan.json consumer is indifferent to null vs [] =="
# .github/workflows/pr-preview.yml:79
echo '{"gnoweb":false,"realms":null}' | jq -r 'if .gnoweb or (.realms | length > 0) then "yes" else "no" end'
echo '{"gnoweb":false,"realms":[]}'   | jq -r 'if .gnoweb or (.realms | length > 0) then "yes" else "no" end'
# Observed: no / no
