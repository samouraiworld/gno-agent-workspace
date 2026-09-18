#!/usr/bin/env bash
# Shows that nothing at head ecf7af0f29abe4737a52803d672bc5a33c17cc60 of
# gnolang/gno#6194 executes misc/gnopreview/main.go past flag parsing:
# no test in the package references a main.go symbol, and pr-preview.yml
# gates every step after "Plan the preview" on a plan that this PR own
# diff makes empty, so render/startGnodev/renderBase/waitReady never run.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash b3-reach-r2-ci-never-renders.sh
set -euo pipefail
BASE=bbd9b2ffe87c9a295ccfd6dfb45224bb02c82d8a
HEAD=ecf7af0f29abe4737a52803d672bc5a33c17cc60

echo "== 1. test references to main.go symbols =="
grep -n "render(\|startGnodev\|renderBase\|findRoot\|readLines\|fileBudget(\|envOr(" \
  misc/gnopreview/crawl_test.go misc/gnopreview/plan_test.go || echo "NONE"
# observed: NONE

echo "== 2. the plan gate pr-preview.yml applies to this PR own diff =="
tmp=$(mktemp -d)
(cd misc/gnopreview && go build -o "$tmp/gnopreview" .)
git diff --name-only "$BASE" "$HEAD" > "$tmp/changed.txt"
"$tmp/gnopreview" plan -changed "$tmp/changed.txt" -root "$PWD" | tee "$tmp/plan.json"
# observed: {"gnoweb":false,"changed_realms":null,"changed_pkgs":null,"realms":[],"dropped":0,"dirs":null}
jq -r "if .gnoweb or (.realms | length > 0) then \"yes\" else \"no\" end" "$tmp/plan.json"
# observed: no  -> skip=true -> Build gnodev, Render the snapshot and
# upload-artifact are all skipped, so the preview is never rendered in CI.
