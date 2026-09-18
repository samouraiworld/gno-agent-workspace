#!/usr/bin/env bash
# b10-refactor-r2: the two shortenings of misc/gnopreview, applied and run.
#
# Repro from a plain clone:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>            # run from the repo root, go1.25.9 on PATH
#
# Rewrite 1  plan.go:212  the gnowebPaths loop is hasPrefixAny, already in the
#            same file and called 30 lines below   345 -> 343 lines, green.
# Rewrite 2  plan.go:335  contains() renames slices.Contains for its three call
#            sites (plan.go, comment.go, main.go)  880 -> 878 lines, green.
# Rejected   sortedKeys() -> slices.Sorted(maps.Keys(m)) is 8 lines to 1 but
#            returns nil for an empty map, flipping plan JSON "realms":[] to
#            null; TestBuildPlan catches it, unreadably: `Realms = []; want []`.
set -eu
cd misc/gnopreview
before=$(cat plan.go comment.go main.go | wc -l)
go test ./... >/dev/null && echo "baseline green, $before lines"

python3 - <<'PY'
s = open('plan.go').read()
s = s.replace('''		for _, gw := range gnowebPaths {
			if strings.HasPrefix(f, gw) {
				plan.Gnoweb = true
			}
		}
''', '''		if hasPrefixAny(f, gnowebPaths) {
			plan.Gnoweb = true
		}
''')
s = s.replace('''
func contains(s []string, v string) bool {
	return slices.Contains(s, v)
}
''', '')
s = s.replace('!contains(plan.Realms, r)', '!slices.Contains(plan.Realms, r)')
open('plan.go', 'w').write(s)

c = open('comment.go').read()
c = c.replace('contains(gnowebSeedRealms, pkgPath)', 'slices.Contains(gnowebSeedRealms, pkgPath)')
c = c.replace('\t"path"\n', '\t"path"\n\t"slices"\n')
open('comment.go', 'w').write(c)

m = open('main.go').read()
m = m.replace('!contains(plan.ChangedRealms, r)', '!slices.Contains(plan.ChangedRealms, r)')
m = m.replace('\t"os/exec"\n', '\t"os/exec"\n\t"slices"\n', 1)
open('main.go', 'w').write(m)
PY
gofmt -w plan.go comment.go main.go
go vet ./... && go test ./... && echo "rewritten green, $(cat plan.go comment.go main.go | wc -l) lines"
git checkout -- plan.go comment.go main.go
