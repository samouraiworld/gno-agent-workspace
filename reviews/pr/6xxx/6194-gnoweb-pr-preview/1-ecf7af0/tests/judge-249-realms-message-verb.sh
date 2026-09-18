#!/usr/bin/env bash
# judge-249: plan_test.go:144 asserts Realms with reflect.DeepEqual but prints
# it with %v, so the nil-vs-empty difference DeepEqual exists to catch is the
# one difference the failure message cannot show. Probe: make sortedKeys return
# nil for an empty map, run the suite, read the four failure lines; then flip
# the verb to %#v and read them again. Fails at ecf7af0 with %v, names the
# difference with %#v. 1 line -> 1 line.
#
# Repro from a plain clone:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>          # run from the repo root, go1.25.9 on PATH
set -eu
cd misc/gnopreview
go test ./... >/dev/null && echo "baseline green"

# Probe: the empty-map case of sortedKeys returns nil instead of a zero-length
# slice, which is exactly what plan.Realms carries when nothing is previewed.
python3 - <<'PY'
s = open('plan.go').read()
s = s.replace('''func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}''', '''func sortedKeys(m map[string]bool) []string {
	return slices.Sorted(maps.Keys(m))
}''')
s = s.replace('\t"io/fs"\n', '\t"io/fs"\n\t"maps"\n')
open('plan.go', 'w').write(s)
PY
gofmt -w plan.go

echo "--- as shipped, %v: ---"
go test ./... 2>&1 | grep 'plan_test.go:144' || true

# The one-character fix: %v -> %#v on the Realms message alone.
sed -i 's|t.Errorf("Realms = %v; want %v", got.Realms, tc.wantRealms)|t.Errorf("Realms = %#v; want %#v", got.Realms, tc.wantRealms)|' plan_test.go

echo "--- with %#v: ---"
go test ./... 2>&1 | grep 'plan_test.go:144' || true

git checkout -- plan.go plan_test.go
